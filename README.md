# FDSN

Federation of Digital Seismic Networks (FDSN) Web Services (FDSN-WS) http://www.fdsn.org/webservices/

Refer to notes in `cmd/*/deploy/DEPLOY.md` for specific deployment requirements.

## Applications

### fdsn-ws

Provides FDSN web services.  

#### Dataselect

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
### fdsn-ws-nrt

Federation of Digital Seismic Networks (FDSN) Web Services (FDSN-WS) http://www.fdsn.org/webservices/ For 
near real time data from a Postgres database.

### slink-db

Listens to a SEEDLink server and saves miniSEED records to a Postgres database.

### fdsn-quake-consumer

Receives notifications for SeisComPML (SC3ML) event data uploads to S3 and stores the SC3ML in the DB.

The following versions of SC3ML can be handled (there is no difference between the quake content for these versions but they are 
handled with separate XSLT to be consistent with upstream changes):

* 0.7
* 0.8
* 0.9

The only version of QuakeML created and stored is 1.2

### fdsn-holdings-consumer

Receives notifications for miniSEED file uploads to S3, indexes the files, and saves the results to the holdings DB. 


## Development

### Building C libraries

Go wrappers to the libmseed and libslink need the C libraries.  The C source is vendored using govendor.  A special govendor 
command was used to vendor the C code from the kit repo:

```
govendor fetch github.com/GeoNet/kit/cvendor/libmseed^
govendor fetch github.com/GeoNet/kit/cvendor/libslink^
```

You will need to build these C libraries in-place before building fdsn-ws.  This will require a C compiler (eg: gcc)
and make (possibly other packages depending on your system.  Alpine requires musl-dev):

```
cd vendor/github.com/GeoNet/kit/cvendor/libmseed/
make
cd ../libslink/
make
```

Build.sh will automatically re-build these C libraries before building any Go executables in Docker.
