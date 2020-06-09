// mseednrt provides a RAM cache for near real time mSEED records that are stored on disk.
package mseednrt

import (
	"context"
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn_pb"
	"github.com/GeoNet/fdsn/internal/mseednrt/fs"
	"github.com/golang/groupcache"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Cache struct {
	dir    string
	list   *groupcache.Group
	record *groupcache.Group

	mu       sync.RWMutex // mu protects channels and err
	channels []fs.NSLC
	err      error

	// buster and t are used to limit the time that List results are cached for.
	buster *int64
	t      <-chan time.Time
}

// InitCache returns a Cache ready for use.
// recordSize is the max size of the RAM cache for mSEED records.
// listSize is the max size of the RAM cache for results from listing records.
// Listings will be cached for a max duration d.
func InitCache(name string, recordSize int64, listSize int64, d time.Duration, dir string) Cache {
	c := Cache{
		dir:    dir,
		buster: new(int64),
		t:      time.NewTicker(d).C,
	}

	c.list = groupcache.NewGroup(name+"list", listSize, groupcache.GetterFunc(c.listGetter))
	c.record = groupcache.NewGroup(name+"record", recordSize, groupcache.GetterFunc(c.recordGetter))

	go func() {
		for range c.t {
			atomic.AddInt64(c.buster, 1)
		}
	}()

	c.channels, c.err = fs.ListChannels(dir)

	// update the channels listing once per minute.
	// could cause pauses using List if the storage is slow
	go func() {
		for range time.NewTicker(time.Minute).C {
			c.mu.Lock()
			c.channels, c.err = fs.ListChannels(dir)
			c.mu.Unlock()
		}
	}()

	return c
}

// Get writes any mSEED data that match the query to w.
//
// Returns the number of bytes written to w
func (c *Cache) Get(n fs.NSLC, start, end time.Time, w io.Writer) (int, error) {
	var idx fdsn_pb.Mseed

	err := c.list.Get(nil, toListKey(n, atomic.LoadInt64(c.buster)), groupcache.ProtoSink(&idx))
	if err != nil {
		return 0, err
	}

	s := start.UnixNano()
	e := end.UnixNano()

	var b []byte
	var num int
	var tot int

	for _, v := range idx.GetRecords() {
		if v.GetStart() >= s && v.GetEnd() <= e {
			c.record.Get(nil, v.Path, groupcache.AllocatingByteSliceSink(&b))
			if err != nil {
				// ignore this error - old record files could be deleted while they are
				// still in a list result.
				continue
			}

			num, err = w.Write(b)
			if err != nil {
				return 0, err
			}

			tot += num
		}
	}

	return tot, nil
}

// List returns a list of NSLC available in the cache that match n.
// Members of n are matched as regexp.
func (c *Cache) List(n fs.NSLC) ([]fs.NSLC, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.err != nil {
		return []fs.NSLC{}, c.err
	}

	var chans []fs.NSLC
	var net, sta, loc, cha bool
	var err error

	for _, v := range c.channels {
		net, err = regexp.MatchString(n.Network, v.Network)
		if err != nil {
			return []fs.NSLC{}, err
		}

		sta, err = regexp.MatchString(n.Station, v.Station)
		if err != nil {
			return []fs.NSLC{}, err
		}

		loc, err = regexp.MatchString(n.Location, v.Location)
		if err != nil {
			return []fs.NSLC{}, err
		}

		cha, err = regexp.MatchString(n.Channel, v.Channel)
		if err != nil {
			return []fs.NSLC{}, err
		}

		if net && sta && loc && cha {
			chans = append(chans, v)
		}
	}

	return chans, nil
}

func (c *Cache) recordGetter(ctx context.Context, key string, dest groupcache.Sink) error {
	b, err := ioutil.ReadFile(key)
	if err != nil {
		return err
	}

	return dest.SetBytes(b)
}

func (c *Cache) listGetter(ctx context.Context, key string, dest groupcache.Sink) error {
	n, err := fromListKey(key)
	if err != nil {
		return err
	}

	l, err := n.ListRecords(c.dir)
	if err != nil {
		return err
	}

	var idx fdsn_pb.Mseed

	for _, v := range l {
		idx.Records = append(idx.Records, &fdsn_pb.Record{
			Path:  v.Path,
			Start: v.Start,
			End:   v.End,
		})
	}

	return dest.SetProto(&idx)
}

func toListKey(n fs.NSLC, b int64) string {
	return strings.Join([]string{n.Network, n.Station, n.Location, n.Channel, fmt.Sprintf("%d", b)}, "_")
}

func fromListKey(key string) (fs.NSLC, error) {
	p := strings.Split(key, "_")

	if len(p) != 5 {
		return fs.NSLC{}, fmt.Errorf("splitting key expected 5 parts got %d", len(p))
	}

	return fs.NSLC{Network: p[0], Station: p[1], Location: p[2], Channel: p[3]}, nil
}
