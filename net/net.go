package net

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	prfl "quake/protocol/flags"
	svc "quake/protocol/server"
	"quake/qtime"
	"sort"
	"strings"
	"time"
)

// qboolean = C.int
// true = 1, false = 0
/*
var (
	controlAddr net.UDPAddr
	controlCon  *net.UDPConn
	serverAddr  net.UDPAddr
)
*/
const (
	maxMessage = 32008
	// make the channel buffer larger than 1 as we need to
	// consider unreliable messages as well and they should not block
	// the channel.
	chanBufLength = 4
)

type Connection struct {
	connectTime time.Duration
	//	con         *net.UDPConn
	addr string
	id   int
	in   <-chan msg
	out  chan<- msg
}

func (c *Connection) ID() int {
	return c.id
}

type msg struct {
	data []byte
	// TODO: should this be with array length?
	// data [maxMessage]byte
}

type QReader struct {
	r *bytes.Reader
}

func NewQReader(data []byte) *QReader {
	return &QReader{bytes.NewReader(data)}
}

var (
	cons               []Connection
	netTime            time.Duration
	loopClient         *Connection
	loopServer         *Connection
	loopConnectPending = false
	netMessage         QReader
	netMessageBackup   QReader
	port               = 26000
	myip               = "127.0.0.1"
	serverName         = "MyServer"
)

func Port() int {
	return port
}

func SetPort(p int) {
	port = p
}

func MyIP() string {
	return myip
}

func ServerName() string {
	return serverName
}

func getCon(id int) (*Connection, error) {
	for _, c := range cons {
		if c.id == id {
			return &c, nil
		}
	}
	fmt.Printf("Go GetCon oob \n")
	return nil, errors.New("Out of bounds connection")
}

func getNextConID() int {
	sort.Slice(cons, func(i, j int) bool { return cons[i].id < cons[j].id })
	pos := 1
	for _, c := range cons {
		if c.id == pos {
			pos = pos + 1
		}
	}
	return pos
}

func ConnectTime(id int) (float64, error) {
	fmt.Printf("Go ConnectTime to %v\n", id)
	con, err := getCon(id)
	if err != nil {
		return 0, err
	}
	return con.connectTime.Seconds(), nil
}

func Address(id int) (string, error) {
	con, err := getCon(id)
	if err != nil {
		return "", err
	}
	return con.addr, nil
}

func SetTime() {
	netTime = qtime.QTime()
}

func Time() float64 {
	return netTime.Seconds()
}

func Connect(host string) (*Connection, error) {
	SetTime()
	// loopback only
	if strings.ToLower(host) != "local" {
		return udpConnect(host, port)
	}
	return localConnect()
}

func udpConnect(host string, port int) (*Connection, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	laddr, newPort, err := handShake(addr)
	if err != nil {
		log.Printf("Handshake err: %v", err)
		return nil, fmt.Errorf("Handshake failed: %v", err)
	}

	addr = net.JoinHostPort(host, fmt.Sprintf("%d", newPort))
	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("Could not resolve address %v: %v", host, err)
	}
	c, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to host %v: %v", host, err)
	}

	clientID := getNextConID()
	c2s := make(chan msg, chanBufLength)
	s2c := make(chan msg, chanBufLength)
	client := &Connection{
		connectTime: netTime,
		addr:        c.RemoteAddr().String(),
		id:          clientID,
		in:          s2c,
		out:         c2s,
	}
	cons = append(cons, *client)
	go readUDP(c, s2c)
	go writeUDP(c, c2s)
	return client, nil
}

const (
	CCREQ_CONNECT = 0x01
	// gameName string 'QUAKE'
	// netProtocolVer byte '3'
	CCREQ_SERVER_INFO = 0x02
	// gameName string 'QUAKE'
	// netProtocolVer byte '3'
	CCREQ_PLAYER_INFO = 0x03
	// playerNum byte
	CCREQ_RULE_INFO = 0x04
	// rule string

	CCREP_ACCEPT = 0x81
	// port int32
	CCREP_REJECT = 0x82
	// reason string
	CCREP_SERVER_INFO = 0x83
	// address string
	// name string
	// level string
	// current_players byte
	// max_players byte
	// protocolVersion byte
	CCREP_PLAYER_INFO = 0x84
	// playerNum byte
	// name string
	// color int32
	// frags int32
	// connectTime int32
	// address string
	CCREP_RULE_INFO = 0x85
	// rule string
	// value string

	NET_PROTOCOL_VERSION = 3
)

func handShake(host string) (*net.UDPAddr, int, error) {
	radd, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		return nil, 0, fmt.Errorf("Could not resolve address %v: %v", host, err)
	}
	c, err := net.DialUDP("udp", nil, radd)
	if err != nil {
		return nil, 0, fmt.Errorf("Could not connect to host %v: %v", host, err)
	}
	defer c.Close()

	var msg bytes.Buffer
	// NETFLAG_CTL(0x80000000) | length
	binary.Write(&msg, binary.BigEndian, uint32(0x8000000C))
	msg.Write([]byte{CCREQ_CONNECT})
	msg.Write([]byte("QUAKE\x00"))
	msg.Write([]byte{NET_PROTOCOL_VERSION})

	i, err := c.Write(msg.Bytes())
	if err != nil {
		return nil, 0, err
	}
	if i != msg.Len() {
		return nil, 0, fmt.Errorf("Did not send full message")
	}

	b := make([]byte, maxMessage)
	i, err = c.Read(b)
	if err != nil {
		return nil, 0, err
	}
	if i < 9 {
		return nil, 0, fmt.Errorf("Return to small: %v", i)
	}
	msg = *bytes.NewBuffer(b[:i])
	var control uint32
	binary.Read(&msg, binary.BigEndian, &control)
	if control&0xffff0000 != 0x80000000 ||
		control&0x0000ffff != uint32(i) {
		return nil, 0, fmt.Errorf("Error in reply")
	}
	ack, err := msg.ReadByte()
	if ack == CCREP_REJECT {
		s, err := msg.ReadString(0x00)
		if err != nil {
			return nil, 0, err
		}
		return nil, 0, fmt.Errorf("Connection request rejected: %s", s)
	}
	if ack != CCREP_ACCEPT {
		return nil, 0, fmt.Errorf("Bad Response")
	}

	var sockAddr uint32
	binary.Read(&msg, binary.LittleEndian, &sockAddr)
	addr := c.LocalAddr()
	laddr, _ := net.ResolveUDPAddr(addr.Network(), addr.String())
	return laddr, int(sockAddr), nil
}

func readUDP(c net.Conn, out chan<- msg) {
	// Read
	// TODO: weird stuff from Datagram_GetMessage
	defer c.Close()
	defer close(out)
	unreliableSequence := uint32(0)
	ackSequence := uint32(0)
	receiveSequence := uint32(0)
	for {
		b := make([]byte, maxMessage)
		i, err := c.Read(b)
		// NETFLAG_LENGTH_MASK = 0x0000ffff
		if err != nil {
			log.Printf("Read failed: %v", err)
			return
		}
		if i < 8 { /* net header == 2 int */
			continue
		}
		// first 4 byte are length
		// second 4 bytes are sequence number
		// all other is data
		b = b[:i]
		var length, sequence uint32
		buf := bytes.NewReader(b)
		// We verified the length already. No error possible.
		binary.Read(buf, binary.BigEndian, &length)
		binary.Read(buf, binary.BigEndian, &sequence)
		flags := length & 0xffff0000
		length = length & 0x0000ffff
		if uint32(i) != length {
			// Just ignore this message. It seems broken.
			continue
		}
		log.Printf("Real: %d, Reported: %d", i, length)
		switch flags {
		case 0x80000000 /*NETFLAG_CTL*/ :
			continue
		case 0x00100000 /*NETFLAG_UNRELIABLE*/ :
			if sequence < unreliableSequence {
				continue
			}
			unreliableSequence = sequence + 1
			out <- msg{data: b[8:]}
			break
		case 0x00040000 /*NETFLAG_ACK*/ :
			ackSequence++
			continue
		case 0x00020000 /*NETFLAG_DATA*/ :
			receiveSequence++
			// Send (8|ACK)(sequence)
			// if sequence != receiveSequence {
			//   continue
			// }
			// receiveSequence++
			// if flags & 0x00080000 /*NETFLAG_EOM*/ {
			// ret = 1
			// out <- msg{data: b[8:]}
			// break
			// }
			continue
		}
	}
}

func writeUDP(c net.Conn, in <-chan msg) {
	unreliableSequence := uint32(0)
	sendSequence := uint32(0)
	defer c.Close()
	for {
		select {
		case msg, isOpen := <-in:
			// first byte of msg indicates reliable/unreliable
			// 1 is reliable, 2 unreliable
			// do not send this byte out
			if isOpen {
				switch msg.data[0] {
				case 1:
					sendSequence++
					_, err := c.Write(msg.data[1:])
					if err != nil {
						log.Printf("Write failed: %v", err)
						return
					}
				case 2:
					// 8 byte 'header' + data
					length := len(msg.data) - 1 /*reliable bit*/ + 8 /*net header*/
					var buf bytes.Buffer
					binary.Write(&buf, binary.BigEndian, length|0x00100000)
					binary.Write(&buf, binary.BigEndian, unreliableSequence)
					unreliableSequence++
					buf.Write(msg.data[1:])
					// keep all in one write operation
					_, err := c.Write(buf.Bytes())
					if err != nil {
						log.Printf("Write failed: %v", err)
						return
					}
				}
			} else {
				log.Printf("c2s is closed")
				return
			}
		}
	}
}

func localConnect() (*Connection, error) {
	loopConnectPending = true
	clientID := getNextConID()
	c2s := make(chan msg, chanBufLength)
	s2c := make(chan msg, chanBufLength)
	loopClient = &Connection{
		connectTime: netTime,
		addr:        "localhost",
		id:          clientID,
		in:          s2c,
		out:         c2s,
	}
	cons = append(cons, *loopClient)
	serverID := getNextConID()
	loopServer = &Connection{
		connectTime: netTime,
		addr:        "LOCAL",
		id:          serverID,
		in:          c2s,
		out:         s2c,
	}
	cons = append(cons, *loopServer)
	return loopClient, nil
}

func CheckNewConnections() int {
	// loopback only
	if !loopConnectPending {
		return 0
	}
	loopConnectPending = false
	// fmt.Printf("Go CheckNewConnections2 %v\n", loopServer.id)
	SetTime()
	//Dangerous chan clear
	for len(loopServer.in) > 0 {
		<-loopServer.in
	}
	for len(loopClient.in) > 0 {
		<-loopClient.in
	}
	// server socket: on the server, connection to the client
	// return server socket... what should be returned?
	return loopServer.id
}

func Close(id int) {
	SetTime()
	// TODO: see loop.Close
	con, err := getCon(id)
	if err == nil {
		con.Close()
	}
}

func (c *Connection) Close() {
	log.Printf("Closing connectiong")
	SetTime()
	close(c.out)
	for i, _ := range cons {
		if cons[i].ID() == c.ID() {
			cons = append(cons[:i], cons[i+1:]...)
			return
		}
	}
	// TODO: see loop.Close
}

func GetMessage(id int) int {
	con, err := getCon(id)
	if err != nil {
		return -1
	}
	r, err := con.GetMessage()
	if err != nil {
		return -1
	}
	if r == nil || r.Len() == 0 {
		return 0
	}
	netMessage = *r
	b, err := r.ReadByte()
	if err != nil {
		return -1
	}
	return int(b)
}

func (c *Connection) GetMessage() (*QReader, error) {
	if c.in == nil {
		return nil, fmt.Errorf("Connection is not established")
	}
	SetTime()
	select {
	case m, isOpen := <-c.in:
		if !isOpen {
			close(c.out)
			return nil, fmt.Errorf("Connection is not open")
		}
		return &QReader{bytes.NewReader(m.data)}, nil
	default:
		return nil, nil
	}
}

func SendMessage(id int, data []byte) int {
	con, err := getCon(id)
	if err != nil {
		return -1
	}
	return con.SendMessage(data)
}

func (c *Connection) SendMessage(data []byte) int {
	if c.out == nil {
		return -1
	}
	SetTime()

	m := make([]byte, 0, len(data)+1)
	buf := bytes.NewBuffer(m)
	buf.WriteByte(1)
	buf.Write(data)

	// there is some mechanism to allow multiple messages into the send buffer
	// in the original. does this need a larger channel buffer?
	c.out <- msg{data: buf.Bytes()}
	for len(c.out) < cap(c.out) {
		c.out <- msg{data: []byte{svc.Nop}}
	}
	return 1
}

func SendUnreliableMessage(id int, data []byte) int {
	con, err := getCon(id)
	if err != nil {
		return -1
	}
	return con.SendUnreliableMessage(data)
}

func (c *Connection) SendUnreliableMessage(data []byte) int {
	if c.out == nil {
		return -1
	}
	SetTime()

	if cap(c.out) < (len(c.out) + 1) {
		return 0
	}

	m := make([]byte, 0, len(data)+1)
	buf := bytes.NewBuffer(m)
	buf.WriteByte(2)
	buf.Write(data)

	// there is some mechanism to allow multiple messages into the send buffer
	// in the original. does this need a larger channel buffer?
	c.out <- msg{data: buf.Bytes()}
	return 1
}

func SendToAll(data []byte) int {
	// blockTime = 5.0
	// if NET_CanSendMessage
	//    init = true
	//    NET_SendMessage
	// else
	//    NET_GetMessage
	// and again to check if message was send
	// if NET_CanSendMessage
	//    send = true
	// else
	//    NET_GetMessage
	// until all are init and send or we exceeded blockTime
	// returns number clients which did not receive

	// loop only
	SendMessage(loopServer.id, data)
	return 0
}

func SendReconnectToAll() int {
	// Svc.StuffText,"reconnect\n"
	s := "reconnect\n\x00"
	m := make([]byte, 0, len(s)+1)
	buf := bytes.NewBuffer(m)
	buf.WriteByte(svc.StuffText)
	buf.WriteString(s)
	return SendToAll(buf.Bytes())
}

func CanSendMessage(id int) bool {
	con, err := getCon(id)
	if err != nil {
		return false
	}
	return con.CanSendMessage()
}

func (c *Connection) CanSendMessage() bool {
	// if channel is disconnected return false
	if c.out == nil {
		return false
	}
	// TODO
	// what does it mean to be disconnected?
	// where does this info come from?
	SetTime()
	return len(c.out) < cap(c.out)
}

func Shutdown() {
	SetTime()
	// nothing to do for loopback
}

func ReadInt8() (int8, error) {
	return netMessage.ReadInt8()
}

func (q *QReader) ReadInt8() (int8, error) {
	var r int8
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadByte() (byte, error) {
	return netMessage.ReadByte()
}

func UnreadByte() {
	netMessage.UnreadByte()
}

func (q *QReader) UnreadByte() {
	q.r.UnreadByte()
}

func (q *QReader) ReadByte() (byte, error) {
	var r byte
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadUint8() (uint8, error) {
	return netMessage.ReadUint8()
}

func (q *QReader) ReadUint8() (uint8, error) {
	var r uint8
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadInt16() (int16, error) {
	return netMessage.ReadInt16()
}

func (q *QReader) ReadInt16() (int16, error) {
	var r int16
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadInt32() (int32, error) {
	return netMessage.ReadInt32()
}

func (q *QReader) ReadInt32() (int32, error) {
	var r int32
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadFloat32() (float32, error) {
	return netMessage.ReadFloat32()
}

func (q *QReader) ReadFloat32() (float32, error) {
	var r float32
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func ReadCoord16() (float32, error) {
	return netMessage.ReadCoord16()
}

// 13.3 fixed point coords, max range +-4096
func (q *QReader) ReadCoord16() (float32, error) {
	i, err := q.ReadInt16()
	return float32(i) * (1.0 / 8.0), err
}

func ReadCoord24() (float32, error) {
	return netMessage.ReadCoord24()
}

// 16.8 fixed point coords, max range +-32768
func (q *QReader) ReadCoord24() (float32, error) {
	// We need to read both before handling the errors if we do not want to change
	// the logic.
	// TODO: Do we need to keep the logic?
	i16, err1 := q.ReadInt16()
	i8, err2 := q.ReadUint8()
	if err1 != nil {
		return 0, err1
	}
	if err2 != nil {
		return 0, err2
	}
	return float32(i16) + (float32(i8) * (1.0 / 255.0)), nil
}

func ReadCoord32f() (float32, error) {
	return netMessage.ReadCoord32f()
}

func (q *QReader) ReadCoord32f() (float32, error) {
	return q.ReadFloat32()
}

func ReadCoord(flags uint16) (float32, error) {
	return netMessage.ReadCoord(flags)
}

// TODO(therjak):
// it is not needed to always check these flags, just change the called function
// whenever cl.protocolflags would be changed
func (q *QReader) ReadCoord(flags uint16) (float32, error) {
	if flags&prfl.COORDFLOAT != 0 {
		return q.ReadFloat32()
	} else if flags&prfl.COORDINT32 != 0 {
		i, err := q.ReadInt32()
		return float32(i) * (1.0 / 16.0), err
	} else if flags&prfl.COORD24BIT != 0 {
		return q.ReadCoord24()
	}
	return q.ReadCoord16()
}

func ReadAngle(flags uint32) (float32, error) {
	return netMessage.ReadAngle(flags)
}

func (q *QReader) ReadAngle(flags uint32) (float32, error) {
	if flags&prfl.ANGLEFLOAT != 0 {
		return q.ReadFloat32()
	} else if flags&prfl.ANGLESHORT != 0 {
		i, err := q.ReadInt16()
		return float32(i) * (360.0 / 65536.0), err
	}
	i, err := q.ReadInt8()
	return float32(i) * (360.0 / 256.0), err
}

func ReadAngle16(flags uint32) (float32, error) {
	return netMessage.ReadAngle16(flags)
}

func (q *QReader) ReadAngle16(flags uint32) (float32, error) {
	if flags&prfl.ANGLEFLOAT != 0 {
		return q.ReadFloat32()
	}
	i, err := q.ReadInt16()
	return float32(i) * (360.0 / 65536.0), err
}

func Replace(data []byte) {
	netMessage = QReader{bytes.NewReader(data)}
}

func GetCurSize() int {
	return netMessage.Len()
}

// Len returns the number of bytes of the unread portion of the slice.
func (q *QReader) Len() int {
	return q.r.Len()
}

func BeginReading() {
	netMessage.BeginReading()
}

func (q *QReader) BeginReading() {
	i, _ := q.r.Seek(0, io.SeekCurrent)
	log.Printf("BeginReading while at %d", i)
	q.r.Seek(0, io.SeekStart)
}

func Backup() {
	netMessageBackup = netMessage
}

func Restore() {
	netMessage = netMessageBackup
}
