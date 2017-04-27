# FDSN

Federation of Digital Seismic Networks (FDSN) Web Services (FDSN-WS).

Refer to notes in `cmd/*/deploy/DEPLOY.md` for specific deployment requirements.

## Building C libraries

Fdsn-ws uses the Go wrappers to the libmseed C library.  The C source is vendored using govendor.  A special govendor 
command was used to vendor the entire collect repository and C code:
```
govendor fetch github.com/GeoNet/collect/^
```

You will need to build these C libraries in-place before building fdsn-ws.  This will require a C compiler (eg: gcc)
and make (possibly other packages depending on your system.  Apline requires musl-dev):
```
cd vendor/github.com/GeoNet/collect/cvendor/libmseed/
make
cd ../libslink/
make
```

Build.sh will automatically re-build these C libraries before building any Go executables in Docker.

## fdsn-ws

Provides FDSN web services.  

Events can be loaded for the QuakeML web services by posting (with appropriate credentials, see below) SC3ML to `/sc3ml`,
this is transformed into a QuakeML 1.2 fragment and stored in the DB (see sc3ml.go).  These fragments are then combined and served for event
requests.

SC3ML is usually delivered with an S3 bucket notification which is consumed by `fdsn-s3-consumer`.
SC3ML can be bulk loaded with `fdsn-ws-event-loader` or any http client.

The following versions of SC3ML can be handled (there is no difference between the quake content for these versions but they are 
handled with separate XSLT to be consistent with upstream changes):

* 0.7
* 0.8
* 0.9

The only version of QuakeML created and stored is 1.2

### Dataselect

FDSN dataselect has been implemented, querying and serving data from miniseed files off an Amazon S3 bucket.

This example uses Curl to download data from a single query to the file test.mseed:
```
curl "http://localhost:8080/fdsnws/dataselect/1/query?network=NZ&station=CHST&location=01&channel=LOG&starttime=2017-01-09T00:00:00&endtime=2017-01-09T23:00:00" -o test.mseed
```
 
This example uses multiple queries using POST, in this case saving to test_post.mseed:
```
curl -v --data-binary @post_input.txt http://localhost:8080/fdsnws/dataselect/1/query -o test_post.mseed
```

The contents of post_input.txt:
```
NZ ALRZ 10 EHN 2017-01-09T00:00:00 2017-01-09T02:00:00
NZ ALRZ 10 AC* 2017-01-02T00:00:00 2017-01-10T00:00:00
NZ ALRZ 10 B?  2017-01-09T00:00:00 2017-01-10T00:00:00
```

## fdsn-s3-consumer

Receives notifications for SeisComPML (SC3ML) event data uploads to S3 and posts the SC3ML to the fdsn-ws event service.
  
## fdsn-ws-event-loader

POSTs SC3ML to an fdsn-ws for bulk loading events.  The same thing can be achieved with curl.

* Refer to the config in the EB env running fdsn-ws for the current key.
* Use HTTPS for the POST.  Until the EB fdsn-ws app is using a GeoNet domain name you will need to skip verifying the TLS cert.

## fdsn-holdings-consumer

Receives notifications for miniSEED file uploads to S3 and PUTs the S3 object key to the fdsn-ws holdings service.

### Data Holdings

A holdings table is used for data select.  This needs to be populated with PUTs to the fdsn-ws URL .../holdings/...key... e.g.,

```
curl -u :test -X PUT http://localhost:8080/holdings/NZ.AKCZ.01.OCF.D.2016.207
```

* This should be kept up to date using S3 bucket notifications, see `cmd/fdsn-holdings-consumer/deploy/DEPLOY.md`.
* Back filling can be done by listing a bucket and then using curl with the list.

There is a dump of ~1/2 a year of holdings data available for dev/test.  Initialize the DB then load the test data:

```
cp etc/data/fdsn-holding-dump.txt.gz /tmp
gunzip /tmp/fdsn-holding-dump.txt.gz
psql -h 127.0.0.1 fdsn fdsn_w -f /tmp/fdsn-holding-dump.txt
```