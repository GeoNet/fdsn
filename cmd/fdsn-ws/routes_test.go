package main

import (
	"bytes"
	"fmt"
	"github.com/GeoNet/collect/mseed"
	"github.com/GeoNet/fdsn/internal/holdings"
	wt "github.com/GeoNet/weft/wefttest"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// setup() adds event 2015p768477 to the DB.
var routes = wt.Requests{
	{ID: wt.L(), URL: "/sc3ml", Method: "POST", Status: http.StatusUnauthorized},
	{ID: wt.L(), URL: "/sc3ml", Method: "PUT", Status: http.StatusMethodNotAllowed},
	{ID: wt.L(), URL: "/sc3ml", Method: "DELETE", Status: http.StatusMethodNotAllowed},
	{ID: wt.L(), URL: "/sc3ml?eventid=2015p768477", Content: "application/xml"},

	{ID: wt.L(), URL: "/holdings/NZ.ABAZ.01.ACE.D.2016.097", Method: "POST", Status: http.StatusMethodNotAllowed},
	{ID: wt.L(), URL: "/holdings/NZ.ABAZ.01.ACE.D.2016.097", Method: "PUT", Status: http.StatusUnauthorized},
	{ID: wt.L(), URL: "/holdings/NZ.ABAZ.01.ACE.D.2016.097", Method: "DELETE", Status: http.StatusUnauthorized},

	// fdsn-ws-event
	{ID: wt.L(), URL: "/fdsnws/event/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/event/1/query?eventid=2015p768477", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/event/1/catalogs", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/contributors", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/application.wadl", Content: "application/xml"},

	// fdsn-ws-dataselect
	{ID: wt.L(), URL: "/fdsnws/dataselect/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-01-09T00:00:00&endtime=2016-01-09T23:00:00&network=NZ&station=CHST&location=01&channel=LOG", Content: "application/vnd.fdsn.mseed"},
	// abbreviated params
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?start=2016-01-09T00:00:00&end=2016-01-09T23:00:00&net=NZ&sta=CHST&loc=01&cha=LOG", Content: "application/vnd.fdsn.mseed"},
	//{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?start=2016-01-09T00:00:00&end=2016-01-09T23:00:00&net=NZ&sta=CHST,ALRZ&loc=01&cha=LOG", Content: "application/vnd.fdsn.mseed"},
	// an invalid network or no files matching query should give 404 (could also give 204 as per spec)
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=2016-01-09T00:00:00&endtime=2016-01-09T23:00:00&network=INVALID_NETWORK&station=CHST&location=01&channel=LOG",
		Content: "text/plain; charset=utf-8",
		Status:  http.StatusNoContent},
	// very old time range, no files:
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/query?starttime=1900-01-09T00:00:00&endtime=1900-01-09T01:00:00&network=NZ&station=CHST&location=01&channel=LOG",
		Content: "text/plain; charset=utf-8",
		Status:  http.StatusNoContent},
	//{ID: wt.L(), URL: "/fdsnws/dataselect/1/query", Content: "text/plain", Status: http.StatusRequestEntityTooLarge},
	{ID: wt.L(), URL: "/fdsnws/dataselect/1/application.wadl", Content: "application/xml"},
}

// Test all routes give the expected response.  Also check with
// cache busters and extra query paramters.
func TestRoutes(t *testing.T) {
	setup(t)
	defer teardown()

	populateHoldings(t)

	for _, r := range routes {
		if b, err := r.Do(ts.URL); err != nil {
			t.Error(err)
			if len(b) > 0 {
				t.Error(string(b))
			}
		}
	}

	if err := routes.DoCheckQuery(ts.URL); err != nil {
		t.Error(err)
	}
}

/*
tests posting an sc3ml file to the server.
*/
func TestPostSc3ml(t *testing.T) {
	setup(t)
	defer teardown()

	var err error
	if _, err = db.Exec(`delete from fdsn.event where publicid = '2015p768477'`); err != nil {
		t.Error(err)
	}

	var count int

	if err = db.QueryRow(`select count(*) from fdsn.event where publicid = '2015p768477'`).Scan(&count); err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("found unpexected quake in the db")
	}

	var f *os.File
	var b []byte

	if f, err = os.Open("etc/2015p768477.xml"); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if b, err = ioutil.ReadAll(f); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.Write(b)

	var req *http.Request
	var res *http.Response

	if req, err = http.NewRequest("POST", ts.URL+"/sc3ml", &buf); err != nil {
		t.Fatal(err)
	}

	// make sure the basic auth passwords will match
	key = "test"
	req.SetBasicAuth("", "test")

	client := &http.Client{}

	if res, err = client.Do(req); err != nil {
		t.Error(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200 got %d", res.StatusCode)
	}

	var mag float64

	if err = db.QueryRow(`select magnitude from fdsn.event where publicid = '2015p768477'`).Scan(&mag); err != nil {
		t.Error(err)
	}

	if mag != 5.691131913 {
		t.Errorf("mag should equal 5.691131913 got %f", mag)
	}
}

// Test getting files from dataselect endpoint.  This is using an S3 bucket so the environment variables from env.list
// must be valid and properly set
func TestDataSelectGet(t *testing.T) {
	setup(t)
	defer teardown()

	populateHoldings(t)

	// Testing GET first
	u, err := url.Parse(ts.URL + "/fdsnws/dataselect/1/query")
	if err != nil {
		t.Fatal(err)
	}

	t1Str := "2016-01-01T22:00:00Z"
	t1, err := time.Parse(time.RFC3339, t1Str)
	if err != nil {
		t.Fatal(err)
	}

	t2Str := "2016-01-02T00:30:00Z"
	t2, err := time.Parse(time.RFC3339, t2Str)
	if err != nil {
		t.Fatal(err)
	}

	values := url.Values{
		"starttime": {t1Str[0 : len(t1Str)-1]},
		"endtime":   {t2Str[0 : len(t2Str)-1]},
		"network":   {"NZ"},
		"station":   {"ALRZ"},
		"location":  {"10"},
		"channel":   {"EHN"},
	}
	u.RawQuery = values.Encode()

	client := http.Client{}
	resp, err := client.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var readBuff bytes.Buffer
	if _, err = readBuff.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 got %d, body:%s", resp.StatusCode, string(readBuff.Bytes()))
	}

	// Read each record in the file, check times/net/sta/etc.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	if readBuff.Len() == 0 {
		t.Error("got empty response")
	}

	for {
		record := readBuff.Next(RECORDLEN)

		if len(record) < RECORDLEN {
			break
		}

		if err = msr.Unpack(record, RECORDLEN, 1, 0); err != nil {
			t.Fatal(err)
		}

		if msr.Starttime().Before(t1) {
			t.Fatal("start time of record was before specified starttime")
		}

		if msr.Starttime().After(t2) {
			t.Fatal("end time of record was after specified endtime")
		}

		expectedHdrs := []string{values["network"][0], values["station"][0], values["location"][0], values["channel"][0]}
		obsHdrs := []string{msr.Network(), msr.Station(), msr.Location(), msr.Channel()}
		for i, header := range obsHdrs {
			// These strings from C have a null terminator which breaks this comparison so trimming it
			header = strings.TrimRight(header, "\x00")
			if header != expectedHdrs[i] {
				t.Fatalf("expected header `%s` but observed: `%s`", header, expectedHdrs[i])
			}
		}

		// Just a couple quick sanity checks of the samples.  If msr.Unpack is getting this far we're probably ok.
		if msr.Numsamples() < 200 || msr.Numsamples() > 700 {
			t.Fatalf("expected Numsamples between 200 and 700 but observed %d", msr.Numsamples())
		}

		if msr.Samprate() != 100.0 {
			t.Fatalf("expected Samprate 100.0 but observed %f", msr.Samprate())
		}
	}

}

func TestDataSelectPost(t *testing.T) {
	setup(t)
	defer teardown()

	populateHoldings(t)
	// testing POST:
	// The curl equivalent is: curl -v --data-binary @post_input.txt http://localhost:8080/fdsnws/dataselect/1/query -o test_post.mseed
	var buf bytes.Buffer

	// note: we ignore the parameters: quality/minimumlength/longestonly
	postContent := []byte(`quality=M
minimumlength=0.0
longestonly=FALSE
NZ ALRZ 10 EHN 2016-01-09T00:00:00 2016-01-09T02:00:00
NZ ALRZ 10 EH* 2016-01-02T00:00:00 2016-01-02T01:00:00
NZ ALRZ 01 V?? 2016-01-09T23:00:00 2016-01-10T00:00:00
NZ ALRZ 01 VEP 2016-01-02T23:57:00 2016-01-03T00:00:00
NZ ALRZ 01 VKI 2016-01-02T00:00:00.000000 2016-01-02T01:00:00.000000
`)

	buf.Write(postContent)

	postClient := http.Client{}
	postResp, err := postClient.Post(ts.URL+"/fdsnws/dataselect/1/query", "text/plain", &buf)
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()

	var postReadBuff bytes.Buffer
	if _, err = postReadBuff.ReadFrom(postResp.Body); err != nil {
		t.Fatal(err)
	}

	if postResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 got %d, body: %s", postResp.StatusCode, string(postReadBuff.Bytes()))
	}

	if postReadBuff.Len() == 0 {
		t.Error("got empty response body")
	}

	tMin, err := time.Parse(time.RFC3339, "2016-01-02T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	tMax, err := time.Parse(time.RFC3339, "2016-01-10T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}

	// Read each record in the file, check times/net/sta/etc.
	msr := mseed.NewMSRecord()
	defer mseed.FreeMSRecord(msr)

	// These records will be multiplexed with varying samplerates/lengths/etc.
	for {
		record := postReadBuff.Next(RECORDLEN)

		if len(record) < RECORDLEN {
			break
		}

		if err = msr.Unpack(record, RECORDLEN, 1, 0); err != nil {
			t.Fatal(err)
		}

		if msr.Starttime().Before(tMin) {
			t.Fatal("start time of record was before tMin:", msr.Starttime(), tMin)
		}

		if msr.Starttime().After(tMax) {
			t.Fatal("end time of record was after tMax:", msr.Starttime(), tMax)
		}

		// Just a couple quick sanity checks of the samples.  If msr.Unpack is getting this far we're probably ok.
		if msr.Numsamples() < 87 || msr.Numsamples() > 700 {
			t.Fatalf("expected Numsamples between 87 and 700 but observed %d", msr.Numsamples())
		}
	}
}

// populateDb inserts a thwack of holdings that exist in the bucket so we can run our integration tests against them
func populateHoldings(t *testing.T) {
	stations := []string{"ALRD", "ALRZ", "CHST"}
	locations := []string{"01", "10"}
	channels := []string{"EHN", "VEP", "VKI", "LOG"}

	t1Str := "2016-01-01T00:00:00Z"
	t1, err := time.Parse(time.RFC3339, t1Str)
	if err != nil {
		t.Fatal(err)
	}

	t2Str := "2016-01-10T00:00:00Z"
	t2, err := time.Parse(time.RFC3339, t2Str)
	if err != nil {
		t.Fatal(err)
	}

	for _, sta := range stations {
		for _, cha := range channels {
			for _, loc := range locations {
				for step := t1; step.Before(t2); step = step.Add(time.Hour * 24) {
					h := holding{
						key: fmt.Sprintf("%d/NZ/%s/%s.D/NZ.%s.%s.%s.D.%d.%03d", step.Year(), sta, cha, sta, loc, cha, step.Year(), step.YearDay()),
						Holding: holdings.Holding{
							Network:    "NZ",
							Station:    sta,
							Channel:    cha,
							Location:   loc,
							Start:      step,
							NumSamples: 500000, // incorrect but we're just faking it
						},
					}

					err = h.save()
					if err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
}
