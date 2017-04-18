// Package slink provides a go wrapper for the libslink C library.
package slink

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libslink
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libslink/libslink.a
//#include <libslink.h>
import "C"

import (
	"errors"
	"unsafe"
)

type SLCD _Ctype_SLCD

func NewSLCD() *SLCD {
	return (*SLCD)(C.sl_newslcd())
}
func FreeSLCD(s *SLCD) {
	C.sl_freeslcd((*_Ctype_struct_slcd_s)(s))
}

func (s *SLCD) NetDly() int {
	return (int)(((*_Ctype_struct_slcd_s)(s)).netdly)
}
func (s *SLCD) SetNetDly(netdly int) {
	(((*_Ctype_struct_slcd_s)(s)).netdly) = (C.int)(netdly)
}
func (s *SLCD) NetTo() int {
	return (int)(((*_Ctype_struct_slcd_s)(s)).netto)
}
func (s *SLCD) SetNetTo(netto int) {
	(((*_Ctype_SLCD)(s)).netto) = (C.int)(netto)
}
func (s *SLCD) KeepAlive() int {
	return (int)(((*_Ctype_SLCD)(s)).keepalive)
}
func (s *SLCD) SetKeepAlive(keepalive int) {
	(((*_Ctype_SLCD)(s)).keepalive) = (C.int)(keepalive)
}
func (s *SLCD) SetSLAddr(sladdr string) {
	(((*_Ctype_SLCD)(s)).sladdr) = C.CString(sladdr)
}
func (s *SLCD) Collect() (*SLPacket, int) {
	var slpack *C.struct_slpacket_s
	err := (int)(C.sl_collect((*C.struct_slcd_s)(s), &slpack))
	return (*SLPacket)(slpack), err
}
func (s *SLCD) CollectNB() (*SLPacket, int) {
	var slpack *C.struct_slpacket_s
	err := (int)(C.sl_collect_nb((*C.struct_slcd_s)(s), &slpack))
	return (*SLPacket)(slpack), err
}

func (s *SLCD) AddStream(net, sta, selectors string, seqnum int, timestamp string) int {
	cnet := C.CString(net)
	defer C.free(unsafe.Pointer(cnet))
	csta := C.CString(sta)
	defer C.free(unsafe.Pointer(csta))
	cselectors := C.CString(selectors)
	defer C.free(unsafe.Pointer(cselectors))
	ctimestamp := C.CString(timestamp)
	defer C.free(unsafe.Pointer(ctimestamp))
	return (int)(C.sl_addstream((*C.struct_slcd_s)(s), cnet, csta, cselectors, (C.int)(seqnum), ctimestamp))
}
func (s *SLCD) SetUniParams(selectors string, seqnum int, timestamp string) int {
	cselectors := C.CString(selectors)
	defer C.free(unsafe.Pointer(cselectors))
	ctimestamp := C.CString(timestamp)
	defer C.free(unsafe.Pointer(ctimestamp))
	return (int)(C.sl_setuniparams((*C.struct_slcd_s)(s), cselectors, (C.int)(seqnum), ctimestamp))
}
func (s *SLCD) RequestInfo(infostr string) int {
	cs := C.CString(infostr)
	defer C.free(unsafe.Pointer(cs))
	return (int)(C.sl_request_info((*C.struct_slcd_s)(s), cs))
}

func (s *SLCD) Terminate() {
	C.sl_terminate((*C.struct_slcd_s)(s))
}

func (s *SLCD) ReadStreamList(streamfile string, defselect string) (int, error) {
	if defselect == "" {
		return s.ReadStreamListDefault(streamfile)
	}
	cstreamfile := C.CString(streamfile)
	defer C.free(unsafe.Pointer(cstreamfile))
	cdefselect := C.CString(defselect)
	defer C.free(unsafe.Pointer(cdefselect))
	count := (int)(C.sl_read_streamlist((*C.struct_slcd_s)(s), cstreamfile, cdefselect))
	if count < 0 {
		return count, errors.New("unable to read stream list")
	}
	return count, nil
}

func (s *SLCD) ReadStreamListDefault(streamfile string) (int, error) {
	cstreamfile := C.CString(streamfile)
	defer C.free(unsafe.Pointer(cstreamfile))
	count := (int)(C.sl_read_streamlist((*C.struct_slcd_s)(s), cstreamfile, nil))
	if count < 0 {
		return count, errors.New("unable to read stream list")
	}
	return count, nil
}

func (s *SLCD) ParseStreamList(streamlist string, defselect string) (int, error) {
	if defselect == "" {
		return s.ParseStreamListDefault(streamlist)
	}
	cstreamlist := C.CString(streamlist)
	defer C.free(unsafe.Pointer(cstreamlist))
	cdefselect := C.CString(defselect)
	defer C.free(unsafe.Pointer(cdefselect))
	count := (int)(C.sl_parse_streamlist((*C.struct_slcd_s)(s), cstreamlist, cdefselect))
	if count < 0 {
		return count, errors.New("unable to parse stream list")
	}
	return count, nil
}

func (s *SLCD) ParseStreamListDefault(streamlist string) (int, error) {
	cstreamlist := C.CString(streamlist)
	defer C.free(unsafe.Pointer(cstreamlist))
	count := (int)(C.sl_parse_streamlist((*C.struct_slcd_s)(s), cstreamlist, nil))
	if count < 0 {
		return count, errors.New("unable to parse stream list")
	}
	return count, nil
}

func (s *SLCD) ConfigLink() int {
	return (int)(C.sl_configlink((*C.struct_slcd_s)(s)))
}

func (s *SLCD) SendInfo(info_level string, verbose int) int {
	cinfo_level := C.CString(info_level)
	defer C.free(unsafe.Pointer(cinfo_level))
	return (int)(C.sl_send_info((*C.struct_slcd_s)(s), cinfo_level, (C.int)(verbose)))
}

func (s *SLCD) Connect(sayhello int) int {
	return (int)(C.sl_connect((*C.struct_slcd_s)(s), (C.int)(sayhello)))
}

func (s *SLCD) Disconnect() int {
	if (*_Ctype_SLCD)(s).link != -1 {
		return (int)(C.sl_disconnect((*C.struct_slcd_s)(s)))
	}
	return 0
}

func (s *SLCD) Ping() (int, string, string) {
	cserverid := C.CString((string)(make([]byte, 100)))
	defer C.free(unsafe.Pointer(cserverid))
	csite := C.CString((string)(make([]byte, 100)))
	defer C.free(unsafe.Pointer(csite))
	ping := (int)(C.sl_ping((*C.struct_slcd_s)(s), cserverid, csite))
	return ping, C.GoString(cserverid), C.GoString(csite)
}

func (s *SLCD) CheckSock(sock, tosec, tousec int) int {
	return (int)(C.sl_checksock((C.int)(sock), (C.int)(tosec), (C.int)(tousec)))
}

func (s *SLCD) SendData(buffer []byte, ident string, resplen int) ([]byte, int) {
	cident := C.CString(ident)
	defer C.free(unsafe.Pointer(cident))
	resp := make([]byte, resplen)
	err := (int)(C.sl_senddata((*C.struct_slcd_s)(s), (unsafe.Pointer)(&buffer), (C.size_t)(len(buffer)), cident, (unsafe.Pointer)(&resp), (C.int)(resplen)))
	return resp, err
}

func (s *SLCD) RecvData(maxbytes int, ident string) ([]byte, int) {
	cident := C.CString(ident)
	defer C.free(unsafe.Pointer(cident))
	buffer := make([]byte, maxbytes)
	err := (int)(C.sl_recvdata((*C.struct_slcd_s)(s), (unsafe.Pointer)(&buffer), (C.size_t)(len(buffer)), cident))
	return buffer, err
}

func (s *SLCD) RecvResp(maxbytes int, command, ident string) ([]byte, int) {
	ccommand := C.CString(command)
	defer C.free(unsafe.Pointer(ccommand))
	cident := C.CString(ident)
	defer C.free(unsafe.Pointer(cident))
	buffer := make([]byte, maxbytes)
	err := (int)(C.sl_recvresp((*C.struct_slcd_s)(s), (unsafe.Pointer)(&buffer), (C.size_t)(len(buffer)), ccommand, cident))
	return buffer, err
}

func (s *SLCD) RecoverState(statefile string) int {
	cstatefile := C.CString(statefile)
	defer C.free(unsafe.Pointer(cstatefile))
	return (int)(C.sl_recoverstate((*C.struct_slcd_s)(s), cstatefile))
}
func (s *SLCD) SaveState(statefile string) int {
	cstatefile := C.CString(statefile)
	defer C.free(unsafe.Pointer(cstatefile))
	return (int)(C.sl_savestate((*C.struct_slcd_s)(s), cstatefile))
}
