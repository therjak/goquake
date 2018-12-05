package quakelib

/*
#ifndef _MOVEDEF_h
#define _MOVEDEF_h
typedef struct movecmd_s {
	float forwardmove;
	float sidemove;
	float upmove;
} movecmd_t;
#endif // _MOVEDEF_h
void SV_DropClient(int, int);
*/
import "C"

import (
	"log"
	"quake/net"
	"quake/protocol/server"
	"unsafe"
)

type SVClient struct {
	active     bool // false = client is free
	spawned    bool // false = don't send datagrams
	sendSignon bool // only valid before spawned

	// reliable messages must be sent periodically
	lastMessage float64

	netConnection int // communications handle

	cmd C.movecmd_t // movement

	// can be added to at any time, copied and clear once per frame
	//  had max length of 64000
	msg     net.Message
	edictId int    // == clientnum + 1
	name    string //[32];  // for printing to other people
	colors  int

	pingTimes [16]float32
	numPings  int // ping_times[num_pings%NUM_PING_TIMES]

	// spawn params are carried from level to level
	spawnParams [16]float32

	// client known data for deltas
	oldFrags int
	id       int // Needed to communicate with the 'c' side
}

var (
	sv_clients []*SVClient
)

//export CreateSVClients
func CreateSVClients() {
	sv_clients = make([]*SVClient, svs.maxClientsLimit)
	for i := range sv_clients {
		sv_clients[i] = &SVClient{
			id: i,
		}
	}
}

//export CleanSVClient
func CleanSVClient(n C.int) {
	// reset everything but netConnection, edictId
	c := sv_clients[int(n)]
	nc := c.netConnection
	ei := c.edictId
	id := c.id
	sv_clients[int(n)] = &SVClient{
		netConnection: nc,
		edictId:       ei,
		id:            id,
	}
}

//export GetClientSpawnParam
func GetClientSpawnParam(c, n C.int) C.float {
	return C.float(sv_clients[int(c)].spawnParams[int(n)])
}

//export SetClientSpawnParam
func SetClientSpawnParam(c, n C.int, v C.float) {
	sv_clients[int(c)].spawnParams[int(n)] = float32(v)
}

//export GetClientPingTime
func GetClientPingTime(c, n C.int) C.float {
	return C.float(sv_clients[int(c)].pingTimes[int(n)])
}

//export SetClientPingTime
func SetClientPingTime(c, n C.int, v C.float) {
	sv_clients[int(c)].pingTimes[int(n)] = float32(v)
}

//export GetClientName
func GetClientName(n C.int) *C.char {
	return C.CString(sv_clients[int(n)].name)
}

//export SetClientName
func SetClientName(n C.int, c *C.char) {
	sv_clients[int(n)].name = C.GoString(c)
}

//export GetClientMoveCmd
func GetClientMoveCmd(n C.int) C.movecmd_t {
	return sv_clients[int(n)].cmd
}

//export SetClientMoveCmd
func SetClientMoveCmd(n C.int, c C.movecmd_t) {
	sv_clients[int(n)].cmd = c
}

//export GetClientLastMessage
func GetClientLastMessage(n C.int) C.double {
	return C.double(sv_clients[int(n)].lastMessage)
}

//export SetClientLastMessage
func SetClientLastMessage(n C.int) {
	sv_clients[int(n)].lastMessage = host.time
}

//export GetClientOldFrags
func GetClientOldFrags(n C.int) C.int {
	return C.int(sv_clients[int(n)].oldFrags)
}

//export SetClientOldFrags
func SetClientOldFrags(n C.int, v C.int) {
	sv_clients[int(n)].oldFrags = int(v)
}

//export GetClientNumPings
func GetClientNumPings(n C.int) C.int {
	return C.int(sv_clients[int(n)].numPings)
}

//export SetClientNumPings
func SetClientNumPings(n C.int, v C.int) {
	sv_clients[int(n)].numPings = int(v)
}

//export GetClientColors
func GetClientColors(n C.int) C.int {
	return C.int(sv_clients[int(n)].colors)
}

//export SetClientColors
func SetClientColors(n C.int, v C.int) {
	sv_clients[int(n)].colors = int(v)
}

//export GetClientEdictId
func GetClientEdictId(n C.int) C.int {
	return C.int(sv_clients[int(n)].edictId)
}

//export SetClientEdictId
func SetClientEdictId(n C.int, v C.int) {
	sv_clients[int(n)].edictId = int(v)
}

//export GetClientNetConnection
func GetClientNetConnection(n C.int) C.int {
	return C.int(sv_clients[int(n)].netConnection)
}

//export SetClientNetConnection
func SetClientNetConnection(n C.int, v C.int) {
	sv_clients[int(n)].netConnection = int(v)
}

//export GetClientActive
func GetClientActive(n C.int) C.int {
	return b2i(sv_clients[int(n)].active)
}

//export SetClientActive
func SetClientActive(n C.int, v C.int) {
	sv_clients[int(n)].active = (v != 0)
}

//export GetClientSpawned
func GetClientSpawned(n C.int) C.int {
	return b2i(sv_clients[int(n)].spawned)
}

//export SetClientSpawned
func SetClientSpawned(n C.int, v C.int) {
	sv_clients[int(n)].spawned = (v != 0)
}

//export GetClientSendSignon
func GetClientSendSignon(n C.int) C.int {
	return b2i(sv_clients[int(n)].sendSignon)
}

//export SetClientSendSignon
func SetClientSendSignon(n C.int, v C.int) {
	sv_clients[int(n)].sendSignon = (v != 0)
}

//export ClientWriteChar
func ClientWriteChar(num, c C.int) {
	sv_clients[int(num)].msg.WriteChar(int(c))
}

//export ClientWriteByte
func ClientWriteByte(num, c C.int) {
	sv_clients[int(num)].msg.WriteByte(int(c))
}

//export ClientWriteShort
func ClientWriteShort(num, c C.int) {
	sv_clients[int(num)].msg.WriteShort(int(c))
}

//export ClientWriteLong
func ClientWriteLong(num, c C.int) {
	sv_clients[int(num)].msg.WriteLong(int(c))
}

//export ClientWriteFloat
func ClientWriteFloat(num C.int, f C.float) {
	sv_clients[int(num)].msg.WriteFloat(float32(f))
}

//export ClientWriteString
func ClientWriteString(num C.int, s *C.char) {
	sv_clients[int(num)].msg.WriteString(C.GoString(s))
}

//export ClientWriteCoord
func ClientWriteCoord(num C.int, f C.float, flags C.uint) {
	sv_clients[int(num)].msg.WriteCoord(float32(f), int(flags))
}

//export ClientWriteAngle
func ClientWriteAngle(num C.int, f C.float, flags C.uint) {
	sv_clients[int(num)].msg.WriteAngle(float32(f), int(flags))
}

//export ClientWriteAngle16
func ClientWriteAngle16(num C.int, f C.float, flags C.uint) {
	sv_clients[int(num)].msg.WriteAngle16(float32(f), int(flags))
}

//export ClientWrite
func ClientWrite(num C.int, data *C.uchar, length C.int) {
	b := C.GoBytes(unsafe.Pointer(data), length)
	sv_clients[int(num)].msg.WriteBytes(b)
}

//export ClientWriteSVMSG
func ClientWriteSVMSG(num C.int) {
	sv_clients[int(num)].msg.WriteBytes(msgBuf.Bytes())
}

//export ClientHasMessage
func ClientHasMessage(num C.int) C.int {
	return b2i(sv_clients[int(num)].msg.HasMessage())
}

//export ClientClearMessage
func ClientClearMessage(num C.int) {
	sv_clients[int(num)].msg.ClearMessage()
}

func (cl *SVClient) SendMessage() int {
	return net.SendMessage(cl.netConnection, cl.msg.Bytes())
}

//export ClientSendMessage
func ClientSendMessage(num C.int) C.int {
	return C.int(sv_clients[int(num)].SendMessage())
}

//export GetClientOverflowed
func GetClientOverflowed(num C.int) C.int {
	// return b2i(sv_clients[int(num)].msg.Len() > 64000)
	// Do we care?
	return 0
}

//export SetClientOverflowed
func SetClientOverflowed(num, v C.int) {
	log.Printf("SetOverflow")
}

func (cl *SVClient) SendNop() {
	if net.SendUnreliableMessage(cl.netConnection, []byte{server.Nop}) == -1 {
		C.SV_DropClient(C.int(cl.id), 1 /* crashed == true */)
	}
	cl.lastMessage = host.time
}

//export SV_SendNop
func SV_SendNop(num C.int) {
	sv_clients[int(num)].SendNop()
}
