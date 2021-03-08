// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"github.com/therjak/goquake/keys"
)

func enterQuitMenu() {
	keyDestination = keys.Console
	hostQuit()
}
