# FDSN additional documentation

GeoNet provides FDSN webservices in two separate services:
- FDSN is the **archive** service
- FDSN-NRT is the **near real-time** service

Diagrams of the implementation of FDSN and FDSN-NRT web services at GeoNet are provided herein.

Further details, limitations and a tutorial on how to access data are provided on the [GeoNet website FDSN Data page](https://www.geonet.org.nz/data/tools/FDSN).

## FDSN limitations

The GeoNet FDSN service implementation provides waveform data that are 7 days behind the present time to ensure the data is as complete as possible.

The `dataselect` service is provisioning data from the GeoNet archive stored as miniseed files off an Amazon S3 bucket.
The `station` service is provisioning station metadata from the GeoNet metadata repository named "delta".
The `event` service is provisioning event information data from the GeoNet earthquake catalogue.


## FDSN-NRT limitations

The GeoNet FDSN-NRT service implementation provides near real time data for the last 8 days. 

The GeoNet FDSN-NRT service implementation between 2017 and 2021 had some limitations related to balancing service performance and costs. 
To reduce the stress of this previous implementation, a GeoNet FDSN-NRT service update was implemented and fully deployed in October 2021.

The latest GeoNet FDSN-NRT service uses a more progressive buffering mechanism while saving the collected miniseed records into the service's storage. Records are only available to the GeoNet FDSN-NRT dataselect service after they've been stored, thus time delays are to be expected while records are in flight in the system buffer.

The duration of the time delay varies for different channels based on the miniseed records' arrival rate: 
- for higher frequency channels the delay will be shorter (< 1 minute)
- for lower frequency channels the delay will be longer (< 5 minutes)

The system is configured to make sure the delay is no longer than 5 minutes for all channels.
