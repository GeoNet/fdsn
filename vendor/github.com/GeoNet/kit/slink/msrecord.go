//nolint //cgo generates code that doesn't pass linting
package slink

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libslink
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libslink/libslink.a
//#include <libslink.h>
import "C"

import (
	"unsafe"
)

type SLMSRecord C.SLMSrecord

func NewSLMSRecord() *SLMSRecord {
	return (*SLMSRecord)(C.sl_msr_new())
}

func FreeSLMSRecord(r **SLMSRecord) {
	C.sl_msr_free((**C.struct_SLMSrecord_s)((unsafe.Pointer)(r)))
}

func (r *SLMSRecord) Print(details int) int {
	return (int)(C.sl_msr_print((*C.struct_SLlog_s)(nil), (*C.struct_SLMSrecord_s)(r), (C.int)(details)))
}

func (r *SLMSRecord) DSampRate() (float64, int) {
	samprate := new(C.double)
	err := (int)(C.sl_msr_dsamprate((*C.struct_SLMSrecord_s)(r), samprate))
	return float64(*samprate), err
}

func (r *SLMSRecord) DNomSampRate() float64 {
	return float64(C.sl_msr_dnomsamprate((*C.struct_SLMSrecord_s)(r)))
}

//nolint:unused // reserved for future use
func (r *SLMSRecord) sl_DEpochSTime() float64 {
	return float64(C.sl_msr_depochstime((*C.struct_SLMSrecord_s)(r)))
}
