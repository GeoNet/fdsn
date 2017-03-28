// +build !devmode

package weft

import (
	"log"
	"net/http"
)

func (res Result) log(r *http.Request) {
	if res.Code != http.StatusOK {
		log.Printf("status: %d serving %s", res.Code, r.RequestURI)
	}
}
