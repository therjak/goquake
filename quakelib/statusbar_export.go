// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

//export Sbar_Changed
func Sbar_Changed() {
	statusbar.MarkChanged()
}

//export Sbar_Lines
func Sbar_Lines() int {
	return statusbar.Lines()
}
