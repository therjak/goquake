// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"
import (
	"fmt"
	cmdl "goquake/commandline"
	"goquake/conlog"
)

var (
	hostRecursionCheck = false
)

//export GoHostError
func GoHostError(msg *C.char) {
	HostError(fmt.Errorf(C.GoString(msg)))
}

func HostError(e error) {
	s := e.Error()

	if hostRecursionCheck {
		Error("Host_Error: recursively entered")
	}
	hostRecursionCheck = true

	screen.EndLoadingPlaque() // reenable screen updates

	conlog.Printf("Host_Error: %s\n", s)

	if sv.active {
		if err := hostShutdownServer(false); err != nil {
			// FIXME: this is recursion
			HostError(err)
		}
	}

	if cmdl.Dedicated() {
		// dedicated servers exit
		Error("Host_Error: %s\n", s)
	}

	if err := cls.Disconnect(); err != nil {
		// FIXME: this is recursion
		HostError(err)
	}
	cls.demoNum = -1

	// for errors during intermissions
	// (changelevel with no map found, etc.)
	cl.intermission = 0

	hostRecursionCheck = false

	panic(s)
}
