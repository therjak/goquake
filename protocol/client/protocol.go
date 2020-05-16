package client

import (
	ptcl "quake/protocol"
	"quake/protos"
	"quake/qmsg"
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
	b := qmsg.NewClientWriter(uint16(protocolFlags))
	for _, c := range pb.GetCmds() {
		switch c.Union.(type) {
		default:
			b.WriteByte(Nop)
		case *protos.Cmd_Disconnect:
			b.WriteByte(Disconnect)
		case *protos.Cmd_StringCmd:
			sc := c.GetStringCmd()
			b.WriteByte(StringCmd)
			b.WriteString(sc)
			b.WriteByte(0)
		case *protos.Cmd_MoveCmd:
			mc := c.GetMoveCmd()
			b.WriteByte(Move)
			b.WriteFloat(mc.GetMessageTime())

			if protocol == ptcl.NetQuake {
				b.WriteAngle(mc.GetPitch())
				b.WriteAngle(mc.GetYaw())
				b.WriteAngle(mc.GetRoll())
			} else {
				b.WriteAngle16(mc.GetPitch())
				b.WriteAngle16(mc.GetYaw())
				b.WriteAngle16(mc.GetRoll())
			}
			b.WriteShort(uint16(mc.GetForward()))
			b.WriteShort(uint16(mc.GetSide()))
			b.WriteShort(uint16(mc.GetUp()))

			bits := byte(0)
			if mc.GetAttack() {
				bits |= 1
			}
			if mc.GetJump() {
				bits |= 2
			}
			b.WriteByte(bits)

			b.WriteByte(byte(mc.GetImpulse()))
		}
	}
	return b.Bytes()
}
