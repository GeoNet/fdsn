

# ms
`import "github.com/GeoNet/kit/seis/ms"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
The ms module has been writen as a lightweight replacement for some parts of the libmseed C library.




## <a name="pkg-index">Index</a>
* [Constants](#pkg-constants)
* [func EncodeBTime(at BTime) []byte](#EncodeBTime)
* [func EncodeBlockette1000(blk Blockette1000) []byte](#EncodeBlockette1000)
* [func EncodeBlockette1001(blk Blockette1001) []byte](#EncodeBlockette1001)
* [func EncodeBlocketteHeader(hdr BlocketteHeader) []byte](#EncodeBlocketteHeader)
* [func EncodeRecordHeader(hdr RecordHeader) []byte](#EncodeRecordHeader)
* [type BTime](#BTime)
  * [func DecodeBTime(data []byte) BTime](#DecodeBTime)
  * [func NewBTime(t time.Time) BTime](#NewBTime)
  * [func (b BTime) Marshal() ([]byte, error)](#BTime.Marshal)
  * [func (b BTime) Time() time.Time](#BTime.Time)
  * [func (b *BTime) Unmarshal(data []byte) error](#BTime.Unmarshal)
* [type Blockette1000](#Blockette1000)
  * [func DecodeBlockette1000(data []byte) Blockette1000](#DecodeBlockette1000)
  * [func (b Blockette1000) Marshal() ([]byte, error)](#Blockette1000.Marshal)
  * [func (b *Blockette1000) Unmarshal(data []byte) error](#Blockette1000.Unmarshal)
* [type Blockette1001](#Blockette1001)
  * [func DecodeBlockette1001(data []byte) Blockette1001](#DecodeBlockette1001)
  * [func (b Blockette1001) Marshal() ([]byte, error)](#Blockette1001.Marshal)
  * [func (b *Blockette1001) Unmarshal(data []byte) error](#Blockette1001.Unmarshal)
* [type BlocketteHeader](#BlocketteHeader)
  * [func DecodeBlocketteHeader(data []byte) BlocketteHeader](#DecodeBlocketteHeader)
  * [func (h BlocketteHeader) Marshal() ([]byte, error)](#BlocketteHeader.Marshal)
  * [func (h *BlocketteHeader) Unmarshal(data []byte) error](#BlocketteHeader.Unmarshal)
* [type Encoding](#Encoding)
* [type Record](#Record)
  * [func NewRecord(buf []byte) (*Record, error)](#NewRecord)
  * [func (m Record) BlockSize() int](#Record.BlockSize)
  * [func (m Record) ByteOrder() WordOrder](#Record.ByteOrder)
  * [func (m Record) Bytes() ([]byte, error)](#Record.Bytes)
  * [func (m Record) Encoding() Encoding](#Record.Encoding)
  * [func (m Record) EndTime() time.Time](#Record.EndTime)
  * [func (m Record) Float64s() ([]float64, error)](#Record.Float64s)
  * [func (m Record) Int32s() ([]int32, error)](#Record.Int32s)
  * [func (m Record) SampleType() SampleType](#Record.SampleType)
  * [func (m Record) StartTime() time.Time](#Record.StartTime)
  * [func (m Record) String() string](#Record.String)
  * [func (m Record) Strings() ([]string, error)](#Record.Strings)
  * [func (m *Record) Unpack(buf []byte) error](#Record.Unpack)
* [type RecordHeader](#RecordHeader)
  * [func DecodeRecordHeader(data []byte) RecordHeader](#DecodeRecordHeader)
  * [func (h RecordHeader) Channel() string](#RecordHeader.Channel)
  * [func (h RecordHeader) Correction() time.Duration](#RecordHeader.Correction)
  * [func (h RecordHeader) IsValid() bool](#RecordHeader.IsValid)
  * [func (h RecordHeader) Less(hdr RecordHeader) bool](#RecordHeader.Less)
  * [func (h RecordHeader) Location() string](#RecordHeader.Location)
  * [func (h RecordHeader) Marshal() ([]byte, error)](#RecordHeader.Marshal)
  * [func (h RecordHeader) Network() string](#RecordHeader.Network)
  * [func (h RecordHeader) SampleCount() int](#RecordHeader.SampleCount)
  * [func (h RecordHeader) SamplePeriod() time.Duration](#RecordHeader.SamplePeriod)
  * [func (h RecordHeader) SampleRate() float64](#RecordHeader.SampleRate)
  * [func (h RecordHeader) SeqNumber() int](#RecordHeader.SeqNumber)
  * [func (h *RecordHeader) SetChannel(s string)](#RecordHeader.SetChannel)
  * [func (h *RecordHeader) SetCorrection(correction time.Duration, applied bool)](#RecordHeader.SetCorrection)
  * [func (h *RecordHeader) SetLocation(s string)](#RecordHeader.SetLocation)
  * [func (h *RecordHeader) SetNetwork(s string)](#RecordHeader.SetNetwork)
  * [func (h *RecordHeader) SetSeqNumber(no int)](#RecordHeader.SetSeqNumber)
  * [func (h *RecordHeader) SetStartTime(t time.Time)](#RecordHeader.SetStartTime)
  * [func (h *RecordHeader) SetStation(s string)](#RecordHeader.SetStation)
  * [func (h RecordHeader) SrcName(quality bool) string](#RecordHeader.SrcName)
  * [func (h RecordHeader) StartTime() time.Time](#RecordHeader.StartTime)
  * [func (h RecordHeader) Station() string](#RecordHeader.Station)
  * [func (h *RecordHeader) Unmarshal(data []byte) error](#RecordHeader.Unmarshal)
* [type SampleType](#SampleType)
* [type WordOrder](#WordOrder)


#### <a name="pkg-files">Package files</a>
[blockette.go](/src/target/blockette.go) [btime.go](/src/target/btime.go) [decode.go](/src/target/decode.go) [doc.go](/src/target/doc.go) [header.go](/src/target/header.go) [record.go](/src/target/record.go) [steim.go](/src/target/steim.go) [unpack.go](/src/target/unpack.go) 


## <a name="pkg-constants">Constants</a>
``` go
const (
    BlocketteHeaderSize = 4
    Blockette1000Size   = 4
    Blockette1001Size   = 4
)
```
``` go
const BTimeSize = 10
```
BTimeSize is the fixed size of an encoded BTime.

``` go
const RecordHeaderSize = 48
```
RecordHeaderSize is the miniseed block fixed header length.




## <a name="EncodeBTime">func</a> [EncodeBTime](/src/target/btime.go?s=1217:1250#L66)
``` go
func EncodeBTime(at BTime) []byte
```
EncodeBTime converts a BTime into a byte slice.



## <a name="EncodeBlockette1000">func</a> [EncodeBlockette1000](/src/target/blockette.go?s=1845:1895#L78)
``` go
func EncodeBlockette1000(blk Blockette1000) []byte
```
EncodeBlockette1000 converts a Blockette1000 into a byte slice.



## <a name="EncodeBlockette1001">func</a> [EncodeBlockette1001](/src/target/blockette.go?s=2957:3007#L125)
``` go
func EncodeBlockette1001(blk Blockette1001) []byte
```


## <a name="EncodeBlocketteHeader">func</a> [EncodeBlocketteHeader](/src/target/blockette.go?s=701:755#L32)
``` go
func EncodeBlocketteHeader(hdr BlocketteHeader) []byte
```
EncodeBlocketteHeader converts a BlocketteHeader into a byte slice.



## <a name="EncodeRecordHeader">func</a> [EncodeRecordHeader](/src/target/header.go?s=7610:7658#L278)
``` go
func EncodeRecordHeader(hdr RecordHeader) []byte
```
EncodeRecordHeader converts a RecordHeader into a byte slice.




## <a name="BTime">type</a> [BTime](/src/target/btime.go?s=170:291#L12)
``` go
type BTime struct {
    Year   uint16
    Doy    uint16
    Hour   uint8
    Minute uint8
    Second uint8
    Unused byte
    S0001  uint16
}

```
BTime is the SEED Representation of Time.







### <a name="DecodeBTime">func</a> [DecodeBTime](/src/target/btime.go?s=870:905#L49)
``` go
func DecodeBTime(data []byte) BTime
```
DecodeBTime returns a BTime from a byte slice.


### <a name="NewBTime">func</a> [NewBTime](/src/target/btime.go?s=577:609#L37)
``` go
func NewBTime(t time.Time) BTime
```
NewBTime builds a BTime from a time.Time.





### <a name="BTime.Marshal">func</a> (BTime) [Marshal](/src/target/btime.go?s=1642:1682#L88)
``` go
func (b BTime) Marshal() ([]byte, error)
```



### <a name="BTime.Time">func</a> (BTime) [Time](/src/target/btime.go?s=336:367#L23)
``` go
func (b BTime) Time() time.Time
```
Time converts a BTime into a time.Time.




### <a name="BTime.Unmarshal">func</a> (\*BTime) [Unmarshal](/src/target/btime.go?s=1556:1600#L83)
``` go
func (b *BTime) Unmarshal(data []byte) error
```



## <a name="Blockette1000">type</a> [Blockette1000](/src/target/blockette.go?s=1379:1488#L56)
``` go
type Blockette1000 struct {
    Encoding     uint8
    WordOrder    uint8
    RecordLength uint8
    Reserved     uint8
}

```
Blockette1000 is a Data Only Seed Blockette (excluding header).







### <a name="DecodeBlockette1000">func</a> [DecodeBlockette1000](/src/target/blockette.go?s=1556:1607#L64)
``` go
func DecodeBlockette1000(data []byte) Blockette1000
```
DecodeBlockette1000 returns a Blockette1000 from a byte slice.





### <a name="Blockette1000.Marshal">func</a> (Blockette1000) [Marshal](/src/target/blockette.go?s=2351:2399#L99)
``` go
func (b Blockette1000) Marshal() ([]byte, error)
```
Marshal converts a Blockette1000 into a byte slice.




### <a name="Blockette1000.Unmarshal">func</a> (\*Blockette1000) [Unmarshal](/src/target/blockette.go?s=2194:2246#L93)
``` go
func (b *Blockette1000) Unmarshal(data []byte) error
```
Unmarshal converts a byte slice into the Blockette1000




## <a name="Blockette1001">type</a> [Blockette1001](/src/target/blockette.go?s=2510:2657#L104)
``` go
type Blockette1001 struct {
    TimingQuality uint8
    MicroSec      int8 //Increased accuracy for starttime
    Reserved      uint8
    FrameCount    uint8
}

```
Blockette1001 is a "Data Extension Blockette" (excluding header).







### <a name="DecodeBlockette1001">func</a> [DecodeBlockette1001](/src/target/blockette.go?s=2725:2776#L112)
``` go
func DecodeBlockette1001(data []byte) Blockette1001
```
DecodeBlockette1001 returns a Blockette1001 from a byte slice.





### <a name="Blockette1001.Marshal">func</a> (Blockette1001) [Marshal](/src/target/blockette.go?s=3444:3492#L146)
``` go
func (b Blockette1001) Marshal() ([]byte, error)
```
Marshal converts a Blockette1001 into a byte slice.




### <a name="Blockette1001.Unmarshal">func</a> (\*Blockette1001) [Unmarshal](/src/target/blockette.go?s=3287:3339#L140)
``` go
func (b *Blockette1001) Unmarshal(data []byte) error
```
Unmarshal converts a byte slice into the Blockette1001




## <a name="BlocketteHeader">type</a> [BlocketteHeader](/src/target/blockette.go?s=194:316#L14)
``` go
type BlocketteHeader struct {
    BlocketteType uint16
    NextBlockette uint16 // Byte of next blockette, 0 if last blockette
}

```
BlocketteHeader stores the header of each miniseed blockette.







### <a name="DecodeBlocketteHeader">func</a> [DecodeBlocketteHeader](/src/target/blockette.go?s=388:443#L20)
``` go
func DecodeBlocketteHeader(data []byte) BlocketteHeader
```
DecodeBlocketteHeader returns a BlocketteHeader from a byte slice.





### <a name="BlocketteHeader.Marshal">func</a> (BlocketteHeader) [Marshal](/src/target/blockette.go?s=1218:1268#L51)
``` go
func (h BlocketteHeader) Marshal() ([]byte, error)
```
Marshal converts a BlocketteHeader into a byte slice.




### <a name="BlocketteHeader.Unmarshal">func</a> (\*BlocketteHeader) [Unmarshal](/src/target/blockette.go?s=1055:1109#L45)
``` go
func (h *BlocketteHeader) Unmarshal(data []byte) error
```
Unmarshal converts a byte slice into the BlocketteHeader




## <a name="Encoding">type</a> [Encoding](/src/target/record.go?s=50:69#L9)
``` go
type Encoding uint8
```

``` go
const (
    EncodingASCII      Encoding = 0
    EncodingInt32      Encoding = 3
    EncodingIEEEFloat  Encoding = 4
    EncodingIEEEDouble Encoding = 5
    EncodingSTEIM1     Encoding = 10
    EncodingSTEIM2     Encoding = 11
)
```









## <a name="Record">type</a> [Record](/src/target/record.go?s=552:671#L37)
``` go
type Record struct {
    RecordHeader

    B1000 Blockette1000 //If Present
    B1001 Blockette1001 //If Present

    Data []byte
}

```






### <a name="NewRecord">func</a> [NewRecord](/src/target/record.go?s=838:881#L48)
``` go
func NewRecord(buf []byte) (*Record, error)
```
NewMSRecord decodes and unpacks the record samples from a byte slice and returns a Record pointer,
or an empty pointer and an error if it could not be decoded.





### <a name="Record.BlockSize">func</a> (Record) [BlockSize](/src/target/record.go?s=2200:2231#L96)
``` go
func (m Record) BlockSize() int
```
PacketSize returns the length of the packet




### <a name="Record.ByteOrder">func</a> (Record) [ByteOrder](/src/target/record.go?s=2492:2529#L109)
``` go
func (m Record) ByteOrder() WordOrder
```
ByteOrder returns the miniseed data byte order.




### <a name="Record.Bytes">func</a> (Record) [Bytes](/src/target/unpack.go?s=1787:1826#L56)
``` go
func (m Record) Bytes() ([]byte, error)
```



### <a name="Record.Encoding">func</a> (Record) [Encoding](/src/target/record.go?s=2365:2400#L104)
``` go
func (m Record) Encoding() Encoding
```
Encoding returns the miniseed data format encoding.




### <a name="Record.EndTime">func</a> (Record) [EndTime](/src/target/record.go?s=1895:1930#L85)
``` go
func (m Record) EndTime() time.Time
```
EndTime returns the calculated time of the last sample.




### <a name="Record.Float64s">func</a> (Record) [Float64s](/src/target/unpack.go?s=3904:3949#L120)
``` go
func (m Record) Float64s() ([]float64, error)
```



### <a name="Record.Int32s">func</a> (Record) [Int32s](/src/target/unpack.go?s=2502:2543#L83)
``` go
func (m Record) Int32s() ([]int32, error)
```



### <a name="Record.SampleType">func</a> (Record) [SampleType](/src/target/record.go?s=2721:2760#L119)
``` go
func (m Record) SampleType() SampleType
```
SampleType returns the type of samples decoded, or UnknownType if no data has been decoded.




### <a name="Record.StartTime">func</a> (Record) [StartTime](/src/target/record.go?s=1653:1690#L74)
``` go
func (m Record) StartTime() time.Time
```
StartTime returns the calculated time of the first sample.




### <a name="Record.String">func</a> (Record) [String](/src/target/record.go?s=1083:1114#L59)
``` go
func (m Record) String() string
```
String implements the Stringer interface and provides a short summary of the miniseed record header.




### <a name="Record.Strings">func</a> (Record) [Strings](/src/target/unpack.go?s=2033:2076#L65)
``` go
func (m Record) Strings() ([]string, error)
```



### <a name="Record.Unpack">func</a> (\*Record) [Unpack](/src/target/unpack.go?s=97:138#L10)
``` go
func (m *Record) Unpack(buf []byte) error
```
Unpack decodes the record from a byte slice.




## <a name="RecordHeader">type</a> [RecordHeader](/src/target/header.go?s=209:1444#L15)
``` go
type RecordHeader struct {
    SequenceNumber [6]byte // ASCII String representing a 6 digit number

    DataQualityIndicator byte // ASCII: D, R, Q or M
    ReservedByte         byte // ASCII: Space

    // These are ascii strings
    StationIdentifier  [5]byte // ASCII: Left justify and pad with spaces
    LocationIdentifier [2]byte // ASCII: Left justify and pad with spaces
    ChannelIdentifier  [3]byte // ASCII: Left justify and pad with spaces
    NetworkIdentifier  [2]byte // ASCII: Left justify and pad with spaces

    RecordStartTime      BTime  // Start time of record
    NumberOfSamples      uint16 // Number of Samples in the data block which may or may not be unpacked.
    SampleRateFactor     int16  // >0: Samples/Second <0: Second/Samples 0: Seconds/Sample, ASCII/OPAQUE DATA records
    SampleRateMultiplier int16  // >0: Multiplication Factor <0: Division Factor

    // Flags are bit masks
    ActivityFlags    byte
    IOAndClockFlags  byte
    DataQualityFlags byte

    NumberOfBlockettesThatFollow uint8 // Total number of blockettes that follow

    TimeCorrection  int32  // 0.0001 second units
    BeginningOfData uint16 // Offset in bytes to the beginning of data.
    FirstBlockette  uint16 // Offset in bytes to the first data blockette in the data record.
}

```






### <a name="DecodeRecordHeader">func</a> [DecodeRecordHeader](/src/target/header.go?s=6337:6386#L224)
``` go
func DecodeRecordHeader(data []byte) RecordHeader
```
DecodeRecordHeader returns a RecordHeader from a byte slice.





### <a name="RecordHeader.Channel">func</a> (RecordHeader) [Channel](/src/target/header.go?s=2150:2188#L73)
``` go
func (h RecordHeader) Channel() string
```



### <a name="RecordHeader.Correction">func</a> (RecordHeader) [Correction](/src/target/header.go?s=3235:3283#L112)
``` go
func (h RecordHeader) Correction() time.Duration
```



### <a name="RecordHeader.IsValid">func</a> (RecordHeader) [IsValid](/src/target/header.go?s=4334:4370#L144)
``` go
func (h RecordHeader) IsValid() bool
```
IsValid performs a simple consistency check of the RecordHeader contents.




### <a name="RecordHeader.Less">func</a> (RecordHeader) [Less](/src/target/header.go?s=5464:5513#L190)
``` go
func (h RecordHeader) Less(hdr RecordHeader) bool
```
Less can be used for sorting record blocks.




### <a name="RecordHeader.Location">func</a> (RecordHeader) [Location](/src/target/header.go?s=1932:1971#L64)
``` go
func (h RecordHeader) Location() string
```



### <a name="RecordHeader.Marshal">func</a> (RecordHeader) [Marshal](/src/target/header.go?s=8706:8753#L315)
``` go
func (h RecordHeader) Marshal() ([]byte, error)
```



### <a name="RecordHeader.Network">func</a> (RecordHeader) [Network](/src/target/header.go?s=2364:2402#L82)
``` go
func (h RecordHeader) Network() string
```



### <a name="RecordHeader.SampleCount">func</a> (RecordHeader) [SampleCount](/src/target/header.go?s=3978:4017#L134)
``` go
func (h RecordHeader) SampleCount() int
```
SampleCount returns the number of samples in the record, independent of whether they are decoded or not.




### <a name="RecordHeader.SamplePeriod">func</a> (RecordHeader) [SamplePeriod](/src/target/header.go?s=4126:4176#L139)
``` go
func (h RecordHeader) SamplePeriod() time.Duration
```
SamplePeriod converts the sample rate into a time interval, or zero.




### <a name="RecordHeader.SampleRate">func</a> (RecordHeader) [SampleRate](/src/target/header.go?s=3749:3791#L129)
``` go
func (h RecordHeader) SampleRate() float64
```
SampleRate returns the decoded header sampling rate in samples per second.




### <a name="RecordHeader.SeqNumber">func</a> (RecordHeader) [SeqNumber](/src/target/header.go?s=1446:1483#L44)
``` go
func (h RecordHeader) SeqNumber() int
```



### <a name="RecordHeader.SetChannel">func</a> (\*RecordHeader) [SetChannel](/src/target/header.go?s=2251:2294#L76)
``` go
func (h *RecordHeader) SetChannel(s string)
```



### <a name="RecordHeader.SetCorrection">func</a> (\*RecordHeader) [SetCorrection](/src/target/header.go?s=2949:3025#L102)
``` go
func (h *RecordHeader) SetCorrection(correction time.Duration, applied bool)
```



### <a name="RecordHeader.SetLocation">func</a> (\*RecordHeader) [SetLocation](/src/target/header.go?s=2035:2079#L67)
``` go
func (h *RecordHeader) SetLocation(s string)
```



### <a name="RecordHeader.SetNetwork">func</a> (\*RecordHeader) [SetNetwork](/src/target/header.go?s=2465:2508#L85)
``` go
func (h *RecordHeader) SetNetwork(s string)
```



### <a name="RecordHeader.SetSeqNumber">func</a> (\*RecordHeader) [SetSeqNumber](/src/target/header.go?s=1609:1652#L51)
``` go
func (h *RecordHeader) SetSeqNumber(no int)
```



### <a name="RecordHeader.SetStartTime">func</a> (\*RecordHeader) [SetStartTime](/src/target/header.go?s=2862:2910#L98)
``` go
func (h *RecordHeader) SetStartTime(t time.Time)
```



### <a name="RecordHeader.SetStation">func</a> (\*RecordHeader) [SetStation](/src/target/header.go?s=1819:1862#L58)
``` go
func (h *RecordHeader) SetStation(s string)
```



### <a name="RecordHeader.SrcName">func</a> (RecordHeader) [SrcName](/src/target/header.go?s=2578:2628#L91)
``` go
func (h RecordHeader) SrcName(quality bool) string
```



### <a name="RecordHeader.StartTime">func</a> (RecordHeader) [StartTime](/src/target/header.go?s=3354:3397#L116)
``` go
func (h RecordHeader) StartTime() time.Time
```



### <a name="RecordHeader.Station">func</a> (RecordHeader) [Station](/src/target/header.go?s=1718:1756#L55)
``` go
func (h RecordHeader) Station() string
```



### <a name="RecordHeader.Unmarshal">func</a> (\*RecordHeader) [Unmarshal](/src/target/header.go?s=8606:8657#L310)
``` go
func (h *RecordHeader) Unmarshal(data []byte) error
```



## <a name="SampleType">type</a> [SampleType](/src/target/record.go?s=371:391#L27)
``` go
type SampleType byte
```

``` go
const (
    UnknownType SampleType = 0
    ByteType    SampleType = 'a'
    IntegerType SampleType = 'i'
    FloatType   SampleType = 'f'
    DoubleType  SampleType = 'd'
)
```









## <a name="WordOrder">type</a> [WordOrder](/src/target/record.go?s=282:302#L20)
``` go
type WordOrder uint8
```

``` go
const (
    LittleEndian WordOrder = 0
    BigEndian    WordOrder = 1
)
```













- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
