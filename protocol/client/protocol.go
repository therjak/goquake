package client

import (
	"quake/protos"
)

const (
	//
	// client to server
	//
	Bad        = 0
	Nop        = 1
	Disconnect = 2
	// [usercmd_t]
	Move = 3
	// [string] message
	StringCmd = 4
)

var (
	protocol      int
	protocolFlags int
)

func SetProtocol(p int) {
	protocol = p
}

func SetProtocolFlags(f int) {
	protocolFlags = f
}

func ToBytes(pb *protos.ClientMessage) []byte {
	return nil
}
