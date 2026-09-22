// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"math"
	"strings"
	"testing"

	"goquake/net"
	"goquake/protocol"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

func approxEqual(a, b, eps float32) bool {
	return math.Abs(float64(a-b)) <= float64(eps)
}

func approxEqualCoord(a, b *protos.Coord, eps float32) bool {
	if a == nil || b == nil {
		return a == b
	}
	return approxEqual(a.GetX(), b.GetX(), eps) &&
		approxEqual(a.GetY(), b.GetY(), eps) &&
		approxEqual(a.GetZ(), b.GetZ(), eps)
}

func TestWriteAndParseTime(t *testing.T) {
	msg := &net.Message{}
	var expected float32 = 123.456
	WriteTime(expected, protocol.NetQuake, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_Time_case {
		t.Fatalf("expected Time command, got %v", cmd.WhichUnion())
	}
	if !approxEqual(cmd.GetTime(), expected, 1e-5) {
		t.Errorf("Time mismatch: got %v, want %v", cmd.GetTime(), expected)
	}
}

func TestWriteAndParseUpdateFrags(t *testing.T) {
	msg := &net.Message{}
	expected := protos.UpdateFrags_builder{
		Player:   3,
		NewFrags: -12,
	}.Build()
	WriteUpdateFrags(expected, protocol.NetQuake, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_UpdateFrags_case {
		t.Fatalf("expected UpdateFrags command, got %v", cmd.WhichUnion())
	}
	uf := cmd.GetUpdateFrags()
	if uf.GetPlayer() != 3 || uf.GetNewFrags() != -12 {
		t.Errorf("UpdateFrags mismatch: got %+v, want %+v", uf, expected)
	}
}

func TestWriteAndParseUpdateColors(t *testing.T) {
	msg := &net.Message{}
	expected := protos.UpdateColors_builder{
		Player:   2,
		NewColor: 0x47,
	}.Build()
	WriteUpdateColors(expected, protocol.NetQuake, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_UpdateColors_case {
		t.Fatalf("expected UpdateColors command, got %v", cmd.WhichUnion())
	}
	uc := cmd.GetUpdateColors()
	if uc.GetPlayer() != 2 || uc.GetNewColor() != 0x47 {
		t.Errorf("UpdateColors mismatch: got %+v, want %+v", uc, expected)
	}
}

func TestWriteAndParseUpdateName(t *testing.T) {
	msg := &net.Message{}
	expected := protos.UpdateName_builder{
		Player:  1,
		NewName: "PlayerOne",
	}.Build()
	WriteUpdateName(expected, protocol.NetQuake, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_UpdateName_case {
		t.Fatalf("expected UpdateName command, got %v", cmd.WhichUnion())
	}
	un := cmd.GetUpdateName()
	if un.GetPlayer() != 1 || un.GetNewName() != "PlayerOne" {
		t.Errorf("UpdateName mismatch: got %+v, want %+v", un, expected)
	}
}

func TestWriteAndParseSetPause(t *testing.T) {
	for _, paused := range []bool{true, false} {
		msg := &net.Message{}
		WriteSetPause(paused, protocol.NetQuake, 0, msg)

		reader := net.NewQReader(msg.Bytes())
		sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed (paused=%v): %v", paused, err)
		}

		if len(sm.GetCmds()) != 1 {
			t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
		}
		cmd := sm.GetCmds()[0]
		if cmd.WhichUnion() != protos.SCmd_SetPause_case {
			t.Fatalf("expected SetPause command, got %v", cmd.WhichUnion())
		}
		if cmd.GetSetPause() != paused {
			t.Errorf("SetPause mismatch: got %v, want %v", cmd.GetSetPause(), paused)
		}
	}
}

func TestWriteAndParseSetAngle(t *testing.T) {
	tests := []struct {
		name     string
		pcol     int
		flags    uint32
		angle    *protos.Coord
		epsAngle float32
	}{
		{
			name:     "NetQuakeByteAngle",
			pcol:     protocol.NetQuake,
			flags:    0,
			angle:    protos.Coord_builder{X: 90.0, Y: 45.0, Z: 0.0}.Build(),
			epsAngle: 360.0/256.0 + 0.01,
		},
		{
			name:     "FitzQuakeShortAngle",
			pcol:     protocol.FitzQuake,
			flags:    protocol.ANGLESHORT,
			angle:    protos.Coord_builder{X: 45.123, Y: -90.456, Z: 12.789}.Build(),
			epsAngle: 360.0/65536.0 + 0.001,
		},
		{
			name:     "FitzQuakeFloatAngle",
			pcol:     protocol.FitzQuake,
			flags:    protocol.ANGLEFLOAT,
			angle:    protos.Coord_builder{X: 45.123, Y: -90.456, Z: 12.789}.Build(),
			epsAngle: 1e-4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &net.Message{}
			WriteSetAngle(tc.angle, tc.pcol, tc.flags, msg)

			reader := net.NewQReader(msg.Bytes())
			sm, err := ParseServerMessage(reader, tc.pcol, tc.flags)
			if err != nil {
				t.Fatalf("ParseServerMessage failed: %v", err)
			}

			if len(sm.GetCmds()) != 1 {
				t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
			}
			cmd := sm.GetCmds()[0]
			if cmd.WhichUnion() != protos.SCmd_SetAngle_case {
				t.Fatalf("expected SetAngle command, got %v", cmd.WhichUnion())
			}
			got := cmd.GetSetAngle()
			if !approxEqualCoord(got, tc.angle, tc.epsAngle) {
				t.Errorf("SetAngle mismatch: got %+v, want %+v (eps: %v)", got, tc.angle, tc.epsAngle)
			}
		})
	}
}

func TestWriteAndParseDamage(t *testing.T) {
	msg := &net.Message{}
	expected := protos.Damage_builder{
		Armor: 25,
		Blood: 10,
		Position: protos.Coord_builder{
			X: 100.5,
			Y: -50.25,
			Z: 12.0,
		}.Build(),
	}.Build()

	WriteDamage(expected, protocol.NetQuake, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_Damage_case {
		t.Fatalf("expected Damage command, got %v", cmd.WhichUnion())
	}
	dmg := cmd.GetDamage()
	if dmg.GetArmor() != expected.GetArmor() || dmg.GetBlood() != expected.GetBlood() {
		t.Errorf("Damage fields mismatch: got armor %v, blood %v; want armor %v, blood %v",
			dmg.GetArmor(), dmg.GetBlood(), expected.GetArmor(), expected.GetBlood())
	}
	if !approxEqualCoord(dmg.GetPosition(), expected.GetPosition(), 0.125) {
		t.Errorf("Damage position mismatch: got %+v, want %+v", dmg.GetPosition(), expected.GetPosition())
	}
}

func TestWriteAndParseParticle(t *testing.T) {
	msg := &net.Message{}
	expected := protos.Particle_builder{
		Origin: protos.Coord_builder{
			X: 10.0,
			Y: 20.0,
			Z: 30.0,
		}.Build(),
		Direction: protos.Coord_builder{
			X: 1.0,
			Y: -2.0,
			Z: 0.5,
		}.Build(),
		Count: 64,
		Color: 15,
	}.Build()

	WriteParticle(expected, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	if len(sm.GetCmds()) != 1 {
		t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
	}
	cmd := sm.GetCmds()[0]
	if cmd.WhichUnion() != protos.SCmd_Particle_case {
		t.Fatalf("expected Particle command, got %v", cmd.WhichUnion())
	}
	p := cmd.GetParticle()
	if p.GetCount() != expected.GetCount() || p.GetColor() != expected.GetColor() {
		t.Errorf("Particle count/color mismatch: got count %d color %d, want count %d color %d",
			p.GetCount(), p.GetColor(), expected.GetCount(), expected.GetColor())
	}
	if !approxEqualCoord(p.GetOrigin(), expected.GetOrigin(), 0.125) {
		t.Errorf("Particle origin mismatch: got %+v, want %+v", p.GetOrigin(), expected.GetOrigin())
	}
	if !approxEqualCoord(p.GetDirection(), expected.GetDirection(), 1.0/16.0+0.001) {
		t.Errorf("Particle direction mismatch: got %+v, want %+v", p.GetDirection(), expected.GetDirection())
	}
}

func TestWriteAndParseParticleCount255(t *testing.T) {
	msg := &net.Message{}
	expected := protos.Particle_builder{
		Origin: protos.Coord_builder{X: 0, Y: 0, Z: 0}.Build(),
		Direction: protos.Coord_builder{X: 0, Y: 0, Z: 0}.Build(),
		Count: 255,
		Color: 1,
	}.Build()

	WriteParticle(expected, 0, msg)

	reader := net.NewQReader(msg.Bytes())
	sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}
	p := sm.GetCmds()[0].GetParticle()
	// count 255 should be expanded to 1024
	if p.GetCount() != 1024 {
		t.Errorf("expected count 255 to parse as 1024, got %d", p.GetCount())
	}
}

func TestWriteAndParseSound(t *testing.T) {
	tests := []struct {
		name  string
		pcol  int
		flags uint32
		sound *protos.Sound
	}{
		{
			name:  "NetQuakeStandardSound",
			pcol:  protocol.NetQuake,
			flags: 0,
			sound: protos.Sound_builder{
				Entity:      10,
				Channel:     2,
				SoundNum:    15,
				Volume:      proto.Int32(200),
				Attenuation: proto.Int32(64),
				Origin: protos.Coord_builder{
					X: 128.0,
					Y: -256.0,
					Z: 32.0,
				}.Build(),
			}.Build(),
		},
		{
			name:  "FitzQuakeLargeSoundAndEntity",
			pcol:  protocol.FitzQuake,
			flags: 0,
			sound: protos.Sound_builder{
				Entity:      8500,
				Channel:     5,
				SoundNum:    350,
				Volume:      proto.Int32(255),
				Attenuation: proto.Int32(128),
				Origin: protos.Coord_builder{
					X: 512.25,
					Y: -1024.5,
					Z: 0.0,
				}.Build(),
			}.Build(),
		},
		{
			name:  "FitzQuakeDefaultVolAtten",
			pcol:  protocol.FitzQuake,
			flags: 0,
			sound: protos.Sound_builder{
				Entity:   1,
				Channel:  0,
				SoundNum: 3,
				Origin: protos.Coord_builder{
					X: 0,
					Y: 0,
					Z: 0,
				}.Build(),
			}.Build(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &net.Message{}
			WriteSound(tc.sound, tc.pcol, tc.flags, msg)

			reader := net.NewQReader(msg.Bytes())
			sm, err := ParseServerMessage(reader, tc.pcol, tc.flags)
			if err != nil {
				t.Fatalf("ParseServerMessage failed: %v", err)
			}

			if len(sm.GetCmds()) != 1 {
				t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
			}
			s := sm.GetCmds()[0].GetSound()
			if s.GetEntity() != tc.sound.GetEntity() {
				t.Errorf("Entity mismatch: got %v, want %v", s.GetEntity(), tc.sound.GetEntity())
			}
			if s.GetChannel() != tc.sound.GetChannel() {
				t.Errorf("Channel mismatch: got %v, want %v", s.GetChannel(), tc.sound.GetChannel())
			}
			if s.GetSoundNum() != tc.sound.GetSoundNum()-1 {
				t.Errorf("SoundNum mismatch: got %v, want %v", s.GetSoundNum(), tc.sound.GetSoundNum()-1)
			}
			if tc.sound.HasVolume() && s.GetVolume() != tc.sound.GetVolume() {
				t.Errorf("Volume mismatch: got %v, want %v", s.GetVolume(), tc.sound.GetVolume())
			}
			if tc.sound.HasAttenuation() && s.GetAttenuation() != tc.sound.GetAttenuation() {
				t.Errorf("Attenuation mismatch: got %v, want %v", s.GetAttenuation(), tc.sound.GetAttenuation())
			}
			if !approxEqualCoord(s.GetOrigin(), tc.sound.GetOrigin(), 0.125) {
				t.Errorf("Origin mismatch: got %+v, want %+v", s.GetOrigin(), tc.sound.GetOrigin())
			}
		})
	}
}

func TestWriteAndParseClientData(t *testing.T) {
	tests := []struct {
		name  string
		pcol  int
		flags uint32
		cd    *protos.ClientData
	}{
		{
			name:  "NetQuakeClientData",
			pcol:  protocol.NetQuake,
			flags: 0,
			cd: protos.ClientData_builder{
				ViewHeight: proto.Int32(22),
				IdealPitch: 12,
				PunchAngle: protos.IntCoord_builder{X: 1, Y: 2, Z: 3}.Build(),
				Velocity:   protos.IntCoord_builder{X: 10, Y: -20, Z: 30}.Build(),
				Items:      0x12345678,
				OnGround:   true,
				InWater:    false,
				WeaponFrame: 5,
				Armor:       100,
				Weapon:      1,
				Health:      100,
				Ammo:        25,
				Shells:      50,
				Nails:       75,
				Rockets:     15,
				Cells:       30,
				ActiveWeapon: 1,
			}.Build(),
		},
		{
			name:  "FitzQuakeExtendedClientData",
			pcol:  protocol.FitzQuake,
			flags: 0,
			cd: protos.ClientData_builder{
				ViewHeight: proto.Int32(DEFAULT_VIEWHEIGHT),
				IdealPitch: -5,
				PunchAngle: protos.IntCoord_builder{X: 0, Y: 0, Z: 0}.Build(),
				Velocity:   protos.IntCoord_builder{X: 100, Y: 120, Z: -50}.Build(),
				Items:      0x87654321,
				OnGround:   false,
				InWater:    true,
				WeaponFrame: 0x0205,
				Armor:       0x0150,
				Weapon:      0x0302,
				WeaponAlpha: 180,
				Health:      -10,
				Ammo:        0x0120,
				Shells:      0x0110,
				Nails:       0x0130,
				Rockets:     0x0105,
				Cells:       0x0140,
				ActiveWeapon: 7,
			}.Build(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &net.Message{}
			WriteClientData(tc.cd, tc.pcol, tc.flags, msg)

			reader := net.NewQReader(msg.Bytes())
			sm, err := ParseServerMessage(reader, tc.pcol, tc.flags)
			if err != nil {
				t.Fatalf("ParseServerMessage failed: %v", err)
			}

			if len(sm.GetCmds()) != 1 {
				t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
			}
			cd := sm.GetCmds()[0].GetClientData()

			if tc.cd.HasViewHeight() && cd.GetViewHeight() != tc.cd.GetViewHeight() {
				t.Errorf("ViewHeight mismatch: got %v, want %v", cd.GetViewHeight(), tc.cd.GetViewHeight())
			}
			if cd.GetIdealPitch() != tc.cd.GetIdealPitch() {
				t.Errorf("IdealPitch mismatch: got %v, want %v", cd.GetIdealPitch(), tc.cd.GetIdealPitch())
			}
			if cd.GetPunchAngle().GetX() != tc.cd.GetPunchAngle().GetX() ||
				cd.GetPunchAngle().GetY() != tc.cd.GetPunchAngle().GetY() ||
				cd.GetPunchAngle().GetZ() != tc.cd.GetPunchAngle().GetZ() {
				t.Errorf("PunchAngle mismatch: got %+v, want %+v", cd.GetPunchAngle(), tc.cd.GetPunchAngle())
			}
			if cd.GetVelocity().GetX() != tc.cd.GetVelocity().GetX() ||
				cd.GetVelocity().GetY() != tc.cd.GetVelocity().GetY() ||
				cd.GetVelocity().GetZ() != tc.cd.GetVelocity().GetZ() {
				t.Errorf("Velocity mismatch: got %+v, want %+v", cd.GetVelocity(), tc.cd.GetVelocity())
			}
			if cd.GetItems() != tc.cd.GetItems() {
				t.Errorf("Items mismatch: got %v, want %v", cd.GetItems(), tc.cd.GetItems())
			}
			if cd.GetOnGround() != tc.cd.GetOnGround() {
				t.Errorf("OnGround mismatch: got %v, want %v", cd.GetOnGround(), tc.cd.GetOnGround())
			}
			if cd.GetInWater() != tc.cd.GetInWater() {
				t.Errorf("InWater mismatch: got %v, want %v", cd.GetInWater(), tc.cd.GetInWater())
			}
			if cd.GetHealth() != tc.cd.GetHealth() {
				t.Errorf("Health mismatch: got %v, want %v", cd.GetHealth(), tc.cd.GetHealth())
			}
			if cd.GetActiveWeapon() != tc.cd.GetActiveWeapon() {
				t.Errorf("ActiveWeapon mismatch: got %v, want %v", cd.GetActiveWeapon(), tc.cd.GetActiveWeapon())
			}
			if cd.GetWeaponFrame() != tc.cd.GetWeaponFrame() {
				t.Errorf("WeaponFrame mismatch: got %v, want %v", cd.GetWeaponFrame(), tc.cd.GetWeaponFrame())
			}
			if cd.GetArmor() != tc.cd.GetArmor() {
				t.Errorf("Armor mismatch: got %v, want %v", cd.GetArmor(), tc.cd.GetArmor())
			}
			if cd.GetWeapon() != tc.cd.GetWeapon() {
				t.Errorf("Weapon mismatch: got %v, want %v", cd.GetWeapon(), tc.cd.GetWeapon())
			}
			if cd.GetAmmo() != tc.cd.GetAmmo() {
				t.Errorf("Ammo mismatch: got %v, want %v", cd.GetAmmo(), tc.cd.GetAmmo())
			}
			if cd.GetShells() != tc.cd.GetShells() {
				t.Errorf("Shells mismatch: got %v, want %v", cd.GetShells(), tc.cd.GetShells())
			}
			if cd.GetNails() != tc.cd.GetNails() {
				t.Errorf("Nails mismatch: got %v, want %v", cd.GetNails(), tc.cd.GetNails())
			}
			if cd.GetRockets() != tc.cd.GetRockets() {
				t.Errorf("Rockets mismatch: got %v, want %v", cd.GetRockets(), tc.cd.GetRockets())
			}
			if cd.GetCells() != tc.cd.GetCells() {
				t.Errorf("Cells mismatch: got %v, want %v", cd.GetCells(), tc.cd.GetCells())
			}
			if cd.GetWeaponAlpha() != tc.cd.GetWeaponAlpha() {
				t.Errorf("WeaponAlpha mismatch: got %v, want %v", cd.GetWeaponAlpha(), tc.cd.GetWeaponAlpha())
			}
		})
	}
}

func TestWriteAndParseEntityUpdate(t *testing.T) {
	tests := []struct {
		name     string
		pcol     int
		flags    uint32
		eu       *protos.EntityUpdate
		epsAngle float32
	}{
		{
			name:  "NetQuakeStandardEntity",
			pcol:  protocol.NetQuake,
			flags: 0,
			eu: protos.EntityUpdate_builder{
				Entity:       12,
				Model:        proto.Int32(3),
				Frame:        proto.Int32(10),
				ColorMap:     proto.Int32(2),
				Skin:         proto.Int32(1),
				Effects:      EffectBrightField,
				OriginX:      proto.Float32(100.5),
				OriginY:      proto.Float32(-200.25),
				OriginZ:      proto.Float32(50.0),
				AngleX:       proto.Float32(45.0),
				AngleY:       proto.Float32(90.0),
				AngleZ:       proto.Float32(0.0),
				LerpMoveStep: true,
			}.Build(),
			epsAngle: 360.0/256.0 + 0.01,
		},
		{
			name:  "FitzQuakeExtendedEntity",
			pcol:  protocol.FitzQuake,
			flags: protocol.ANGLESHORT,
			eu: protos.EntityUpdate_builder{
				Entity:       512, // U_LONGENTITY
				Model:        proto.Int32(0x0210), // U_MODEL2
				Frame:        proto.Int32(0x0120), // U_FRAME2
				ColorMap:     proto.Int32(5),
				Skin:         proto.Int32(2),
				Effects:      EffectDimLight,
				OriginX:      proto.Float32(350.0),
				OriginY:      proto.Float32(-150.0),
				OriginZ:      proto.Float32(75.5),
				AngleX:       proto.Float32(12.34),
				AngleY:       proto.Float32(56.78),
				AngleZ:       proto.Float32(90.12),
				Alpha:        proto.Int32(128),
				LerpFinish:   proto.Int32(200),
				LerpMoveStep: false,
			}.Build(),
			epsAngle: 360.0/65536.0 + 0.001,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &net.Message{}
			WriteEntityUpdate(tc.eu, tc.pcol, tc.flags, msg)

			reader := net.NewQReader(msg.Bytes())
			sm, err := ParseServerMessage(reader, tc.pcol, tc.flags)
			if err != nil {
				t.Fatalf("ParseServerMessage failed: %v", err)
			}

			if len(sm.GetCmds()) != 1 {
				t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
			}
			cmd := sm.GetCmds()[0]
			if cmd.WhichUnion() != protos.SCmd_EntityUpdate_case {
				t.Fatalf("expected EntityUpdate command, got %v", cmd.WhichUnion())
			}
			got := cmd.GetEntityUpdate()

			if got.GetEntity() != tc.eu.GetEntity() {
				t.Errorf("Entity mismatch: got %v, want %v", got.GetEntity(), tc.eu.GetEntity())
			}
			if got.GetModel() != tc.eu.GetModel() {
				t.Errorf("Model mismatch: got %v, want %v", got.GetModel(), tc.eu.GetModel())
			}
			if got.GetFrame() != tc.eu.GetFrame() {
				t.Errorf("Frame mismatch: got %v, want %v", got.GetFrame(), tc.eu.GetFrame())
			}
			if got.GetColorMap() != tc.eu.GetColorMap() {
				t.Errorf("ColorMap mismatch: got %v, want %v", got.GetColorMap(), tc.eu.GetColorMap())
			}
			if got.GetSkin() != tc.eu.GetSkin() {
				t.Errorf("Skin mismatch: got %v, want %v", got.GetSkin(), tc.eu.GetSkin())
			}
			if got.GetEffects() != tc.eu.GetEffects() {
				t.Errorf("Effects mismatch: got %v, want %v", got.GetEffects(), tc.eu.GetEffects())
			}
			if got.GetLerpMoveStep() != tc.eu.GetLerpMoveStep() {
				t.Errorf("LerpMoveStep mismatch: got %v, want %v", got.GetLerpMoveStep(), tc.eu.GetLerpMoveStep())
			}
			if !approxEqual(got.GetOriginX(), tc.eu.GetOriginX(), 0.125) {
				t.Errorf("OriginX mismatch: got %v, want %v", got.GetOriginX(), tc.eu.GetOriginX())
			}
			if !approxEqual(got.GetOriginY(), tc.eu.GetOriginY(), 0.125) {
				t.Errorf("OriginY mismatch: got %v, want %v", got.GetOriginY(), tc.eu.GetOriginY())
			}
			if !approxEqual(got.GetOriginZ(), tc.eu.GetOriginZ(), 0.125) {
				t.Errorf("OriginZ mismatch: got %v, want %v", got.GetOriginZ(), tc.eu.GetOriginZ())
			}
			if !approxEqual(got.GetAngleX(), tc.eu.GetAngleX(), tc.epsAngle) {
				t.Errorf("AngleX mismatch: got %v, want %v", got.GetAngleX(), tc.eu.GetAngleX())
			}
			if !approxEqual(got.GetAngleY(), tc.eu.GetAngleY(), tc.epsAngle) {
				t.Errorf("AngleY mismatch: got %v, want %v", got.GetAngleY(), tc.eu.GetAngleY())
			}
			if !approxEqual(got.GetAngleZ(), tc.eu.GetAngleZ(), tc.epsAngle) {
				t.Errorf("AngleZ mismatch: got %v, want %v", got.GetAngleZ(), tc.eu.GetAngleZ())
			}
			if tc.eu.HasAlpha() && got.GetAlpha() != tc.eu.GetAlpha() {
				t.Errorf("Alpha mismatch: got %v, want %v", got.GetAlpha(), tc.eu.GetAlpha())
			}
			if tc.eu.HasLerpFinish() && got.GetLerpFinish() != tc.eu.GetLerpFinish() {
				t.Errorf("LerpFinish mismatch: got %v, want %v", got.GetLerpFinish(), tc.eu.GetLerpFinish())
			}
		})
	}
}

func TestParseTempEntityAllTypes(t *testing.T) {
	testCoord := func(c *protos.Coord, x, y, z float32) bool {
		return approxEqual(c.GetX(), x, 0.125) &&
			approxEqual(c.GetY(), y, 0.125) &&
			approxEqual(c.GetZ(), z, 0.125)
	}

	tests := []struct {
		name      string
		writeFunc func(m *net.Message)
		validate  func(t *testing.T, te *protos.TempEntity)
	}{
		{
			name: "TE_SPIKE",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_SPIKE)
				m.WriteCoord(10, 0)
				m.WriteCoord(20, 0)
				m.WriteCoord(30, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Spike_case {
					t.Fatalf("expected Spike, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetSpike(), 10, 20, 30) {
					t.Errorf("Spike coord mismatch: %+v", te.GetSpike())
				}
			},
		},
		{
			name: "TE_SUPERSPIKE",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_SUPERSPIKE)
				m.WriteCoord(11, 0)
				m.WriteCoord(21, 0)
				m.WriteCoord(31, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_SuperSpike_case {
					t.Fatalf("expected SuperSpike, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetSuperSpike(), 11, 21, 31) {
					t.Errorf("SuperSpike coord mismatch: %+v", te.GetSuperSpike())
				}
			},
		},
		{
			name: "TE_GUNSHOT",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_GUNSHOT)
				m.WriteCoord(12, 0)
				m.WriteCoord(22, 0)
				m.WriteCoord(32, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Gunshot_case {
					t.Fatalf("expected Gunshot, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetGunshot(), 12, 22, 32) {
					t.Errorf("Gunshot coord mismatch: %+v", te.GetGunshot())
				}
			},
		},
		{
			name: "TE_EXPLOSION",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_EXPLOSION)
				m.WriteCoord(13, 0)
				m.WriteCoord(23, 0)
				m.WriteCoord(33, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Explosion_case {
					t.Fatalf("expected Explosion, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetExplosion(), 13, 23, 33) {
					t.Errorf("Explosion coord mismatch: %+v", te.GetExplosion())
				}
			},
		},
		{
			name: "TE_TAREXPLOSION",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_TAREXPLOSION)
				m.WriteCoord(14, 0)
				m.WriteCoord(24, 0)
				m.WriteCoord(34, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_TarExplosion_case {
					t.Fatalf("expected TarExplosion, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetTarExplosion(), 14, 24, 34) {
					t.Errorf("TarExplosion coord mismatch: %+v", te.GetTarExplosion())
				}
			},
		},
		{
			name: "TE_LIGHTNING1",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_LIGHTNING1)
				m.WriteShort(42)
				m.WriteCoord(1, 0)
				m.WriteCoord(2, 0)
				m.WriteCoord(3, 0)
				m.WriteCoord(4, 0)
				m.WriteCoord(5, 0)
				m.WriteCoord(6, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Lightning1_case {
					t.Fatalf("expected Lightning1, got %v", te.WhichUnion())
				}
				line := te.GetLightning1()
				if line.GetEntity() != 42 {
					t.Errorf("Entity: got %d, want 42", line.GetEntity())
				}
				if !testCoord(line.GetStart(), 1, 2, 3) || !testCoord(line.GetEnd(), 4, 5, 6) {
					t.Errorf("Start/End coords mismatch: %+v", line)
				}
			},
		},
		{
			name: "TE_LIGHTNING2",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_LIGHTNING2)
				m.WriteShort(43)
				m.WriteCoord(7, 0)
				m.WriteCoord(8, 0)
				m.WriteCoord(9, 0)
				m.WriteCoord(10, 0)
				m.WriteCoord(11, 0)
				m.WriteCoord(12, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Lightning2_case {
					t.Fatalf("expected Lightning2, got %v", te.WhichUnion())
				}
				line := te.GetLightning2()
				if line.GetEntity() != 43 {
					t.Errorf("Entity: got %d, want 43", line.GetEntity())
				}
			},
		},
		{
			name: "TE_WIZSPIKE",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_WIZSPIKE)
				m.WriteCoord(15, 0)
				m.WriteCoord(25, 0)
				m.WriteCoord(35, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_WizSpike_case {
					t.Fatalf("expected WizSpike, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetWizSpike(), 15, 25, 35) {
					t.Errorf("WizSpike coord mismatch: %+v", te.GetWizSpike())
				}
			},
		},
		{
			name: "TE_KNIGHTSPIKE",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_KNIGHTSPIKE)
				m.WriteCoord(16, 0)
				m.WriteCoord(26, 0)
				m.WriteCoord(36, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_KnightSpike_case {
					t.Fatalf("expected KnightSpike, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetKnightSpike(), 16, 26, 36) {
					t.Errorf("KnightSpike coord mismatch: %+v", te.GetKnightSpike())
				}
			},
		},
		{
			name: "TE_LIGHTNING3",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_LIGHTNING3)
				m.WriteShort(44)
				m.WriteCoord(1, 0)
				m.WriteCoord(2, 0)
				m.WriteCoord(3, 0)
				m.WriteCoord(4, 0)
				m.WriteCoord(5, 0)
				m.WriteCoord(6, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Lightning3_case {
					t.Fatalf("expected Lightning3, got %v", te.WhichUnion())
				}
				line := te.GetLightning3()
				if line.GetEntity() != 44 {
					t.Errorf("Entity: got %d, want 44", line.GetEntity())
				}
			},
		},
		{
			name: "TE_LAVASPLASH",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_LAVASPLASH)
				m.WriteCoord(17, 0)
				m.WriteCoord(27, 0)
				m.WriteCoord(37, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_LavaSplash_case {
					t.Fatalf("expected LavaSplash, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetLavaSplash(), 17, 27, 37) {
					t.Errorf("LavaSplash coord mismatch: %+v", te.GetLavaSplash())
				}
			},
		},
		{
			name: "TE_TELEPORT",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_TELEPORT)
				m.WriteCoord(18, 0)
				m.WriteCoord(28, 0)
				m.WriteCoord(38, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Teleport_case {
					t.Fatalf("expected Teleport, got %v", te.WhichUnion())
				}
				if !testCoord(te.GetTeleport(), 18, 28, 38) {
					t.Errorf("Teleport coord mismatch: %+v", te.GetTeleport())
				}
			},
		},
		{
			name: "TE_EXPLOSION2",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_EXPLOSION2)
				m.WriteCoord(19, 0)
				m.WriteCoord(29, 0)
				m.WriteCoord(39, 0)
				m.WriteByte(10) // startColor
				m.WriteByte(20) // stopColor
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Explosion2_case {
					t.Fatalf("expected Explosion2, got %v", te.WhichUnion())
				}
				exp := te.GetExplosion2()
				if !testCoord(exp.GetPosition(), 19, 29, 39) {
					t.Errorf("Explosion2 pos mismatch: %+v", exp.GetPosition())
				}
				if exp.GetStartColor() != 10 || exp.GetStopColor() != 20 {
					t.Errorf("Explosion2 color mismatch: got start %d stop %d, want 10 and 20",
						exp.GetStartColor(), exp.GetStopColor())
				}
			},
		},
		{
			name: "TE_BEAM",
			writeFunc: func(m *net.Message) {
				m.WriteByte(TempEntity)
				m.WriteByte(TE_BEAM)
				m.WriteShort(99)
				m.WriteCoord(100, 0)
				m.WriteCoord(200, 0)
				m.WriteCoord(300, 0)
				m.WriteCoord(400, 0)
				m.WriteCoord(500, 0)
				m.WriteCoord(600, 0)
			},
			validate: func(t *testing.T, te *protos.TempEntity) {
				if te.WhichUnion() != protos.TempEntity_Beam_case {
					t.Fatalf("expected Beam, got %v", te.WhichUnion())
				}
				beam := te.GetBeam()
				if beam.GetEntity() != 99 {
					t.Errorf("Beam entity: got %d, want 99", beam.GetEntity())
				}
				if !testCoord(beam.GetStart(), 100, 200, 300) || !testCoord(beam.GetEnd(), 400, 500, 600) {
					t.Errorf("Beam coords mismatch: %+v", beam)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &net.Message{}
			tc.writeFunc(m)

			reader := net.NewQReader(m.Bytes())
			sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
			if err != nil {
				t.Fatalf("ParseServerMessage failed: %v", err)
			}
			if len(sm.GetCmds()) != 1 {
				t.Fatalf("expected 1 cmd, got %d", len(sm.GetCmds()))
			}
			cmd := sm.GetCmds()[0]
			if cmd.WhichUnion() != protos.SCmd_TempEntity_case {
				t.Fatalf("expected TempEntity, got %v", cmd.WhichUnion())
			}
			tc.validate(t, cmd.GetTempEntity())
		})
	}
}

func TestParseBaseline1And2(t *testing.T) {
	t.Run("SpawnBaseline1", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(SpawnBaseline)
		m.WriteShort(10) // entity index
		m.WriteByte(2)   // model index
		m.WriteByte(3)   // frame
		m.WriteByte(1)   // colormap
		m.WriteByte(0)   // skin
		for i := 0; i < 3; i++ {
			m.WriteCoord(float32(i*10), 0)
			m.WriteAngle(float32(i*20), 0)
		}

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}
		if len(sm.GetCmds()) != 1 {
			t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
		}
		cmd := sm.GetCmds()[0]
		if cmd.WhichUnion() != protos.SCmd_SpawnBaseline_case {
			t.Fatalf("expected SpawnBaseline, got %v", cmd.WhichUnion())
		}
		eb := cmd.GetSpawnBaseline()
		if eb.GetIndex() != 10 {
			t.Errorf("index: got %d, want 10", eb.GetIndex())
		}
		b := eb.GetBaseline()
		if b.GetModelIndex() != 2 || b.GetFrame() != 3 || b.GetColorMap() != 1 || b.GetSkin() != 0 {
			t.Errorf("baseline fields mismatch: %+v", b)
		}
	})

	t.Run("SpawnBaseline2_Extended", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(SpawnBaseline2)
		m.WriteShort(25) // entity index
		// bits: LargeModel | LargeFrame | Alpha
		bits := EntityBaselineLargeModel | EntityBaselineLargeFrame | EntityBaselineAlpha
		m.WriteByte(bits)
		m.WriteShort(300) // 16-bit modelindex
		m.WriteShort(500) // 16-bit frame
		m.WriteByte(2)   // colormap
		m.WriteByte(4)   // skin
		for i := 0; i < 3; i++ {
			m.WriteCoord(float32(i*15), 0)
			m.WriteAngle(float32(i*30), 0)
		}
		m.WriteByte(128) // alpha

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.FitzQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}
		eb := sm.GetCmds()[0].GetSpawnBaseline()
		if eb.GetIndex() != 25 {
			t.Errorf("index: got %d, want 25", eb.GetIndex())
		}
		b := eb.GetBaseline()
		if b.GetModelIndex() != 300 || b.GetFrame() != 500 || b.GetAlpha() != 128 {
			t.Errorf("extended baseline mismatch: %+v", b)
		}
	})

	t.Run("SpawnStatic1", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(SpawnStatic)
		m.WriteByte(1) // model index
		m.WriteByte(0) // frame
		m.WriteByte(0) // colormap
		m.WriteByte(0) // skin
		for i := 0; i < 3; i++ {
			m.WriteCoord(0, 0)
			m.WriteAngle(0, 0)
		}

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}
		cmd := sm.GetCmds()[0]
		if cmd.WhichUnion() != protos.SCmd_SpawnStatic_case {
			t.Fatalf("expected SpawnStatic, got %v", cmd.WhichUnion())
		}
		b := cmd.GetSpawnStatic()
		if b.GetModelIndex() != 1 {
			t.Errorf("SpawnStatic model index: got %d, want 1", b.GetModelIndex())
		}
	})

	t.Run("SpawnStatic2", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(SpawnStatic2)
		m.WriteByte(EntityBaselineLargeModel)
		m.WriteShort(400) // modelindex
		m.WriteByte(5)   // frame (byte)
		m.WriteByte(0)   // colormap
		m.WriteByte(0)   // skin
		for i := 0; i < 3; i++ {
			m.WriteCoord(0, 0)
			m.WriteAngle(0, 0)
		}

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.FitzQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}
		b := sm.GetCmds()[0].GetSpawnStatic()
		if b.GetModelIndex() != 400 || b.GetFrame() != 5 {
			t.Errorf("SpawnStatic2 mismatch: %+v", b)
		}
	})
}

func TestParseServerInfo(t *testing.T) {
	t.Run("NetQuakeServerInfo", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(ServerInfo)
		m.WriteLong(protocol.NetQuake)
		m.WriteByte(16)        // maxClients
		m.WriteByte(GameCoop)  // gameType
		m.WriteString("start") // levelName
		m.WriteString("progs/player.mdl")
		m.WriteString("progs/eyes.mdl")
		m.WriteString("") // end models
		m.WriteString("sound/weapons/guncock.wav")
		m.WriteString("") // end sounds

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}

		if len(sm.GetCmds()) != 1 {
			t.Fatalf("expected 1 command, got %d", len(sm.GetCmds()))
		}
		cmd := sm.GetCmds()[0]
		if cmd.WhichUnion() != protos.SCmd_ServerInfo_case {
			t.Fatalf("expected ServerInfo, got %v", cmd.WhichUnion())
		}
		si := cmd.GetServerInfo()
		if si.GetProtocol() != protocol.NetQuake {
			t.Errorf("protocol: got %d, want %d", si.GetProtocol(), protocol.NetQuake)
		}
		if si.GetMaxClients() != 16 {
			t.Errorf("maxClients: got %d, want 16", si.GetMaxClients())
		}
		if si.GetGameType() != GameCoop {
			t.Errorf("gameType: got %d, want %d", si.GetGameType(), GameCoop)
		}
		if si.GetLevelName() != "start" {
			t.Errorf("levelName: got %q, want 'start'", si.GetLevelName())
		}
		if len(si.GetModelPrecache()) != 2 || si.GetModelPrecache()[0] != "progs/player.mdl" {
			t.Errorf("model precache mismatch: %v", si.GetModelPrecache())
		}
		if len(si.GetSoundPrecache()) != 1 || si.GetSoundPrecache()[0] != "sound/weapons/guncock.wav" {
			t.Errorf("sound precache mismatch: %v", si.GetSoundPrecache())
		}
	})

	t.Run("RMQServerInfoWithFlags", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(ServerInfo)
		m.WriteLong(protocol.RMQ)
		m.WriteLong(0x00010002) // flags
		m.WriteByte(8)          // maxClients
		m.WriteByte(GameDeathmatch)
		m.WriteString("dm1")
		m.WriteString("") // end models
		m.WriteString("") // end sounds

		reader := net.NewQReader(m.Bytes())
		sm, err := ParseServerMessage(reader, protocol.RMQ, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage failed: %v", err)
		}
		si := sm.GetCmds()[0].GetServerInfo()
		if si.GetProtocol() != protocol.RMQ {
			t.Errorf("protocol: got %d, want %d", si.GetProtocol(), protocol.RMQ)
		}
		if si.GetFlags() != 0x00010002 {
			t.Errorf("flags: got %x, want 0x00010002", si.GetFlags())
		}
	})

	t.Run("InvalidProtocolVersion", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(ServerInfo)
		m.WriteLong(999999) // unknown protocol

		reader := net.NewQReader(m.Bytes())
		_, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on invalid protocol version in ServerInfo, got nil")
		}
	})
}

func TestParseSimpleCommands(t *testing.T) {
	m := &net.Message{}

	// Nop
	m.WriteByte(Nop)
	// Disconnect
	m.WriteByte(Disconnect)
	// Version
	m.WriteByte(Version)
	m.WriteLong(protocol.FitzQuake)
	// SetView
	m.WriteByte(SetView)
	m.WriteShort(5)
	// Print
	m.WriteByte(Print)
	m.WriteString("test print\n")
	// CenterPrint
	m.WriteByte(CenterPrint)
	m.WriteString("center message")
	// StuffText
	m.WriteByte(StuffText)
	m.WriteString("reconnect\n")
	// LightStyle
	m.WriteByte(LightStyle)
	m.WriteByte(2)
	m.WriteString("mmnnmm")
	// StopSound
	m.WriteByte(StopSound)
	m.WriteShort(123)
	// SignonNum
	m.WriteByte(SignonNum)
	m.WriteByte(1)
	// KilledMonster
	m.WriteByte(KilledMonster)
	// FoundSecret
	m.WriteByte(FoundSecret)
	// UpdateStat
	m.WriteByte(UpdateStat)
	m.WriteByte(StatHealth)
	m.WriteLong(75)
	// CDTrack
	m.WriteByte(CDTrack)
	m.WriteByte(3)
	m.WriteByte(4)
	// Intermission
	m.WriteByte(Intermission)
	// Finale
	m.WriteByte(Finale)
	m.WriteString("victory text")
	// Cutscene
	m.WriteByte(Cutscene)
	m.WriteString("cutscene text")
	// SellScreen
	m.WriteByte(SellScreen)
	// Skybox
	m.WriteByte(Skybox)
	m.WriteString("unit1_")
	// BF
	m.WriteByte(BF)
	// Fog
	m.WriteByte(Fog)
	m.WriteByte(128) // density: 128/255
	m.WriteByte(255) // r
	m.WriteByte(0)   // g
	m.WriteByte(128) // b
	m.WriteByte(50)  // time: 50/100
	// SpawnStaticSound
	m.WriteByte(SpawnStaticSound)
	m.WriteCoord(10, 0)
	m.WriteCoord(20, 0)
	m.WriteCoord(30, 0)
	m.WriteByte(4)   // num
	m.WriteByte(255) // vol
	m.WriteByte(64)  // att
	// SpawnStaticSound2
	m.WriteByte(SpawnStaticSound2)
	m.WriteCoord(15, 0)
	m.WriteCoord(25, 0)
	m.WriteCoord(35, 0)
	m.WriteShort(350) // num (uint16)
	m.WriteByte(200)  // vol
	m.WriteByte(128)  // att
	// Achievement
	m.WriteByte(Achievement)
	m.WriteString("ach_completed_game")

	reader := net.NewQReader(m.Bytes())
	sm, err := ParseServerMessage(reader, protocol.FitzQuake, 0)
	if err != nil {
		t.Fatalf("ParseServerMessage failed: %v", err)
	}

	cmds := sm.GetCmds()
	expectedCount := 24
	if len(cmds) != expectedCount {
		t.Fatalf("expected %d commands, got %d", expectedCount, len(cmds))
	}

	idx := 0
	checkNext := func(expectedUnion any, desc string) *protos.SCmd {
		t.Helper()
		cmd := cmds[idx]
		if cmd.WhichUnion() != expectedUnion {
			t.Fatalf("cmd %d (%s): got union %v, want %v", idx, desc, cmd.WhichUnion(), expectedUnion)
		}
		idx++
		return cmd
	}

	// Nop
	checkNext(protos.SCmd_Union_not_set_case, "Nop")

	// Disconnect
	cDisc := checkNext(protos.SCmd_Disconnect_case, "Disconnect")
	if !cDisc.GetDisconnect() {
		t.Errorf("Disconnect expected true")
	}

	// Version
	cVer := checkNext(protos.SCmd_Version_case, "Version")
	if cVer.GetVersion() != protocol.FitzQuake {
		t.Errorf("Version: got %d, want %d", cVer.GetVersion(), protocol.FitzQuake)
	}

	// SetView
	cView := checkNext(protos.SCmd_SetViewEntity_case, "SetView")
	if cView.GetSetViewEntity() != 5 {
		t.Errorf("SetViewEntity: got %d, want 5", cView.GetSetViewEntity())
	}

	// Print
	cPrint := checkNext(protos.SCmd_Print_case, "Print")
	if cPrint.GetPrint() != "test print\n" {
		t.Errorf("Print: got %q", cPrint.GetPrint())
	}

	// CenterPrint
	cCenter := checkNext(protos.SCmd_CenterPrint_case, "CenterPrint")
	if cCenter.GetCenterPrint() != "center message" {
		t.Errorf("CenterPrint: got %q", cCenter.GetCenterPrint())
	}

	// StuffText
	cStuff := checkNext(protos.SCmd_StuffText_case, "StuffText")
	if cStuff.GetStuffText() != "reconnect\n" {
		t.Errorf("StuffText: got %q", cStuff.GetStuffText())
	}

	// LightStyle
	cLight := checkNext(protos.SCmd_LightStyle_case, "LightStyle")
	if cLight.GetLightStyle().GetIdx() != 2 || cLight.GetLightStyle().GetNewStyle() != "mmnnmm" {
		t.Errorf("LightStyle: got %+v", cLight.GetLightStyle())
	}

	// StopSound
	cStop := checkNext(protos.SCmd_StopSound_case, "StopSound")
	if cStop.GetStopSound() != 123 {
		t.Errorf("StopSound: got %d", cStop.GetStopSound())
	}

	// SignonNum
	cSignon := checkNext(protos.SCmd_SignonNum_case, "SignonNum")
	if cSignon.GetSignonNum() != 1 {
		t.Errorf("SignonNum: got %d", cSignon.GetSignonNum())
	}

	// KilledMonster
	checkNext(protos.SCmd_KilledMonster_case, "KilledMonster")

	// FoundSecret
	checkNext(protos.SCmd_FoundSecret_case, "FoundSecret")

	// UpdateStat
	cStat := checkNext(protos.SCmd_UpdateStat_case, "UpdateStat")
	if cStat.GetUpdateStat().GetStat() != StatHealth || cStat.GetUpdateStat().GetValue() != 75 {
		t.Errorf("UpdateStat: got %+v", cStat.GetUpdateStat())
	}

	// CDTrack
	cCD := checkNext(protos.SCmd_CdTrack_case, "CDTrack")
	if cCD.GetCdTrack().GetTrackNumber() != 3 || cCD.GetCdTrack().GetLoopTrack() != 4 {
		t.Errorf("CDTrack: got %+v", cCD.GetCdTrack())
	}

	// Intermission
	checkNext(protos.SCmd_Intermission_case, "Intermission")

	// Finale
	cFin := checkNext(protos.SCmd_Finale_case, "Finale")
	if cFin.GetFinale() != "victory text" {
		t.Errorf("Finale: got %q", cFin.GetFinale())
	}

	// Cutscene
	cCut := checkNext(protos.SCmd_Cutscene_case, "Cutscene")
	if cCut.GetCutscene() != "cutscene text" {
		t.Errorf("Cutscene: got %q", cCut.GetCutscene())
	}

	// SellScreen
	checkNext(protos.SCmd_SellScreen_case, "SellScreen")

	// Skybox
	cSky := checkNext(protos.SCmd_Skybox_case, "Skybox")
	if cSky.GetSkybox() != "unit1_" {
		t.Errorf("Skybox: got %q", cSky.GetSkybox())
	}

	// BF
	checkNext(protos.SCmd_BackgroundFlash_case, "BF")

	// Fog
	cFog := checkNext(protos.SCmd_Fog_case, "Fog")
	fog := cFog.GetFog()
	if !approxEqual(fog.GetDensity(), 128.0/255.0, 1e-4) ||
		!approxEqual(fog.GetRed(), 1.0, 1e-4) ||
		!approxEqual(fog.GetGreen(), 0.0, 1e-4) ||
		!approxEqual(fog.GetBlue(), 128.0/255.0, 1e-4) ||
		!approxEqual(fog.GetTime(), 50.0/100.0, 1e-4) {
		t.Errorf("Fog mismatch: %+v", fog)
	}

	// SpawnStaticSound
	cSnd := checkNext(protos.SCmd_SpawnStaticSound_case, "SpawnStaticSound")
	ss := cSnd.GetSpawnStaticSound()
	if ss.GetIndex() != 4 || ss.GetVolume() != 255 || ss.GetAttenuation() != 64 {
		t.Errorf("SpawnStaticSound mismatch: %+v", ss)
	}

	// SpawnStaticSound2
	cSnd2 := checkNext(protos.SCmd_SpawnStaticSound_case, "SpawnStaticSound2")
	ss2 := cSnd2.GetSpawnStaticSound()
	if ss2.GetIndex() != 350 || ss2.GetVolume() != 200 || ss2.GetAttenuation() != 128 {
		t.Errorf("SpawnStaticSound2 mismatch: %+v", ss2)
	}

	// Achievement
	cAch := checkNext(protos.SCmd_Achievement_case, "Achievement")
	if cAch.GetAchievement() != "ach_completed_game" {
		t.Errorf("Achievement: got %q", cAch.GetAchievement())
	}
}

func TestParseServerMessageErrors(t *testing.T) {
	t.Run("UnknownCommandByte", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(120) // Invalid command (high bit not set)
		reader := net.NewQReader(m.Bytes())
		_, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on unknown command, got nil")
		}
		if !strings.Contains(err.Error(), "Illegible server message") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("TruncatedCommandPayload", func(t *testing.T) {
		m := &net.Message{}
		m.WriteByte(Version)
		m.WriteByte(0x01) // Only 1 byte instead of int32 (4 bytes)
		reader := net.NewQReader(m.Bytes())
		_, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err == nil {
			t.Fatal("expected error on truncated command, got nil")
		}
	})

	t.Run("EmptyMessage", func(t *testing.T) {
		reader := net.NewQReader([]byte{})
		sm, err := ParseServerMessage(reader, protocol.NetQuake, 0)
		if err != nil {
			t.Fatalf("ParseServerMessage on empty buffer failed: %v", err)
		}
		if len(sm.GetCmds()) != 0 {
			t.Errorf("expected 0 commands, got %d", len(sm.GetCmds()))
		}
	})
}
