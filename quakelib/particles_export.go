// SPDX-License-Identifier: GPL-2.0-or-later
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
