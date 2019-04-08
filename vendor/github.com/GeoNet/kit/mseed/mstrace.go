//nolint //cgo generates code that doesn't pass linting
package mseed

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libmseed
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libmseed/libmseed.a
//#include <libmseed.h>
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"
)

type MSTrace C.MSTrace

func NewMSTrace() *MSTrace {
	return (*MSTrace)(C.mst_init(nil))
}

func FreeMSTrace(t *MSTrace) {
	C.mst_free((**C.struct_MSTrace_s)((unsafe.Pointer)(&t)))
}

func (t *MSTrace) String() string {
	return fmt.Sprintf("%-14s %s %s %g %d", t.SrcName(0), t.Starttime(), t.Endtime(), t.Samprate(), t.Samplecnt())
}

func (t *MSTrace) Network() string {
	return strings.TrimRight(C.GoStringN(&t.network[0], 2), " ")
}
func (t *MSTrace) Station() string {
	return strings.TrimRight(C.GoStringN(&t.station[0], 5), " ")
}
func (t *MSTrace) Location() string {
	return strings.TrimRight(C.GoStringN(&t.location[0], 2), " ")
}
func (t *MSTrace) Channel() string {
	return strings.TrimRight(C.GoStringN(&t.channel[0], 3), " ")
}
func (t *MSTrace) Dataquality() byte {
	return byte(t.dataquality)
}

func (t *MSTrace) Starttime() time.Time {
	sec := int64(t.starttime) / 1000000
	nsec := 1000 * (int64(t.starttime) % 1000000)
	return time.Unix(sec, nsec).UTC()
}

func (t *MSTrace) Samprate() float32 {
	return float32(t.samprate)
}
func (t *MSTrace) Samplecnt() int32 {
	return int32(t.samplecnt)
}
func (t *MSTrace) Numsamples() int32 {
	return int32(t.numsamples)
}
func (t *MSTrace) Sampletype() byte {
	return byte(t.sampletype)
}

func (t *MSTrace) MsgSamples() (string, error) {
	if t.sampletype != 'a' {
		return "", errors.New("not an ascii formatted record")
	}
	return C.GoStringN((*C.char)(t.datasamples), C.int(t.numsamples)), nil
}

func (t *MSTrace) DataSamples() ([]int32, error) {
	if t.sampletype == 'a' {
		return nil, errors.New("not a numerical formatted record")
	}
	samples := make([]int32, t.numsamples)

	switch {
	case t.sampletype == 'i':
		ptr := (*[1 << 30](C.int))(unsafe.Pointer(t.datasamples))
		for i := 0; i < int(t.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	case t.sampletype == 'f':
		ptr := (*[1 << 30](C.float))(unsafe.Pointer(t.datasamples))
		for i := 0; i < int(t.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	case t.sampletype == 'd':
		ptr := (*[1 << 30](C.double))(unsafe.Pointer(t.datasamples))
		for i := 0; i < int(t.numsamples); i++ {
			samples[i] = (int32)(ptr[i])
		}
	default:
		return nil, errors.New("format not coded")
	}

	return samples, nil
}

func (t *MSTrace) Endtime() time.Time {
	sec := int64(t.endtime) / 1000000
	nsec := 1000 * (int64(t.endtime) % 1000000)
	return time.Unix(sec, nsec).UTC()
}

func (t *MSTrace) SrcName(quality int8) string {
	csrcname := C.CString("NN_SSSSS_LL_CHA_Q_0")
	defer C.free(unsafe.Pointer(csrcname))
	C.mst_srcname((*C.struct_MSTrace_s)(t), csrcname, C.flag(quality))
	return C.GoString(csrcname)
}
