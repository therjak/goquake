// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/keys"
)

func enterQuitMenu() {
	keyDestination = keys.Console
	if err := hostQuit(); err != nil {
		HostError(err)
	}
}
