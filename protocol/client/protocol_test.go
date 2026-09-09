// SPDX-License-Identifier: GPL-2.0-or-later

package client

import (
	"math"
	"reflect"
	"strings"
	"testing"

	ptcl "goquake/protocol"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

func TestGoQuakeProtocolRoundTrip(t *testing.T) {
	usrCmd := &protos.UsrCmd{}
	usrCmd.SetMessageTime(123.456)
	usrCmd.SetPitch(10.5)
	usrCmd.SetYaw(20.5)
	usrCmd.SetRoll(30.5)
	usrCmd.SetForward(200)
	usrCmd.SetSide(-150)
	usrCmd.SetUp(100)
	usrCmd.SetAttack(true)
	usrCmd.SetJump(false)
	usrCmd.SetImpulse(7)

	original := &protos.ClientMessage{}
	original.SetCmds([]*protos.Cmd{
		{}, // Nop
		protos.Cmd_builder{
			Disconnect: proto.Bool(true),
		}.Build(),
		protos.Cmd_builder{
			StringCmd: proto.String("status"),
		}.Build(),
		protos.Cmd_builder{
			MoveCmd: usrCmd,
		}.Build(),
	})

	data, err := ToBytes(original, ptcl.GoQuake, 0)
	if err != nil {
		t.Fatalf("ToBytes failed for GoQuake: %v", err)
	}

	decoded, err := FromBytes(data, ptcl.GoQuake, 0)
	if err != nil {
		t.Fatalf("FromBytes failed for GoQuake: %v", err)
	}

	if !proto.Equal(original, decoded) {
		t.Errorf("GoQuake roundtrip mismatch.\nGot:  %+v\nWant: %+v", decoded, original)
	}
}

func TestFromBytesGoQuakeInvalid(t *testing.T) {
	invalidData := []byte{0xff, 0xff, 0xff, 0xff}
	_, err := FromBytes(invalidData, ptcl.GoQuake, 0)
	if err == nil {
		t.Fatal("expected error unmarshaling invalid GoQuake protobuf data, got nil")
	}
}

func TestNetQuakeProtocolRoundTrip(t *testing.T) {
	// In NetQuake:
	// Angles are written using WriteAngle (byte angle: 256 steps per 360 deg).
	// Forward, Side, Up are written as int16.
	// MessageTime is float32.
	// Attack bit 1, Jump bit 2.
	// Impulse is 1 byte.
	usrCmd := &protos.UsrCmd{}
	usrCmd.SetMessageTime(42.5)
	// Angles chosen to align with 256 discrete steps: (k / 256.0) * 360.0
	// 64 * 360 / 256 = 90.0 deg
	usrCmd.SetPitch(90.0)
	usrCmd.SetYaw(45.0)
	usrCmd.SetRoll(0.0)
	usrCmd.SetForward(150)
	usrCmd.SetSide(-200)
	usrCmd.SetUp(50)
	usrCmd.SetAttack(true)
	usrCmd.SetJump(true)
	usrCmd.SetImpulse(2)

	msg := &protos.ClientMessage{}
	msg.SetCmds([]*protos.Cmd{
		{}, // Nop
		protos.Cmd_builder{
			Disconnect: proto.Bool(true),
		}.Build(),
		protos.Cmd_builder{
			StringCmd: proto.String("say hello"),
		}.Build(),
		protos.Cmd_builder{
			MoveCmd: usrCmd,
		}.Build(),
	})

	data, err := ToBytes(msg, ptcl.NetQuake, 0)
	if err != nil {
		t.Fatalf("ToBytes failed for NetQuake: %v", err)
	}

	decoded, err := FromBytes(data, ptcl.NetQuake, 0)
	if err != nil {
		t.Fatalf("FromBytes failed for NetQuake: %v", err)
	}

	cmds := decoded.GetCmds()
	if len(cmds) != 4 {
		t.Fatalf("expected 4 commands, got %d", len(cmds))
	}

	// 1. Nop
	if cmds[0].WhichUnion() != protos.Cmd_Union_not_set_case {
		t.Errorf("cmd 0 expected Nop, got: %v", cmds[0].WhichUnion())
	}

	// 2. Disconnect
	if !cmds[1].GetDisconnect() {
		t.Errorf("cmd 1 expected Disconnect=true, got %v", cmds[1].GetDisconnect())
	}

	// 3. StringCmd
	if cmds[2].GetStringCmd() != "say hello" {
		t.Errorf("cmd 2 expected 'say hello', got %q", cmds[2].GetStringCmd())
	}

	// 4. MoveCmd
	mc := cmds[3].GetMoveCmd()
	if mc == nil {
		t.Fatalf("cmd 3 expected MoveCmd, got nil")
	}

	if mc.GetMessageTime() != 42.5 {
		t.Errorf("MessageTime: got %v, want %v", mc.GetMessageTime(), 42.5)
	}

	const angleEps = 360.0 / 256.0 + 0.01
	if math.Abs(float64(mc.GetPitch()-90.0)) > angleEps {
		t.Errorf("Pitch: got %v, want 90.0", mc.GetPitch())
	}
	if math.Abs(float64(mc.GetYaw()-45.0)) > angleEps {
		t.Errorf("Yaw: got %v, want 45.0", mc.GetYaw())
	}
	if math.Abs(float64(mc.GetRoll()-0.0)) > angleEps {
		t.Errorf("Roll: got %v, want 0.0", mc.GetRoll())
	}

	if mc.GetForward() != 150 {
		t.Errorf("Forward: got %v, want 150", mc.GetForward())
	}
	if mc.GetSide() != -200 {
		t.Errorf("Side: got %v, want -200", mc.GetSide())
	}
	if mc.GetUp() != 50 {
		t.Errorf("Up: got %v, want 50", mc.GetUp())
	}
	if !mc.GetAttack() {
		t.Errorf("Attack: got false, want true")
	}
	if !mc.GetJump() {
		t.Errorf("Jump: got false, want true")
	}
	if mc.GetImpulse() != 2 {
		t.Errorf("Impulse: got %v, want 2", mc.GetImpulse())
	}
}

func TestFitzQuakeProtocolRoundTrip(t *testing.T) {
	// FitzQuake writes angle using WriteAngle16 (short angle or float angle depending on flags)
	tests := []struct {
		name  string
		flags uint32
	}{
		{
			name:  "DefaultShortAngle",
			flags: 0,
		},
		{
			name:  "FloatAngle",
			flags: ptcl.ANGLEFLOAT,
		},
		{
			name:  "ShortAngleFlag",
			flags: ptcl.ANGLESHORT,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			usrCmd := &protos.UsrCmd{}
			usrCmd.SetMessageTime(10.0)
			usrCmd.SetPitch(45.123)
			usrCmd.SetYaw(-90.456)
			usrCmd.SetRoll(12.789)
			usrCmd.SetForward(-400)
			usrCmd.SetSide(300)
			usrCmd.SetUp(-100)
			usrCmd.SetAttack(false)
			usrCmd.SetJump(true)
			usrCmd.SetImpulse(15)

			msg := &protos.ClientMessage{}
			msg.SetCmds([]*protos.Cmd{
				protos.Cmd_builder{
					MoveCmd: usrCmd,
				}.Build(),
			})

			data, err := ToBytes(msg, ptcl.FitzQuake, tc.flags)
			if err != nil {
				t.Fatalf("ToBytes failed: %v", err)
			}

			decoded, err := FromBytes(data, ptcl.FitzQuake, tc.flags)
			if err != nil {
				t.Fatalf("FromBytes failed: %v", err)
			}

			cmds := decoded.GetCmds()
			if len(cmds) != 1 {
				t.Fatalf("expected 1 command, got %d", len(cmds))
			}

			mc := cmds[0].GetMoveCmd()
			if mc == nil {
				t.Fatalf("expected MoveCmd, got nil")
			}

			if mc.GetMessageTime() != 10.0 {
				t.Errorf("MessageTime: got %v, want 10.0", mc.GetMessageTime())
			}

			var eps float64 = 360.0 / 65536.0 + 0.001
			if tc.flags&ptcl.ANGLEFLOAT != 0 {
				eps = 1e-4
			}

			if math.Abs(float64(mc.GetPitch()-45.123)) > eps {
				t.Errorf("Pitch: got %v, want ~45.123", mc.GetPitch())
			}
			if math.Abs(float64(mc.GetYaw()-(-90.456))) > eps {
				t.Errorf("Yaw: got %v, want ~ -90.456", mc.GetYaw())
			}
			if math.Abs(float64(mc.GetRoll()-12.789)) > eps {
				t.Errorf("Roll: got %v, want ~12.789", mc.GetRoll())
			}

			if mc.GetForward() != -400 {
				t.Errorf("Forward: got %v, want -400", mc.GetForward())
			}
			if mc.GetSide() != 300 {
				t.Errorf("Side: got %v, want 300", mc.GetSide())
			}
			if mc.GetUp() != -100 {
				t.Errorf("Up: got %v, want -100", mc.GetUp())
			}
			if mc.GetAttack() {
				t.Errorf("Attack: got true, want false")
			}
			if !mc.GetJump() {
				t.Errorf("Jump: got false, want true")
			}
			if mc.GetImpulse() != 15 {
				t.Errorf("Impulse: got %v, want 15", mc.GetImpulse())
			}
		})
	}
}

func TestFromBytesErrors(t *testing.T) {
	t.Run("UnknownCommand", func(t *testing.T) {
		data := []byte{250} // Invalid command ID
		_, err := FromBytes(data, ptcl.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on unknown command, got nil")
		}
		if !strings.Contains(err.Error(), "unknown command char") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("TruncatedMoveCmd", func(t *testing.T) {
		// Move command header with partial payload
		data := []byte{Move, 0x00, 0x00}
		_, err := FromBytes(data, ptcl.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on truncated move cmd, got nil")
		}
		if !strings.Contains(err.Error(), "badread") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("TruncatedStringCmd", func(t *testing.T) {
		// StringCmd without null terminator
		data := []byte{StringCmd, 't', 'e', 's', 't'}
		_, err := FromBytes(data, ptcl.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on truncated string cmd, got nil")
		}
		if !strings.Contains(err.Error(), "badread") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestToBytesByteRepresentation(t *testing.T) {
	// Verify raw bytes produced by ToBytes for simple commands in NetQuake mode
	msg := &protos.ClientMessage{}
	msg.SetCmds([]*protos.Cmd{
		{}, // Nop (1)
		protos.Cmd_builder{
			Disconnect: proto.Bool(true),
		}.Build(), // Disconnect (2)
		protos.Cmd_builder{
			StringCmd: proto.String("cmd"),
		}.Build(), // StringCmd (4) + "cmd\0"
	})

	data, err := ToBytes(msg, ptcl.NetQuake, 0)
	if err != nil {
		t.Fatalf("ToBytes failed: %v", err)
	}

	expected := []byte{
		Nop,
		Disconnect,
		StringCmd, 'c', 'm', 'd', 0,
	}

	if !reflect.DeepEqual(data, expected) {
		t.Errorf("byte mismatch.\nGot:  %v\nWant: %v", data, expected)
	}
}
