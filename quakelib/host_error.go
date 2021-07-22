// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"
import (
	"fmt"

	"goquake/conlog"
)

var (
	hostRecursionCheck = false
)

//export GoHostError
func GoHostError(msg *C.char) {
	HostError(C.GoString(msg))
}

func HostEndGame(msg string) {
	conlog.DPrintf("Host_EndGame: %s\n", msg)

	if sv.active {
		hostShutdownServer(false)
	}

	if cls.state == ca_dedicated {
		// dedicated servers exit
		Error("Host_EndGame: %s\n", msg)
	}

	if cls.demoNum != -1 {
		CL_NextDemo()
	} else {
		cls.Disconnect()
	}
	// TODO: There must be a better way than to panic
	panic(msg)
}

func HostError(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)

	if hostRecursionCheck {
		Error("Host_Error: recursively entered")
	}
	hostRecursionCheck = true

	screen.EndLoadingPlaque() // reenable screen updates

	conlog.Printf("Host_Error: %s\n", s)

	if sv.active {
		hostShutdownServer(false)
	}

	if cls.state == ca_dedicated {
		// dedicated servers exit
		Error("Host_Error: %s\n", s)
	}

	cls.Disconnect()
	cls.demoNum = -1

	// for errors during intermissions
	// (changelevel with no map found, etc.)
	cl.intermission = 0

	hostRecursionCheck = false

	panic(s)
}
