// +build !devmode

package weft

import (
	"log"
	"net/http"
)

func (res Result) log(r *http.Request) {
	if res.Code != http.StatusOK {
		log.Printf("WARN: %d serving %s: %s", res.Code, r.RequestURI, res.Msg)
	}
}
