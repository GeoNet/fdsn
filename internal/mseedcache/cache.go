// Package mseedcache provides a RAM cache indexing mechanism for accelerating access to mSEED day files.
package mseedcache

import (
	"fmt"
	"github.com/GeoNet/fdsn/internal/fdsn_pb"
	"github.com/GeoNet/kit/mseed"
	"github.com/golang/groupcache"
	"io"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const dateFormat = "2006-01-02"

// the record length of the miniSEED records.  Constant for all GNS miniSEED files.
const recordLength int = 512
const recordLength64 int64 = 512

type Cache struct {
	getterFunc   GetterFunc
	listerFunc   ListerFunc
	modifiedFunc ModifiedFunc
	getRangeFunc GetRangeFunc
	index        *groupcache.Group
	list         *groupcache.Group

	// buster and t are used to limit the time that List results are cached for.
	buster *int64
	t      <-chan time.Time
}

// NSLC uniquely identifies a stream of data
type NSLC struct {
	Network, Station, Location, Channel string
}

type DayFile struct {
	NSLC
	Date time.Time // Date is a calendar date - year, month, day.
}

// A GetterFunc provides access to the mSEED day file referred to by d.
type GetterFunc func(d DayFile) (io.ReadCloser, error)

// A GetterRangeFunc provides access to the mSEED day file referred to be d between the byte range from-to (inclusive).
type GetRangeFunc func(d DayFile, from, to int64) (io.ReadCloser, error)

// ModifiedFunc returns the modification time for the mSEED file referred to by d.
type ModifiedFunc func(d DayFile) (time.Time, error)

// A ListerFunc provides a listing of NSLC that are available for a given date (year, month, day).
type ListerFunc func(date time.Time) ([]NSLC, error)

// InitCache returns a Cache ready for use.
// indexSize is the max size of the RAM cache for mSEED file indexes.
// listSize is the max size of the RAM cache for results from ListerFunc.  Listings will be cached for a max duration d.
func InitCache(name string, indexSize int64, listSize int64, d time.Duration, g GetterFunc, l ListerFunc, m ModifiedFunc, r GetRangeFunc) Cache {
	c := Cache{
		getterFunc:   g,
		listerFunc:   l,
		modifiedFunc: m,
		getRangeFunc: r,
		buster:       new(int64),
		t:            time.NewTicker(d).C,
	}

	c.index = groupcache.NewGroup(name+"index", indexSize, groupcache.GetterFunc(c.indexGetter))
	c.list = groupcache.NewGroup(name+"list", listSize, groupcache.GetterFunc(c.listGetter))

	go func() {
		for range c.t {
			atomic.AddInt64(c.buster, 1)
		}
	}()

	return c
}

// List returns a list of day files that would be needed to provide data for the given query.
// start and end are padded to allow for blocking of mSEED records at day boundaries.
// NSLC members are matched as regexp.
func (c *Cache) List(n NSLC, start, end time.Time) ([]DayFile, error) {
	start = start.Add(time.Minute * -1).Truncate(time.Hour * 24)
	end = end.Add(time.Minute)

	var d []DayFile

	var lst fdsn_pb.Listing

	for start.Before(end) {
		err := c.list.Get(nil, toListKey(start, atomic.LoadInt64(c.buster)), groupcache.ProtoSink(&lst))
		if err != nil {
			return []DayFile{}, err
		}

		var net, sta, loc, cha bool

		for _, v := range lst.GetChannels() {
			net, err = regexp.MatchString(n.Network, v.GetNetwork())
			if err != nil {
				return []DayFile{}, err
			}

			sta, err = regexp.MatchString(n.Station, v.GetStation())
			if err != nil {
				return []DayFile{}, err
			}

			loc, err = regexp.MatchString(n.Location, v.GetLocation())
			if err != nil {
				return []DayFile{}, err
			}

			cha, err = regexp.MatchString(n.Channel, v.GetChannel())
			if err != nil {
				return []DayFile{}, err
			}

			if net && sta && loc && cha {
				d = append(d, DayFile{
					Date: start,
					NSLC: NSLC{
						Network:  v.GetNetwork(),
						Station:  v.GetStation(),
						Location: v.GetLocation(),
						Channel:  v.GetChannel(),
					},
				})
			}
		}

		start = start.Add(time.Hour * 24)
	}

	return d, nil
}

// Get writes any mSEED data that match the query to w.
//
// An mSEED index file is cached to accelerate access to data.  The index is refreshed if the source mSEED file is modified.
//
// Returns the number of bytes written to w
func (c *Cache) Get(d DayFile, start, end time.Time, w io.Writer) (int64, error) {
	mod, err := c.modifiedFunc(d)
	if err != nil {
		return 0, err
	}

	var idx fdsn_pb.Mseed

	err = c.index.Get(nil, toIndexKey(d, mod), groupcache.ProtoSink(&idx))
	if err != nil {
		return 0, err
	}

	s := start.UnixNano()
	e := end.UnixNano()

	var first int64
	// if the end time is after the end of the file then will always return all the records
	last := idx.Records[len(idx.Records)-1].Number

	for _, v := range idx.Records {
		if s >= v.Start && s <= v.End {
			first = v.Number
		}
		if e >= v.Start && e <= v.End {
			last = v.Number
		}
	}

	in, err := c.getRangeFunc(d, first*recordLength64, (last+1)*recordLength64)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	return io.Copy(w, in)
}

func (c *Cache) listGetter(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	day, err := fromListKey(key)
	if err != nil {
		return err
	}

	n, err := c.listerFunc(day)
	if err != nil {
		return err
	}

	var l fdsn_pb.Listing

	for _, v := range n {
		l.Channels = append(l.Channels, &fdsn_pb.NSLC{Network: v.Network, Station: v.Station, Location: v.Location, Channel: v.Channel})
	}

	return dest.SetProto(&l)
}

func (c *Cache) indexGetter(ctx groupcache.Context, key string, dest groupcache.Sink) error {
	d, err := fromIndexKey(key)
	if err != nil {
		return err
	}

	in, err := c.getterFunc(d)
	if err != nil {
		return err
	}
	defer in.Close()

	var idx fdsn_pb.Mseed

	record := make([]byte, recordLength)
	var i int64

	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

loop:
	for {
		_, err = io.ReadFull(in, record)
		switch {
		case err == io.EOF:
			break loop
		case err != nil:
			return err
		}

		err = msr.Unpack(record, recordLength, 1, 0)
		if err != nil {
			return err
		}

		idx.Records = append(idx.Records, &fdsn_pb.Record{
			Number: i,
			Start:  msr.Starttime().UnixNano(),
			End:    msr.Endtime().UnixNano(),
		})

		i++
	}

	return dest.SetProto(&idx)
}

func toListKey(d time.Time, b int64) string {
	return fmt.Sprintf("%s_%d", d.Format(dateFormat), b)
}

func fromListKey(key string) (time.Time, error) {
	p := strings.Split(key, "_")

	if len(p) != 2 {
		return time.Time{}, fmt.Errorf("splitting key expected 2 parts got %d", len(p))
	}

	return time.Parse(dateFormat, p[0])
}

func toIndexKey(d DayFile, modificationTime time.Time) string {
	return fmt.Sprintf("%s_%s_%s_%s_%s_%s", d.Network, d.Station, d.Location, d.Channel, d.Date.Format(dateFormat), modificationTime.Format(time.RFC3339))
}

func fromIndexKey(key string) (DayFile, error) {
	p := strings.Split(key, "_")

	if len(p) != 6 {
		return DayFile{}, fmt.Errorf("splitting key expected 6 parts got %d", len(p))
	}

	t, err := time.Parse(dateFormat, p[4])
	if err != nil {
		return DayFile{}, err
	}

	d := DayFile{
		Date: t,
		NSLC: NSLC{
			Network:  p[0],
			Station:  p[1],
			Location: p[2],
			Channel:  p[3],
		},
	}

	return d, nil
}
