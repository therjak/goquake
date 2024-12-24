// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"fmt"

	"goquake/math/vec"
	"goquake/net"
	"goquake/protocol"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
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

	has := func(flag int) bool {
		return bits&flag != 0
	}

	if has(SU_EXTEND1) {
		m, err := msg.ReadByte()
		if err != nil {
			return nil, err
		}
		bits |= int(m) << 16
	}
	if has(SU_EXTEND2) {
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

	readUpperByte := func(v *int32) error {
		s, err := msg.ReadByte()
		if err != nil {
			return err
		}
		*v |= int32(s) << 8
		return nil
	}

	if has(SU_VIEWHEIGHT) {
		m, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.SetViewHeight(int32(m))
	}

	if has(SU_IDEALPITCH) {
		p, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.SetIdealPitch(int32(p))
	}

	clientData.SetPunchAngle(&protos.IntCoord{})
	clientData.SetVelocity(&protos.IntCoord{})

	if has(SU_PUNCH1) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetPunchAngle().SetX(int32(v))
	}
	if has(SU_VELOCITY1) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetVelocity().SetX(int32(v))
	}
	if has(SU_PUNCH2) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetPunchAngle().SetY(int32(v))
	}
	if has(SU_VELOCITY2) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetVelocity().SetY(int32(v))
	}
	if has(SU_PUNCH3) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetPunchAngle().SetZ(int32(v))
	}
	if has(SU_VELOCITY3) {
		v, err := msg.ReadInt8()
		if err != nil {
			return nil, err
		}
		clientData.GetVelocity().SetZ(int32(v))
	}

	// [always sent]	if (bits & SU_ITEMS) != 0
	items, err := msg.ReadUint32()
	if err != nil {
		return nil, err
	}
	clientData.SetItems(items)

	clientData.SetInWater(bits&SU_INWATER != 0)
	clientData.SetOnGround(bits&SU_ONGROUND != 0)

	var weaponFrame int32
	var armor int32
	var weapon int32
	if has(SU_WEAPONFRAME) {
		if err := readByte(&weaponFrame); err != nil {
			return nil, err
		}
	}
	if has(SU_ARMOR) {
		if err := readByte(&armor); err != nil {
			return nil, err
		}
	}
	if has(SU_WEAPON) {
		if err := readByte(&weapon); err != nil {
			return nil, err
		}
	}

	health, err := msg.ReadInt16()
	if err != nil {
		return nil, err
	}
	clientData.SetHealth(int32(health))

	var ammo int32
	var shells int32
	var nails int32
	var rockets int32
	var cells int32
	if err := readByte(&ammo); err != nil {
		return nil, err
	}
	if err := readByte(&shells); err != nil {
		return nil, err
	}
	if err := readByte(&nails); err != nil {
		return nil, err
	}
	if err := readByte(&rockets); err != nil {
		return nil, err
	}
	if err := readByte(&cells); err != nil {
		return nil, err
	}
	active, err := msg.ReadByte()
	if err != nil {
		return nil, err
	}
	clientData.SetActiveWeapon(int32(active))

	if has(SU_WEAPON2) {
		if err := readUpperByte(&weapon); err != nil {
			return nil, err
		}
	}
	if has(SU_ARMOR2) {
		if err := readUpperByte(&armor); err != nil {
			return nil, err
		}
	}
	if has(SU_AMMO2) {
		if err := readUpperByte(&ammo); err != nil {
			return nil, err
		}
	}
	if has(SU_SHELLS2) {
		if err := readUpperByte(&shells); err != nil {
			return nil, err
		}
	}
	if has(SU_NAILS2) {
		if err := readUpperByte(&nails); err != nil {
			return nil, err
		}
	}
	if has(SU_ROCKETS2) {
		if err := readUpperByte(&rockets); err != nil {
			return nil, err
		}
	}
	if has(SU_CELLS2) {
		if err := readUpperByte(&cells); err != nil {
			return nil, err
		}
	}
	if has(SU_WEAPONFRAME2) {
		if err := readUpperByte(&weaponFrame); err != nil {
			return nil, err
		}
	}
	if has(SU_WEAPONALPHA) {
		v, err := msg.ReadByte()
		if err != nil {
			return nil, err
		}
		clientData.SetWeaponAlpha(int32(v))
	}
	clientData.SetAmmo(ammo)
	clientData.SetShells(shells)
	clientData.SetNails(nails)
	clientData.SetRockets(rockets)
	clientData.SetCells(cells)
	if has(SU_WEAPONFRAME) || has(SU_WEAPONFRAME2) {
		clientData.SetWeaponFrame(weaponFrame)
	}
	if has(SU_ARMOR) || has(SU_ARMOR2) {
		clientData.SetArmor(armor)
	}
	if has(SU_WEAPON) || has(SU_WEAPON2) {
		clientData.SetWeapon(weapon)
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
	return protos.Coord_builder{
		X: x,
		Y: y,
		Z: z,
	}.Build(), nil
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
	return protos.Coord_builder{
		X: x,
		Y: y,
		Z: z,
	}.Build(), nil
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
		return protos.TempEntity_builder{
			Spike: pos,
		}.Build(), nil
	case TE_SUPERSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			SuperSpike: pos,
		}.Build(), nil
	case TE_GUNSHOT:
		// bullet hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			Gunshot: pos,
		}.Build(), nil
	case TE_EXPLOSION:
		// rocket explosion
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			Explosion: pos,
		}.Build(), nil
	case TE_TAREXPLOSION:
		// tarbaby explosion
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			TarExplosion: pos,
		}.Build(), nil
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
		return protos.TempEntity_builder{
			Lightning1: protos.Line_builder{
				Entity: int32(ent),
				Start:  s,
				End:    e,
			}.Build(),
		}.Build(), nil
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
		return protos.TempEntity_builder{
			Lightning2: protos.Line_builder{
				Entity: int32(ent),
				Start:  s,
				End:    e,
			}.Build(),
		}.Build(), nil
	case TE_WIZSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			WizSpike: pos,
		}.Build(), nil
	case TE_KNIGHTSPIKE:
		// spike hitting wall
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			KnightSpike: pos,
		}.Build(), nil
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
		return protos.TempEntity_builder{
			Lightning3: protos.Line_builder{
				Entity: int32(ent),
				Start:  s,
				End:    e,
			}.Build(),
		}.Build(), nil
	case TE_LAVASPLASH:
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			LavaSplash: pos,
		}.Build(), nil
	case TE_TELEPORT:
		pos, err := readCoordVec()
		if err != nil {
			return nil, err
		}
		return protos.TempEntity_builder{
			Teleport: pos,
		}.Build(), nil
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
		return protos.TempEntity_builder{
			Explosion2: protos.Explosion2_builder{
				Position:   pos,
				StartColor: int32(color.start),
				StopColor:  int32(color.end),
			}.Build(),
		}.Build(), nil
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
		return protos.TempEntity_builder{
			Beam: protos.Line_builder{
				Entity: int32(ent),
				Start:  s,
				End:    e,
			}.Build(),
		}.Build(), nil
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
		message.SetVolume(int32(volume))
	}

	if fieldMask&SoundAttenuation != 0 {
		a, err := msg.ReadByte() // byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SetAttenuation(int32(a))
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
		message.SetEntity(int32(e))
		message.SetChannel(int32(c))
	} else {
		s, err := msg.ReadInt16() // int16 + byte
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SetEntity(int32(s >> 3))
		message.SetChannel(int32(s & 7))
	}

	if fieldMask&SoundLargeSound != 0 {
		n, err := msg.ReadInt16() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SetSoundNum(int32(n - 1))
	} else {
		n, err := msg.ReadByte() // int16
		if err != nil {
			return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
		}
		message.SetSoundNum(int32(n - 1))
	}
	cord, err := readCoord(msg, protocolFlags)
	if err != nil {
		return nil, fmt.Errorf("CL_ParseStartSoundPacket: %v", err)
	}
	message.SetOrigin(cord)
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
			bl.SetModelIndex(int32(i))
		}
	} else {
		if i, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.SetModelIndex(int32(i))
		}
	}
	if bits&EntityBaselineLargeFrame != 0 {
		if f, err := msg.ReadUint16(); err != nil {
			return nil, err
		} else {
			bl.SetFrame(int32(f))
		}
	} else {
		if f, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.SetFrame(int32(f))
		}
	}

	// colormap: no idea what this is good for. It is not really used.
	if cm, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		bl.SetColorMap(int32(cm))
	}
	if s, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		bl.SetSkin(int32(s))
	}

	var o, a vec.Vec3
	if o[0], err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a[0], err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	if o[1], err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a[1], err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	if o[2], err = msg.ReadCoord(protocolFlags); err != nil {
		return nil, err
	}
	if a[2], err = msg.ReadAngle(protocolFlags); err != nil {
		return nil, err
	}
	bl.SetOrigin(protos.Coord_builder{
		X: o[0],
		Y: o[1],
		Z: o[2],
	}.Build())
	bl.SetAngles(protos.Coord_builder{
		X: a[0],
		Y: a[1],
		Z: a[2],
	}.Build())

	if bits&EntityBaselineAlpha != 0 {
		if a, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			bl.SetAlpha(int32(a))
		}
	}

	return bl, nil
}

func parseServerInfo(msg *net.QReader) (*protos.ServerInfo, error) {
	si := &protos.ServerInfo{}
	var err error

	v, err := msg.ReadInt32()
	if err != nil {
		return nil, err
	}
	si.SetProtocol(v)

	switch si.GetProtocol() {
	case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
	default:
		return nil, fmt.Errorf("Server returned version %d, not %d or %d or %d or %d", si.GetProtocol(),
			protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
	}

	if si.GetProtocol() == protocol.RMQ {
		if flags, err := msg.ReadUint32(); err != nil {
			return nil, err
		} else {
			si.SetFlags(int32(flags))
		}
	}

	if mc, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		si.SetMaxClients(int32(mc))
	}

	if gt, err := msg.ReadByte(); err != nil {
		return nil, err
	} else {
		si.SetGameType(int32(gt))
	}

	lname, err := msg.ReadString()
	if err != nil {
		return nil, err
	}
	si.SetLevelName(lname)

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
	si.SetModelPrecache(modelNames)

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
	si.SetSoundPrecache(sounds)

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
	case protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
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
	eu.SetEntity(num)
	eu.SetLerpMoveStep(bits&U_STEP != 0)

	if bits&U_MODEL != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.SetModel(int32(v))
		}
	}
	if bits&U_FRAME != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.SetFrame(int32(v))
		}
	}
	if bits&U_COLORMAP != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.SetColorMap(int32(v))
		}
	}
	if bits&U_SKIN != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.SetSkin(int32(v))
		}
	}
	if bits&U_EFFECTS != 0 {
		if v, err := msg.ReadByte(); err != nil {
			return nil, err
		} else {
			eu.SetEffects(int32(v))
		}
	}

	// Why optional in each component? It is near impossible to have only one component changed.
	// I guess changing to per component is a one way street :(
	if bits&U_ORIGIN1 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetOriginX(v)
		}
	}
	if bits&U_ANGLE1 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetAngleX(v)
		}
	}
	if bits&U_ORIGIN2 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetOriginY(v)
		}
	}
	if bits&U_ANGLE2 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetAngleY(v)
		}
	}
	if bits&U_ORIGIN3 != 0 {
		if v, err := msg.ReadCoord(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetOriginZ(v)
		}
	}
	if bits&U_ANGLE3 != 0 {
		if v, err := msg.ReadAngle(protocolFlags); err != nil {
			return nil, err
		} else {
			eu.SetAngleZ(v)
		}
	}

	switch pcol {
	case protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
		if bits&U_ALPHA != 0 {
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.SetAlpha(int32(v))
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
				eu.SetFrame(eu.GetFrame() | int32(v)<<8)
			}
		}
		if bits&U_MODEL2 != 0 {
			// Can only be set if U_MODEL is set as well
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.SetModel(eu.GetModel() | int32(v)<<8)
			}
		}
		if bits&U_LERPFINISH != 0 {
			if v, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				eu.SetLerpFinish(int32(v))
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
			switch {
			case b < 0:
				eu.SetAlpha(0)
			case b == 0, b >= 255:
				eu.SetAlpha(255)
			default:
				eu.SetAlpha(int32(b))
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
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					EntityUpdate: eu,
				}.Build()))
			}
			continue
		}

		switch cmd {
		default:
			return nil, fmt.Errorf("Illegible server message, previous was %s", svc_strings[lastcmd])

		case Nop:
			sm.SetCmds(append(sm.GetCmds(), &protos.SCmd{}))
		case Time:
			if t, err := msg.ReadFloat32(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Time: proto.Float32(t),
				}.Build()))
			}
		case ClientData:
			if cdp, err := parseClientData(msg); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					ClientData: cdp,
				}.Build()))
			}
		case Version:
			if i, err := msg.ReadInt32(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Version: proto.Int32(int32(i)),
				}.Build()))
			}
		case Disconnect:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				Disconnect: proto.Bool(true),
			}.Build()))
		case Print:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Print: proto.String(s),
				}.Build()))
			}
		case CenterPrint:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					CenterPrint: proto.String(s),
				}.Build()))
			}
		case StuffText:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					StuffText: proto.String(s),
				}.Build()))
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
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Damage: protos.Damage_builder{
						Armor:    int32(data.Armor),
						Blood:    int32(data.Blood),
						Position: pos,
					}.Build(),
				}.Build()))
			}
		case ServerInfo:
			if si, err := parseServerInfo(msg); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					ServerInfo: si,
				}.Build()))
			}
		case SetAngle:
			if a, err := readAngle(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SetAngle: a,
				}.Build()))
			}
		case SetView:
			if ve, err := msg.ReadUint16(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SetViewEntity: proto.Int32(int32(ve)),
				}.Build()))
			}
		case LightStyle:
			cmd := &protos.LightStyle{}
			if idx, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				cmd.SetIdx(int32(idx))
			}
			if str, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				cmd.SetNewStyle(str)
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				LightStyle: cmd,
			}.Build()))
		case Sound:
			if spp, err := parseSoundMessage(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Sound: spp,
				}.Build()))
			}
		case StopSound:
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					StopSound: proto.Int32(int32(i)),
				}.Build()))
			}
		case UpdateName:
			un := &protos.UpdateName{}
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				un.SetPlayer(int32(i))
			}
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				un.SetNewName(s)
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				UpdateName: un,
			}.Build()))
		case UpdateFrags:
			var data struct {
				Player   byte
				NewFrags int16
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			uf := protos.UpdateFrags_builder{
				Player:   int32(data.Player),
				NewFrags: int32(data.NewFrags),
			}.Build()
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				UpdateFrags: uf,
			}.Build()))
		case UpdateColors:
			var data struct {
				Player   byte
				NewColor byte
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			uc := protos.UpdateColors_builder{
				Player:   int32(data.Player),
				NewColor: int32(data.NewColor),
			}.Build()
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				UpdateColors: uc,
			}.Build()))
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
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				Particle: protos.Particle_builder{
					Origin: org,
					Direction: protos.Coord_builder{
						X: float32(data.Dir[0]) * (1.0 / 16),
						Y: float32(data.Dir[1]) * (1.0 / 16),
						Z: float32(data.Dir[2]) * (1.0 / 16),
					}.Build(),
					Count: count,
					Color: int32(data.Color),
				}.Build(),
			}.Build()))
		case SpawnBaseline:
			eb := &protos.EntityBaseline{}
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				eb.SetIndex(int32(i))
			}
			if pb, err := parseBaseline(msg, protocolFlags, 1); err != nil {
				return nil, err
			} else {
				eb.SetBaseline(pb)
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				SpawnBaseline: eb,
			}.Build()))

		case SpawnStatic:
			if pb, err := parseBaseline(msg, protocolFlags, 1); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SpawnStatic: pb,
				}.Build()))
			}

		case TempEntity:
			if tep, err := parseTempEntity(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					TempEntity: tep,
				}.Build()))
			}
		case SetPause:
			// this byte was used to pause cd audio, other pause as well?
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SetPause: proto.Bool(i != 0),
				}.Build()))
			}
		case SignonNum:
			if i, err := msg.ReadByte(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SignonNum: proto.Int32(int32(i)),
				}.Build()))
			}
		case KilledMonster:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				KilledMonster: &protos.Empty{},
			}.Build()))
		case FoundSecret:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				FoundSecret: &protos.Empty{},
			}.Build()))
		case UpdateStat:
			var data struct {
				Stat byte
				Val  int32
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				UpdateStat: protos.UpdateStat_builder{
					Stat:  int32(data.Stat),
					Value: int32(data.Val),
				}.Build(),
			}.Build()))
		case SpawnStaticSound:
			ss := &protos.StaticSound{}
			if org, err := readCoord(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				ss.SetOrigin(org)
			}
			var data struct {
				Num uint8
				Vol uint8
				Att uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			ss.SetIndex(int32(data.Num))
			ss.SetVolume(int32(data.Vol))
			ss.SetAttenuation(int32(data.Att))
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				SpawnStaticSound: ss,
			}.Build()))
		case CDTrack:
			var data struct {
				TrackNumber uint8
				Loop        uint8 // was for cl.looptrack
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				CdTrack: protos.CDTrack_builder{
					TrackNumber: int32(data.TrackNumber),
					LoopTrack:   int32(data.Loop),
				}.Build(),
			}.Build()))
		case Intermission:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				Intermission: &protos.Empty{},
			}.Build()))
		case Finale:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Finale: proto.String(s),
				}.Build()))
			}
		case Cutscene:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Cutscene: proto.String(s),
				}.Build()))
			}
		case SellScreen:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				SellScreen: &protos.Empty{},
			}.Build()))
		case Skybox:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Skybox: proto.String(s),
				}.Build()))
			}
		case BF:
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				BackgroundFlash: &protos.Empty{},
			}.Build()))
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
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				Fog: protos.Fog_builder{
					Density: float32(data.Density) / 255.0,
					Red:     float32(data.Red) / 255.0,
					Green:   float32(data.Green) / 255.0,
					Blue:    float32(data.Blue) / 255.0,
					Time:    float32(data.Time) / 100.0,
				}.Build(),
			}.Build()))
		case SpawnBaseline2:
			sb := &protos.EntityBaseline{}
			if i, err := msg.ReadInt16(); err != nil {
				return nil, err
			} else {
				sb.SetIndex(int32(i))
			}
			if pb, err := parseBaseline(msg, protocolFlags, 2); err != nil {
				return nil, err
			} else {
				sb.SetBaseline(pb)
			}
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				SpawnBaseline: sb,
			}.Build()))

		case SpawnStatic2:
			if pb, err := parseBaseline(msg, protocolFlags, 2); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					SpawnStatic: pb,
				}.Build()))
			}
		case SpawnStaticSound2:
			ss := &protos.StaticSound{}
			if org, err := readCoord(msg, protocolFlags); err != nil {
				return nil, err
			} else {
				ss.SetOrigin(org)
			}
			var data struct {
				Num uint16
				Vol uint8
				Att uint8
			}
			if err := msg.Read(&data); err != nil {
				return nil, err
			}
			ss.SetIndex(int32(data.Num))
			ss.SetVolume(int32(data.Vol))
			ss.SetAttenuation(int32(data.Att))
			sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
				SpawnStaticSound: ss,
			}.Build()))
		case Achievement:
			if s, err := msg.ReadString(); err != nil {
				return nil, err
			} else {
				sm.SetCmds(append(sm.GetCmds(), protos.SCmd_builder{
					Achievement: proto.String(s),
				}.Build()))
			}
		}
		lastcmd = cmd
	}
}

func writeCoord(c *protos.Coord, protocolFlags uint32, m *net.Message) {
	m.WriteCoord(c.GetX(), protocolFlags)
	m.WriteCoord(c.GetY(), protocolFlags)
	m.WriteCoord(c.GetZ(), protocolFlags)
}

func writeAngle(a *protos.Coord, protocolFlags uint32, m *net.Message) {
	m.WriteAngle(a.GetX(), protocolFlags)
	m.WriteAngle(a.GetY(), protocolFlags)
	m.WriteAngle(a.GetZ(), protocolFlags)
}

func WriteParticle(p *protos.Particle, protocolFlags uint32, m *net.Message) {
	m.WriteByte(Particle)
	writeCoord(p.GetOrigin(), protocolFlags, m)
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
	m.WriteChar(df(p.GetDirection().GetX()))
	m.WriteChar(df(p.GetDirection().GetY()))
	m.WriteChar(df(p.GetDirection().GetZ()))
	m.WriteByte(int(p.GetCount()))
	m.WriteByte(int(p.GetColor()))
}

func WriteSound(s *protos.Sound, pcol int, flags uint32, m *net.Message) {
	fieldMask := 0
	if s.GetEntity() >= 8192 {
		if pcol == protocol.NetQuake {
			return
		}
		fieldMask |= SoundLargeEntity
	}
	if s.GetSoundNum() >= 256 || s.GetChannel() >= 8 {
		if pcol == protocol.NetQuake {
			return
		}
		fieldMask |= SoundLargeSound
	}
	if s.HasVolume() {
		fieldMask |= SoundVolume
	}
	if s.HasAttenuation() {
		fieldMask |= SoundAttenuation
	}
	m.WriteByte(Sound)
	m.WriteByte(fieldMask)
	if s.HasVolume() {
		m.WriteByte(int(s.GetVolume()))
	}
	if s.HasAttenuation() {
		m.WriteByte(int(s.GetAttenuation()))
	}
	if fieldMask&SoundLargeEntity != 0 {
		m.WriteShort(int(s.GetEntity()))
		m.WriteByte(int(s.GetChannel()))
	} else {
		m.WriteShort(int(s.GetEntity()<<3 | s.GetChannel()))
	}
	if fieldMask&SoundLargeSound != 0 {
		m.WriteShort(int(s.GetSoundNum()))
	} else {
		m.WriteByte(int(s.GetSoundNum()))
	}
	writeCoord(s.GetOrigin(), flags, m)
}

func WriteDamage(d *protos.Damage, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(Damage)
	m.WriteByte(int(d.GetArmor()))
	m.WriteByte(int(d.GetBlood()))
	writeCoord(d.GetPosition(), flags, m)
}

func WriteSetAngle(a *protos.Coord, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(SetAngle)
	writeAngle(a, flags, m)
}

func WriteClientData(cd *protos.ClientData, pcol int, flags uint32, m *net.Message) {
	bits := 0
	if cd.HasViewHeight() {
		bits |= SU_VIEWHEIGHT
	}
	if cd.GetIdealPitch() != 0 {
		bits |= SU_IDEALPITCH
	}
	bits |= SU_ITEMS
	bits |= SU_WEAPON
	if cd.GetOnGround() {
		bits |= SU_ONGROUND
	}
	if cd.GetInWater() {
		bits |= SU_INWATER
	}
	if cd.GetPunchAngle().GetX() != 0 {
		bits |= SU_PUNCH1
	}
	if cd.GetPunchAngle().GetY() != 0 {
		bits |= SU_PUNCH2
	}
	if cd.GetPunchAngle().GetZ() != 0 {
		bits |= SU_PUNCH3
	}
	if cd.GetVelocity().GetX() != 0 {
		bits |= SU_VELOCITY1
	}
	if cd.GetVelocity().GetY() != 0 {
		bits |= SU_VELOCITY2
	}
	if cd.GetVelocity().GetZ() != 0 {
		bits |= SU_VELOCITY3
	}
	if cd.GetWeaponFrame() != 0 {
		bits |= SU_WEAPONFRAME
	}
	if cd.GetArmor() != 0 {
		bits |= SU_ARMOR
	}

	if pcol != protocol.NetQuake {
		if (cd.GetWeapon() & 0xFF00) != 0 {
			bits |= SU_WEAPON2
		}
		if (cd.GetArmor() & 0xFF00) != 0 {
			bits |= SU_ARMOR2
		}
		if (cd.GetAmmo() & 0xFF00) != 0 {
			bits |= SU_AMMO2
		}
		if (cd.GetShells() & 0xFF00) != 0 {
			bits |= SU_SHELLS2
		}
		if (cd.GetNails() & 0xFF00) != 0 {
			bits |= SU_NAILS2
		}
		if (cd.GetRockets() & 0xFF00) != 0 {
			bits |= SU_ROCKETS2
		}
		if (cd.GetCells() & 0xFF00) != 0 {
			bits |= SU_CELLS2
		}
		if (bits&SU_WEAPONFRAME != 0) &&
			(cd.GetWeaponFrame()&0xFF00) != 0 {
			bits |= SU_WEAPONFRAME2
		}
		if cd.GetWeaponAlpha() != 0 {
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
		m.WriteChar(int(cd.GetViewHeight()))
	}
	if (bits & SU_IDEALPITCH) != 0 {
		m.WriteChar(int(cd.GetIdealPitch()))
	}
	if (bits & SU_PUNCH1) != 0 {
		m.WriteChar(int(cd.GetPunchAngle().GetX()))
	}
	if (bits & SU_VELOCITY1) != 0 {
		m.WriteChar(int(cd.GetVelocity().GetX()))
	}
	if (bits & SU_PUNCH2) != 0 {
		m.WriteChar(int(cd.GetPunchAngle().GetY()))
	}
	if (bits & SU_VELOCITY2) != 0 {
		m.WriteChar(int(cd.GetVelocity().GetY()))
	}
	if (bits & SU_PUNCH3) != 0 {
		m.WriteChar(int(cd.GetPunchAngle().GetZ()))
	}
	if (bits & SU_VELOCITY3) != 0 {
		m.WriteChar(int(cd.GetVelocity().GetZ()))
	}
	m.WriteLong(int(cd.GetItems()))

	if (bits & SU_WEAPONFRAME) != 0 {
		m.WriteByte(int(cd.GetWeaponFrame()))
	}
	if (bits & SU_ARMOR) != 0 {
		m.WriteByte(int(cd.GetArmor()))
	}
	m.WriteByte(int(cd.GetWeapon()))
	m.WriteShort(int(cd.GetHealth()))
	m.WriteByte(int(cd.GetAmmo()))
	m.WriteByte(int(cd.GetShells()))
	m.WriteByte(int(cd.GetNails()))
	m.WriteByte(int(cd.GetRockets()))
	m.WriteByte(int(cd.GetCells()))
	m.WriteByte(int(cd.GetActiveWeapon()))

	if (bits & SU_WEAPON2) != 0 {
		m.WriteByte(int(cd.GetWeapon() >> 8))
	}
	if (bits & SU_ARMOR2) != 0 {
		m.WriteByte(int(cd.GetArmor()) >> 8)
	}
	if (bits & SU_AMMO2) != 0 {
		m.WriteByte(int(cd.GetAmmo()) >> 8)
	}
	if (bits & SU_SHELLS2) != 0 {
		m.WriteByte(int(cd.GetShells()) >> 8)
	}
	if (bits & SU_NAILS2) != 0 {
		m.WriteByte(int(cd.GetNails()) >> 8)
	}
	if (bits & SU_ROCKETS2) != 0 {
		m.WriteByte(int(cd.GetRockets()) >> 8)
	}
	if (bits & SU_CELLS2) != 0 {
		m.WriteByte(int(cd.GetCells()) >> 8)
	}
	if (bits & SU_WEAPONFRAME2) != 0 {
		m.WriteByte(int(cd.GetWeaponFrame()) >> 8)
	}
	if (bits & SU_WEAPONALPHA) != 0 {
		m.WriteByte(int(cd.GetWeaponAlpha()))
	}
}

func WriteTime(t float32, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(Time)
	m.WriteFloat(t)
}

func WriteUpdateFrags(uf *protos.UpdateFrags, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(UpdateFrags)
	m.WriteByte(int(uf.GetPlayer()))
	m.WriteShort(int(uf.GetNewFrags()))
}

func WriteEntityUpdate(eu *protos.EntityUpdate, pcol int, flags uint32, m *net.Message) {
	bits := 0
	if eu.HasOriginX() {
		bits |= U_ORIGIN1
	}
	if eu.HasOriginY() {
		bits |= U_ORIGIN2
	}
	if eu.HasOriginZ() {
		bits |= U_ORIGIN3
	}
	if eu.HasAngleX() {
		bits |= U_ANGLE1
	}
	if eu.HasAngleY() {
		bits |= U_ANGLE2
	}
	if eu.HasAngleZ() {
		bits |= U_ANGLE3
	}
	if eu.GetLerpMoveStep() {
		bits |= U_STEP // don't mess up the step animation
	}
	if eu.HasColorMap() {
		bits |= U_COLORMAP
	}
	if eu.HasSkin() {
		bits |= U_SKIN
	}
	if eu.HasFrame() {
		bits |= U_FRAME
	}
	if eu.GetEffects() != 0 {
		bits |= U_EFFECTS
	}
	if eu.HasModel() {
		bits |= U_MODEL
	}

	if pcol != protocol.NetQuake {
		if eu.HasAlpha() {
			bits |= U_ALPHA
		}
		if eu.HasFrame() &&
			eu.GetFrame()&0xFF00 != 0 {
			bits |= U_FRAME2
		}
		if eu.HasModel() &&
			eu.GetModel()&0xFF00 != 0 {
			bits |= U_MODEL2
		}
		if eu.HasLerpFinish() {
			bits |= U_LERPFINISH
		}
		if bits >= 65536 {
			bits |= U_EXTEND1
		}
		if bits >= 16777216 {
			bits |= U_EXTEND2
		}
	}

	if eu.GetEntity() >= 256 {
		bits |= U_LONGENTITY
	}

	if bits >= 256 {
		bits |= U_MOREBITS
	}

	m.WriteByte(bits | U_SIGNAL)

	if bits&U_MOREBITS != 0 {
		m.WriteByte(bits >> 8)
	}

	if bits&U_EXTEND1 != 0 {
		m.WriteByte(bits >> 16)
	}
	if bits&U_EXTEND2 != 0 {
		m.WriteByte(bits >> 24)
	}

	if bits&U_LONGENTITY != 0 {
		m.WriteShort(int(eu.GetEntity()))
	} else {
		m.WriteByte(int(eu.GetEntity()))
	}

	if eu.HasModel() {
		m.WriteByte(int(eu.GetModel()))
	}
	if eu.HasFrame() {
		m.WriteByte(int(eu.GetFrame()))
	}
	if eu.HasColorMap() {
		m.WriteByte(int(eu.GetColorMap()))
	}
	if eu.HasSkin() {
		m.WriteByte(int(eu.GetSkin()))
	}
	if eu.GetEffects() != 0 {
		m.WriteByte(int(eu.GetEffects()))
	}
	if eu.HasOriginX() {
		m.WriteCoord(eu.GetOriginX(), flags)
	}
	if eu.HasAngleX() {
		m.WriteAngle(eu.GetAngleX(), flags)
	}
	if eu.HasOriginY() {
		m.WriteCoord(eu.GetOriginY(), flags)
	}
	if eu.HasAngleY() {
		m.WriteAngle(eu.GetAngleY(), flags)
	}
	if eu.HasOriginZ() {
		m.WriteCoord(eu.GetOriginZ(), flags)
	}
	if eu.HasAngleZ() {
		m.WriteAngle(eu.GetAngleZ(), flags)
	}

	if bits&U_ALPHA != 0 {
		m.WriteByte(int(eu.GetAlpha()))
	}
	if bits&U_FRAME2 != 0 {
		m.WriteByte(int(eu.GetFrame()) >> 8)
	}
	if bits&U_MODEL2 != 0 {
		m.WriteByte(int(eu.GetModel()) >> 8)
	}
	if bits&U_LERPFINISH != 0 {
		m.WriteByte(int(eu.GetLerpFinish()))
	}
}

func WriteUpdateColors(uc *protos.UpdateColors, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(UpdateColors)
	m.WriteByte(int(uc.GetPlayer()))
	m.WriteByte(int(uc.GetNewColor()))
}

func WriteUpdateName(un *protos.UpdateName, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(UpdateName)
	m.WriteByte(int(un.GetPlayer()))
	m.WriteString(un.GetNewName())
}

func WriteSetPause(p bool, pcol int, flags uint32, m *net.Message) {
	m.WriteByte(SetPause)
	m.WriteByte(func() int {
		if p {
			return 1
		}
		return 0
	}())
}
