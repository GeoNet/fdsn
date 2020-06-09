// fs provides utilities for mSEED records on disk.
package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// NSLC uniquely identifies a stream of data
type NSLC struct {
	Network, Station, Location, Channel string
}

type Record struct {
	Path       string
	Start, End int64
}

func (n NSLC) Path(dir string, start time.Time) string {
	return strings.Join([]string{dir, n.Network, n.Station, n.Location, n.Channel,
		fmt.Sprintf("%d", start.UTC().Weekday()),
		fmt.Sprintf("%02d", start.UTC().Hour())},
		string(os.PathSeparator))
}

func (n NSLC) RecordPath(dir string, start, end time.Time) string {
	return strings.Join([]string{dir, n.Network, n.Station, n.Location, n.Channel,
		fmt.Sprintf("%d", start.UTC().Weekday()),
		fmt.Sprintf("%02d", start.UTC().Hour()),
		fmt.Sprintf("%d-%d", start.UTC().UnixNano(), end.UTC().UnixNano())},
		string(os.PathSeparator))
}

func (n NSLC) ListRecords(dir string) ([]Record, error) {
	var recs []Record
	pth := strings.Join([]string{dir, n.Network, n.Station, n.Location, n.Channel}, string(os.PathSeparator))
	wds, err := ioutil.ReadDir(pth)
	if err != nil {
		return recs, err
	}

	for _, w := range wds { // weekdays
		hrs, err := ioutil.ReadDir(strings.Join([]string{pth, w.Name()}, string(os.PathSeparator)))
		if err != nil {
			return recs, err
		}

		for _, h := range hrs {
			rs, err := ioutil.ReadDir(strings.Join([]string{pth, w.Name(), h.Name()}, string(os.PathSeparator)))
			if err != nil {
				return recs, err
			}

			pt := strings.Join([]string{pth, w.Name(), h.Name()}, string(os.PathSeparator))
			for _, r := range rs {
				// making sure we'll only return the filename with valid format

				p := strings.Split(r.Name(), "-")

				if len(p) != 2 {
					continue
				}

				start, err := strconv.ParseInt(p[0], 10, 64)
				if err != nil {
					continue
				}

				end, err := strconv.ParseInt(p[0], 10, 64)
				if err != nil {
					continue
				}

				recs = append(recs, Record{Path: pt + string(os.PathSeparator) + r.Name(), Start: start, End: end})
			}
		}
	}

	return recs, nil
}

func ListChannels(dir string) ([]NSLC, error) {
	networks, err := ioutil.ReadDir(dir)
	if err != nil {
		return []NSLC{}, err
	}

	var chans []NSLC

	for _, n := range networks {
		stations, err := ioutil.ReadDir(strings.Join([]string{dir, n.Name()}, string(os.PathSeparator)))
		if err != nil {
			return []NSLC{}, err
		}

		for _, s := range stations {
			locations, err := ioutil.ReadDir(strings.Join([]string{dir, n.Name(), s.Name()}, string(os.PathSeparator)))
			if err != nil {
				return []NSLC{}, err
			}

			for _, l := range locations {
				channels, err := ioutil.ReadDir(strings.Join([]string{dir, n.Name(), s.Name(), l.Name()}, string(os.PathSeparator)))
				if err != nil {
					return []NSLC{}, err
				}

				for _, c := range channels {
					chans = append(chans, NSLC{Network: n.Name(), Station: s.Name(), Location: l.Name(), Channel: c.Name()})
				}
			}
		}
	}

	return chans, nil
}
