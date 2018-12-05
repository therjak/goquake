package quakelib

var (
	//loopClient         *connection
	//loopServer         *connection
	//netMessage         *bytes.Reader
	//netMessageBackup   *bytes.Reader
	tcpipAvailable = true // TODO: better start with false
)

func udp_init() bool {
	return true
	// contolAddr, err := net.ResolveUDPAddr("udp", ":0")
	//if err != nil {
	//	return -1
	//}
	// serverAddr...
	// DialUDP("udp", nil, serverAddr) chooses automatically a local address
	// conn, err := net.DialUDP("udp", controlAddr, serverAddr)
	// handle err
	// defer conn.Close()
	// con.Write / con.Read

	// server:
	// serverCon, err := net.ListenUDP("udp", severAddr)
	// handle err
	// defer serverCon.Close()
	// buf := make([]byte, 1024)
	// n, addr, err := serverCon.ReadFromUDP(buf)
	// -> received n bytes from addr
	// n, err := serverCon.WriteTo(buf, addr)
	// -> send n bytes to addr

}

/*
chan: len to check for elements queued
Datagram_Init
-- loop: near empty, -1 error, 0 success
Datagram_Listen
-- loop is empty
Datagram_SearchForHosts
-- loop: fill hostcache[]{name, map, users, maxusers, driver, cname}
Datagram_Connect
-- -1 on error
Datagram_CheckNewConnections
-- loop returns the server socket and resets client and server
-- -1 on error
Datagram_GetMessage
-- loop fill stuff in net_message, tell other side canSend = true
-- 0 if no data waiting
-- 1 message was received
-- 2 unreliable message received
-- -1 connection died
Datagram_SendMessage
-- loop: canSend = false, stuff data in a buffer to be read in getmessage
--       buffer[0] = 1, buffer[1]+[2] = size. probably not needed in go?
--       returns -1 on error , 1 on success
Datagram_SendUnreliableMessage
-- loop: nearly the same as SendMessage, just not syserror on error
--       sends -1 on error, 0 on not send, 1 on success
Datagram_CanSendMessage
-- loop: just check if the chanel is empty (if len(chan) = 1)
Datagram_CanSendUnreliableMessage
-- loop: true
Datagram_Close
-- loop: clear local static of sock, reset sock data
Datagram_Shutdown
-- loop is empty
*/
