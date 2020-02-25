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

//export ParticlesDrawShowTris
func ParticlesDrawShowTris() {
	// C.R_DrawParticles_ShowTris()
}
