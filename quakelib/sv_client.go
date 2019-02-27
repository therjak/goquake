package quakelib

/*
#include <stdlib.h>
#include <string.h>

#ifndef _MOVEDEF_h
#define _MOVEDEF_h
typedef struct movecmd_s {
	float forwardmove;
	float sidemove;
	float upmove;
} movecmd_t;
#endif // _MOVEDEF_h
void SV_DropClient(int, int);
void SV_ConnectClient(int);
int HostClient(void);
*/
import "C"

import (
	"bytes"
	"fmt"
	"log"
	"quake/cvars"
	"quake/net"
	"quake/protocol"
	"quake/protocol/server"
	"time"
	"unsafe"
)

type SVClient struct {
	active     bool // false = client is free
	spawned    bool // false = don't send datagrams
	sendSignon bool // only valid before spawned

	// reliable messages must be sent periodically
	lastMessage float64

	netConnection *net.Connection // communications handle

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

	badRead bool
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

//export SV_CheckForNewClients
func SV_CheckForNewClients() {
	CheckForNewClients()
}

//export SV_ClientPrint2
func SV_ClientPrint2(client C.int, msg *C.char) {
	sv_clients[int(client)].ClientPrint(C.GoString(msg))
	//	ClientWriteByte(client, server.Print)
	//	ClientWriteString(client, msg)
}

//export SV_BroadcastPrint2
func SV_BroadcastPrint2(msg *C.char) {
	SV_BroadcastPrint(C.GoString(msg))
}

func SV_BroadcastPrintf(format string, v ...interface{}) {
	SV_BroadcastPrint(fmt.Sprintf(format, v...))
}

func SV_BroadcastPrint(m string) {
	for _, c := range sv_clients {
		if c.active && c.spawned {
			c.ClientPrint(m)
		}
	}
}

func HostClient() *SVClient {
	return sv_clients[HostClientID()]
}

func HostClientID() int {
	return int(C.HostClient())
}

func (c *SVClient) ClientPrint(msg string) {
	c.msg.WriteByte(server.Print)
	c.msg.WriteString(msg)
}

func (c *SVClient) ClientCommands(msg string) {
	c.msg.WriteByte(server.StuffText)
	c.msg.WriteString(msg)
}

func (c *SVClient) PingTime() float32 {
	r := float32(0)
	for _, p := range c.pingTimes {
		r += p
	}
	return r / float32(len(c.pingTimes))
}

func CheckForNewClients() {
	for {
		con := net.CheckNewConnections()
		if con == nil {
			return
		}
		foundFree := false
		for _, c := range sv_clients {
			if c.active {
				continue
			}
			foundFree = true
			c.netConnection = con
			C.SV_ConnectClient(C.int(c.id))
			break
		}
		if !foundFree {
			Error("Host_CheckForNewClients: no free clients")
		}
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

func ClientWriteByte2(num, c int) {
	sv_clients[num].msg.WriteByte(c)
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

func ClientWriteString2(num int, s string) {
	sv_clients[num].msg.WriteString(s)
}

//export ClientWriteCoord
func ClientWriteCoord(num C.int, f C.float) {
	sv_clients[int(num)].msg.WriteCoord(float32(f), int(sv.protocolFlags))
}

//export ClientWriteAngle
func ClientWriteAngle(num C.int, f C.float) {
	sv_clients[int(num)].msg.WriteAngle(float32(f), int(sv.protocolFlags))
}

//export ClientWriteAngle16
func ClientWriteAngle16(num C.int, f C.float) {
	sv_clients[int(num)].msg.WriteAngle16(float32(f), int(sv.protocolFlags))
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

//export ClientCanSendMessage
func ClientCanSendMessage(num C.int) C.int {
	return b2i(sv_clients[int(num)].CanSendMessage())
}

func (cl *SVClient) CanSendMessage() bool {
	return cl.netConnection.CanSendMessage()
}

//export ClientClose
func ClientClose(num C.int) {
	sv_clients[int(num)].Close()
}

func (cl *SVClient) Close() {
	cl.netConnection.Close()
	cl.netConnection = nil
	cl.active = false
	cl.name = ""
	cl.oldFrags = -999999
}

//export ClientConnectTime
func ClientConnectTime(num C.int) C.double {
	return C.double(sv_clients[int(num)].ConnectTime())
}

func (cl *SVClient) ConnectTime() float64 {
	return cl.netConnection.ConnectTime()
}

//export ClientAddress
func ClientAddress(num C.int, ret *C.char, n C.size_t) {
	s := sv_clients[int(num)].Address()
	if len(s) >= int(n) {
		s = s[:n-1]
	}
	sp := C.CString(s)
	defer C.free(unsafe.Pointer(sp))
	C.strncpy(ret, sp, n)
}

func (cl *SVClient) Address() string {
	return cl.netConnection.Address()
}

//export ClientClearMessage
func ClientClearMessage(num C.int) {
	sv_clients[int(num)].msg.ClearMessage()
}

func (cl *SVClient) SendMessage() int {
	return cl.netConnection.SendMessage(cl.msg.Bytes())
}

//export ClientSendMessage
func ClientSendMessage(num C.int) C.int {
	return C.int(sv_clients[int(num)].SendMessage())
}

//export ClientGetMessage
func ClientGetMessage(num C.int) C.int {
	return C.int(sv_clients[int(num)].GetMessage())
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
	if cl.netConnection.SendUnreliableMessage([]byte{server.Nop}) == -1 {
		C.SV_DropClient(C.int(cl.id), 1 /* crashed == true */)
	}
	cl.lastMessage = host.time
}

//export SV_SendNop
func SV_SendNop(num C.int) {
	sv_clients[int(num)].SendNop()
}

//export SV_SendDisconnectToAll
func SV_SendDisconnectToAll() {
	SendToAll([]byte{server.Disconnect})
}

func SendReconnectToAll() {
	s := "reconnect\n\x00"
	m := make([]byte, 0, len(s)+1)
	buf := bytes.NewBuffer(m)
	buf.WriteByte(server.StuffText)
	buf.WriteString(s)
	SendToAll(buf.Bytes())
}

func SendToAll(data []byte) {
	// We try for 5 seconds to send the message to everyone
	s := make([]bool, len(sv_clients))
	start := time.Now()
TimeoutLoop:
	for {
		if time.Now().Sub(start) > 5*time.Second {
			return
		}
		for i, c := range sv_clients {
			if s[i] {
				continue
			}
			if !c.active {
				s[i] = true
				continue
			}
			if c.CanSendMessage() {
				c.netConnection.SendMessage(data)
				s[i] = true
			}
		}
		for _, c := range s {
			if !c {
				// There is no need to spin too fast, we are waiting for
				// the last ACK of one of the clients.
				time.Sleep(time.Millisecond)
				continue TimeoutLoop
			}
		}
		return
	}
}

var (
	// There is only one reader which gets switched for each client
	msg_badread = false
	netMessage  *net.QReader
)

func (cl *SVClient) GetMessage() int {
	msg_badread = false
	r, err := cl.netConnection.GetMessage()
	if err != nil {
		return -1
	}
	if r == nil || r.Len() == 0 {
		return 0
	}
	netMessage = r
	b, err := r.ReadByte()
	if err != nil {
		return -1
	}
	return int(b)
}

//export MSG_BadRead
func MSG_BadRead() C.int {
	// poor mans error handling :(
	return b2i(msg_badread)
}

//export MSG_ReadChar
func MSG_ReadChar() C.int {
	i, err := netMessage.ReadInt8()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadByte
func MSG_ReadByte() C.int {
	i, err := netMessage.ReadByte()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadShort
func MSG_ReadShort() C.int {
	i, err := netMessage.ReadInt16()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadLong
func MSG_ReadLong() C.int {
	i, err := netMessage.ReadInt32()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.int(i)
}

//export MSG_ReadFloat
func MSG_ReadFloat() C.float {
	f, err := netMessage.ReadFloat32()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord16
func MSG_ReadCoord16() C.float {
	f, err := netMessage.ReadCoord16()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord24
func MSG_ReadCoord24() C.float {
	f, err := netMessage.ReadCoord24()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord32f
func MSG_ReadCoord32f() C.float {
	f, err := netMessage.ReadCoord32f()
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadCoord
func MSG_ReadCoord() C.float {
	f, err := netMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadAngle
func MSG_ReadAngle() C.float {
	f, err := netMessage.ReadAngle(uint32(sv.protocolFlags))
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export MSG_ReadAngle16
func MSG_ReadAngle16() C.float {
	f, err := netMessage.ReadAngle16(uint32(sv.protocolFlags))
	if err != nil {
		msg_badread = true
		return -1
	}
	return C.float(f)
}

//export SV_SendServerinfo
func SV_SendServerinfo(client C.int) {
	c := sv_clients[int(client)]
	m := &c.msg
	m.WriteByte(server.Print)
	m.WriteString(
		fmt.Sprintf("%s\nFITZQUAKE %1.2f SERVER (%d CRC)\n",
			[]byte{2}, FITZQUAKE_VERSION, progsdat.CRC))

	m.WriteByte(int(server.ServerInfo))
	m.WriteLong(int(sv.protocol))

	if sv.protocol == protocol.RMQ {
		m.WriteLong(int(sv.protocolFlags))
	}

	c.msg.WriteByte(svs.maxClients)

	if !cvars.Coop.Bool() && cvars.DeathMatch.Bool() {
		m.WriteByte(server.GameDeathmatch)
	} else {
		m.WriteByte(server.GameCoop)
	}

	m.WriteString(PR_GetStringWrap(int(EntVars(0).Message)))

	for i, mn := range sv.modelPrecache[1:] {
		if sv.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		log.Printf("Model: %v", mn)
		m.WriteString(mn)
	}
	m.WriteByte(0)

	for i, sn := range sv.soundPrecache[1:] {
		if sv.protocol == protocol.NetQuake && i >= 256 {
			break
		}
		m.WriteString(sn)
	}
	m.WriteByte(0)

	m.WriteByte(server.CDTrack)
	m.WriteByte(int(EntVars(0).Sounds))
	m.WriteByte(int(EntVars(0).Sounds))

	m.WriteByte(server.SetView)
	m.WriteShort(c.edictId)

	m.WriteByte(server.SignonNum)
	m.WriteByte(1)

	c.sendSignon = true
	c.spawned = false
}

//export SV_ModelIndex
func SV_ModelIndex(name *C.char) C.int {
	if name == nil {
		return 0
	}
	n := C.GoString(name)
	if len(n) == 0 {
		return 0
	}
	for i, m := range sv.modelPrecache {
		if m == n {
			return C.int(i)
		}
	}
	Error("SV_ModelINdex: model %v not precached", n)
	return 0
}
