package keys

type Destination byte

const (
	Game = Destination(iota)
	Console
	Message
	Menu
)
