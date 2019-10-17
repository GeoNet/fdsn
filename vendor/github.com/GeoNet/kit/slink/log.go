//nolint //cgo generates code that doesn't pass linting
// Package slink provides a go wrapper for the libslink C library.
package slink

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libslink
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libslink/libslink.a
//#include <libslink.h>
import "C"

import (
	"strings"
)

var logFunc, errFunc func(string)

//export logPrint
func logPrint(msg *C.char) {
	if logFunc != nil {
		logFunc(strings.TrimSpace(C.GoString(msg)))
	}
}

//export errPrint
func errPrint(msg *C.char) {
	if errFunc != nil {
		errFunc(strings.TrimSpace(C.GoString(msg)))
	}
}
