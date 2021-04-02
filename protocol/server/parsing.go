// SPDX-License-Identifier: GPL-2.0-or-later
package server

import (
	"fmt"

	"github.com/therjak/goquake/net"
	"github.com/therjak/goquake/protos"
)

func ParseClientData(msg *net.QReader) (*protos.ClientData, error) {
	clientData := &protos.ClientData{}

	m, err := msg.ReadUint16()
	if err != nil {
		return nil, err
	}
	bits := int(m)

	if bits&SU_EXTEND1 != 0 {
		m, err := msg.ReadByte()
		if err != nil {
			return nil, err
		}
		bits |= int(m) << 16
	}
	if bits&SU_EXTEND2 != 0 {
		m, err := msg.ReadByte()
		if err != nil {
			return nil, err
		}
		bits |= int(m) << 24
	}

	readByte := func(v *int32) error {
		nv, err := msg.ReadByte()
		if err != nil {
			return err
		}
		*v = int32(nv)
		return nil
	}
	readByteIf := func(flag int, v *int32) error {
		if bits&flag != 0 {
			return readByte(v)
		}
		return nil
	}

	readInt8 := func(v *int32) error {
		nv, err := msg.ReadInt8()
		if err != nil {
			return err
		}
		*v = int32(nv)
		return nil
	}
	readInt8If := func(flag int, v *int32) error {
		if bits&flag != 0 {
			return readInt8(v)
		}
		return nil
	}

	readUpperByte := func(v *int32) error {
		s, err := msg.ReadByte()
		if err != nil {
			return err
		}
		*v |= int32(s) << 8
		return nil
	}
	readUpperByteIf := func(flag int, v *int32) error {
		if bits&flag != 0 {
			return readUpperByte(v)
		}
		return nil
	}

	if bits&SU_VIEWHEIGHT != 0 {
		m, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.ViewHeight = &protos.OptionalInt32{
			Value: int32(m),
		}
	}

	if err := readInt8If(SU_IDEALPITCH, &clientData.IdealPitch); err != nil {
		return nil, err
	}

	clientData.PunchAngle = &protos.IntCoord{}
	clientData.Velocity = &protos.IntCoord{}

	if err := readInt8If(SU_PUNCH1, &clientData.PunchAngle.X); err != nil {
		return nil, err
	}
	if err := readInt8If(SU_VELOCITY1, &clientData.Velocity.X); err != nil {
		return nil, err
	}
	if err := readInt8If(SU_PUNCH2, &clientData.PunchAngle.Y); err != nil {
		return nil, err
	}
	if err := readInt8If(SU_VELOCITY2, &clientData.Velocity.Y); err != nil {
		return nil, err
	}
	if err := readInt8If(SU_PUNCH3, &clientData.PunchAngle.Z); err != nil {
		return nil, err
	}
	if err := readInt8If(SU_VELOCITY3, &clientData.Velocity.Z); err != nil {
		return nil, err
	}

	// [always sent]	if (bits & SU_ITEMS) != 0
	items, err := msg.ReadUint32()
	if err != nil {
		return nil, err
	}
	clientData.Items = items

	clientData.InWater = bits&SU_INWATER != 0
	clientData.OnGround = bits&SU_ONGROUND != 0

	if err := readByteIf(SU_WEAPONFRAME, &clientData.WeaponFrame); err != nil {
		return nil, err
	}
	if err := readByteIf(SU_ARMOR, &clientData.Armor); err != nil {
		return nil, err
	}
	if err := readByteIf(SU_WEAPON, &clientData.Weapon); err != nil {
		return nil, err
	}

	health, err := msg.ReadInt16()
	if err != nil {
		return nil, err
	}
	clientData.Health = int32(health)

	if err := readByte(&clientData.Ammo); err != nil {
		return nil, err
	}
	if err := readByte(&clientData.Shells); err != nil {
		return nil, err
	}
	if err := readByte(&clientData.Nails); err != nil {
		return nil, err
	}
	if err := readByte(&clientData.Rockets); err != nil {
		return nil, err
	}
	if err := readByte(&clientData.Cells); err != nil {
		return nil, err
	}
	if err := readByte(&clientData.ActiveWeapon); err != nil {
		return nil, err
	}

	if err := readUpperByteIf(SU_WEAPON2, &clientData.Weapon); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_ARMOR2, &clientData.Armor); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_AMMO2, &clientData.Ammo); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_SHELLS2, &clientData.Shells); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_NAILS2, &clientData.Nails); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_ROCKETS2, &clientData.Rockets); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_CELLS2, &clientData.Cells); err != nil {
		return nil, err
	}
	if err := readUpperByteIf(SU_WEAPONFRAME2, &clientData.WeaponFrame); err != nil {
		return nil, err
	}
	if err := readByteIf(SU_WEAPONALPHA, &clientData.WeaponAlpha); err != nil {
		return nil, err
	}
	return clientData, nil
}

func ParseTempEntity(msg *net.QReader, protocolFlags uint32) (*protos.TempEntity, error) {
	readCoordVec := func() (*protos.Coord, error) {
		x, err := msg.ReadCoord(protocolFlags)
		if err != nil {
			return nil, err
		}
		y, err := msg.ReadCoord(protocolFlags)
		if err != nil {
			return nil, err
		}
		z, err := msg.ReadCoord(protocolFlags)
		if err != nil {
			return nil, err
		}
		return &protos.Coord{
			X: x,
			Y: y,
			Z: z,
		}, nil
	}
	t, err := msg.ReadByte()
	if err != nil {
		return nil, err
	}
	switch t {
	case TE_SPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Spike{pos},
		}, nil
	case TE_SUPERSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_SuperSpike{pos},
		}, nil
	case TE_GUNSHOT:
		// bullet hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Gunshot{pos},
		}, nil
	case TE_EXPLOSION:
		// rocket explosion
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Explosion{pos},
		}, nil
	case TE_TAREXPLOSION:
		// tarbaby explosion
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_TarExplosion{pos},
		}, nil
	case TE_LIGHTNING1:
		// lightning bolts
		ent, err := msg.ReadInt16()
		if err != nil {
			return nil, err
		}
		s, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		e, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Lightning1{
				&protos.Line{
					Entity: int32(ent),
					Start:  s,
					End:    e,
				},
			},
		}, nil
	case TE_LIGHTNING2:
		// lightning bolts
		ent, err := msg.ReadInt16()
		if err != nil {
			return nil, err
		}
		s, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		e, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Lightning2{
				&protos.Line{
					Entity: int32(ent),
					Start:  s,
					End:    e,
				},
			},
		}, nil
	case TE_WIZSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_WizSpike{pos},
		}, nil
	case TE_KNIGHTSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_KnightSpike{pos},
		}, nil
	case TE_LIGHTNING3:
		// lightning bolts
		ent, err := msg.ReadInt16()
		if err != nil {
			return nil, err
		}
		s, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		e, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Lightning3{
				&protos.Line{
					Entity: int32(ent),
					Start:  s,
					End:    e,
				},
			},
		}, nil
	case TE_LAVASPLASH:
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_LavaSplash{pos},
		}, nil
	case TE_TELEPORT:
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Teleport{pos},
		}, nil
	case TE_EXPLOSION2:
		// color mapped explosion
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		var color struct {
			start byte
			end   byte
		}
		if err = msg.Read(&color); err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Explosion2{
				&protos.Explosion2{
					Position:   pos,
					StartColor: int32(color.start),
					StopColor:  int32(color.end),
				},
			},
		}, nil
	case TE_BEAM:
		// grappling hook beam
		ent, err := msg.ReadInt16()
		if err != nil {
			return nil, err
		}
		s, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		e, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return &protos.TempEntity{
			Union: &protos.TempEntity_Beam{
				&protos.Line{
					Entity: int32(ent),
					Start:  s,
					End:    e,
				},
			},
		}, nil
	}
	return nil, fmt.Errorf("CL_ParseTEnt: bad type")
}
