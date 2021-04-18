// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//void CL_ParseUpdate(int num, int modNum);
//void CL_ClearState(void);
//void CLPrecacheModelClear(void);
//void FinishCL_ParseServerInfo(void);
import "C"

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/therjak/goquake/bsp"
	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/execute"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/mdl"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/protocol"
	svc "github.com/therjak/goquake/protocol/server"
	"github.com/therjak/goquake/protos"
	"github.com/therjak/goquake/snd"
	"github.com/therjak/goquake/spr"
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
	}
)

func CL_ParseBaseline(pb *protos.Baseline, e *Entity) {
	e.Baseline = state{
		ModelIndex: int(pb.GetModelIndex()),
		Frame:      int(pb.GetFrame()),
		ColorMap:   int(pb.GetColorMap()),
		Skin:       int(pb.GetSkin()),
		Origin:     v3FC(pb.GetOrigin()),
		Angles:     v3FC(pb.GetAngles()),
		Alpha:      byte(pb.GetAlpha()),
	}
}

func parse3Coord() (vec.Vec3, error) {
	x, err := cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	y, err := cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	z, err := cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	return vec.Vec3{x, y, z}, nil
}

func parse3Angle() (vec.Vec3, error) {
	x, err := cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	y, err := cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	z, err := cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		return vec.Vec3{}, err
	}
	return vec.Vec3{x, y, z}, nil
}

func CL_ParseServerMessage() {
	// if recording demos, copy the message out
	switch cvars.ClientShowNet.String() {
	case "1":
		// This is not known
		// conlog.Printf("%d ", CL_MSG_GetCurSize());
	case "2":
		conlog.Printf("------------------\n")
	}

	cl.onGround = false
	// unless the server says otherwise parse the message

	lastcmd := byte(0)
	for {
		if cls.msgBadRead {
			fmt.Printf("Bad server message\n")
			HostError("CL_ParseServerMessage: Bad server message")
		}

		if cls.inMessage.Len() == 0 {
			if cvars.ClientShowNet.String() == "2" {
				// conlog.Printf("%3d:%s\n", CL_MSG_ReadCount() - 1, "END OF MESSAGE");
			}
			// end of message
			return
		}
		cmd, _ := cls.inMessage.ReadByte()

		// if the high bit of the command byte is set, it is a fast update
		if cmd&svc.U_SIGNAL != 0 {
			if cvars.ClientShowNet.String() == "2" {
				// conlog.Printf("%3i:%s\n", CL_MSG_ReadCount() - 1, "fast update");
			}
			eu, err := svc.ParseEntityUpdate(cls.inMessage, cl.protocol, cl.protocolFlags, cmd&127)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.ParseEntityUpdate(eu)
			continue
		}

		if cvars.ClientShowNet.String() == "2" {
			// conlog.Printf("%3i:%s\n", CL_MSG_ReadCount() - 1, svc_strings[cmd]);
		}

		// other commands
		switch cmd {
		default:
			HostError("Illegible server message, previous was %s", svc_strings[lastcmd])

		case svc.Nop:
			//	conlog.Printf("svc_nop\n");

		case svc.Time:
			cl.messageTimeOld = cl.messageTime
			t, err := cls.inMessage.ReadFloat32()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.messageTime = float64(t)

		case svc.ClientData:
			cdp, err := svc.ParseClientData(cls.inMessage)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.parseClientData(cdp)

		case svc.Version:
			i, err := cls.inMessage.ReadInt32()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			switch i {
			case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
			default:
				HostError("Server returned version %d, not %d or %d or %d or %d", i,
					protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
			}
			cl.protocol = int(i)

		case svc.Disconnect:
			HostEndGame("Server disconnected\n")

		case svc.Print:
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			conlog.Printf("%s", s)

		case svc.CenterPrint:
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			screen.CenterPrint(s)
			console.CenterPrint(s)

		case svc.StuffText:
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cbuf.AddText(s)

		case svc.Damage:
			armor, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			blood, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			pos, err := parse3Coord()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.parseDamage(int(armor), int(blood), pos)

		case svc.ServerInfo:
			si, err := svc.ParseServerInfo(cls.inMessage)
			if err != nil {
				cls.msgBadRead = true
				conlog.Printf("\nParseServerInfo: %v")
				continue
			}
			CL_ParseServerInfo(si)
			screen.recalcViewRect = true // leave intermission full screen

		case svc.SetAngle:
			a, err := parse3Angle()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.pitch = a[0]
			cl.yaw = a[1]
			cl.roll = a[2]

		case svc.SetView:
			ve, err := cls.inMessage.ReadUint16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.viewentity = int(ve)

		case svc.LightStyle:
			idx, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			str, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			err = readLightStyle(idx, str)
			if err != nil {
				Error("svc_lightstyle: %v", err)
			}

		case svc.Sound:
			spp, err := svc.ParseSoundMessage(cls.inMessage, cl.protocolFlags)
			if err != nil {
				HostError("%v", err)
			}
			err = CL_ParseStartSoundPacket(spp)
			if err != nil {
				HostError("%v", err)
			}

		case svc.StopSound:
			i, err := cls.inMessage.ReadInt16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			snd.Stop(int(i)>>3, int(i)&7)

		case svc.UpdateName:
			statusbar.MarkChanged()
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			if int(i) >= cl.maxClients {
				HostError("CL_ParseServerMessage: svc_updatename > MAX_SCOREBOARD")
			}
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.scores[i].name = s

		case svc.UpdateFrags:
			statusbar.MarkChanged()
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			if int(i) >= cl.maxClients {
				HostError("CL_ParseServerMessage: svc_updatefrags > MAX_SCOREBOARD")
			}
			f, err := cls.inMessage.ReadInt16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.scores[i].frags = int(f)

		case svc.UpdateColors:
			statusbar.MarkChanged()
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			if int(i) >= cl.maxClients {
				HostError("CL_ParseServerMessage: svc_updatecolors > MAX_SCOREBOARD")
			}
			c, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.scores[i].topColor = int((c & 0xf0) >> 4)
			cl.scores[i].bottomColor = int(c & 0x0f)
			CL_NewTranslation(int(i))

		case svc.Particle:
			var dir vec.Vec3
			org, err := parse3Coord()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			var data struct {
				Dir   [3]int8
				Count uint8
				Color uint8
			}
			err = cls.inMessage.Read(&data)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			dir[0] = float32(data.Dir[0]) * (1.0 / 16)
			dir[1] = float32(data.Dir[1]) * (1.0 / 16)
			dir[2] = float32(data.Dir[2]) * (1.0 / 16)
			count := int(data.Count)
			color := int(data.Color)
			if count == 255 {
				count = 1024
			}
			particlesRunEffect(org, dir, color, count, float32(cl.time))

		case svc.SpawnBaseline:
			i, err := cls.inMessage.ReadInt16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			// force cl.num_entities up
			e := cl.GetOrCreateEntity(int(i))

			pb, err := svc.ParseBaseline(cls.inMessage, cl.protocolFlags, 1)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			CL_ParseBaseline(pb, e)

		case svc.SpawnStatic:
			pb, err := svc.ParseBaseline(cls.inMessage, cl.protocolFlags, 1)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.ParseStatic(pb)

		case svc.TempEntity:
			tep, err := svc.ParseTempEntity(cls.inMessage, cl.protocolFlags)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cls.parseTempEntity(tep)

		case svc.SetPause:
			// this byte was used to pause cd audio, other pause as well?
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.paused = (i != 0)

		case svc.SignonNum:
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			if int(i) <= cls.signon {
				HostError("Received signon %d when at %d", i, cls.signon)
			}
			cls.signon = int(i)
			// if signonnum==2, signon packet has been fully parsed, so
			// check for excessive static entities and entity fragments
			if i == 2 {
				if len(cl.staticEntities) > 128 {
					conlog.DWarning("%d static entities exceeds standard limit of 128.\n",
						len(cl.staticEntities))
				}
			}
			CL_SignonReply()

		case svc.KilledMonster:
			cl.stats.monsters++

		case svc.FoundSecret:
			cl.stats.secrets++

		case svc.UpdateStat:
			i, err := cls.inMessage.ReadByte()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			v, err := cls.inMessage.ReadInt32()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			//if i < 0 || i >= MAX_CL_STATS {
			//	Go_Error_I("svc_updatestat: %v is invalid", i)
			//}
			// Only used for STAT_TOTALSECRETS, STAT_TOTALMONSTERS, STAT_SECRETS,
			// STAT_MONSTERS
			cl_setStats(int(i), int(v))

		case svc.SpawnStaticSound:
			org, err := parse3Coord()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			var data struct {
				Num uint8
				Vol uint8
				Att uint8
			}
			err = cls.inMessage.Read(&data)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			snd.Start(0, 0, cl.soundPrecache[data.Num-1], org, float32(data.Vol)/255, float32(data.Att)/64, loopingSound)

		case svc.CDTrack:
			// nobody uses cds anyway. just ignore
			var data struct {
				TrackNumber uint8
				Loop        uint8 // was for cl.looptrack
			}
			err := cls.inMessage.Read(&data)
			if err != nil {
				cls.msgBadRead = true
				continue
			}

		case svc.Intermission:
			cl.intermission = 1
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen

		case svc.Finale:
			cl.intermission = 2
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			screen.CenterPrint(s)
			console.CenterPrint(s)

		case svc.Cutscene:
			cl.intermission = 3
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			screen.CenterPrint(s)
			console.CenterPrint(s)

		case svc.SellScreen:
			execute.Execute("help", execute.Command, sv_player)

		case svc.Skybox:
			s, err := cls.inMessage.ReadString()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			sky.LoadBox(s)

		case svc.BF:
			execute.Execute("bf", execute.Command, sv_player)

		case svc.Fog:
			{
				var data struct {
					Density uint8
					Red     uint8
					Green   uint8
					Blue    uint8
					Time    uint8
				}
				err := cls.inMessage.Read(&data)
				if err != nil {
					cls.msgBadRead = true
					continue
				}
				density := float32(data.Density) / 255.0
				red := float32(data.Red) / 255.0
				green := float32(data.Green) / 255.0
				blue := float32(data.Blue) / 255.0
				time := float64(data.Time) / 100.0
				if time < 0 {
					time = 0
				}
				fog.Update(density, red, green, blue, time)
			}
		case svc.SpawnBaseline2:
			i, err := cls.inMessage.ReadInt16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			// force cl.num_entities up
			e := cl.GetOrCreateEntity(int(i))

			pb, err := svc.ParseBaseline(cls.inMessage, cl.protocolFlags, 2)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			CL_ParseBaseline(pb, e)

		case svc.SpawnStatic2:
			pb, err := svc.ParseBaseline(cls.inMessage, cl.protocolFlags, 2)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			cl.ParseStatic(pb)

		case svc.SpawnStaticSound2:
			org, err := parse3Coord()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			var data struct {
				Num uint16
				Vol uint8
				Att uint8
			}
			err = cls.inMessage.Read(&data)
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			snd.Start(0, 0, cl.soundPrecache[data.Num-1], org, float32(data.Vol)/255, float32(data.Att)/64, loopingSound)
		}

		lastcmd = cmd
	}
}

func CL_ParseServerInfo(si *protos.ServerInfo) {
	// protos.ServerInfo
	conlog.DPrintf("Serverinfo packet received.\n")

	// bring up loading plaque for map changes within a demo.
	// it will be hidden in CL_SignonReply.
	if cls.demoPlayback {
		screen.BeginLoadingPlaque()
	}

	C.CL_ClearState()

	cl.protocol = int(si.Protocol)
	cl.protocolFlags = uint32(si.Flags)

	if cl.protocol == protocol.RMQ {
		const supportedflags uint32 = protocol.PRFL_SHORTANGLE |
			protocol.PRFL_FLOATANGLE |
			protocol.PRFL_24BITCOORD |
			protocol.PRFL_FLOATCOORD |
			protocol.PRFL_EDICTSCALE |
			protocol.PRFL_INT32COORD

		if cl.protocolFlags&^supportedflags != 0 {
			conlog.Warning("PROTOCOL_RMQ protocolflags %d contains unsupported flags\n", cl.protocolFlags)
		}
	}

	if si.MaxClients < 1 || si.MaxClients > 16 {
		HostError("Bad maxclients (%d) from server", si.MaxClients)
	}
	cl.maxClients = int(si.MaxClients)
	cl.scores = make([]score, cl.maxClients)
	cl.gameType = int(si.GameType)
	cl.levelName = si.LevelName

	// seperate the printfs so the server message can have a color
	console.printBar()
	conlog.Printf("%c%s\n", 2, cl.levelName)

	conlog.Printf("Using protocol %d\n", cl.protocol)

	cl.modelPrecache = cl.modelPrecache[:]
	if len(si.ModelPrecache) >= 2048 {
		HostError("Server sent too many model precaches")
	}
	if len(si.ModelPrecache) >= 256 {
		conlog.DWarning("%d models exceeds standard limit of 256.\n", len(si.ModelPrecache))
	}

	cl.soundPrecache = cl.soundPrecache[:0]
	if len(si.SoundPrecache) >= 2048 {
		HostError("Server sent too many sound precaches")
	}
	if len(si.SoundPrecache) >= 256 {
		conlog.DWarning("%d sounds exceeds standard limit of 256.\n", len(si.SoundPrecache))
	}

	mapName := si.ModelPrecache[0]
	// now we try to load everything else until a cache allocation fails
	cl.mapName = strings.TrimSuffix(filepath.Base(mapName), filepath.Ext(mapName))

	C.CLPrecacheModelClear()
	for i, mn := range si.ModelPrecache {
		m, ok := models[mn]
		CLPrecacheModel(mn, i+1) // keep C side happy
		if !ok {
			loadModel(mn)
			m, ok = models[mn]
			if !ok {
				HostError("Model %s not found", mn)
			}
		}
		cl.modelPrecache = append(cl.modelPrecache, m)
		CL_KeepaliveMessage()
	}

	for _, s := range si.SoundPrecache {
		sfx := snd.PrecacheSound(s)
		cl.soundPrecache = append(cl.soundPrecache, sfx)
		CL_KeepaliveMessage()
	}

	// TODO: clean this stuff up
	cl.worldModel, _ = cl.modelPrecache[0].(*bsp.Model)
	for _, t := range cl.worldModel.Textures {
		if t != nil && strings.HasPrefix(t.Name(), "sky") {
			sky.LoadTexture(t.Data, t.Name(), cl.mapName)
		}
	}

	C.FinishCL_ParseServerInfo()

	// we don't consider identical messages to be duplicates if the map has changed in between
	console.lastCenter = ""
}

//ParseEntityUpdate parses an entity update message from the server
//If an entities model or origin changes from frame to frame, it must be
//relinked. Other attributes can change without relinking.
func (c *Client) ParseEntityUpdate(eu *protos.EntityUpdate) {
	if cls.signon == 3 {
		// first update is the final signon stage
		cls.signon = 4
		CL_SignonReply()
	}
	num := int(eu.Entity)
	e := c.GetOrCreateEntity(num)
	e.SyncC()
	forceLink := e.MsgTime != c.messageTimeOld

	if e.MsgTime+0.2 < c.messageTime {
		// most entities think every 0.1s, if we missed one we would be lerping from the wrong frame
		e.LerpFlags |= lerpResetAnim
	}
	if eu.LerpMoveStep {
		e.ForceLink = true
		e.LerpFlags |= lerpMoveStep
	} else {
		e.LerpFlags &^= lerpMoveStep
	}

	e.MsgTime = c.messageTime
	e.Frame = e.Baseline.Frame
	oldSkinNum := e.SkinNum
	e.SkinNum = e.Baseline.Skin
	// shift known values for interpolation
	e.MsgOrigin[1] = e.MsgOrigin[0]
	e.MsgAngles[1] = e.MsgAngles[0]
	e.MsgOrigin[0] = e.Baseline.Origin
	e.MsgAngles[0] = e.Baseline.Angles
	e.Alpha = e.Baseline.Alpha
	e.SyncBase = 0

	modNum := e.Baseline.ModelIndex
	if eu.Model != nil {
		modNum = int(eu.Model.Value)
	}
	if modNum >= model.MAX_MODELS {
		Error("CL_ParseModel: mad modnum")
	}
	if eu.Frame != nil {
		e.Frame = int(eu.Frame.Value)
	}
	if eu.Skin != nil {
		e.SkinNum = int(eu.Skin.Value)
	}
	if e.SkinNum != oldSkinNum {
		if num > 0 && num <= cl.maxClients {
			// C.R_TranslateNewPlaykerSkin(num - 1)
		}
	}
	e.Effects = int(eu.Effects)
	if eu.OriginX != nil {
		e.MsgOrigin[0][0] = eu.OriginX.Value
	}
	if eu.OriginY != nil {
		e.MsgOrigin[0][1] = eu.OriginY.Value
	}
	if eu.OriginZ != nil {
		e.MsgOrigin[0][2] = eu.OriginZ.Value
	}
	if eu.AngleX != nil {
		e.MsgAngles[0][0] = eu.AngleX.Value
	}
	if eu.AngleY != nil {
		e.MsgAngles[0][1] = eu.AngleY.Value
	}
	if eu.AngleZ != nil {
		e.MsgAngles[0][2] = eu.AngleZ.Value
	}

	if eu.Alpha != nil {
		e.Alpha = byte(eu.Alpha.Value)
	}
	if eu.LerpFinish != nil {
		e.LerpFinish = e.MsgTime + float64(eu.LerpFinish.Value)
		e.LerpFlags |= lerpFinish
	} else {
		e.LerpFlags &^= lerpFinish
	}

	if modNum > 0 && modNum <= len(cl.modelPrecache) {
		model := cl.modelPrecache[modNum-1] // server sends this 1 based, modelPrecache is 0 based
		if model != e.Model {
			e.Model = model
			// automatic animation (torches, etc) can be either all together or randomized
			if model != nil {
				e.SyncBase = 0
				switch m := model.(type) {
				case *mdl.Model:
					if m.SyncType != 0 {
						e.SyncBase = float32(cRand.Uint32n(0x7fff)) / 0x7fff
					}
				case *spr.Model:
					if m.SyncType != 0 {
						e.SyncBase = float32(cRand.Uint32n(0x7fff)) / 0x7fff
					}
				}
			} else {
				// hack to make nil model players work
				forceLink = true
			}
			if num > 0 && num <= cl.maxClients {
				// R_TranslateNewPlayreSkin(num -1)
			}
			// do not lerp animation across model changes
			e.LerpFlags |= lerpResetAnim
		}
	} else {
		conlog.Printf("len(modelPrecache): %v, modNum: %v", len(cl.modelPrecache), modNum)
		e.Model = nil
		forceLink = true
		e.LerpFlags |= lerpResetAnim
	}

	C.CL_ParseUpdate(C.int(num), C.int(modNum))

	if forceLink {
		e.MsgOrigin[1] = e.MsgOrigin[0]
		e.Origin = e.MsgOrigin[0]
		e.MsgAngles[1] = e.MsgAngles[0]
		e.Angles = e.MsgAngles[0]
		e.ForceLink = true
	}
	e.Sync()
}

func (c *Client) ParseStatic(pb *protos.Baseline) {
	ent := c.CreateStaticEntity()
	CL_ParseBaseline(pb, ent)
	// copy it to the current state

	ent.Model = c.modelPrecache[ent.Baseline.ModelIndex-1]
	ent.LerpFlags |= lerpResetAnim // TODO(therjak): shouldn't this be an override instead of an OR?
	ent.Frame = ent.Baseline.Frame
	ent.SkinNum = ent.Baseline.Skin
	ent.Effects = 0
	ent.Alpha = ent.Baseline.Alpha
	ent.Origin = ent.Baseline.Origin
	ent.Angles = ent.Baseline.Angles
	ent.ParseStaticC(ent.Baseline.ModelIndex)
	ent.Sync()

	ent.R_AddEfrags() // clean up after removal of c-efrags
}
