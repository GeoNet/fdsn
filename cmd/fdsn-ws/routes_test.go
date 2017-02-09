package main

import (
	"bytes"
	wt "github.com/GeoNet/weft/wefttest"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

// setup() adds event 2015p768477 to the DB.
var routes = wt.Requests{
	{ID: wt.L(), URL: "/sc3ml", Method: "POST", Status: http.StatusUnauthorized},
	{ID: wt.L(), URL: "/sc3ml", Method: "PUT", Status: http.StatusMethodNotAllowed},
	{ID: wt.L(), URL: "/sc3ml", Method: "DELETE", Status: http.StatusMethodNotAllowed},
	{ID: wt.L(), URL: "/sc3ml?eventid=2015p768477", Content: "application/xml"},

	// fdsn-ws-event
	{ID: wt.L(), URL: "/fdsnws/event/1", Content: "text/html"},
	{ID: wt.L(), URL: "/fdsnws/event/1/query?eventid=2015p768477", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/version", Content: "text/plain"},
	{ID: wt.L(), URL: "/fdsnws/event/1/catalogs", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/contributors", Content: "application/xml"},
	{ID: wt.L(), URL: "/fdsnws/event/1/application.wadl", Content: "application/xml"},
}

// Test all routes give the expected response.  Also check with
// cache busters and extra query paramters.
func TestRoutes(t *testing.T) {
	setup(t)
	defer teardown()

	for _, r := range routes {
		if b, err := r.Do(ts.URL); err != nil {
			t.Error(err)
			t.Error(string(b))
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
