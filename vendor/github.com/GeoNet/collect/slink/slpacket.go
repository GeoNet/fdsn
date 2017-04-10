package slink

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libslink
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libslink/libslink.a
//#include <libslink.h>
import "C"

import (
	//"bytes"
	"unsafe"
)

const (
	SLRECSIZE  int = 512
	SLHEADSIZE int = 8
)

const (
	SLPACKET    int = 1
	SLTERMINATE int = 0
	SLNOPACKET  int = -1
)

type Type int

const (
	SLDATA Type = iota // waveform data record
	SLDET              // detection record
	SLCAL              // calibration record
	SLTIM              // timing record
	SLMSG              // message record
	SLBLK              // general record
	SLNUM              // used as the error indicator (same as SLCHA)
	SLCHA              // for requesting channel info or detectors
	SLINF              // a non-terminating XML formatted message in a miniSEED log record, used for INFO responses
	SLINFT             // a terminating XML formatted message in a miniSEED log record, used for INFO responses
	SLKEEP             // an XML formatted message in a miniSEED log record, used for keepalive/heartbeat responses
)

type SLPacket _Ctype_SLpacket

func (p *SLPacket) Sequence() int {
	return (int)(C.sl_sequence((*C.struct_slpacket_s)(p)))
}
func (p *SLPacket) PacketType() Type {
	return (Type)(C.sl_packettype((*C.struct_slpacket_s)(p)))
}

func (p *SLPacket) GetMSRecord() []byte {
	return C.GoBytes(unsafe.Pointer(&p.msrecord[0]), (C.int)(SLRECSIZE))

}
func (p *SLPacket) GetSLHead() []byte {
	return C.GoBytes(unsafe.Pointer(&p.slhead[0]), (C.int)(SLHEADSIZE))
}

func (p *SLPacket) ParseRecord(blktflag, unpackflag int8) *SLMSRecord {
	msr := C.sl_msr_new()
	C.sl_msr_parse((*C.struct_SLlog_s)(nil), (*_Ctype_char)(unsafe.Pointer(&p.msrecord[0])), &msr, (C.int8_t)(blktflag), (C.int8_t)(unpackflag))
	return (*SLMSRecord)(msr)
}
