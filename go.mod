module github.com/GeoNet/fdsn

go 1.13

require (
	github.com/GeoNet/kit v0.0.0-20210610214455-dafd8e077cdc
	github.com/aws/aws-sdk-go v1.39.6
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/gorilla/schema v1.2.0
	github.com/lib/pq v1.10.2
	github.com/pkg/errors v0.9.1
	google.golang.org/protobuf v1.27.1 // indirect
)

replace github.com/GeoNet/kit => ../kit
