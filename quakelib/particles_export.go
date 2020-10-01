package quakelib

import "C"

//export ParticlesInit
func ParticlesInit() {
	particlesInit()
}

//export ParticlesClear
func ParticlesClear() {
	particlesClear()
}

//export ParticlesDraw
func ParticlesDraw() {
	particlesDraw()
}
