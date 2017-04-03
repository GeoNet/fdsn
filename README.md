# FDSN

Federation of Digital Seismic Networks (FDSN) Web Services (FDSN-WS).

Refer to notes in `cmd/*/deploy/DEPLOY.md` for specific deployment requirements.

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
