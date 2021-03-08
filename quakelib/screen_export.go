// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

//export SCR_Init
func SCR_Init() {
	screen.initialized = true
}

//export SCR_ResetTileClearUpdates
func SCR_ResetTileClearUpdates() {
	screen.ResetTileClearUpdates()
}
