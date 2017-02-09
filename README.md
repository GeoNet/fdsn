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

## fdsn-s3-consumer

Receives notifications for SeisComPML (SC3ML) event data uploads to S3 and posts the SC3ML to the fdsn-ws event service.
  
## fdsn-ws-event-loader

POSTs SC3ML to an fdsn-ws for bulk loading events.  The same thing can be achieved with curl.

* Refer to the config in the EB env running fdsn-ws for the current key.
* Use HTTPS for the POST.  Until the EB fdsn-ws app is using a GeoNet domain name you will need to skip verifying the TLS cert.
