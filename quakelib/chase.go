package quakelib

//#include "cgo_help.h"
import "C"
import "quake/math/vec"

func p2v3(p *C.float) vec.Vec3 {
	return vec.Vec3{
		float32(C.cf(0, p)),
		float32(C.cf(1, p)),
		float32(C.cf(2, p)),
	}
}

//export TraceLine
func TraceLine(start, end, impact *C.float) {
	s := p2v3(start)
	e := p2v3(end)
	trace := trace{}
	recursiveHullCheck(&cl.worldModel.Hulls[0], 0, 0, 1, s, e, &trace)
	*C.cfp(0, impact) = C.float(trace.EndPos[0])
	*C.cfp(1, impact) = C.float(trace.EndPos[1])
	*C.cfp(2, impact) = C.float(trace.EndPos[2])
}
