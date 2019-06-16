package quakelib

//#include "trace.h"
//#include "cgo_help.h"
import "C"

//export TraceLine
func TraceLine(start, end, impact *C.float) {
	s := p2v3(start)
	e := p2v3(end)
	trace := C.trace_t{}
	recursiveHullCheck(&cl.worldModel.Hulls[0], 0, 0, 1, s, e, &trace)
	*C.cfp(0, impact) = trace.endpos[0]
	*C.cfp(1, impact) = trace.endpos[1]
	*C.cfp(2, impact) = trace.endpos[2]
}
