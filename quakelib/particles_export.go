package quakelib

//#include "q_stdinc.h"
//#include "render.h"
//void R_InitParticles(void);
//void R_EntityParticles(entity_t *ent);
//void R_ClearParticles(void);
//void R_ParticleExplosion(vec3_t org);
//void R_ParticleExplosion2(vec3_t org, int colorStart, int colorLength);
//void R_BlobExplosion(vec3_t org);
//void R_RunParticleEffect(vec3_t org, vec3_t dir, int color, int count);
//void R_LavaSplash(vec3_t org);
//void R_TeleportSplash(vec3_t org);
//void R_RocketTrail(vec3_t start, vec3_t end, int type);
//void CL_RunParticles(void);
//void R_DrawParticles(void);
//void R_DrawParticles_ShowTris(void);
import "C"

//export ParticlesInit
func ParticlesInit() {
	C.R_InitParticles()
	particlesInit()
}

//export ParticlesAddEntity
func ParticlesAddEntity(ent *C.entity_t) {
	C.R_EntityParticles(ent)
	e := Entity{ent}
	particlesAddEntity(e.origin(), float32(cl.time))
}

//export ParticlesClear
func ParticlesClear() {
	C.R_ClearParticles()
	particlesClear()
}

//export ParticlesAddExplosion
func ParticlesAddExplosion(org *C.float) {
	C.R_ParticleExplosion(org)
	particlesAddExplosion(p2v3(org), float32(cl.time))
}

//export ParticlesAddExplosion2
func ParticlesAddExplosion2(org *C.float, colorStart, colorLength C.int) {
	C.R_ParticleExplosion2(org, colorStart, colorLength)
	particlesAddExplosion2(p2v3(org), int(colorStart), int(colorLength), float32(cl.time))
}

//export ParticlesAddBlobExplosion
func ParticlesAddBlobExplosion(org *C.float) {
	C.R_BlobExplosion(org)
	particlesAddBlobExplosion(p2v3(org), float32(cl.time))
}

//export ParticlesRunEffect
func ParticlesRunEffect(org, dir *C.float, color, count C.int) {
	C.R_RunParticleEffect(org, dir, color, count)
}

//export ParticlesAddLavaSplash
func ParticlesAddLavaSplash(org *C.float) {
	C.R_LavaSplash(org)
	particlesAddLavaSplash(p2v3(org), float32(cl.time))
}

//export ParticlesAddTeleportSplash
func ParticlesAddTeleportSplash(org *C.float) {
	C.R_TeleportSplash(org)
	particlesAddTeleportSplash(p2v3(org), float32(cl.time))
}

//export ParticlesAddRocketTrail
func ParticlesAddRocketTrail(start, end *C.float, typ C.int) {
	C.R_RocketTrail(start, end, typ)
	particlesAddRocketTrail(p2v3(start), p2v3(end), int(typ), float32(cl.time))
}

//export ParticlesRun
func ParticlesRun() {
	C.CL_RunParticles()
	particlesRun(float32(cl.time), float32(cl.oldTime))
}

//export ParticlesDraw
func ParticlesDraw() {
	C.R_DrawParticles()
}

//export ParticlesDrawShowTris
func ParticlesDrawShowTris() {
	C.R_DrawParticles_ShowTris()
}
