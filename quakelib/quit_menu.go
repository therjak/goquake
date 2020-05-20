package quakelib

import (
	"github.com/therjak/goquake/keys"
)

func enterQuitMenu() {
	keyDestination = keys.Console
	hostQuit()
}
