module github.com/GeoNet/fdsn

go 1.13

require (
	github.com/GeoNet/kit v0.0.0-20190918224938-2cc27e012059
	github.com/aws/aws-sdk-go v1.24.6
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6
	github.com/gorilla/schema v1.1.1-0.20190322171712-d768c7020973
	github.com/lib/pq v1.2.0
	github.com/pkg/errors v0.8.2-0.20190227000051-27936f6d90f9
)

replace github.com/GeoNet/kit => ../kit
