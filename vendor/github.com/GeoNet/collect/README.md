# SEED Projects


## SEEDLink

Projects that connect to a SEEDLink server

* http://ds.iris.edu/ds/nodes/dmc/services/seedlink/
* https://seiscode.iris.washington.edu/projects/libmseed
* https://ds.iris.edu/ds/nodes/dmc/manuals/libslink/

### Local Development

Needs libmseed and libslink compiled.  These are vendored into this repo.

```
cd cvendor/libmseed/
make
cd ../libslink
make
```

#### shakenz-slink

Connects to a SEEDLink server, requests strong motion data and calculates real time PGA and PGV values.
The config file shakenz-slink.pb must be in the same directory.  Modify the config file shakenz-slink/env.list 
as you see fit.

```
export SLINK_HOST=https://url.to.seedlink/
go build
./shakenz-slink
```

##### Running from Docker
Build haz-db locally if not available on AWS ECR (see the haz repo for more info):

```
cd ~/src/github.com/GeoNet/haz
./build.sh database
```

##### Build the shakenz-slink image
Run from the collect repo directory:
```
./build-cgo.sh shakenz-slink
```

##### Running From Docker-Compose
Docker-compose can be used to launch both shakenz-slink and haz-db images with the proper network config.  Other 
config is handled in the file shakenz-slink/env.list as environment variables.  Look at docker-compose.yml for 
more info.
```
docker-compose up
```

Shut down with:
```
docker-compose down
```
