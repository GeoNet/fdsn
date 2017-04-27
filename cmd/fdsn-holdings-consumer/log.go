package main

import "log"

var Prefix string

// set the log prefix in main instead of importing a pkg to do this
// ensures start up order.
func init() {
	if Prefix != "" {
		log.SetPrefix(Prefix + " ")
	}
}
