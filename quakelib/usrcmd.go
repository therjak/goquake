package quakelib

import (
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/input"
	clc "github.com/therjak/goquake/protocol/client"
	"github.com/therjak/goquake/protos"
)

var (
	in_impulse int
)

var (
	mouseMove mouseMoveBuffer
)

type mouseMoveBuffer struct {
	x, y int
}

func mouseMotion(x, y int) {
	mouseMove.x += x
	mouseMove.y += y
}

func resetMouseMotion() {
	mouseMove.x = 0
	mouseMove.y = 0
}

type userMove struct {
	forward, side, up float32 // move
}

func (u *userMove) keyboardMove() {
	if input.Strafe.Down() {
		r := input.Right.GetImpulse()
		l := input.Left.GetImpulse()
		u.side += cvars.ClientSideSpeed.Value() * (r - l)
	}
	r := input.MoveRight.GetImpulse()
	l := input.MoveLeft.GetImpulse()
	u.side += cvars.ClientSideSpeed.Value() * (r - l)

	up := input.Up.GetImpulse()
	dw := input.Down.GetImpulse()
	u.up += cvars.ClientUpSpeed.Value() * (up - dw)

	if !input.KLook.Down() {
		f := input.Forward.GetImpulse()
		b := input.Back.GetImpulse()
		u.forward += cvars.ClientForwardSpeed.Value() * f
		u.forward -= cvars.ClientBackSpeed.Value() * b
	}

	if cvars.ClientForwardSpeed.Value() > 200 && cvars.ClientMoveSpeedKey.Value() != 0 {
		u.forward /= cvars.ClientMoveSpeedKey.Value()
	}

	if (cvars.ClientForwardSpeed.Value() > 200) != input.Speed.Down() {
		u.forward *= cvars.ClientMoveSpeedKey.Value()
		u.side *= cvars.ClientMoveSpeedKey.Value()
		u.up *= cvars.ClientMoveSpeedKey.Value()
	}
}

func resetInput() {
	input.Forward.ResetImpulse()
	input.Back.ResetImpulse()
	input.Up.ResetImpulse()
	input.Down.ResetImpulse()
	input.MoveRight.ResetImpulse()
	input.MoveLeft.ResetImpulse()
	input.Right.ResetImpulse()
	input.Left.ResetImpulse()
	mouseMove.x = 0
	mouseMove.y = 0
}

func clamp(x, min, max float32) float32 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func (u *userMove) mouseMove() {
	x := float32(mouseMove.x) * cvars.Sensitivity.Value()
	y := float32(mouseMove.y) * cvars.Sensitivity.Value()
	if input.Strafe.Down() || (input.MLook.Down() && (cvars.LookStrafe.Value() != 0)) {
		u.side += cvars.MouseSide.Value() * x
	}
	if !input.MLook.Down() || input.Strafe.Down() {
		if input.Strafe.Down() { // && noclip_anglehack {
			u.up -= cvars.MouseForward.Value() * y
		} else {
			u.forward -= cvars.MouseForward.Value() * y
		}
	}
}

// Send unreliable message (CL_SendMove)
func send(v userView, m userMove) {
	cmd := &protos.UsrCmd{
		MessageTime: float32(cl.messageTime),
		Pitch:       v.pitch,
		Yaw:         v.yaw,
		Roll:        v.roll,
		Forward:     m.forward,
		Side:        m.side,
		Up:          m.up,
		Attack:      input.Attack.WentDown(),
		Jump:        input.Jump.WentDown(),
		Impulse:     int32(in_impulse),
	}
	pb := &protos.ClientMessage{}
	pb.Cmds = append(pb.Cmds, &protos.Cmd{
		Union: &protos.Cmd_MoveCmd{
			cmd,
		}})

	cl.cmdForwardMove = m.forward
	in_impulse = 0

	if cls.demoPlayback {
		return
	}
	// allways dump the first two message, because it may contain leftover inputs from the last level
	cl.movemessages++
	if 2 >= cl.movemessages {
		return
	}

	if cls.connection.SendUnreliableMessage(clc.ToBytes(pb, cl.protocol, cl.protocolFlags)) == -1 {
		conlog.Printf("CL_SendMove: lost server connection\n")
		cls.Disconnect()
	}
}

type userView struct {
	pitch, yaw, roll float32
}

func (v *userView) mouseMove() {
	x := float32(mouseMove.x) * cvars.Sensitivity.Value()
	y := float32(mouseMove.y) * cvars.Sensitivity.Value()
	if !input.Strafe.Down() && (!input.MLook.Down() || (cvars.LookStrafe.Value() == 0)) {
		v.yaw -= cvars.MouseYaw.Value() * x
	}
	if input.MLook.Down() {
		if x != 0 || y != 0 {
			cl.stopPitchDrift()
		}
	}
	if input.MLook.Down() && !input.Strafe.Down() {
		v.pitch += cvars.MousePitch.Value() * y
		v.pitch = clamp(v.pitch, cvars.ClientMinPitch.Value(), cvars.ClientMaxPitch.Value())
	}
}

func HandleMove() {
	v := userView{
		pitch: cl.pitch,
		yaw:   cl.yaw,
		roll:  cl.roll,
	}
	v.mouseMove()

	m := userMove{}
	m.keyboardMove()
	m.mouseMove()

	send(v, m)

	resetInput()

	cl.pitch = v.pitch
	cl.yaw = v.yaw
	cl.roll = v.roll
}

func init() {
	cmd.AddCommand("impulse", func(args []cmd.QArg, _ int) { in_impulse = args[0].Int() })
}
