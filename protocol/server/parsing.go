// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"

	"goquake/net"
	"goquake/protocol"
	"goquake/protos"
)

var (
	svc_strings = []string{
		"svc_bad", "svc_nop", "svc_disconnect", "svc_updatestat",
		"svc_version",   // [long] server version
		"svc_setview",   // [short] entity number
		"svc_sound",     // <see code>
		"svc_time",      // [float] server time
		"svc_print",     // [string] null terminated string
		"svc_stufftext", // [string] stuffed into client's console buffer
		// the string should be \n terminated
		"svc_setangle", // [vec3] set the view angle to this absolute value

		"svc_serverinfo", // [long] version
		// [string] signon string
		// [string]..[0]model cache [string]...[0]sounds cache
		// [string]..[0]item cache
		"svc_lightstyle",   // [byte] [string]
		"svc_updatename",   // [byte] [string]
		"svc_updatefrags",  // [byte] [short]
		"svc_clientdata",   // <shortbits + data>
		"svc_stopsound",    // <see code>
		"svc_updatecolors", // [byte] [byte]
		"svc_particle",     // [vec3] <variable>
		"svc_damage",       // [byte] impact [byte] blood [vec3] from

		"svc_spawnstatic", "OBSOLETE svc_spawnbinary", "svc_spawnbaseline",

		"svc_temp_entity", // <variable>
		"svc_setpause", "svc_signonnum", "svc_centerprint", "svc_killedmonster",
		"svc_foundsecret", "svc_spawnstaticsound", "svc_intermission",
		"svc_finale",  // [string] music [string] text
		"svc_cdtrack", // [byte] track [byte] looptrack
		"svc_sellscreen", "svc_cutscene",
		"",                      // 35
		"",                      // 36
		"svc_skybox",            // 37 [string] skyname
		"",                      // 38
		"",                      // 39
		"svc_bf",                // 40 no data
		"svc_fog",               // 41 [byte] density [byte] red [byte] green [byte] blue [float] time
		"svc_spawnbaseline2",    // 42 support for large modelindex, large framenum, alpha, using flags
		"svc_spawnstatic2",      // 43 support for large modelindex, large framenum, alpha, using flags
		"svc_spawnstaticsound2", //	44 [coord3] [short] samp [byte] vol [byte] aten
		"",                      // 44
		"",                      // 45
		"",                      // 46
		"",                      // 47
		"",                      // 48
		"",                      // 49
		"",                      // 50
		"",                      // 51
		"svc_achievement",       // 52
	}
)

func parseClientData(msg *net.QReader) (*protos.ClientData, error) {
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

func readAngle(msg *net.QReader, protocolFlags uint32) (*protos.Coord, error) {
	x, err := msg.ReadAngle(protocolFlags)
	if err != nil {
		return nil, err
	}
	y, err := msg.ReadAngle(protocolFlags)
	if err != nil {
		return nil, err
	}
	z, err := msg.ReadAngle(protocolFlags)
	if err != nil {
		return nil, err
	}
	return &protos.Coord{
		X: x,
		Y: y,
		Z: z,
	}, nil
}

func parseTempEntity(msg *net.QReader, protocolFlags uint32) (*protos.TempEntity, error) {
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

func parseSoundMessage(msg *net.QReader, protocolFlags uint32) (*protos.Sound, error) {
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

	if fieldMask&SoundLargeEntity != 0 {
		e, err := msg.ReadInt16() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		c, err := msg.ReadByte() // byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.Entity = int32(e)
		message.Channel = int32(c)
	} else {
		s, err := msg.ReadInt16() // int16 + byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.Entity = int32(s >> 3)
		message.Channel = int32(s & 7)
	}

	if fieldMask&SoundLargeSound != 0 {
		n, err := msg.ReadInt16() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SoundNum = int32(n - 1)
	} else {
		n, err := msg.ReadByte() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SoundNum = int32(n - 1)
	}
	cord, err := readCoord(msg, protocolFlags)
	if err != nil {
		return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
	}
	message.Origin = cord
	return message, nil
}

func parseBaseline(msg *net.QReader, protocolFlags uint32, version int) (*protos.Baseline, error) {
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

func parseServerInfo(msg *net.QReader) (*protos.ServerInfo, error) {
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

func parseEntityUpdate(msg *net.QReader, pcol int, protocolFlags uint32, cmd byte) (*protos.EntityUpdate, error) {
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
			// RMQ, currently ignored
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

func ParseServerMessage(msg *net.QReader, protocol int, protocolFlags uint32) (*protos.ServerMessage, error) {
	sm := &protos.ServerMessage{}
	lastcmd := byte(0)
	for {
		if msg.Len() == 0 {
			// end of message
			return sm, nil
		}
		cmd, _ := msg.ReadByte() // we already checked for at least 1 byte

		// if the high bit of the command byte is set, it is a fast update
		if cmd&U_SIGNAL != 0 {
			if eu, err := parseEntityUpdate(msg, protocol, protocolFlags, cmd&127); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_EntityUpdate{eu},
				})
			}
			continue
		}

		switch cmd {
		default:
			return nil, fmt.Errorf("Illegible server message, previous was %s", svc_strings[lastcmd])

		case Nop:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{})
		case Time:
			if t, err := msg.ReadFloat32(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Time{t},
				})
			}
		case ClientData:
			if cdp, err := parseClientData(msg); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_ClientData{cdp},
				})
			}
		case Version:
			if i, err := msg.ReadInt32(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Version{int32(i)},
				})
			}
		case Disconnect:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_Disconnect{true},
			})
		case Print:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Print{s},
				})
			}
		case CenterPrint:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_CenterPrint{s},
				})
			}
		case StuffText:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_StuffText{s},
				})
			}
		case Damage:
			var data struct {
				Armor byte
				Blood byte
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			if pos, err := readCoord(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Damage{&protos.Damage{
						Armor:    int32(data.Armor),
						Blood:    int32(data.Blood),
						Position: pos,
					}},
				})
			}
		case ServerInfo:
			if si, err := parseServerInfo(msg); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_ServerInfo{si},
				})
			}
		case SetAngle:
			if a, err := readAngle(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SetAngle{a},
				})
			}
		case SetView:
			if ve, err := msg.ReadUint16(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SetViewEntity{int32(ve)},
				})
			}
		case LightStyle:
			cmd := &protos.LightStyle{}
			if idx, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				cmd.Idx = int32(idx)
			}
			if str, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				cmd.NewStyle = str
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_LightStyle{cmd},
			})
		case Sound:
			if spp, err := parseSoundMessage(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Sound{spp},
				})
			}
		case StopSound:
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_StopSound{int32(i)},
				})
			}
		case UpdateName:
			un := &protos.UpdateName{}
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				un.Player = int32(i)
			}
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				un.NewName = s
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_UpdateName{un},
			})
		case UpdateFrags:
			var data struct {
				Player   byte
				NewFrags int16
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			uf := &protos.UpdateFrags{
				Player:   int32(data.Player),
				NewFrags: int32(data.NewFrags),
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_UpdateFrags{uf},
			})
		case UpdateColors:
			var data struct {
				Player   byte
				NewColor byte
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			uc := &protos.UpdateColors{
				Player:   int32(data.Player),
				NewColor: int32(data.NewColor),
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_UpdateColors{uc},
			})
		case Particle:
			org, err := readCoord(msg, protocolFlags)
			if err != nil {
				return nil, err
			}
			var data struct {
				Dir   [3]int8
				Count uint8
				Color uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			count := int32(data.Count)
			if count == 255 {
				// there is no size difference in protos between 255 and 1024 so just keep the logic here
				count = 1024
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_Particle{&protos.Particle{
					Origin: org,
					Direction: &protos.Coord{
						X: float32(data.Dir[0]) * (1.0 / 16),
						Y: float32(data.Dir[1]) * (1.0 / 16),
						Z: float32(data.Dir[2]) * (1.0 / 16),
					},
					Count: count,
					Color: int32(data.Color),
				}},
			})
		case SpawnBaseline:
			eb := &protos.EntityBaseline{}
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				eb.Index = int32(i)
			}
			if pb, err := parseBaseline(msg, protocolFlags, 1); err != nil {
				return nil, err
			} else {
				eb.Baseline = pb
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_SpawnBaseline{eb},
			})

		case SpawnStatic:
			if pb, err := parseBaseline(msg, protocolFlags, 1); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SpawnStatic{pb},
				})
			}

		case TempEntity:
			if tep, err := parseTempEntity(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_TempEntity{tep},
				})
			}
		case SetPause:
			// this byte was used to pause cd audio, other pause as well?
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SetPause{i != 0},
				})
			}
		case SignonNum:
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SignonNum{int32(i)},
				})
			}
		case KilledMonster:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_KilledMonster{},
			})
		case FoundSecret:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_FoundSecret{},
			})
		case UpdateStat:
			var data struct {
				Stat byte
				Val  int32
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_UpdateStat{&protos.UpdateStat{
					Stat:  int32(data.Stat),
					Value: int32(data.Val),
				}},
			})
		case SpawnStaticSound:
			ss := &protos.StaticSound{}
			if org, err := readCoord(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				ss.Origin = org
			}
			var data struct {
				Num uint8
				Vol uint8
				Att uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			ss.Index = int32(data.Num)
			ss.Volume = int32(data.Vol)
			ss.Attenuation = int32(data.Att)
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_SpawnStaticSound{ss},
			})
		case CDTrack:
			var data struct {
				TrackNumber uint8
				Loop        uint8 // was for cl.looptrack
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_CdTrack{&protos.CDTrack{
					TrackNumber: int32(data.TrackNumber),
					LoopTrack:   int32(data.Loop),
				}},
			})
		case Intermission:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_Intermission{},
			})
		case Finale:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Finale{s},
				})
			}
		case Cutscene:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Cutscene{s},
				})
			}
		case SellScreen:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_SellScreen{},
			})
		case Skybox:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Skybox{s},
				})
			}
		case BF:
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_BackgroundFlash{},
			})
		case Fog:
			var data struct {
				Density uint8
				Red     uint8
				Green   uint8
				Blue    uint8
				Time    uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_Fog{&protos.Fog{
					Density: float32(data.Density) / 255.0,
					Red:     float32(data.Red) / 255.0,
					Green:   float32(data.Green) / 255.0,
					Blue:    float32(data.Blue) / 255.0,
					Time:    float32(data.Time) / 100.0,
				}},
			})
		case SpawnBaseline2:
			sb := &protos.EntityBaseline{}
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				sb.Index = int32(i)
			}
			if pb, err := parseBaseline(msg, protocolFlags, 2); err != nil {
				return nil, err
			} else {
				sb.Baseline = pb
			}
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_SpawnBaseline{sb},
			})

		case SpawnStatic2:
			if pb, err := parseBaseline(msg, protocolFlags, 2); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_SpawnStatic{pb},
				})
			}
		case SpawnStaticSound2:
			ss := &protos.StaticSound{}
			if org, err := readCoord(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				ss.Origin = org
			}
			var data struct {
				Num uint16
				Vol uint8
				Att uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			ss.Index = int32(data.Num)
			ss.Volume = int32(data.Vol)
			ss.Attenuation = int32(data.Att)
			sm.Cmds = append(sm.Cmds, &protos.SCmd{
				Union: &protos.SCmd_SpawnStaticSound{ss},
			})
		case Achievement:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.Cmds = append(sm.Cmds, &protos.SCmd{
					Union: &protos.SCmd_Achievement{s},
				})
			}
		}
		lastcmd = cmd
	}
}

func writeCoord(c *protos.Coord, protocolFlags uint32, m *net.Message) {
	m.WriteCoord(c.X, protocolFlags)
	m.WriteCoord(c.Y, protocolFlags)
	m.WriteCoord(c.Z, protocolFlags)
}

func writeAngle(a *protos.Coord, protocolFlags uint32, m *net.Message) {
	m.WriteAngle(a.X, protocolFlags)
	m.WriteAngle(a.Y, protocolFlags)
	m.WriteAngle(a.Z, protocolFlags)
}

func WriteParticle(p *protos.Particle, protocolFlags uint32, m *net.Message) {
	m.WriteByte(Particle)
	writeCoord(p.Origin, protocolFlags, m)
	df := func(d float32) int {
		v := d * 16
		if v > 127 {
			return 127
		}
		if v < -128 {
			return -128
		}
		return int(v)
	}
	m.WriteChar(df(p.Direction.X))
	m.WriteChar(df(p.Direction.Y))
	m.WriteChar(df(p.Direction.Z))
	m.WriteByte(int(p.Count))
	m.WriteByte(int(p.Color))
}

func WriteSound(s *protos.Sound, pcol int, flags uint32, m *net.Message) {
	fieldMask := 0
	if s.Entity >= 8192 {
		if pcol == protocol.NetQuake {
			return
		}
		fieldMask |= SoundLargeEntity
	}
	if s.SoundNum >= 256 || s.Channel >= 8 {
		if pcol == protocol.NetQuake {
			return
		}
		fieldMask |= SoundLargeSound
	}
	v := s.Volume
	if v != nil {
		fieldMask |= SoundVolume
	}
	a := s.Attenuation
	if a != nil {
		fieldMask |= SoundAttenuation
	}
	m.WriteByte(Sound)
	m.WriteByte(fieldMask)
	if v != nil {
		m.WriteByte(int(v.Value))
	}
	if a != nil {
		m.WriteByte(int(a.Value))
	}
	if fieldMask&SoundLargeEntity != 0 {
		m.WriteShort(int(s.Entity))
		m.WriteByte(int(s.Channel))
	} else {
		m.WriteShort(int(s.Entity<<3 | s.Channel))
	}
	if fieldMask&SoundLargeSound != 0 {
		m.WriteShort(int(s.SoundNum))
	} else {
		m.WriteByte(int(s.SoundNum))
	}
	writeCoord(s.Origin, flags, m)
}

func WriteDamage(d *protos.Damage, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(Damage)
	m.WriteByte(int(d.Armor))
	m.WriteByte(int(d.Blood))
	writeCoord(d.Position, flags, m)
}

func WriteSetAngle(a *protos.Coord, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(SetAngle)
	writeAngle(a, flags, m)
}

func WriteClientData(cd *protos.ClientData, pcol int, flags uint32, m *net.Message) {
	bits := 0
	if cd.ViewHeight != nil {
		bits |= SU_VIEWHEIGHT
	}
	if cd.IdealPitch != 0 {
		bits |= SU_IDEALPITCH
	}
	bits |= SU_ITEMS
	bits |= SU_WEAPON
	if cd.OnGround {
		bits |= SU_ONGROUND
	}
	if cd.InWater {
		bits |= SU_INWATER
	}
	if cd.PunchAngle.X != 0 {
		bits |= SU_PUNCH1
	}
	if cd.PunchAngle.Y != 0 {
		bits |= SU_PUNCH2
	}
	if cd.PunchAngle.Z != 0 {
		bits |= SU_PUNCH3
	}
	if cd.Velocity.X != 0 {
		bits |= SU_VELOCITY1
	}
	if cd.Velocity.Y != 0 {
		bits |= SU_VELOCITY2
	}
	if cd.Velocity.Z != 0 {
		bits |= SU_VELOCITY3
	}
	if cd.WeaponFrame != 0 {
		bits |= SU_WEAPONFRAME
	}
	if cd.Armor != 0 {
		bits |= SU_ARMOR
	}

	if pcol != protocol.NetQuake {
		if (cd.Weapon & 0xFF00) != 0 {
			bits |= SU_WEAPON2
		}
		if (cd.Armor & 0xFF00) != 0 {
			bits |= SU_ARMOR2
		}
		if (cd.Ammo & 0xFF00) != 0 {
			bits |= SU_AMMO2
		}
		if (cd.Shells & 0xFF00) != 0 {
			bits |= SU_SHELLS2
		}
		if (cd.Nails & 0xFF00) != 0 {
			bits |= SU_NAILS2
		}
		if (cd.Rockets & 0xFF00) != 0 {
			bits |= SU_ROCKETS2
		}
		if (cd.Cells & 0xFF00) != 0 {
			bits |= SU_CELLS2
		}
		if (bits&SU_WEAPONFRAME != 0) &&
			(cd.WeaponFrame&0xFF00) != 0 {
			bits |= SU_WEAPONFRAME2
		}
		if cd.WeaponAlpha != 0 {
			bits |= SU_WEAPONALPHA
		}
		if bits >= 65536 {
			bits |= SU_EXTEND1
		}
		if bits >= 16777216 {
			bits |= SU_EXTEND2
		}
	}
	m.WriteByte(ClientData)
	m.WriteShort(bits)
	if (bits & SU_EXTEND1) != 0 {
		m.WriteByte(bits >> 16)
	}
	if (bits & SU_EXTEND2) != 0 {
		m.WriteByte(bits >> 24)
	}
	if (bits & SU_VIEWHEIGHT) != 0 {
		m.WriteChar(int(cd.ViewHeight.Value))
	}
	if (bits & SU_IDEALPITCH) != 0 {
		m.WriteChar(int(cd.IdealPitch))
	}
	if (bits & SU_PUNCH1) != 0 {
		m.WriteChar(int(cd.PunchAngle.X))
	}
	if (bits & SU_VELOCITY1) != 0 {
		m.WriteChar(int(cd.Velocity.X))
	}
	if (bits & SU_PUNCH2) != 0 {
		m.WriteChar(int(cd.PunchAngle.Y))
	}
	if (bits & SU_VELOCITY2) != 0 {
		m.WriteChar(int(cd.Velocity.Y))
	}
	if (bits & SU_PUNCH3) != 0 {
		m.WriteChar(int(cd.PunchAngle.Z))
	}
	if (bits & SU_VELOCITY3) != 0 {
		m.WriteChar(int(cd.Velocity.Z))
	}
	m.WriteLong(int(cd.Items))

	if (bits & SU_WEAPONFRAME) != 0 {
		m.WriteByte(int(cd.WeaponFrame))
	}
	if (bits & SU_ARMOR) != 0 {
		m.WriteByte(int(cd.Armor))
	}
	m.WriteByte(int(cd.Weapon))
	m.WriteShort(int(cd.Health))
	m.WriteByte(int(cd.Ammo))
	m.WriteByte(int(cd.Shells))
	m.WriteByte(int(cd.Nails))
	m.WriteByte(int(cd.Rockets))
	m.WriteByte(int(cd.Cells))
	m.WriteByte(int(cd.ActiveWeapon))

	if (bits & SU_WEAPON2) != 0 {
		m.WriteByte(int(cd.Weapon >> 8))
	}
	if (bits & SU_ARMOR2) != 0 {
		m.WriteByte(int(cd.Armor) >> 8)
	}
	if (bits & SU_AMMO2) != 0 {
		m.WriteByte(int(cd.Ammo) >> 8)
	}
	if (bits & SU_SHELLS2) != 0 {
		m.WriteByte(int(cd.Shells) >> 8)
	}
	if (bits & SU_NAILS2) != 0 {
		m.WriteByte(int(cd.Nails) >> 8)
	}
	if (bits & SU_ROCKETS2) != 0 {
		m.WriteByte(int(cd.Rockets) >> 8)
	}
	if (bits & SU_CELLS2) != 0 {
		m.WriteByte(int(cd.Cells) >> 8)
	}
	if (bits & SU_WEAPONFRAME2) != 0 {
		m.WriteByte(int(cd.WeaponFrame) >> 8)
	}
	if (bits & SU_WEAPONALPHA) != 0 {
		m.WriteByte(int(cd.WeaponAlpha))
	}
}

func WriteTime(t float32, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(Time)
	m.WriteFloat(t)
}

func WriteUpdateFrags(uf *protos.UpdateFrags, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(UpdateFrags)
	m.WriteByte(int(uf.Player))
	m.WriteShort(int(uf.NewFrags))
}
