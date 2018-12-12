package quakelib

import "C"

import (
	"quake/net"
	svc "quake/protocol/server"
)

//export NETtcpipAvailable
func NETtcpipAvailable() C.int {
	return b2i(tcpipAvailable)
}

//export NET_SetTime
func NET_SetTime() {
	net.SetTime()
}

//export NET_GetTime
func NET_GetTime() C.double {
	return C.double(net.Time())
}

//export NET_Listen
func NET_Listen(b C.int) {
	// fmt.Printf("Go Listen: %v\n", b)
	// b is a boolean workaround
	// nothing to do for loopback
}

//export NET_CheckNewConnections
func NET_CheckNewConnections() C.int {
	return C.int(net.CheckNewConnections())
}

//export NET_SendDisconnectToAll
func NET_SendDisconnectToAll() C.int {
	return C.int(net.SendToAll([]byte{svc.Disconnect}))
}

//export NET_Shutdown
func NET_Shutdown() {
	net.Shutdown()
}

//export UDP_Init2
func UDP_Init2() C.int {
	return b2i(udp_init())
}
