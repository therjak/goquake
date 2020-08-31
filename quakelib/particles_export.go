package quakelib

//#include "q_stdinc.h"
//#include "render.h"
import "C"

//export ParticlesInit
func ParticlesInit() {
	particlesInit()
}

//export ParticlesAddEntity
func ParticlesAddEntity(ent *C.entity_t) {
	e := Entity{ptr: ent}
	particlesAddEntity(e.origin(), float32(cl.time))
}

//export ParticlesClear
func ParticlesClear() {
	particlesClear()
}

//export ParticlesAddRocketTrail
func ParticlesAddRocketTrail(start, end *C.float, typ C.int) {
	particlesAddRocketTrail(p2v3(start), p2v3(end), int(typ), float32(cl.time))
}

//export ParticlesDraw
func ParticlesDraw() {
	particlesDraw()
}
