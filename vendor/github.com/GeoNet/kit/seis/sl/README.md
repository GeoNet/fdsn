

# sl
`import "github.com/GeoNet/kit/seis/sl"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
The sl module has been writen as a lightweight replacement for the C libslink library.
It is aimed at clients that need to connect and decode data from a seedlink server.

The seedlink code is not a direct replacement for libslink. It can run in two modes, either as a
raw connection to the client connection (Conn) which allows mechanisms to monitor or have a finer
control of the SeedLink connection, or in the collection mode (SLink) where a connection is established
and received miniseed blocks can be processed with a call back function. A context can be passed into
the collection loop to allow interuption or as a shutdown mechanism. It is not passed to the underlying
seedlink connection messaging which is managed via a deadline mechanism, e.g. the `SetTimeout` option.

An example raw Seedlink application can be as simple as:


	 if err := sl.NewSLink().Collect(func(seq string, data []byte) (bool, error) {
		   //... process miniseed data
	
	        return false, nil
	 }); err != nil {
	         log.Fatal(err)
	 }

An example using Seedlink collection mechanism with state could look like.


	 slink := sl.NewSLink(
	  sl.SetServer(slinkHost),
	  sl.SetNetTo(60*time.Second),
	  sl.SetKeepAlive(time.Second),
	  sl.SetStreams(streamList),
	  sl.SetSelectors(selectors),
	  sl.SetStart(beginTime),
	 )
	
	 slconn := sl.NewSLConn(slink, sl.SetStateFile("example.json"), sl.SetFlush(time.Minute))
	 if err := slconn.Collect(func(seq string, data []byte) (bool, error) {
		   //... process miniseed data
	
	        return false, nil
	 }); err != nil {
	         log.Fatal(err)
	 }




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [type CollectFunc](#CollectFunc)
* [type Conn](#Conn)
  * [func NewConn(service string, timeout time.Duration) (*Conn, error)](#NewConn)
  * [func (c *Conn) Collect() (*Packet, error)](#Conn.Collect)
  * [func (c *Conn) CommandCat() ([]byte, error)](#Conn.CommandCat)
  * [func (c *Conn) CommandClose() ([]byte, error)](#Conn.CommandClose)
  * [func (c *Conn) CommandData(sequence string, starttime time.Time) error](#Conn.CommandData)
  * [func (c *Conn) CommandEnd() error](#Conn.CommandEnd)
  * [func (c *Conn) CommandHello() ([]byte, error)](#Conn.CommandHello)
  * [func (c *Conn) CommandId() ([]byte, error)](#Conn.CommandId)
  * [func (c *Conn) CommandSelect(selection string) error](#Conn.CommandSelect)
  * [func (c *Conn) CommandStation(station, network string) error](#Conn.CommandStation)
  * [func (c *Conn) CommandTime(starttime, endtime time.Time) error](#Conn.CommandTime)
  * [func (c *Conn) GetInfo(level string) (*Info, error)](#Conn.GetInfo)
  * [func (c *Conn) GetInfoLevel(level string) ([]byte, error)](#Conn.GetInfoLevel)
* [type Info](#Info)
  * [func (s *Info) Unmarshal(data []byte) error](#Info.Unmarshal)
* [type Packet](#Packet)
  * [func NewPacket(data []byte) (*Packet, error)](#NewPacket)
* [type PacketError](#PacketError)
  * [func NewPacketError(message string) *PacketError](#NewPacketError)
  * [func (e *PacketError) Error() string](#PacketError.Error)
* [type SLConn](#SLConn)
  * [func NewSLConn(slink *SLink, opts ...SLConnOpt) *SLConn](#NewSLConn)
  * [func (s *SLConn) Collect(fn CollectFunc) error](#SLConn.Collect)
  * [func (s *SLConn) CollectWithContext(ctx context.Context, fn CollectFunc) error](#SLConn.CollectWithContext)
* [type SLConnOpt](#SLConnOpt)
  * [func SetDelay(v time.Duration) SLConnOpt](#SetDelay)
  * [func SetFlush(v time.Duration) SLConnOpt](#SetFlush)
  * [func SetStateFile(v string) SLConnOpt](#SetStateFile)
* [type SLink](#SLink)
  * [func NewSLink(opts ...SLinkOpt) *SLink](#NewSLink)
  * [func (s *SLink) AddState(stations ...Station)](#SLink.AddState)
  * [func (s *SLink) Collect(fn CollectFunc) error](#SLink.Collect)
  * [func (s *SLink) CollectWithContext(ctx context.Context, fn CollectFunc) error](#SLink.CollectWithContext)
  * [func (s *SLink) SetEnd(t time.Time)](#SLink.SetEnd)
  * [func (s *SLink) SetKeepAlive(d time.Duration)](#SLink.SetKeepAlive)
  * [func (s *SLink) SetNetTo(d time.Duration)](#SLink.SetNetTo)
  * [func (s *SLink) SetSelectors(selectors string)](#SLink.SetSelectors)
  * [func (s *SLink) SetSequence(sequence int)](#SLink.SetSequence)
  * [func (s *SLink) SetStart(t time.Time)](#SLink.SetStart)
  * [func (s *SLink) SetState(stations ...Station)](#SLink.SetState)
  * [func (s *SLink) SetStreams(streams string)](#SLink.SetStreams)
  * [func (s *SLink) SetTimeout(d time.Duration)](#SLink.SetTimeout)
* [type SLinkOpt](#SLinkOpt)
  * [func SetEnd(t time.Time) SLinkOpt](#SetEnd)
  * [func SetKeepAlive(d time.Duration) SLinkOpt](#SetKeepAlive)
  * [func SetNetTo(d time.Duration) SLinkOpt](#SetNetTo)
  * [func SetSelectors(selectors string) SLinkOpt](#SetSelectors)
  * [func SetSequence(sequence int) SLinkOpt](#SetSequence)
  * [func SetServer(v string) SLinkOpt](#SetServer)
  * [func SetStart(t time.Time) SLinkOpt](#SetStart)
  * [func SetState(stations ...Station) SLinkOpt](#SetState)
  * [func SetStreams(streams string) SLinkOpt](#SetStreams)
  * [func SetStrict(strict bool) SLinkOpt](#SetStrict)
  * [func SetTimeout(d time.Duration) SLinkOpt](#SetTimeout)
* [type State](#State)
  * [func (s *State) Add(station Station)](#State.Add)
  * [func (s *State) Find(stn Station) (Station, bool)](#State.Find)
  * [func (s *State) Marshal() ([]byte, error)](#State.Marshal)
  * [func (s *State) ReadFile(path string) error](#State.ReadFile)
  * [func (s *State) Stations() []Station](#State.Stations)
  * [func (s *State) Unmarshal(data []byte) error](#State.Unmarshal)
  * [func (s *State) WriteFile(path string) error](#State.WriteFile)
* [type Station](#Station)
  * [func UnpackStation(seq string, data []byte) Station](#UnpackStation)
  * [func (s Station) Key() Station](#Station.Key)


#### <a name="pkg-files">Package files</a>
[conn.go](/src/target/conn.go) [doc.go](/src/target/doc.go) [info.go](/src/target/info.go) [packet.go](/src/target/packet.go) [slconn.go](/src/target/slconn.go) [slink.go](/src/target/slink.go) [state.go](/src/target/state.go) [station.go](/src/target/station.go) [stream.go](/src/target/stream.go) 


## <a name="pkg-constants">Constants</a>
``` go
const (
    PacketSize = 8 + 512
)
```




## <a name="CollectFunc">type</a> [CollectFunc](/src/target/slink.go?s=4607:4658#L183)
``` go
type CollectFunc func(string, []byte) (bool, error)
```
CollectFunc is a function run on each returned seedlink packet. It should return a true value
to stop collecting data without an error message. A non-nil returned error will also stop
collection but with an assumed errored state.










## <a name="Conn">type</a> [Conn](/src/target/conn.go?s=1434:1581#L57)
``` go
type Conn struct {
    net.Conn
    // contains filtered or unexported fields
}

```






### <a name="NewConn">func</a> [NewConn](/src/target/conn.go?s=1773:1839#L71)
``` go
func NewConn(service string, timeout time.Duration) (*Conn, error)
```
NewConn returns a new connection to the named seedlink server with a given command timeout. It is expected that the
Close function be called when the connection is no longer required.





### <a name="Conn.Collect">func</a> (\*Conn) [Collect](/src/target/conn.go?s=9359:9400#L384)
``` go
func (c *Conn) Collect() (*Packet, error)
```
Collect returns a seedlink packet if available within the optional timout. Any error returned should be
checked that it isn't a timeout, this should be handled as appropriate for the request.




### <a name="Conn.CommandCat">func</a> (\*Conn) [CommandCat](/src/target/conn.go?s=6641:6684#L296)
``` go
func (c *Conn) CommandCat() ([]byte, error)
```
CommandStationList sends a CAT command to the seedlink server.




### <a name="Conn.CommandClose">func</a> (\*Conn) [CommandClose](/src/target/conn.go?s=6492:6537#L291)
``` go
func (c *Conn) CommandClose() ([]byte, error)
```
CommandClose sends a BYE command to the seedlink server.




### <a name="Conn.CommandData">func</a> (\*Conn) [CommandData](/src/target/conn.go?s=8008:8078#L333)
``` go
func (c *Conn) CommandData(sequence string, starttime time.Time) error
```
CommandData sends a DATA command to the seedlink server.




### <a name="Conn.CommandEnd">func</a> (\*Conn) [CommandEnd](/src/target/conn.go?s=9008:9041#L375)
``` go
func (c *Conn) CommandEnd() error
```
CommandEnd sends an END command to the seedlink server.




### <a name="Conn.CommandHello">func</a> (\*Conn) [CommandHello](/src/target/conn.go?s=6349:6394#L286)
``` go
func (c *Conn) CommandHello() ([]byte, error)
```
CommandHello sends a HELLO command to the seedlink server.




### <a name="Conn.CommandId">func</a> (\*Conn) [CommandId](/src/target/conn.go?s=6206:6248#L281)
``` go
func (c *Conn) CommandId() ([]byte, error)
```
CommandId sends an INFO ID command to the seedlink server.




### <a name="Conn.CommandSelect">func</a> (\*Conn) [CommandSelect](/src/target/conn.go?s=7719:7771#L323)
``` go
func (c *Conn) CommandSelect(selection string) error
```
CommandSelect sends a SELECT command to the seedlink server.




### <a name="Conn.CommandStation">func</a> (\*Conn) [CommandStation](/src/target/conn.go?s=6786:6846#L301)
``` go
func (c *Conn) CommandStation(station, network string) error
```
CommandStation sends a STATION command to the seedlink server.




### <a name="Conn.CommandTime">func</a> (\*Conn) [CommandTime](/src/target/conn.go?s=8492:8554#L353)
``` go
func (c *Conn) CommandTime(starttime, endtime time.Time) error
```
CommandTime sends a TIME command to the seedlink server.




### <a name="Conn.GetInfo">func</a> (\*Conn) [GetInfo](/src/target/conn.go?s=5910:5961#L266)
``` go
func (c *Conn) GetInfo(level string) (*Info, error)
```
GetInfo requests the seedlink server return an INFO request for the given level. The results
are returned as a decoded Info pointer, or an error otherwise.




### <a name="Conn.GetInfoLevel">func</a> (\*Conn) [GetInfoLevel](/src/target/conn.go?s=5417:5474#L252)
``` go
func (c *Conn) GetInfoLevel(level string) ([]byte, error)
```
GetInfoLevel requests the seedlink server return an INFO request for the given level.




## <a name="Info">type</a> [Info](/src/target/info.go?s=40:858#L7)
``` go
type Info struct {
    XMLName xml.Name `xml:"seedlink"`

    Software     string `xml:"software,attr"`
    Organization string `xml:"organization,attr"`
    Started      string `xml:"started,attr"`
    Capability   []struct {
        Name string `xml:"name,attr"`
    } `xml:"capability"`
    Station []struct {
        Name        string `xml:"name,attr"`
        Network     string `xml:"network,attr"`
        Description string `xml:"description,attr"`
        BeginSeq    string `xml:"begin_seq,attr"`
        EndSeq      string `xml:"end_seq,attr"`
        StreamCheck string `xml:"stream_check,attr"`
        Stream      []struct {
            Location  string `xml:"location,attr"`
            Seedname  string `xml:"seedname,attr"`
            Type      string `xml:"type,attr"`
            BeginTime string `xml:"begin_time,attr"`
            EndTime   string `xml:"end_time,attr"`
        } `xml:"stream"`
    } `xml:"station"`
}

```









### <a name="Info.Unmarshal">func</a> (\*Info) [Unmarshal](/src/target/info.go?s=860:903#L33)
``` go
func (s *Info) Unmarshal(data []byte) error
```



## <a name="Packet">type</a> [Packet](/src/target/packet.go?s=64:205#L11)
``` go
type Packet struct {
    SL   [2]byte   // ASCII String == "SL"
    Seq  [6]byte   // ASCII sequence number
    Data [512]byte // Fixed size payload
}

```






### <a name="NewPacket">func</a> [NewPacket](/src/target/packet.go?s=411:455#L31)
``` go
func NewPacket(data []byte) (*Packet, error)
```




## <a name="PacketError">type</a> [PacketError](/src/target/packet.go?s=207:250#L17)
``` go
type PacketError struct {
    // contains filtered or unexported fields
}

```






### <a name="NewPacketError">func</a> [NewPacketError](/src/target/packet.go?s=252:300#L21)
``` go
func NewPacketError(message string) *PacketError
```




### <a name="PacketError.Error">func</a> (\*PacketError) [Error](/src/target/packet.go?s=351:387#L27)
``` go
func (e *PacketError) Error() string
```



## <a name="SLConn">type</a> [SLConn](/src/target/slconn.go?s=102:201#L10)
``` go
type SLConn struct {
    *SLink

    StateFile string
    Flush     time.Duration
    Delay     time.Duration
}

```
SLConn is a wrapper around SLink to manage state.







### <a name="NewSLConn">func</a> [NewSLConn](/src/target/slconn.go?s=868:923#L45)
``` go
func NewSLConn(slink *SLink, opts ...SLConnOpt) *SLConn
```
NewSLConn builds a SLConn from a SLink and any extra options.





### <a name="SLConn.Collect">func</a> (\*SLConn) [Collect](/src/target/slconn.go?s=2266:2312#L112)
``` go
func (s *SLConn) Collect(fn CollectFunc) error
```
Collect calls CollectWithContext with a background Context and a handler function.




### <a name="SLConn.CollectWithContext">func</a> (\*SLConn) [CollectWithContext](/src/target/slconn.go?s=1029:1107#L55)
``` go
func (s *SLConn) CollectWithContext(ctx context.Context, fn CollectFunc) error
```



## <a name="SLConnOpt">type</a> [SLConnOpt](/src/target/slconn.go?s=268:296#L19)
``` go
type SLConnOpt func(*SLConn)
```
SLinkOpt is a function for setting SLink internal parameters.







### <a name="SetDelay">func</a> [SetDelay](/src/target/slconn.go?s=714:754#L38)
``` go
func SetDelay(v time.Duration) SLConnOpt
```
SetDelay sets how long to wait until retrying a network connection


### <a name="SetFlush">func</a> [SetFlush](/src/target/slconn.go?s=555:595#L31)
``` go
func SetFlush(v time.Duration) SLConnOpt
```
SetFlush sets how often the state file should be flushed


### <a name="SetStateFile">func</a> [SetStateFile](/src/target/slconn.go?s=346:383#L22)
``` go
func SetStateFile(v string) SLConnOpt
```
SetStateFile sets the connection state file.





## <a name="SLink">type</a> [SLink](/src/target/slink.go?s=156:393#L12)
``` go
type SLink struct {
    Server  string
    Timeout time.Duration

    NetTo     time.Duration
    KeepAlive time.Duration
    Strict    bool

    Start    time.Time
    End      time.Time
    Sequence int

    Streams   string
    Selectors string

    State []Station
}

```
SLink is a wrapper around an SLConn to provide
handling of timeouts and keep alive messages.







### <a name="NewSLink">func</a> [NewSLink](/src/target/slink.go?s=2577:2615#L111)
``` go
func NewSLink(opts ...SLinkOpt) *SLink
```
NewSlink returns a SLink pointer for the given server, optional settings can be passed as SLinkOpt functions.





### <a name="SLink.AddState">func</a> (\*SLink) [AddState](/src/target/slink.go?s=4186:4231#L174)
``` go
func (s *SLink) AddState(stations ...Station)
```
AddState appends the list of station state information.




### <a name="SLink.Collect">func</a> (\*SLink) [Collect](/src/target/slink.go?s=7889:7934#L301)
``` go
func (s *SLink) Collect(fn CollectFunc) error
```
Collect calls CollectWithContext with a background Context and a handler function.




### <a name="SLink.CollectWithContext">func</a> (\*SLink) [CollectWithContext](/src/target/slink.go?s=5499:5576#L194)
``` go
func (s *SLink) CollectWithContext(ctx context.Context, fn CollectFunc) error
```
CollectWithContext makes a connection to the seedlink server, recovers initial client information and
the sets the connection into streaming mode. Recovered packets are passed to a given function
to process, if this function returns a true value or a non-nil error value the collection will
stop and the function will return.
If a call returns with a timeout error a check is made whether a keepalive is needed or whether
the function should return as no data has been received for an extended period of time. It is
assumed the calling function will attempt a reconnection with an updated set of options, specifically
any start or end time parameters. The Context parameter can be used to to cancel the data collection
independent of the function as this may never be called if no appropriate has been received.




### <a name="SLink.SetEnd">func</a> (\*SLink) [SetEnd](/src/target/slink.go?s=3620:3655#L154)
``` go
func (s *SLink) SetEnd(t time.Time)
```
SetEndTime sets the initial end time of the request.




### <a name="SLink.SetKeepAlive">func</a> (\*SLink) [SetKeepAlive](/src/target/slink.go?s=3237:3282#L139)
``` go
func (s *SLink) SetKeepAlive(d time.Duration)
```
SetKeepAlive sets the time interval needed without any packets for
a check message is sent.




### <a name="SLink.SetNetTo">func</a> (\*SLink) [SetNetTo](/src/target/slink.go?s=3079:3120#L133)
``` go
func (s *SLink) SetNetTo(d time.Duration)
```
SetNetTo sets the overall timeout after which a reconnection is tried.




### <a name="SLink.SetSelectors">func</a> (\*SLink) [SetSelectors](/src/target/slink.go?s=3891:3937#L164)
``` go
func (s *SLink) SetSelectors(selectors string)
```
SetSelectors sets the channel selectors used for seedlink connections.




### <a name="SLink.SetSequence">func</a> (\*SLink) [SetSequence](/src/target/slink.go?s=3369:3410#L144)
``` go
func (s *SLink) SetSequence(sequence int)
```
SetSequence sets the start sequence for the initial request.




### <a name="SLink.SetStart">func</a> (\*SLink) [SetStart](/src/target/slink.go?s=3502:3539#L149)
``` go
func (s *SLink) SetStart(t time.Time)
```
SetStartTime sets the initial starting time of the request.




### <a name="SLink.SetState">func</a> (\*SLink) [SetState](/src/target/slink.go?s=4032:4077#L169)
``` go
func (s *SLink) SetState(stations ...Station)
```
SetState sets the default list of station state information.




### <a name="SLink.SetStreams">func</a> (\*SLink) [SetStreams](/src/target/slink.go?s=3748:3790#L159)
``` go
func (s *SLink) SetStreams(streams string)
```
SetStreams sets the channel streams used for seedlink connections.




### <a name="SLink.SetTimeout">func</a> (\*SLink) [SetTimeout](/src/target/slink.go?s=2941:2984#L128)
``` go
func (s *SLink) SetTimeout(d time.Duration)
```
SetTimeout sets the timeout value used for connection requests.




## <a name="SLinkOpt">type</a> [SLinkOpt](/src/target/slink.go?s=460:486#L31)
``` go
type SLinkOpt func(*SLink)
```
SLinkOpt is a function for setting SLink internal parameters.







### <a name="SetEnd">func</a> [SetEnd](/src/target/slink.go?s=1593:1626#L76)
``` go
func SetEnd(t time.Time) SLinkOpt
```
SetEndTime sets the end of the initial request from the seedlink server.


### <a name="SetKeepAlive">func</a> [SetKeepAlive](/src/target/slink.go?s=1096:1139#L55)
``` go
func SetKeepAlive(d time.Duration) SLinkOpt
```
SetKeepAlive sets the time to send an ID message to server if no packets have been received.


### <a name="SetNetTo">func</a> [SetNetTo](/src/target/slink.go?s=913:952#L48)
``` go
func SetNetTo(d time.Duration) SLinkOpt
```
SetNetTo sets the time to after which the connection is closed after no packets have been received.


### <a name="SetSelectors">func</a> [SetSelectors](/src/target/slink.go?s=1943:1987#L90)
``` go
func SetSelectors(selectors string) SLinkOpt
```
SetSelectors sets the default list of selectors to use for seedlink stream requests.


### <a name="SetSequence">func</a> [SetSequence](/src/target/slink.go?s=1255:1294#L62)
``` go
func SetSequence(sequence int) SLinkOpt
```
SetSequence sets the start sequence for the initial request.


### <a name="SetServer">func</a> [SetServer](/src/target/slink.go?s=556:589#L34)
``` go
func SetServer(v string) SLinkOpt
```
SetServer sets the seedlink server in the form of "host<:port>".


### <a name="SetStart">func</a> [SetStart](/src/target/slink.go?s=1428:1463#L69)
``` go
func SetStart(t time.Time) SLinkOpt
```
SetStart sets the start of the initial request from the seedlink server.


### <a name="SetState">func</a> [SetState](/src/target/slink.go?s=2152:2195#L97)
``` go
func SetState(stations ...Station) SLinkOpt
```
SetState sets the default list of station state information, only used during the initial connection.


### <a name="SetStreams">func</a> [SetStreams](/src/target/slink.go?s=1759:1799#L83)
``` go
func SetStreams(streams string) SLinkOpt
```
SetStreams sets the list of stations and streams to from the seedlink server.


### <a name="SetStrict">func</a> [SetStrict](/src/target/slink.go?s=2374:2410#L104)
``` go
func SetStrict(strict bool) SLinkOpt
```
SetStrict sets whether a package error should restart the collection system, rather than be skipped.


### <a name="SetTimeout">func</a> [SetTimeout](/src/target/slink.go?s=719:760#L41)
``` go
func SetTimeout(d time.Duration) SLinkOpt
```
SetTimeout sets the timeout for seedlink server commands and packet requests.





## <a name="State">type</a> [State](/src/target/state.go?s=154:236#L12)
``` go
type State struct {
    // contains filtered or unexported fields
}

```
State maintains the current state information for a seedlink connection.










### <a name="State.Add">func</a> (\*State) [Add](/src/target/state.go?s=865:901#L45)
``` go
func (s *State) Add(station Station)
```
Add inserts or updates the station collection details into the connection state.




### <a name="State.Find">func</a> (\*State) [Find](/src/target/state.go?s=1172:1221#L58)
``` go
func (s *State) Find(stn Station) (Station, bool)
```



### <a name="State.Marshal">func</a> (\*State) [Marshal](/src/target/state.go?s=1719:1760#L89)
``` go
func (s *State) Marshal() ([]byte, error)
```



### <a name="State.ReadFile">func</a> (\*State) [ReadFile](/src/target/state.go?s=1881:1924#L99)
``` go
func (s *State) ReadFile(path string) error
```



### <a name="State.Stations">func</a> (\*State) [Stations](/src/target/state.go?s=311:347#L20)
``` go
func (s *State) Stations() []Station
```
Stations returns a sorted slice of current station state information.




### <a name="State.Unmarshal">func</a> (\*State) [Unmarshal](/src/target/state.go?s=1513:1557#L75)
``` go
func (s *State) Unmarshal(data []byte) error
```



### <a name="State.WriteFile">func</a> (\*State) [WriteFile](/src/target/state.go?s=2073:2117#L113)
``` go
func (s *State) WriteFile(path string) error
```



## <a name="Station">type</a> [Station](/src/target/station.go?s=167:345#L11)
``` go
type Station struct {
    Network   string    `json:"network"`
    Station   string    `json:"station"`
    Sequence  int       `json:"sequence"`
    Timestamp time.Time `json:"timestamp"`
}

```
Station stores the latest state information for the given network and station combination.







### <a name="UnpackStation">func</a> [UnpackStation](/src/target/station.go?s=620:671#L27)
``` go
func UnpackStation(seq string, data []byte) Station
```
UnpackStation builds a Station based on a raw miniseed block header.





### <a name="Station.Key">func</a> (Station) [Key](/src/target/station.go?s=448:478#L19)
``` go
func (s Station) Key() Station
```
Key returns a blank Station except for the Network and Station entries, this useful as a map key.








- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
