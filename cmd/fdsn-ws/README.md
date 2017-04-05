#FDSN-WS

##fdsn-station
###Deploying
fdsn-station service requires a full fdsn-station xml as the data source.
Simply upload the fdsn-station xml file to a S3 bucket, then set environment variable:
```
AWS_REGION=
FDSN_STATION_XML_BUCKET= (bucket name only)
FDSN_STATION_XML_META_KEY= (eg. file name)
```
(NOTE: An environment with credential to access S3 is required.)

The fdsn-station service will download and cache it in "etc/" for later use (when the service restarts).

Downloading and unmarshaling the fdsn-station xml could take long : Unmarshaling a 174MB xml takes about 10 seconds in my MacBookPro(2017), and downloading it takes much longer. 
The service (/fdsnws/station/1/query) will keep returning 500 error with "Station data not ready" message until the xml is fully unmarshaled.

###Testing
There's a small fdsn-station-test.xml file in "etc/" for testing.
Simply set `FDSN_STATION_XML_META_KEY` to `fdsn-station-test.xml` then run the test.

##Generating `fdsn_station_type.go`
`fdsn_station_type.go` is generated from etc/fdsn-station-1.0.xsd by tool `xsdgen` from https://github.com/droyo/go-xml.
In the directory fdsn-ws, issue the command:
```
xsdgen -o fdsn_station_type.go etc/fdsn-station-1.0.xsd
```
However xsdgen doesn't really generates the final go file we wanted. These are issues and workarounds to apply:

* The generated fields' tags in struct contains xmlns. This caused unnecessary url for EACH tag when we marshaling the  xml.
To resolve this, we'll have to manually remove all "http://www.fdsn.org/xml/station/1" in the tags of generated go file.

* The incorrect interpreted "RootType" in the struct.
In the xsd definition "RootType" is a type name, not a field name. But xsdgen misinterpreted it.
To resolve this, find the "type RootType struct" in the generated go file and change from:
```
type RootType struct {
    SchemaVersion   float64   `xml:"schemaVersion,attr"`
...
```
to
```
type FDSNStationXML struct {
    XmlNs           string      `xml:"xmlns,attr" default:"http://www.fdsn.org/xml/station/1"`
    SchemaVersion   float64     `xml:"schemaVersion,attr"`
...
```