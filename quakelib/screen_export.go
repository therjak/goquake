// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

//export SCR_ResetTileClearUpdates
func SCR_ResetTileClearUpdates() {
	screen.ResetTileClearUpdates()
}
