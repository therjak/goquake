package quakelib

import "C"

import (
	"quake/net"
)

//export NET_SetTime
func NET_SetTime() {
	net.SetTime()
}

//export NET_GetTime
func NET_GetTime() C.double {
	return C.double(net.Time())
}

//export NET_Shutdown
func NET_Shutdown() {
	net.Shutdown()
}
