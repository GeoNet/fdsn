package slink

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libslink
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libslink/libslink.a
//#include <libslink.h>
import "C"

import (
	"unsafe"
)

type SLMSRecord _Ctype_SLMSrecord

func NewSLMSRecord() *SLMSRecord {
	return (*SLMSRecord)(C.sl_msr_new())
}

func FreeSLMSRecord(r **SLMSRecord) {
	C.sl_msr_free((**_Ctype_struct_SLMSrecord_s)((unsafe.Pointer)(r)))
}

func (r *SLMSRecord) Print(details int) int {
	return (int)(C.sl_msr_print((*_Ctype_struct_SLlog_s)(nil), (*_Ctype_struct_SLMSrecord_s)(r), (C.int)(details)))
}

func (r *SLMSRecord) DSampRate() (float64, int) {
	samprate := new(C.double)
	err := (int)(C.sl_msr_dsamprate((*_Ctype_struct_SLMSrecord_s)(r), samprate))
	return float64(*samprate), err
}

func (r *SLMSRecord) DNomSampRate() float64 {
	return float64(C.sl_msr_dnomsamprate((*_Ctype_struct_SLMSrecord_s)(r)))
}

func (r *SLMSRecord) sl_DEpochSTime() float64 {
	return float64(C.sl_msr_depochstime((*_Ctype_struct_SLMSrecord_s)(r)))
}
