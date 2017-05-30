## go wrapper for libslink ##

http://ds.iris.edu/ds/nodes/dmc/software/downloads/libslink/

An initial minimal wrapper for the libslink library used
to create clients for connecting to seedlink servers.

The library needs to be compiled and placed, together with
the libslink.h and slplatform.h files, somewhere where
the go build (cgo) routines can find them.

The logging aspect of the library hasn't been wrapped.

To test the build an operational seedlink server should be
running locally on port 18000.

Mark Chadwick
