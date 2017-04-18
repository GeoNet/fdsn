## go wrapper for libmseed ##

http://ds.iris.edu/ds/nodes/dmc/software/downloads/libmseed/

An initial minimal wrapper for the libmseed library used to
handle miniseed blocks.

The library needs to be compiled and placed, together with
the libmseed.h and lmplatform.h files, somewhere where
the go build (cgo) routines can find them.

Mark Chadwick

