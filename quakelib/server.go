package quakelib

// void SV_DropClient(int,int);
import "C"

import (
	cmdl "quake/commandline"
	"quake/execute"
	"quake/net"
)

//int maxclients
//int maxclientslimit
//int serverflags
//qboolean changelevel_issued
type ServerStatic struct {
	maxClients        int
	maxClientsLimit   int
	serverFlags       int // TODO: is int the correct way?
	changeLevelIssued bool
	// [maxClients]clients...
}

type Server struct {
	active   bool
	paused   bool
	loadGame bool

	datagram         net.Message
	reliableDatagram net.Message
	signon           net.Message

	protocol      uint16
	protocolFlags uint16
}

var (
	svs = ServerStatic{}
	sv  = Server{}
)

//export SV_SetProtocolFlags
func SV_SetProtocolFlags(flags C.ushort) {
	sv.protocolFlags = uint16(flags)
}

//export SV_ProtocolFlags
func SV_ProtocolFlags() C.ushort {
	return C.ushort(sv.protocolFlags)
}

//export SV_Paused
func SV_Paused() C.int {
	return b2i(sv.paused)
}

//export SV_SetPaused
func SV_SetPaused(b C.int) {
	sv.paused = (b != 0)
}

//export SV_LoadGame
func SV_LoadGame() C.int {
	return b2i(sv.loadGame)
}

//export SV_SetLoadGame
func SV_SetLoadGame(b C.int) {
	sv.loadGame = (b != 0)
}

//export SV_Clear
func SV_Clear() {
	sv = Server{}
}

//export SV_Active
func SV_Active() C.int {
	return b2i(sv.active)
}

//export SV_SetActive
func SV_SetActive(v C.int) {
	sv.active = (v != 0)
}

var (
	msgBuf       = net.Message{}
	msgBufMaxLen = 0
)

//export SV_MS_Clear
func SV_MS_Clear() {
	msgBuf.ClearMessage()
}

//export SV_MS_SetMaxLen
func SV_MS_SetMaxLen(v C.int) {
	msgBufMaxLen = int(v)
}

//export SV_MS_MaxLen
func SV_MS_MaxLen() C.int {
	return C.int(msgBufMaxLen)
}

//export SV_MS_WriteByte
func SV_MS_WriteByte(v C.int) {
	msgBuf.WriteByte(int(v))
}

//export SV_MS_WriteChar
func SV_MS_WriteChar(v C.int) {
	msgBuf.WriteChar(int(v))
}

//export SV_MS_WriteShort
func SV_MS_WriteShort(v C.int) {
	msgBuf.WriteShort(int(v))
}

//export SV_MS_WriteLong
func SV_MS_WriteLong(v C.int) {
	msgBuf.WriteLong(int(v))
}

//export SV_MS_WriteFloat
func SV_MS_WriteFloat(f C.float) {
	msgBuf.WriteFloat(float32(f))
}

//export SV_MS_WriteAngle
func SV_MS_WriteAngle(v C.float) {
	msgBuf.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_MS_WriteCoord
func SV_MS_WriteCoord(v C.float) {
	msgBuf.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_MS_WriteString
func SV_MS_WriteString(s *C.char) {
	msgBuf.WriteString(C.GoString(s))
}

//export SV_MS_Len
func SV_MS_Len() C.int {
	return C.int(msgBuf.Len())
}

//export SV_SO_WriteByte
func SV_SO_WriteByte(v C.int) {
	sv.signon.WriteByte(int(v))
}

//export SV_SO_WriteChar
func SV_SO_WriteChar(v C.int) {
	sv.signon.WriteChar(int(v))
}

//export SV_SO_WriteShort
func SV_SO_WriteShort(v C.int) {
	sv.signon.WriteShort(int(v))
}

//export SV_SO_WriteLong
func SV_SO_WriteLong(v C.int) {
	sv.signon.WriteLong(int(v))
}

//export SV_SO_WriteAngle
func SV_SO_WriteAngle(v C.float) {
	sv.signon.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_SO_WriteCoord
func SV_SO_WriteCoord(v C.float) {
	sv.signon.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_SO_WriteString
func SV_SO_WriteString(s *C.char) {
	sv.signon.WriteString(C.GoString(s))
}

//export SV_SO_Len
func SV_SO_Len() C.int {
	return C.int(sv.signon.Len())
}

//export SV_DG_WriteByte
func SV_DG_WriteByte(v C.int) {
	sv.datagram.WriteByte(int(v))
}

//export SV_DG_WriteChar
func SV_DG_WriteChar(v C.int) {
	sv.datagram.WriteChar(int(v))
}

//export SV_DG_WriteShort
func SV_DG_WriteShort(v C.int) {
	sv.datagram.WriteShort(int(v))
}

//export SV_DG_WriteLong
func SV_DG_WriteLong(v C.int) {
	sv.datagram.WriteLong(int(v))
}

//export SV_DG_WriteAngle
func SV_DG_WriteAngle(v C.float) {
	sv.datagram.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_DG_WriteCoord
func SV_DG_WriteCoord(v C.float) {
	sv.datagram.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_DG_WriteString
func SV_DG_WriteString(s *C.char) {
	sv.datagram.WriteString(C.GoString(s))
}

//export SV_DG_Len
func SV_DG_Len() C.int {
	return C.int(sv.datagram.Len())
}

//export SV_DG_ClearMessage
func SV_DG_ClearMessage() {
	sv.datagram.ClearMessage()
}

//export SV_DG_SendOut
func SV_DG_SendOut(client C.int) C.int {
	b := msgBuf.Bytes()
	// If there is space add the server datagram
	if len(b)+sv.datagram.Len() < 32000 {
		b = append(b, sv.datagram.Bytes()...)
	}
	con := sv_clients[int(client)].netConnection
	if con.SendUnreliableMessage(b) == -1 {
		C.SV_DropClient(client, 1)
		return 0
	}
	return 1
}

//export SV_RD_WriteByte
func SV_RD_WriteByte(v C.int) {
	sv.reliableDatagram.WriteByte(int(v))
}

//export SV_RD_WriteChar
func SV_RD_WriteChar(v C.int) {
	sv.reliableDatagram.WriteChar(int(v))
}

//export SV_RD_WriteShort
func SV_RD_WriteShort(v C.int) {
	sv.reliableDatagram.WriteShort(int(v))
}

//export SV_RD_WriteLong
func SV_RD_WriteLong(v C.int) {
	sv.reliableDatagram.WriteLong(int(v))
}

//export SV_RD_WriteAngle
func SV_RD_WriteAngle(v C.float) {
	sv.reliableDatagram.WriteAngle(float32(v), int(sv.protocolFlags))
}

//export SV_RD_WriteCoord
func SV_RD_WriteCoord(v C.float) {
	sv.reliableDatagram.WriteCoord(float32(v), int(sv.protocolFlags))
}

//export SV_RD_WriteString
func SV_RD_WriteString(s *C.char) {
	sv.reliableDatagram.WriteString(C.GoString(s))
}

//export SV_RD_SendOut
func SV_RD_SendOut() {
	b := sv.reliableDatagram.Bytes()
	for _, cl := range sv_clients {
		if cl.active {
			cl.msg.WriteBytes(b)
		}
	}
	sv.reliableDatagram.ClearMessage()
}

//export SVS_GetServerFlags
func SVS_GetServerFlags() C.int {
	return C.int(svs.serverFlags)
}

//export SVS_SetServerFlags
func SVS_SetServerFlags(flags C.int) {
	svs.serverFlags = int(flags)
}

//export SVS_IsChangeLevelIssued
func SVS_IsChangeLevelIssued() C.int {
	return b2i(svs.changeLevelIssued)
}

//export SVS_SetChangeLevelIssued
func SVS_SetChangeLevelIssued(s C.int) {
	if s == 0 {
		svs.changeLevelIssued = false
		return
	}
	svs.changeLevelIssued = true
}

//export SVS_GetMaxClients
func SVS_GetMaxClients() C.int {
	return C.int(svs.maxClients)
}

//export SVS_SetMaxClients
func SVS_SetMaxClients(n C.int) {
	svs.maxClients = int(n)
}

//export SVS_GetMaxClientsLimit
func SVS_GetMaxClientsLimit() C.int {
	return C.int(svs.maxClientsLimit)
}

//export SVS_SetMaxClientsLimit
func SVS_SetMaxClientsLimit(n C.int) {
	svs.maxClientsLimit = int(n)
}

//export SV_SendReconnect
func SV_SendReconnect() {
	SendReconnectToAll()
	if !cmdl.Dedicated() {
		execute.Execute("reconnect\n", execute.Command)
	}
}
