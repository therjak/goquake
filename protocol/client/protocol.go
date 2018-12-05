package client

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
