package quakelib

//#include <string.h>
//#include <stdlib.h>
//void Host_Error(const char * error, ...);
//void host_error_go(char* error) {
//  char string[1024];
//  strncpy(string, error, 1024);
//  free(error);
//  Host_Error(string);
//}

import "C"
import (
	"fmt"
	"quake/conlog"
)

var (
	hostRecursionCheck = false
)

//export GoHostError
func GoHostError(msg *C.char) {
	HostError(C.GoString(msg))
}

//export Host_EndGame
func Host_EndGame(msg *C.char) {
	HostEndGame(C.GoString(msg))
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

	SCR_EndLoadingPlaque() // reenable screen updates

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
	// C.host_error_go(C.CString(s))
}
