package mseed

//#cgo CFLAGS: -I/usr/local/include
//#cgo LDFLAGS: /usr/local/lib/libmseed.a
//#include <libmseed.h>
import "C"

import (
	"errors"
	"strings"
	"time"
	"unsafe"
)

type MSRecord _Ctype_struct_MSRecord_s

func NewMSRecord() *MSRecord {
	return (*MSRecord)(C.msr_init(nil))
}
func FreeMSRecord(m *MSRecord) {
	C.msr_free((**_Ctype_struct_MSRecord_s)(unsafe.Pointer(&m)))
}
func (m *MSRecord) SequenceNumber() int32 {
	return int32(m.sequence_number)
}
func (m *MSRecord) Network() string {
	return strings.TrimRight(C.GoStringN(&m.network[0], 2), " ")
}
func (m *MSRecord) Station() string {
	return strings.TrimRight(C.GoStringN(&m.station[0], 5), " ")
}
func (m *MSRecord) Location() string {
	return strings.TrimRight(C.GoStringN(&m.location[0], 2), " ")
}
func (m *MSRecord) Channel() string {
	return strings.TrimRight(C.GoStringN(&m.channel[0], 3), " ")
}
func (m *MSRecord) Dataquality() byte {
	return byte(m.dataquality)
}
func (m *MSRecord) Starttime() time.Time {
	sec := int64(m.starttime) / 1000000
	nsec := 1000 * (int64(m.starttime) % 1000000)
	return time.Unix(sec, nsec).UTC()
}
func (m *MSRecord) Samprate() float32 {
	return float32(m.samprate)
}
func (m *MSRecord) Samplecnt() int32 {
	return int32(m.samplecnt)
}
func (m *MSRecord) Encoding() int8 {
	return int8(m.encoding)
}
func (m *MSRecord) Byteorder() int8 {
	return int8(m.byteorder)
}
func (m *MSRecord) Numsamples() int32 {
	return int32(m.numsamples)
}
func (m *MSRecord) Sampletype() byte {
	return byte(m.sampletype)
}
func (m *MSRecord) MsgSamples() (string, error) {
	if m.sampletype != 'a' {
		return "", errors.New("not an ascii formatted record")
	}
	return C.GoStringN((*_Ctype_char)(m.datasamples), C.int(m.numsamples)), nil
}
func (m *MSRecord) DataSamples() ([]int32, error) {
	if m.sampletype == 'a' {
		return nil, errors.New("not a numerical formatted record")
	}
	samples := make([]int32, m.numsamples)

	switch {
	case m.sampletype == 'i':
		ptr := (*[1 << 30](C.int))(unsafe.Pointer(m.datasamples))
		for i := 0; i < int(m.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	case m.sampletype == 'f':
		ptr := (*[1 << 30](C.float))(unsafe.Pointer(m.datasamples))
		for i := 0; i < int(m.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	case m.sampletype == 'd':
		ptr := (*[1 << 30](C.double))(unsafe.Pointer(m.datasamples))
		for i := 0; i < int(m.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	default:
		return nil, errors.New("format not coded")
	}

	return samples, nil
}

func (m *MSRecord) Endtime() time.Time {
	endtime := C.msr_endtime((*_Ctype_struct_MSRecord_s)(m))
	sec := int64(endtime) / 1000000
	nsec := 1000 * (int64(endtime) % 1000000)
	return time.Unix(sec, nsec).UTC()
}

func (m *MSRecord) Print(details int8) {
	C.msr_print((*_Ctype_struct_MSRecord_s)(m), C.flag(details))
}

func (m *MSRecord) Unpack(buf []byte, maxlen int, dataflag int, verbose int) {
	C.msr_unpack(((*C.char)(unsafe.Pointer(&buf[0]))), C.int(maxlen), (**_Ctype_struct_MSRecord_s)((unsafe.Pointer)(&m)), C.flag(dataflag), C.flag(verbose))
}

func (m *MSRecord) SrcName(quality int8) string {
	csrcname := C.CString("NN_SSSSS_LL_CHA_Q_0")
	defer C.free(unsafe.Pointer(csrcname))
	C.msr_srcname((*_Ctype_struct_MSRecord_s)(m), csrcname, C.flag(quality))
	return C.GoString(csrcname)
}
