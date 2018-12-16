package net

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"quake/qtime"
	"strconv"
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
	in           <-chan msg
	out          chan<- msg
	canWriteChan <-chan bool
	canWrite     bool
}

type msg struct {
	data []byte
}

var (
	netTime            time.Duration
	loopClient         *Connection
	loopServer         *Connection
	loopConnectPending = false
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

func (c *Connection) ConnectTime() float64 {
	return c.connectTime.Seconds()
}

func (c *Connection) Address() string {
	if c.con != nil {
		return c.con.RemoteAddr().String()
	}
	// For the loopback variant
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

	c2s := make(chan msg, chanBufLength)
	s2c := make(chan msg, chanBufLength)
	// canWrite needs to be buffered as the chan is only read from
	// if a new reliable message should be send. And we do not want
	// to block the receiving chan.
	canWrite := make(chan bool, 1)
	client := &Connection{
		connectTime:  netTime,
		con:          c,
		in:           s2c,
		out:          c2s,
		canWriteChan: canWrite,
		canWrite:     true,
	}
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

	netProtocolVersion = 3

	NETFLAG_LENGTH_MASK = 0x0000ffff
	NETFLAG_FLAG_MASK   = 0xffff0000
	NETFLAG_DATA        = 0x00010000
	NETFLAG_ACK         = 0x00020000
	NETFLAG_NAK         = 0x00040000
	NETFLAG_EOM         = 0x00080000
	NETFLAG_UNRELIABLE  = 0x00100000
	NETFLAG_CTL         = 0x80000000

	MAX_DATAGRAM = 32000
	quake        = "QUAKE\x00"
)

const (
	// NETFLAG_CTL(0x80000000) | length, CCREQ_CONNECT, QUAKE\0,netProtocolVersion
	connectRequest = "\x80\x00\x00\x0c\x01QUAKE\x00\x03"
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

	i, err := c.Write([]byte(connectRequest))
	if err != nil {
		return nil, 0, err
	}
	if i != 0x0c {
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
	msg := *bytes.NewBuffer(b[:i])
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
	b := make([]byte, maxMessage)
	for {
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
		var length, sequence uint32
		reader := bytes.NewReader(b[:i])
		// We verified the length already. No read error possible.
		binary.Read(reader, binary.BigEndian, &length)
		binary.Read(reader, binary.BigEndian, &sequence)
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
			io.Copy(&unreliableBuf, reader)
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
			io.Copy(&reliableBuf, reader)
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
	c2s := make(chan msg, chanBufLength)
	s2c := make(chan msg, chanBufLength)
	loopClient = &Connection{
		connectTime: netTime,
		addr:        "localhost",
		in:          s2c,
		out:         c2s,
		canWrite:    true,
	}
	loopServer = &Connection{
		connectTime: netTime,
		addr:        "LOCAL",
		in:          c2s,
		out:         s2c,
		canWrite:    true,
	}
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
	// TODO: see loop.Close
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
		return NewQReader(m.data), nil
	default:
		return nil, nil
	}
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
	StopListen()
	// nothing to do for loopback
	// otherwise we should close the 'init' connection
}

var (
	listenConn *net.UDPConn
)

func Listen() {
	StopListen()
	addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Printf("Listen could not create addr: %v", err)
		return
	}
	con, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("Listen could not create connection: %v", err)
		return
	}
	listenConn = con
	go listenToNewClients(listenConn)
}

func StopListen() {
	if listenConn != nil {
		listenConn.Close()
		listenConn = nil
	}
}

const (
	// CCREP_REJECT | 7+21 = x1c
	versionError = "\x80\x00\x00\x1d\x82Incompatible version.\n\x00"
	// CCREP_REJECT | 7+15 = x16
	serverFullError = "\x80\x00\x00\x17\x82	Server is full.\n\x00"
)

func listenToNewClients(conn *net.UDPConn) {
	log.Printf("Start listening")
	buf := make([]byte, maxMessage)
	//var sendBuf bytes.Buffer
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("ReadFromUDP error: %v", err)
			return
		}
		if n < 8 {
			continue
		}
		reader := bytes.NewBuffer(buf[:n])
		var length uint32
		binary.Read(reader, binary.BigEndian, &length)
		flags := length & NETFLAG_FLAG_MASK
		length = length & NETFLAG_LENGTH_MASK
		if flags != NETFLAG_CTL {
			continue
		}
		if length != uint32(n) {
			continue
		}
		command, err := reader.ReadByte()
		if err != nil {
			continue
		}
		switch command {
		default:
			continue
		case CCREQ_SERVER_INFO:
			// TODO
			continue
		case CCREQ_PLAYER_INFO:
			// TODO
			continue
		case CCREQ_RULE_INFO:
			// TODO
			continue
		case CCREQ_CONNECT:
			q, err := reader.ReadString('\x00')
			if err != nil || q != quake {
				log.Printf("ReadString: %v, %v", q, err)
				// If the client does not speak quake no aswer is ok
				continue
			}
			v, err := reader.ReadByte()
			if err != nil {
				// message is broken, ignore
				continue
			}
			if v != netProtocolVersion {
				log.Printf("ProtoVersion: %v", v)
				conn.WriteToUDP([]byte(versionError), addr)
				continue
			}
			log.Printf("Would connect")
			// check if already connected
			// if yes and connect time is under 2sec send CCREP_ACCEPT again
			// if yes otherwise, close their old connection and let them retry
			// check for max connections
			// if full send CCREP_REJECT, reason 'Server is full.\n'

		}

		/*
			CCREP_ACCEPT = 0x81
			CCREP_REJECT = 0x82
			CCREP_SERVER_INFO = 0x83
			CCREP_PLAYER_INFO = 0x84
			CCREP_RULE_INFO = 0x85
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

				...
						conn.WriteToUDP(sendBuf.Bytes(), addr)
		*/
	}
}
