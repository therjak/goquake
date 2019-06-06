package quakelib

import (
	"quake/cmd"
	"quake/keys"
)

func enterQuitMenu() {
	keyDestination = keys.Console
	hostQuit([]cmd.QArg{})
}
