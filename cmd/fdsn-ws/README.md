#FDSN-WS

##fdsn-station
###Deploying
fdsn-station service requires a full fdsn-station gob as the data source.
Simply upload the fdsn-station gob file to a S3 bucket, then set environment variable:
```
AWS_REGION=
FDSN_STATION_GOB_BUCKET= (bucket name only)
FDSN_STATION_GOB_META_KEY= (eg. file name)
```
(NOTE: An environment with credential to access S3 is required.)

The fdsn-station service will download and cache it in "etc/" for later use (when the service restarts).

Downloading the fdsn-station gob could take long - 
the service (/fdsnws/station/1/query) will keep returning 500 error with "Station data not ready" message until the gob is fully unmarshaled.

###Testing
There's a small fdsn-station-test.gob file in "etc/" for testing.
Simply set `FDSN_STATION_GOB_META_KEY` to `fdsn-station-test.gob` then run the test.

##Generating `fdsn_station_type.go`
`fdsn_station_type.go` is generated from etc/fdsn-station-1.0.xsd by tool `xsdgen` from https://github.com/droyo/go-xml.
In the directory fdsn-ws, issue the command:
```
xsdgen -r 'RootType -> FDSNStationXML' -pkg main -o fdsn_station_type.go etc/fdsn-station-1.0.xsd 
```
However the generated fields' tags in struct contains xmlns. This caused unnecessary url for EACH tag when we marshaling the  xml.
To resolve this, we'll have to manually remove all "http://www.fdsn.org/xml/station/1" in the tags of generated go file.

##Creating GOB file
About GOB file: https://blog.golang.org/gobs-of-data .
There's a command line tool in `cmd/xml2gob` to transcode station xml into gob file.
The tool requires fdsn_station_type.go to unmarshal xml into struct first. 
So remeber to keep the `xml2gob/fdsn_station_type.go` in sync with the one in `cmd/fdsn-ws`.