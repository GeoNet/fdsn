package main

import (
	"bytes"
	"github.com/GeoNet/fdsn/internal/weft"
	"net/http"
)

func s3ml(r *http.Request, h http.Header, b *bytes.Buffer) *weft.Result {
	switch r.Method {
	case "GET":
		if res := weft.CheckQuery(r, []string{"eventid"}, []string{}); !res.Ok {
			return res
		}

		var s string
		err := db.QueryRow(`SELECT Sc3ml FROM fdsn.event where publicid = $1`, r.URL.Query().Get("eventid")).Scan(&s)
		if err != nil {
			return weft.InternalServerError(err)
		}

		b.WriteString(s)

		h.Set("Content-Type", "application/xml")

		return &weft.StatusOK

	default:
		return &weft.MethodNotAllowed
	}

}
