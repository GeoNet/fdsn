// +build devmode

package weft

import (
	"log"
	"net/http"
)

func (res Result) log(r *http.Request) {
	log.Printf("status: %d %t serving %s", res.Code, res.Ok, r.RequestURI)
	if res.Msg != "" {
		log.Printf("msg: %s", res.Msg)
	}
}
