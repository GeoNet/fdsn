package main

import (
	"bytes"
	"github.com/GeoNet/fdsn/internal/valid"
	"github.com/GeoNet/kit/weft"
	"net/http"
)

func s3ml(r *http.Request, h http.Header, b *bytes.Buffer) error {
	err := weft.CheckQuery(r, []string{"GET"}, []string{"eventid"}, []string{})
	if err != nil {
		return err
	}

	publicID := r.URL.Query().Get("eventid")

	err = valid.PublicID(publicID)
	if err != nil {
		return err
	}

	var s string
	err = db.QueryRow(`SELECT Sc3ml FROM fdsn.event where publicid = $1`, publicID).Scan(&s)
	if err != nil {
		return err
	}

	h.Set("Content-Type", "application/xml")

	_, err = b.WriteString(s)

	return err
}
