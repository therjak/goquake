// SPDX-License-Identifier: GPL-2.0-or-later

package client

import (
	"fmt"

	"goquake/net"
	ptcl "goquake/protocol"
	"goquake/protos"
	"goquake/qmsg"

	"google.golang.org/protobuf/proto"
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

func ToBytes(pb *protos.ClientMessage, protocol int, protocolFlags uint32) ([]byte, error) {
	if protocol == ptcl.GoQuake {
		return proto.Marshal(pb)
	}
	b := qmsg.NewClientWriter(uint16(protocolFlags))
	for _, c := range pb.GetCmds() {
		switch c.WhichUnion() {
		default:
			b.WriteByte(Nop)
		case protos.Cmd_Disconnect_case:
			b.WriteByte(Disconnect)
		case protos.Cmd_StringCmd_case:
			sc := c.GetStringCmd()
			b.WriteByte(StringCmd)
			b.WriteString(sc)
			b.WriteByte(0)
		case protos.Cmd_MoveCmd_case:
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
	return b.Bytes(), nil
}

func FromBytes(data []byte, protocol int, flags uint32) (*protos.ClientMessage, error) {
	pb := &protos.ClientMessage{}
	if protocol == ptcl.GoQuake {
		err := proto.Unmarshal(data, pb)
		return pb, err
	}
	netMessage := net.NewQReader(data)
	readAngle := netMessage.ReadAngle16
	if protocol == ptcl.NetQuake {
		readAngle = netMessage.ReadAngle
	}
	var err error
	for netMessage.Len() != 0 {
		ccmd, _ := netMessage.ReadInt8()
		switch ccmd {
		default:
			return nil, fmt.Errorf("SV_ReadClientMessage: unknown command char %v\n", ccmd)
		case Nop:
			pb.SetCmds(append(pb.GetCmds(), &protos.Cmd{}))
		case Disconnect:
			pb.SetCmds(append(pb.GetCmds(), protos.Cmd_builder{
				Disconnect: proto.Bool(true),
			}.Build()))
		case Move:
			cmd := &protos.UsrCmd{}
			cmd.MessageTime, err = netMessage.ReadFloat32()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}

			cmd.Pitch, err = readAngle(flags)
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			cmd.Yaw, err = readAngle(flags)
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			cmd.Roll, err = readAngle(flags)
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}

			forward, err := netMessage.ReadInt16()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			side, err := netMessage.ReadInt16()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			upward, err := netMessage.ReadInt16()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			cmd.SetForward(float32(forward))
			cmd.SetSide(float32(side))
			cmd.SetUp(float32(upward))

			bits, err := netMessage.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			cmd.SetAttack((bits & 1) != 0)
			cmd.SetJump((bits & 2) != 0)

			impulse, err := netMessage.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			cmd.SetImpulse(int32(impulse))

			pb.SetCmds(append(pb.GetCmds(), protos.Cmd_builder{
				MoveCmd: cmd,
			}.Build()))
		case StringCmd:
			s, err := netMessage.ReadString()
			if err != nil {
				return nil, fmt.Errorf("SV_ReadClientMessage: badread %v\n", err)
			}
			pb.SetCmds(append(pb.GetCmds(), protos.Cmd_builder{
				StringCmd: proto.String(s),
			}.Build()))
		}
	}
	return pb, nil
}
