// SPDX-License-Identifier: GPL-2.0-or-later
package server

import (
	"fmt"

	"github.com/therjak/goquake/net"
	"github.com/therjak/goquake/protocol"
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

func readCoord(msg *net.QReader, protocolFlags uint32) (*protos.Coord, error) {
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

func ParseTempEntity(msg *net.QReader, protocolFlags uint32) (*protos.TempEntity, error) {
	readCoordVec := func() (*protos.Coord, error) {
		return readCoord(msg, protocolFlags)
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

func ParseSoundMessage(msg *net.QReader, protocolFlags uint32) (*protos.Sound, error) {
	message := &protos.Sound{}

	fieldMask, err := msg.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
	}

	if fieldMask&SoundVolume != 0 {
		volume, err := msg.ReadByte() // byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.Volume = &protos.OptionalInt32{
			Value: int32(volume),
		}
	}

	if fieldMask&SoundAttenuation != 0 {
		a, err := msg.ReadByte() // byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.Attenuation = &protos.OptionalInt32{
			Value: int32(a),
		}
	}

	ent := uint16(0)
	channel := byte(0)
	if fieldMask&SoundLargeEntity != 0 {
		e, err := msg.ReadInt16() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		c, err := msg.ReadByte() // byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		ent = uint16(e)
		channel = c
	} else {
		s, err := msg.ReadInt16() // int16 + byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		ent = uint16(s >> 3)
		channel = byte(s & 7)
	}
	message.Entity = int32(ent)
	message.Channel = int32(channel)

	soundNum := uint16(0)
	if fieldMask&SoundLargeSound != 0 {
		n, err := msg.ReadInt16() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		soundNum = uint16(n - 1)
	} else {
		n, err := msg.ReadByte() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		soundNum = uint16(n - 1)
	}
	message.SoundNum = int32(soundNum)
	cord, err := readCoord(msg, protocolFlags)
	if err != nil {
		return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
	}
	message.Origin = cord
	return message, nil
}

func ParseBaseline(msg *net.QReader, protocolFlags uint32, version int) (*protos.Baseline, error) {
	bl := &protos.Baseline{}
	var err error
	bits := byte(0)
	if version == 2 {
		bits, err = msg.ReadByte()
		if err != nil {
			return nil, err
		}
	}
	if bits&EntityBaselineLargeModel != 0 {
		if i, err := msg.ReadUint16(); err != nil {
			return nil, err
		} else {
			bl.ModelIndex = int32(i)
		}
	} else {
		if i, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.ModelIndex = int32(i)
		}
	}
	if bits&EntityBaselineLargeFrame != 0 {
		if f, err := msg.ReadUint16(); err != nil {
			return nil, err
		} else {
			bl.Frame = int32(f)
		}
	} else {
		if f, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.Frame = int32(f)
		}
	}

	// colormap: no idea what this is good for. It is not really used.
	if cm, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		bl.ColorMap = int32(cm)
	}
	if s, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		bl.Skin = int32(s)
	}

	o := &protos.Coord{}
	a := &protos.Coord{}
	if o.X, err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a.X, err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	if o.Y, err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a.Y, err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	if o.Z, err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a.Z, err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	bl.Origin = o
	bl.Angles = a

	if bits&EntityBaselineAlpha != 0 {
		if a, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.Alpha = int32(a)
		}
	}

	return bl, nil
}

func ParseServerInfo(msg *net.QReader) (*protos.ServerInfo, error) {
	si := &protos.ServerInfo{}
	var err error

	if si.Protocol, err = msg.ReadInt32(); err != nil {
		return nil, err
	}
	switch si.Protocol {
	case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
	default:
		return nil, fmt.Errorf("Server returned version %d, not %d or %d or %d or %d", si.Protocol,
			protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
	}

	if si.Protocol == protocol.RMQ {
		if flags, err := msg.ReadUint32(); err != nil {
			return nil, err
		} else {
			si.Flags = int32(flags)
		}
	}

	if mc, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		si.MaxClients = int32(mc)
	}

	if gt, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		si.GameType = int32(gt)
	}

	if si.LevelName, err = msg.ReadString(); err != nil {
		return nil, err
	}

	var modelNames []string
	for {
		m, err := msg.ReadString()
		if err != nil {
			return nil, err
		}
		if m == "" {
			break
		}
		modelNames = append(modelNames, m)
	}
	si.ModelPrecache = modelNames

	var sounds []string
	for {
		s, err := msg.ReadString()
		if err != nil {
			return nil, err
		}
		if s == "" {
			break
		}
		sounds = append(sounds, s)
	}
	si.SoundPrecache = sounds

	return si, nil
}

func ParseEntityUpdate(msg *net.QReader, pcol int, protocolFlags uint32, cmd byte) (*protos.EntityUpdate, error) {
	eu := &protos.EntityUpdate{}
	bits := uint32(cmd)
	if bits&U_MOREBITS != 0 {
		if b, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bits |= uint32(b) << 8
		}
	}
	switch pcol {
	case protocol.FitzQuake, protocol.RMQ:
		if bits&U_EXTEND1 != 0 {
			if b, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				bits |= uint32(b) << 16
			}
		}
		if bits&U_EXTEND2 != 0 {
			if b, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				bits |= uint32(b) << 24
			}
		}
	}
	num, err := func() (int32, error) {
		if bits&U_LONGENTITY != 0 {
			s, err := msg.ReadInt16()
			return int32(s), err
		}
		b, err := msg.ReadByte()
		return int32(b), err
	}()
	if err != nil {
		return nil, err
	}
	eu.Entity = num
	eu.LerpMoveStep = bits&U_STEP != 0

	if bits&U_MODEL != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.Model = &protos.OptionalInt32{Value: int32(v)}
		}
	}
	if bits&U_FRAME != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.Frame = &protos.OptionalInt32{Value: int32(v)}
		}
	}
	if bits&U_COLORMAP != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.ColorMap = &protos.OptionalInt32{Value: int32(v)}
		}
	}
	if bits&U_SKIN != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.Skin = &protos.OptionalInt32{Value: int32(v)}
		}
	}
	if bits&U_EFFECTS != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.Effects = int32(v)
		}
	}

	// Why optional in each component? It is near impossible to have only one component changed.
	// I guess changing to per component is a one way street :(
	if bits&U_ORIGIN1 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.OriginX = &protos.OptionalFloat{Value: v}
		}
	}
	if bits&U_ANGLE1 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.AngleX = &protos.OptionalFloat{Value: v}
		}
	}
	if bits&U_ORIGIN2 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.OriginY = &protos.OptionalFloat{Value: v}
		}
	}
	if bits&U_ANGLE2 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.AngleY = &protos.OptionalFloat{Value: v}
		}
	}
	if bits&U_ORIGIN3 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.OriginZ = &protos.OptionalFloat{Value: v}
		}
	}
	if bits&U_ANGLE3 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.AngleZ = &protos.OptionalFloat{Value: v}
		}
	}

	switch pcol {
	case protocol.FitzQuake, protocol.RMQ:
		if bits&U_ALPHA != 0 {
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.Alpha = &protos.OptionalInt32{Value: int32(v)}
			}
		}
		if bits&U_SCALE != 0 {
			// RMQ, currenty ignored
			if _, err := msg.ReadByte(); err != nil {
				return nil, err
			}
		}
		if bits&U_FRAME2 != 0 {
			// Can only be set if U_FRAME is set as well
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.Frame.Value |= int32(v) << 8
			}
		}
		if bits&U_MODEL2 != 0 {
			// Can only be set if U_MODEL is set as well
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.Model.Value |= int32(v) << 8
			}
		}
		if bits&U_LERPFINISH != 0 {
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.LerpFinish = &protos.OptionalFloat{Value: float32(v) / 255}
			}
		}
	case protocol.NetQuake:
		if bits&U_TRANS != 0 {
			// HACK: if this bit is set, assume this is protocol NEHAHRA
			a, err := msg.ReadFloat32()
			if err != nil {
				return nil, err
			}
			b, err := msg.ReadFloat32() // alpha
			if err != nil {
				return nil, err
			}
			if a == 2 {
				// fullbright (not using this yet)
				_, err := msg.ReadFloat32()
				if err != nil {
					return nil, err
				}
			}
			b *= 255
			eu.Alpha = &protos.OptionalInt32{}
			switch {
			case b < 0:
				eu.Alpha.Value = 0
			case b == 0, b >= 255:
				eu.Alpha.Value = 255
			default:
				eu.Alpha.Value = int32(b)
			}
		}
	}

	return eu, nil
}
