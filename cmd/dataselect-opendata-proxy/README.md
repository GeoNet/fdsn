# dataselect-opendata-proxy

A FDSN dataselect web service that serves miniSEED waveform data by using [GeoNet open data waveform data] (`https://www.geonet.org.nz/data/access/aws`) as the backend.

This service is intented to run on client's local computer/premisis, as a middle man between end user and GeoNet Open Data bucket. **The end user simply runs this service, then sets his FDSN service to localhost, or to the on-premisis host.** 

Currently, this proxy service removed all request restrictions been set on GeoNet FDSN. Users using this proxy service won't be rejected when requesting a large amount of data in one shot.

## How it works

The service implements the [FDSN Web Services dataselect specification](http://www.fdsn.org/webservices/FDSN-WS-Specifications-1.1.pdf) (v1.1). 

For each request it:

1. Parses the network/station/location/channel parameters and time range.
2. Enumerates calendar days covering the query window (starting one day before the requested start time to catch files whose data crosses the day boundary).
3. Lists matching S3 keys under the open data waveform prefix using the path pattern:
   ```
   waveforms/miniseed/{yyyy}/{yyyy}.{doy}/{station}.{network}/{yyyy}.{doy}.{station}.{location}-{channel}.{network}.D
   ```
4. Fetches each matching file in sorted key order and streams miniSEED records that fall within the requested time window directly to the client.

The GeoNet Open Data S3 bucket is public — no AWS credentials are required.

## Running

```
go build ./cmd/dataselect-opendata-proxy
./dataselect-opendata-proxy
```

The server listens on `:8080`.

### Environment variables

| Variable | Description |
|----------|-------------|
| `LOG_EXTRA` | Set to `true` to log POST request bodies |
