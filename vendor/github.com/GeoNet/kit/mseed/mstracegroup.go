//nolint //cgo generates code that doesn't pass linting
package mseed

//#cgo CFLAGS: -I${SRCDIR}/../cvendor/libmseed
//#cgo LDFLAGS: ${SRCDIR}/../cvendor/libmseed/libmseed.a
//#include <libmseed.h>
//MSTrace *mst_groupfirst(MSTraceGroup *mstg) {return (MSTrace *) mstg->traces;}
//MSTrace *mst_groupnext(MSTrace *mst) {return (MSTrace *) mst->next;}
import "C"

import (
	"unsafe"
)

type MSTraceGroup C.struct_MSTraceGroup_s

func NewMSTraceGroup() *MSTraceGroup {
	return (*MSTraceGroup)(C.mst_initgroup(nil))
}

func FreeMSTraceGroup(g *MSTraceGroup) {
	C.mst_freegroup((**C.struct_MSTraceGroup_s)((unsafe.Pointer)(&g)))
}

func (g *MSTraceGroup) AddMSRtoGroup(m *MSRecord, dataquality int, timetol float64, sampratetol float64) {
	C.mst_addmsrtogroup((*C.struct_MSTraceGroup_s)(g), (*C.struct_MSRecord_s)(m), C.flag(dataquality), C.double(timetol), C.double(sampratetol))
}

func (g *MSTraceGroup) PrintTraceList(timeformat int, details int, gaps int) {
	C.mst_printtracelist((*C.struct_MSTraceGroup_s)(g), C.flag(timeformat), C.flag(details), C.flag(gaps))
}

func (g *MSTraceGroup) First() *MSTrace {
	return (*MSTrace)(C.mst_groupfirst((*C.struct_MSTraceGroup_s)(g)))
}

func (t *MSTrace) Next() *MSTrace {
	return (*MSTrace)(C.mst_groupnext((*C.struct_MSTrace_s)(t)))
}
