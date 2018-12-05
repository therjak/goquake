package quakelib

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"quake/net"
	svc "quake/protocol/server"
	"unsafe"
)

var (
	msg_badread = false
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

//export NET_ConnectTime
func NET_ConnectTime(id C.int) C.double {
	t, err := net.ConnectTime(int(id - 1))
	if err != nil {
		return C.double(0)
	}
	return C.double(t)
}

//export NET_Address
func NET_Address(id C.int, ret *C.char, n C.size_t) {
	s, err := net.Address(int(id - 1))
	if len(s) >= int(n) {
		s = s[:n-1]
	}
	if err != nil {
		s = ""
	}
	sp := C.CString(s)
	defer C.free(unsafe.Pointer(sp))
	C.strncpy(ret, sp, n)
}

//export NET_Listen
func NET_Listen(b C.int) {
	// fmt.Printf("Go Listen: %v\n", b)
	// b is a boolean workaround
	// nothing to do for loopback
}

//export NET_Connect
func NET_Connect(host *C.char) C.int {
	// returns the connection to the server, will be stored on the client
	// return 0 == error. C stuff...
	c, err := net.Connect(C.GoString(host))
	if err != nil {
		return C.int(0)
	}
	return C.int(c.ID())
}

//export NET_CheckNewConnections
func NET_CheckNewConnections() C.int {
	return C.int(net.CheckNewConnections())
}

//export NET_Close
func NET_Close(id C.int) {
	net.Close(int(id))
}

//export NET_GetMessage
func NET_GetMessage(id C.int) C.int {
	msg_badread = false
	return C.int(net.GetMessage(int(id)))
}

//export NET_SendMessage
func NET_SendMessage(id C.int, data *C.uchar, size C.int) C.int {
	return C.int(net.SendMessage(int(id), C.GoBytes(unsafe.Pointer(data), size)))
}

//export NET_SendUnreliableMessage
func NET_SendUnreliableMessage(id C.int, data *C.uchar, size C.int) C.int {
	return C.int(net.SendUnreliableMessage(int(id), C.GoBytes(unsafe.Pointer(data), size)))
}

//export NET_SendDisconnectToAll
func NET_SendDisconnectToAll() C.int {
	return C.int(net.SendToAll([]byte{svc.Disconnect}))
}

//export NET_CanSendMessage
func NET_CanSendMessage(id C.int) C.int {
	if net.CanSendMessage(int(id)) {
		return 1
	}
	return 0
}

//export NET_Shutdown
func NET_Shutdown() {
	net.Shutdown()
}

//export MSG_BadRead
func MSG_BadRead() C.int {
	// poor mans error handling :(
	if msg_badread {
		return 1
	}
	return 0
}

//export MSG_ReadChar
func MSG_ReadChar() C.int {
	i, err := net.ReadInt8()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadByte
func MSG_ReadByte() C.int {
	i, err := net.ReadByte()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadShort
func MSG_ReadShort() C.int {
	i, err := net.ReadInt16()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadLong
func MSG_ReadLong() C.int {
	i, err := net.ReadInt32()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadFloat
func MSG_ReadFloat() C.float {
	f, err := net.ReadFloat32()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord16
func MSG_ReadCoord16() C.float {
	f, err := net.ReadCoord16()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord24
func MSG_ReadCoord24() C.float {
	f, err := net.ReadCoord24()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord32f
func MSG_ReadCoord32f() C.float {
	f, err := net.ReadCoord32f()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord
func MSG_ReadCoord() C.float {
	f, err := net.ReadCoord(cl.protocolFlags)
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadAngle
func MSG_ReadAngle(flags C.uint) C.float {
	f, err := net.ReadAngle(uint32(flags))
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadAngle16
func MSG_ReadAngle16(flags C.uint) C.float {
	f, err := net.ReadAngle16(uint32(flags))
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)

}

//export UDP_Init2
func UDP_Init2() C.int {
	return b2i(udp_init())
}
