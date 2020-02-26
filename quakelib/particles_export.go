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
	e := Entity{ent}
	particlesAddEntity(e.origin(), float32(cl.time))
}

//export ParticlesClear
func ParticlesClear() {
	particlesClear()
}

//export ParticlesAddExplosion
func ParticlesAddExplosion(org *C.float) {
	particlesAddExplosion(p2v3(org), float32(cl.time))
}

//export ParticlesAddExplosion2
func ParticlesAddExplosion2(org *C.float, colorStart, colorLength C.int) {
	particlesAddExplosion2(p2v3(org), int(colorStart), int(colorLength), float32(cl.time))
}

//export ParticlesAddBlobExplosion
func ParticlesAddBlobExplosion(org *C.float) {
	particlesAddBlobExplosion(p2v3(org), float32(cl.time))
}

//export ParticlesRunEffect
func ParticlesRunEffect(org, dir *C.float, color, count C.int) {
	particlesRunEffect(p2v3(org), p2v3(dir), int(color), int(count), float32(cl.time))
}

//export ParticlesAddLavaSplash
func ParticlesAddLavaSplash(org *C.float) {
	particlesAddLavaSplash(p2v3(org), float32(cl.time))
}

//export ParticlesAddTeleportSplash
func ParticlesAddTeleportSplash(org *C.float) {
	particlesAddTeleportSplash(p2v3(org), float32(cl.time))
}

//export ParticlesAddRocketTrail
func ParticlesAddRocketTrail(start, end *C.float, typ C.int) {
	particlesAddRocketTrail(p2v3(start), p2v3(end), int(typ), float32(cl.time))
}

//export ParticlesRun
func ParticlesRun() {
	particlesRun(float32(cl.time), float32(cl.oldTime))
}

//export ParticlesDraw
func ParticlesDraw() {
	particlesDraw()
}
