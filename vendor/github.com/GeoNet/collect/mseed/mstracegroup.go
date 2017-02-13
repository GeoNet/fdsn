package mseed

//#cgo CFLAGS: -I/usr/local/include
//#cgo LDFLAGS: /usr/local/lib/libmseed.a
//#include <libmseed.h>
//MSTrace *mst_groupfirst(MSTraceGroup *mstg) {return (MSTrace *) mstg->traces;}
//MSTrace *mst_groupnext(MSTrace *mst) {return (MSTrace *) mst->next;}
import "C"

import (
	"unsafe"
)

type MSTraceGroup _Ctype_struct_MSTraceGroup_s

func NewMSTraceGroup() *MSTraceGroup {
	return (*MSTraceGroup)(C.mst_initgroup(nil))
}

func FreeMSTraceGroup(g *MSTraceGroup) {
	C.mst_freegroup((**_Ctype_struct_MSTraceGroup_s)((unsafe.Pointer)(&g)))
}

func (g *MSTraceGroup) AddMSRtoGroup(m *MSRecord, dataquality int, timetol float64, sampratetol float64) {
	C.mst_addmsrtogroup((*_Ctype_struct_MSTraceGroup_s)(g), (*_Ctype_struct_MSRecord_s)(m), C.flag(dataquality), C.double(timetol), C.double(sampratetol))
}

func (g *MSTraceGroup) PrintTraceList(timeformat int, details int, gaps int) {
	C.mst_printtracelist((*_Ctype_struct_MSTraceGroup_s)(g), C.flag(timeformat), C.flag(details), C.flag(gaps))
}

func (g *MSTraceGroup) First() *MSTrace {
	return (*MSTrace)(C.mst_groupfirst((*_Ctype_struct_MSTraceGroup_s)(g)))
}

func (t *MSTrace) Next() *MSTrace {
	return (*MSTrace)(C.mst_groupnext((*_Ctype_struct_MSTrace_s)(t)))
}
