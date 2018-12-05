package quakelib

// void Host_Quit_f(void);
import "C"

import (
	"quake/keys"
)

func enterQuitMenu() {
	keyDestination = keys.Console
	C.Host_Quit_f()
}
