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
const (
	maxMessage = 32008
	// make the channel buffer larger than 1 as we need to
	// consider unreliable messages as well and they should not block
	// the channel.
	chanBufLength = 4
)

type Connection struct {
	connectTime  time.Duration
	con          net.Conn
	addr         string
	id           int
	in           <-chan msg
	out          chan<- msg
	canWriteChan <-chan bool
	canWrite     bool
}

func (c *Connection) ID() int {
	return c.id
}

type msg struct {
	data []byte
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
	serverName         = "127.0.0.1" //"MyServer"
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
	fmt.Printf("Go GetCon oob %v\n", id)
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
	return con.ConnectTime(), nil
}

func (c *Connection) ConnectTime() float64 {
	return c.connectTime.Seconds()
}

func Address(id int) (string, error) {
	con, err := getCon(id)
	if err != nil {
		return "", err
	}
	return con.Address(), nil
}

func (c *Connection) Address() string {
	return c.addr
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
	// canWrite needs to be buffered as the chan is only read from
	// if a new reliable message should be send. And we do not want
	// to block the receiving chan.
	canWrite := make(chan bool, 1)
	client := &Connection{
		connectTime:  netTime,
		con:          c,
		addr:         c.RemoteAddr().String(),
		id:           clientID,
		in:           s2c,
		out:          c2s,
		canWriteChan: canWrite,
		canWrite:     true,
	}
	cons = append(cons, *client)
	acks := make(chan uint32, 1)
	go readUDP(c, s2c, acks)
	go writeUDP(c, c2s, acks, canWrite)
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

	NETFLAG_LENGTH_MASK = 0x0000ffff
	NETFLAG_FLAG_MASK   = 0xffff0000
	NETFLAG_DATA        = 0x00010000
	NETFLAG_ACK         = 0x00020000
	NETFLAG_NAK         = 0x00040000
	NETFLAG_EOM         = 0x00080000
	NETFLAG_UNRELIABLE  = 0x00100000
	NETFLAG_CTL         = 0x80000000

	MAX_DATAGRAM = 32000
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
	if control&NETFLAG_FLAG_MASK != NETFLAG_CTL ||
		control&NETFLAG_LENGTH_MASK != uint32(i) {
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

func readUDP(c net.Conn, out chan<- msg, acks chan<- uint32) {
	// Read
	defer c.Close()

	defer close(out)
	defer close(acks)

	unreliableSequence := uint32(0)
	receiveSequence := uint32(0)
	var reliableBuf bytes.Buffer
	var unreliableBuf bytes.Buffer
	var ack bytes.Buffer
	reliableBuf.WriteByte(1)
	for {
		b := make([]byte, maxMessage)
		i, err := c.Read(b)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read failed: %v", err)
			}
			return
		}
		if i < 8 { /* net header == 2 int32 */
			continue
		}
		// first 4 byte are flag|length
		// second 4 bytes are sequence number
		// all other is data
		b = b[:i]
		var length, sequence uint32
		buf := bytes.NewBuffer(b)
		// We verified the length already. No read error possible.
		binary.Read(buf, binary.BigEndian, &length)
		binary.Read(buf, binary.BigEndian, &sequence)
		flags := length & NETFLAG_FLAG_MASK
		length = length & NETFLAG_LENGTH_MASK
		if uint32(i) != length {
			// Just ignore this message. It seems broken.
			continue
		}
		if flags&NETFLAG_CTL != 0 {
			continue
		} else if flags&NETFLAG_UNRELIABLE != 0 {
			if sequence < unreliableSequence {
				// Got a stale datagram
				continue
			}
			unreliableSequence = sequence + 1
			unreliableBuf.Reset()
			// we need to pass the information of unreliable forward, add the 2
			unreliableBuf.WriteByte(2)
			unreliableBuf.Write(buf.Bytes())
			// make sure the data moved out is a different slice
			o := make([]byte, unreliableBuf.Len())
			copy(o, unreliableBuf.Bytes())
			m := msg{data: o}
			out <- m
			continue
		} else if flags&NETFLAG_ACK != 0 {
			acks <- sequence
			continue
		} else if flags&NETFLAG_DATA != 0 {
			// We may have received this packet already but the ack was not received.
			ack.Reset()
			binary.Write(&ack, binary.BigEndian, uint32(8|NETFLAG_ACK))
			binary.Write(&ack, binary.BigEndian, uint32(sequence))
			c.Write(ack.Bytes())
			if sequence != receiveSequence {
				// not the packet we expect, ignore,
				// could be a resend because of missed ACK
				continue
			}
			receiveSequence++
			reliableBuf.Write(buf.Bytes())
			if flags&NETFLAG_EOM != 0 {
				// we need to pass the information of reliable forward, add the 1
				// make sure the data moved out is a different slice
				o := make([]byte, reliableBuf.Len())
				copy(o, reliableBuf.Bytes())
				out <- msg{data: o}
				reliableBuf.Reset()
				reliableBuf.WriteByte(1)
			}
			continue
		}
	}
}

func writeUDP(c net.Conn, in <-chan msg, acks <-chan uint32, canWrite chan<- bool) {
	unreliableSequence := uint32(0)
	sendSequence := uint32(0)
	ackSequence := uint32(0)
	var reliableMsg []byte
	var sendBuf bytes.Buffer
	defer c.Close()
	defer close(canWrite)
	resendTimer := time.NewTimer(time.Second)
	if !resendTimer.Stop() {
		<-resendTimer.C
	}
	for {
		// handle ack
		select {
		case sequence, ok := <-acks:
			if !ok {
				return
			}
			if sequence != sendSequence-1 {
				log.Printf("Wrong sendSequence")
				continue
			}
			if sequence != ackSequence {
				log.Printf("Wrong AckSequence")
				continue
			}
			ackSequence++
			if ackSequence != sendSequence {
				log.Printf("ack sequencing error")
			}
			// remove last message
			if len(reliableMsg) > MAX_DATAGRAM {
				reliableMsg = reliableMsg[MAX_DATAGRAM:]
			} else {
				reliableMsg = reliableMsg[:0]
			}
			if !resendTimer.Stop() {
				if len(resendTimer.C) != 0 {
					<-resendTimer.C
				}
			}
			if len(reliableMsg) != 0 {
				// So we got our last reliableMsg acked and the packet was larger than
				// MAX_DATAGRAM, so send next packet
				length := MAX_DATAGRAM + 8
				eom := 0
				if len(reliableMsg) <= MAX_DATAGRAM {
					length = len(reliableMsg) + 8
					eom = NETFLAG_EOM
				}
				sendBuf.Reset()
				binary.Write(&sendBuf, binary.BigEndian, uint32(length|NETFLAG_DATA|eom))
				binary.Write(&sendBuf, binary.BigEndian, uint32(sendSequence))
				sendSequence++
				sendBuf.Write(reliableMsg[:length-8])
				_, err := c.Write(sendBuf.Bytes())
				if err != nil {
					log.Printf("Write failed: %v", err)
					return
				}
				resendTimer.Reset(time.Second)
				continue
			} else {
				if len(canWrite) == 0 {
					// We only need to ensure the next one who asks gets
					// the right answer. It does not matter how many we
					// send. And as the channel would never be drained
					// we better not try to write more than 1 message.
					// As this is udp we have no guarantee of how many
					// ACKS we receive for the last reliable msg.
					canWrite <- true
				}
			}

		case <-resendTimer.C:
			length := MAX_DATAGRAM + 8
			eom := 0
			if len(reliableMsg) <= MAX_DATAGRAM {
				length = len(reliableMsg) + 8
				eom = NETFLAG_EOM
			}
			sendBuf.Reset()
			binary.Write(&sendBuf, binary.BigEndian, uint32(length|NETFLAG_DATA|eom))
			binary.Write(&sendBuf, binary.BigEndian, uint32(sendSequence-1))
			sendBuf.Write(reliableMsg[:length-8])
			_, err := c.Write(sendBuf.Bytes())
			if err != nil {
				log.Printf("Write failed: %v", err)
				return
			}
			resendTimer.Reset(time.Second)

		case msg, isOpen := <-in:
			// first byte of msg indicates reliable/unreliable
			// 1 is reliable, 2 unreliable
			// do not send this byte out
			if !isOpen {
				log.Printf("c2s is closed")
				return
			}
			switch msg.data[0] {
			case 1:
				reliableMsg = msg.data[1:]

				length := MAX_DATAGRAM + 8
				eom := 0
				if len(reliableMsg) <= MAX_DATAGRAM {
					length = len(reliableMsg) + 8
					eom = NETFLAG_EOM
				}
				sendBuf.Reset()
				binary.Write(&sendBuf, binary.BigEndian, uint32(length|NETFLAG_DATA|eom))
				binary.Write(&sendBuf, binary.BigEndian, uint32(sendSequence))
				sendSequence++
				sendBuf.Write(reliableMsg[:length-8])
				_, err := c.Write(sendBuf.Bytes())
				if err != nil {
					log.Printf("Write failed: %v", err)
					return
				}
				resendTimer.Reset(time.Second)
			case 2:
				// 8 byte 'header' + data
				length := len(msg.data) - 1 /*reliable bit*/ + 8 /*net header*/
				sendBuf.Reset()
				binary.Write(&sendBuf, binary.BigEndian, uint32(length|NETFLAG_UNRELIABLE))
				binary.Write(&sendBuf, binary.BigEndian, uint32(unreliableSequence))
				unreliableSequence++
				sendBuf.Write(msg.data[1:])
				// keep all in one write operation
				_, err := c.Write(sendBuf.Bytes())
				if err != nil {
					log.Printf("Write failed: %v", err)
					return
				}
			default:
				log.Printf("WTF %d", msg.data[0])
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
		canWrite:    true,
	}
	cons = append(cons, *loopClient)
	serverID := getNextConID()
	loopServer = &Connection{
		connectTime: netTime,
		addr:        "LOCAL",
		id:          serverID,
		in:          c2s,
		out:         s2c,
		canWrite:    true,
	}
	cons = append(cons, *loopServer)
	return loopClient, nil
}

func CheckNewConnections() *Connection {
	// loopback only
	if !loopConnectPending {
		return nil
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
	return loopServer
}

func Close(id int) {
	SetTime()
	con, err := getCon(id)
	if err == nil {
		con.Close()
	}
}

func (c *Connection) Close() {
	SetTime()
	if c.con != nil {
		c.con.Close()
	} else {
		// loop server/client
		close(c.out)
	}
	c.canWriteChan = nil
	c.canWrite = false
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
	if c.canWriteChan != nil {
		// TODO: there should be a better way to handle both loopback and udp
		c.canWrite = false
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
		log.Printf("Ignored sending as c.out is full? %d, %d",
			cap(c.out), len(c.out))
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
		log.Printf("CanSend nil false")
		return false
	}
	SetTime()
	if c.canWriteChan != nil {
		select {
		case can, ok := <-c.canWriteChan:
			log.Printf("CanSend can: %v", can)
			c.canWrite = ok && can
		default:
		}
	}
	if !c.canWrite {
		log.Printf("CanSend canWrite false")
		return false
	}

	return len(c.out) < cap(c.out)
}

func Shutdown() {
	SetTime()
	// nothing to do for loopback
	// otherwise we should close the 'init' connection
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
