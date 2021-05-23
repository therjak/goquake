// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"fmt"
)

//export VID_Init_Go
func VID_Init_Go() {
	err := videoInit()
	if err != nil {
		fmt.Printf("%v", err)
		Error(fmt.Sprintf("%v", err))
	}
	// All our shaders:
	CreateAliasDrawer()
	CreateSpriteDrawer()
	CreateSkyDrawer()
	CreateParticleDrawer()
	CreatePostProcess()
	CreateConeDrawer()
	CreateUiDrawer()
}
