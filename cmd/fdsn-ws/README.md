#FDSN-WS

##fdsn-station
###Deploying
fdsn-station service requires a full fdsn-station xml as the data source.
Simply upload the fdsn-station xml file to a S3 bucket, then set environment variable:
```
AWS_REGION=
STATION_XML_BUCKET= (bucket name only)
STATION_XML_META_KEY= (eg. file name)
```
(NOTE: An environment with credential to access S3 is required.)

The fdsn-station service will download and cache it in "etc/" for later use (when the service restarts).

Downloading and unmarshaling the fdsn-station xml could take long : Unmarshaling a 174MB xml takes about 10 seconds in my MacBookPro(2017), and downloading it takes much longer. 
So the initial time will be longer than usual - could be more than 10 seconds.
The service periodically (defaults to 300 seconds, set by STATION_RELOAD_INTERVAL) checks if the data source xml in the S3 bucket has been updated.

###Testing
There's a small fdsn-station-test.xml file in "etc/" for testing.
Simply set `STATION_XML_META_KEY` to `fdsn-station-test.xml` then run the test.

##Generating `fdsn_station_type.go`
`fdsn_station_type.go` is generated from etc/fdsn-station-1.0.xsd by tool `xsdgen` from https://github.com/droyo/go-xml.
In the directory fdsn-ws, issue the command:
```
xsdgen -r 'RootType -> FDSNStationXML' -pkg main -o fdsn_station_type.go etc/fdsn-station-1.0.xsd 
```
However there're something you'll have to do manually.
1. The generated fields' tags in struct contains xmlns. This caused unnecessary url for EACH tag when we marshaling the  xml.
To resolve this, we'll have to manually remove all "http://www.fdsn.org/xml/station/1" in the tags of generated go file.
2. There's no "omitempty" attribute in the tag, so you'll have to add "omitempty" to all tags.
For example, change
```
Unit       string  `xml:"unit,attr"`
```
to
```
Unit       string  `xml:"unit,attr,omitempty"`
```
However, some fields can't add omitempty attribute.
All fields named "Value" with float64 type and without field name:
```
Value      float64 `xml:",chardata"`
```
must leave unchanged. 
