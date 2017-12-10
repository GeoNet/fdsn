package main

import (
	"database/sql"
	"github.com/GeoNet/fdsn/internal/fdsn"
	"github.com/GeoNet/kit/mseed"
	"github.com/golang/groupcache"
	"io"
	"os"
	"testing"
	"time"
)

func TestHoldingsSearch(t *testing.T) {
	testSetUp(t)
	defer testTearDown()

	testLoadFirst("etc/NZ.ABAZ.10.EHE.D.2016.079", t)

	s, err := time.Parse(time.RFC3339Nano, "2016-03-18T00:00:00.0Z")
	if err != nil {
		t.Error(err)
	}

	e, err := time.Parse(time.RFC3339Nano, "2016-03-20T00:00:00.0Z")
	if err != nil {
		t.Error(err)
	}

	d := fdsn.DataSearch{
		Network:  "^NZ$",
		Station:  "^ABAZ$",
		Channel:  "^EHE$",
		Location: "^10$",
		Start:    s,
		End:      e,
	}

	k, err := holdingsSearchNrt(d)
	if err != nil {
		t.Error(err)
	}

	if len(k) != 1 {
		t.Errorf("expected 1 result got %d", len(k))
	}

	if k[0] != "NZ_ABAZ_EHE_10_2016-03-19T00:00:01.968393Z" {
		t.Errorf("expected key NZ_ABAZ_EHE_10_2016-03-19T00:00:01.968393Z got %s", k[0])
	}
}

func TestGetRecord(t *testing.T) {
	testSetUp(t)
	defer testTearDown()

	testLoadFirst("etc/NZ.ABAZ.10.EHE.D.2016.079", t)

	var r []byte

	err := recordCache.Get(nil, "NZ_ABAZ_EHE_10_2016-03-19T00:00:01.968393Z", groupcache.AllocatingByteSliceSink(&r))
	if err != nil {
		t.Error(err)
	}
	if len(r) != 512 {
		t.Errorf("expected 512 bytes got %d", len(r))
	}

	// make sure we can unpack the miniSEED
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	err = msr.Unpack(r, 512, 1, 0)
	if err != nil {
		t.Error(err)
	}

	if msr.Network() != "NZ" {
		t.Errorf("expected network NZ got %s", msr.Network())
	}

	if msr.Station() != "ABAZ" {
		t.Errorf("expected station ABAZ got %s", msr.Station())
	}

	if msr.Channel() != "EHE" {
		t.Errorf("expected channel EHE got %s", msr.Channel())
	}

	if msr.Location() != "10" {
		t.Errorf("expected location 10 got %s", msr.Location())
	}

	_, err = msr.DataSamples()
	if err != nil {
		t.Errorf("error reading data %s", err.Error())
	}
}

func TestStartEnd(t *testing.T) {
	ts := "2017-03-18T00:00:00.0Z"
	testSetUp(t)
	defer func() { // clean up
		_, err := db.Exec("DELETE FROM fdsn.record WHERE start_time>=$1", ts)
		if err != nil {
			t.Error(err)
		}
		testTearDown()
	}()

	var err error

	tm, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Error(err)
	}

	// Create 10 records with each record is 10 seconds long.
	for i := 0; i < 10; i++ {
		_, err = db.Exec(`INSERT INTO fdsn.record (streamPK, start_time, latency_data, latency_tx, raw)
							SELECT streamPK, $1, 0, 0, '' FROM fdsn.stream WHERE network='NZ' AND station='ABAZ' AND channel='ACE' AND location='01'`, tm.Format(time.RFC3339Nano))
		if err != nil {
			t.Error(err)
		}
		tm = tm.Add(time.Second * 10)
	}

	// Now we set start time at '2017-03-18T00:00:15.0Z' which is between 00:00:10 and 00:00:20
	// And end time at '2017-03-18T00:01:15.0Z' which is between 00:01:10 and 00:01:20
	// The search result should start with 00:00:10 and till 00:01:10
	s, _ := time.Parse(time.RFC3339Nano, "2017-03-18T00:00:15.0Z")
	e, _ := time.Parse(time.RFC3339Nano, "2017-03-18T00:01:15.0Z")
	d := fdsn.DataSearch{
		Network:  "^NZ$",
		Station:  "^ABAZ$",
		Channel:  "^ACE$",
		Location: "^01$",
		Start:    s,
		End:      e,
	}

	k, err := holdingsSearchNrt(d)
	if err != nil {
		t.Error(err)
	}

	if len(k) != 7 {
		t.Errorf("expected 7 result got %d", len(k))
	}

	if k[0] != "NZ_ABAZ_ACE_01_2017-03-18T00:00:10Z" {
		t.Errorf("expect 'NZ_ABAZ_ACE_01_2017-03-18T00:00:10Z' but got '%s'", k[0])
	}

	if k[6] != "NZ_ABAZ_ACE_01_2017-03-18T00:01:10Z" {
		t.Errorf("expect 'NZ_ABAZ_ACE_01_2017-03-18T00:01:10Z' but got '%s'", k[6])
	}

}

func BenchmarkHoldingsSearch(b *testing.B) {
	testSetUp(b)
	defer testTearDown()

	// run benchmarks with more data if needed by loading all the data.
	//testLoad("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	testLoadFirst("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	s, err := time.Parse(time.RFC3339Nano, "2016-03-18T00:15:00.0Z")
	if err != nil {
		b.Error(err)
	}

	e, err := time.Parse(time.RFC3339Nano, "2016-03-20T00:30:00.0Z")
	if err != nil {
		b.Error(err)
	}

	d := fdsn.DataSearch{
		Network:  "^NZ$",
		Station:  "^ABAZ$",
		Channel:  "^EHE$",
		Location: "^10$",
		Start:    s,
		End:      e,
	}

	// exclude the set up from benchmark.
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		_, err = holdingsSearchNrt(d)
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkGetRecordCache(b *testing.B) {
	testSetUp(b)
	defer testTearDown()

	// run benchmarks with more data if needed by loading all the data.
	//testLoad("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	testLoadFirst("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	var r []byte
	var err error

	// exclude the set up from benchmark.
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		err = recordCache.Get(nil, "NZ_ABAZ_EHE_10_2016-03-19T00:00:01.968393Z", groupcache.AllocatingByteSliceSink(&r))
		if err != nil {
			b.Error(err)
		}
		if len(r) != 512 {
			b.Errorf("expected 512 bytes got %d", len(r))
		}
	}
}

// BenchmarkGetRecordDB is for comparison to BenchmarkGetRecordCache
// It hits the DB directly with no caching.
func BenchmarkGetRecordDB(b *testing.B) {
	testSetUp(b)
	defer testTearDown()

	// run benchmarks with more data if needed by loading all the data.
	//testLoad("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	testLoadFirst("etc/NZ.ABAZ.10.EHE.D.2016.079", b)

	start, err := time.Parse(time.RFC3339Nano, "2016-03-19T00:00:01.968393Z")
	if err != nil {
		b.Error(err)
	}

	var r []byte

	for n := 0; n < b.N; n++ {
		db.QueryRow(`SELECT raw FROM fdsn.record WHERE streampk =
        (SELECT streampk FROM fdsn.stream WHERE network = $1 AND station = $2 AND channel = $3 AND location = $4)
	AND start_time = $5`, "NZ", "ABAZ", "EHE", "10", start).Scan(&r)
		if err != nil {
			b.Error(err)
		}
		if len(r) != 512 {
			b.Errorf("expected 512 bytes got %d", len(r))
		}
	}
}

// funcs for setting up test data.

func testSetUp(t testing.TB) {
	var err error
	db, err = sql.Open("postgres", "host=localhost connect_timeout=5 user=fdsn_w password=test dbname=fdsn sslmode=disable")
	if err != nil {
		t.Fatalf("error with DB config: %s", err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	recordStmt, err = db.Prepare(`SELECT raw FROM fdsn.record WHERE streampk =
                                  (SELECT streampk FROM fdsn.stream WHERE network = $1 AND station = $2 AND channel = $3 AND location = $4)
	                          AND start_time = $5`)

	if recordCache == nil {
		recordCache = groupcache.NewGroup("record", 1000000000, groupcache.GetterFunc(recordGetter))
	}
}

func testTearDown() {
	db.Close()
}

// testLoad inserts all the records in file.
// Existing data for the stream in file are deleted first.
func testLoad(file string, t testing.TB) {
	in, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	r := make([]byte, 512)
	first := true

	for {
		_, err = io.ReadFull(in, r)
		switch {
		case err == io.EOF:
			return
		case err != nil:
			t.Fatal(err)
		}

		err = msr.Unpack(r, 512, 1, 0)
		if err != nil {
			t.Error(err)
			continue
		}

		network := msr.Network()
		station := msr.Station()
		channel := msr.Channel()
		location := msr.Location()

		// not bothering setting min and max

		if first {
			// first time through delete all the data and then readd the stream.
			_, err = db.Exec(`DELETE FROM fdsn.stream WHERE network = $1 AND station=$2 AND channel=$3 AND location=$4`,
				network, station, channel, location)
			if err != nil {
				t.Error(err)
			}
			_, err = db.Exec(`INSERT INTO fdsn.stream (network, station, channel, location) VALUES($1, $2, $3, $4)`,
				network, station, channel, location)
			if err != nil {
				t.Error(err)
			}
			first = false
		}

		_, err = db.Exec(`INSERT INTO fdsn.record (streamPK, start_time, raw, latency)
	SELECT streamPK, $5, $6, $7
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`, network, station, channel, location, msr.Starttime(), r, 0)
		if err != nil {
			t.Error(err)
		}
	}
}

// testLoadDataFirst inserts the first record in file.
// Existing data for the stream in file are deleted first.
func testLoadFirst(file string, t testing.TB) {
	in, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	r := make([]byte, 512)

	_, err = io.ReadFull(in, r)
	switch {
	case err == io.EOF:
		return
	case err != nil:
		t.Fatal(err)
	}

	err = msr.Unpack(r, 512, 1, 0)
	if err != nil {
		t.Error(err)
		return
	}

	network := msr.Network()
	station := msr.Station()
	channel := msr.Channel()
	location := msr.Location()

	// not bothering setting min and max

	// delete all the data and then readd the stream.
	_, err = db.Exec(`DELETE FROM fdsn.stream WHERE network=$1 AND station=$2 AND channel=$3 AND location=$4`,
		network, station, channel, location)
	if err != nil {
		t.Error(err)
	}
	_, err = db.Exec(`INSERT INTO fdsn.stream (network, station, channel, location) VALUES($1, $2, $3, $4)`,
		network, station, channel, location)
	if err != nil {
		t.Error(err)
	}

	_, err = db.Exec(`INSERT INTO fdsn.record (streamPK, start_time, raw, latency_tx, latency_data)
	SELECT streamPK, $5, $6, $7, $8
	FROM fdsn.stream
	WHERE network = $1
	AND station = $2
	AND channel = $3
	AND location = $4`, network, station, channel, location, msr.Starttime(), r, 0, 0)
	if err != nil {
		t.Error(err)
	}
}
