// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	cmdl "goquake/commandline"
	"goquake/conlog"
)

var (
	hostRecursionCheck = false
)

func HostError(e error) {
	s := e.Error()

	if hostRecursionCheck {
		QError("Host_Error: recursively entered")
	}
	hostRecursionCheck = true

	screen.EndLoadingPlaque() // reenable screen updates

	conlog.Printf("Host_Error: %s\n", s)

	if sv.Active() {
		if err := hostShutdownServer(false); err != nil {
			// FIXME: this is recursion
			HostError(err)
		}
	}

	if cmdl.Dedicated() {
		// dedicated servers exit
		QError("Host_Error: %s\n", s)
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
