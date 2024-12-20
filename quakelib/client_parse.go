// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"path/filepath"
	"strings"

	"goquake/bsp"
	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/math/vec"
	"goquake/mdl"
	"goquake/model"
	"goquake/protocol"
	"goquake/protos"
	"goquake/spr"
)

func parseBaseline(pb *protos.Baseline, e *Entity) {
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

func CL_ParseServerMessage(pb *protos.ServerMessage) (serverState, error) {
	switch cvars.ClientShowNet.String() {
	case "1", "2":
		conlog.Printf("------------------\n")
	}

	for _, scmd := range pb.GetCmds() {
		switch cmd := scmd.Union.(type) {
		default:
			// nop
		case *protos.SCmd_EntityUpdate:
			cl.ParseEntityUpdate(cmd.EntityUpdate)
		case *protos.SCmd_Time:
			cl.messageTimeOld = cl.messageTime
			cl.messageTime = float64(cmd.Time)
		case *protos.SCmd_ClientData:
			cl.parseClientData(cmd.ClientData)
		case *protos.SCmd_Version:
			switch cmd.Version {
			case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
				cl.protocol = int(cmd.Version)
			default:
				return serverRunning, fmt.Errorf("Server returned version %d, not %d or %d or %d or %d", cmd.Version,
					protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
			}
		case *protos.SCmd_Disconnect:
			if err := handleServerDisconnected("Server disconnected\n"); err != nil {
				return serverDisconnected, err
			}
			return serverDisconnected, nil
		case *protos.SCmd_Print:
			conlog.Printf("%s", cmd.Print)
		case *protos.SCmd_CenterPrint:
			screen.CenterPrint(cmd.CenterPrint)
			console.CenterPrint(cmd.CenterPrint)
		case *protos.SCmd_StuffText:
			cbuf.AddText(cmd.StuffText)
		case *protos.SCmd_Damage:
			d := cmd.Damage
			pos := d.Position
			cl.parseDamage(int(d.GetArmor()), int(d.GetBlood()), vec.Vec3{
				pos.GetX(), pos.GetY(), pos.GetZ(),
			})
		case *protos.SCmd_ServerInfo:
			if err := CL_ParseServerInfo(cmd.ServerInfo); err != nil {
				return serverRunning, err
			}
			screen.recalcViewRect = true // leave intermission full screen
		case *protos.SCmd_SetAngle:
			cl.pitch = cmd.SetAngle.GetX()
			cl.yaw = cmd.SetAngle.GetY()
			cl.roll = cmd.SetAngle.GetZ()
		case *protos.SCmd_SetViewEntity:
			cl.viewentity = int(cmd.SetViewEntity)
		case *protos.SCmd_LightStyle:
			if err := readLightStyle(cmd.LightStyle.GetIdx(), cmd.LightStyle.GetNewStyle()); err != nil {
				Error("svc_lightstyle: %v", err)
			}
		case *protos.SCmd_Sound:
			if err := CL_ParseStartSoundPacket(cmd.Sound); err != nil {
				return serverRunning, err
			}
		case *protos.SCmd_StopSound:
			snd.Stop(int(cmd.StopSound)>>3, int(cmd.StopSound)&7)
		case *protos.SCmd_UpdateName:
			player := int(cmd.UpdateName.GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatename > MAX_SCOREBOARD")
			}
			cl.scores[player].name = cmd.UpdateName.GetNewName()
		case *protos.SCmd_UpdateFrags:
			player := int(cmd.UpdateFrags.GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatefrags > MAX_SCOREBOARD")
			}
			cl.scores[player].frags = int(cmd.UpdateFrags.GetNewFrags())
		case *protos.SCmd_UpdateColors:
			player := int(cmd.UpdateColors.GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatecolors > MAX_SCOREBOARD")
			}
			c := cmd.UpdateColors.GetNewColor()
			cl.scores[player].topColor = int((c & 0xf0) >> 4)
			cl.scores[player].bottomColor = int(c & 0x0f)
			updatePlayerSkin(player)
		case *protos.SCmd_Particle:
			org := cmd.Particle.GetOrigin()
			dir := cmd.Particle.GetDirection()
			particlesRunEffect(
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				vec.Vec3{dir.GetX(), dir.GetY(), dir.GetZ()},
				int(cmd.Particle.GetColor()), int(cmd.Particle.GetCount()), float32(cl.time))
		case *protos.SCmd_SpawnBaseline:
			i := cmd.SpawnBaseline.GetIndex()
			// force cl.num_entities up
			e := cl.GetOrCreateEntity(int(i))
			parseBaseline(cmd.SpawnBaseline.GetBaseline(), e)
		case *protos.SCmd_SpawnStatic:
			cl.parseStatic(cmd.SpawnStatic)
		case *protos.SCmd_TempEntity:
			cls.parseTempEntity(cmd.TempEntity)
		case *protos.SCmd_SetPause:
			// this was used to pause cd audio, other pause as well?
			cl.paused = cmd.SetPause
		case *protos.SCmd_SignonNum:
			i := int(cmd.SignonNum)
			if i <= cls.signon {
				return serverRunning, fmt.Errorf("Received signon %d when at %d", i, cls.signon)
			}
			cls.signon = i
			// if signonnum==2, signon packet has been fully parsed, so
			// check for excessive static entities and entity fragments
			if i == 2 {
				if len(cl.staticEntities) > 128 {
					conlog.DWarning("%d static entities exceeds standard limit of 128.\n",
						len(cl.staticEntities))
				}
			}
			CL_SignonReply()
		case *protos.SCmd_KilledMonster:
			cl.stats.monsters++
		case *protos.SCmd_FoundSecret:
			cl.stats.secrets++
		case *protos.SCmd_UpdateStat:
			// Only used for STAT_TOTALSECRETS, STAT_TOTALMONSTERS, STAT_SECRETS,
			// STAT_MONSTERS
			cl_setStats(int(cmd.UpdateStat.GetStat()), int(cmd.UpdateStat.GetValue()))
		case *protos.SCmd_SpawnStaticSound:
			s := cmd.SpawnStaticSound
			org := s.GetOrigin()
			snd.Start(0, 0, cl.soundPrecache[s.GetIndex()-1],
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				float32(s.GetVolume())/255, float32(s.GetAttenuation())/64, loopingSound)
		case *protos.SCmd_CdTrack:
			// We do not play cds
		case *protos.SCmd_Intermission:
			cl.intermission = 1
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			restoreViewAngles()
		case *protos.SCmd_Finale:
			cl.intermission = 2
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			screen.CenterPrint(cmd.Finale)
			console.CenterPrint(cmd.Finale)
			restoreViewAngles()
		case *protos.SCmd_Cutscene:
			cl.intermission = 3
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			screen.CenterPrint(cmd.Cutscene)
			console.CenterPrint(cmd.Cutscene)
			restoreViewAngles()
		case *protos.SCmd_SellScreen:
			// Origin seems to be progs.dat
			enterMenuHelp()
		case *protos.SCmd_Skybox:
			sky.LoadBox(cmd.Skybox)
		case *protos.SCmd_BackgroundFlash:
			// Origin seems to be progs.dat
			cl.bonusFlash()
		case *protos.SCmd_Fog:
			f := cmd.Fog
			fog.Update(f.GetDensity(), f.GetRed(), f.GetGreen(), f.GetBlue(), float64(f.GetTime()))
		case *protos.SCmd_Achievement:
			conlog.DPrintf("Ignoring svc_achievement (%s)\n", cmd.Achievement)
		}
	}
	return serverRunning, nil
}

func restoreViewAngles() {
	e := cl.Entities(cl.viewentity)
	e.Angles = e.MsgAngles[0]
}

func CL_ParseServerInfo(si *protos.ServerInfo) error {
	conlog.DPrintf("Serverinfo packet received.\n")

	// bring up loading plaque for map changes within a demo.
	// it will be hidden in CL_SignonReply.
	if cls.demoPlayback {
		screen.BeginLoadingPlaque()
	}

	cl.ClearState()

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
		return fmt.Errorf("Bad maxclients (%d) from server", si.MaxClients)
	}
	cl.maxClients = int(si.MaxClients)
	cl.scores = make([]score, cl.maxClients)
	cl.gameType = int(si.GameType)
	cl.levelName = si.LevelName

	// separate the printfs so the server message can have a color
	console.printBar()
	conlog.Printf("%c%s\n", 2, cl.levelName)

	conlog.Printf("Using protocol %d\n", cl.protocol)

	cl.modelPrecache = cl.modelPrecache[:0]
	if len(si.ModelPrecache) >= 2048 {
		return fmt.Errorf("Server sent too many model precaches")
	}
	if len(si.ModelPrecache) >= 256 {
		conlog.DWarning("%d models exceeds standard limit of 256.\n", len(si.ModelPrecache))
	}

	cl.soundPrecache = cl.soundPrecache[:0]
	if len(si.SoundPrecache) >= 2048 {
		return fmt.Errorf("Server sent too many sound precaches")
	}
	if len(si.SoundPrecache) >= 256 {
		conlog.DWarning("%d sounds exceeds standard limit of 256.\n", len(si.SoundPrecache))
	}

	mapName := si.ModelPrecache[0]
	// now we try to load everything else until a cache allocation fails
	cl.mapName = strings.TrimSuffix(filepath.Base(mapName), filepath.Ext(mapName))

	for _, mn := range si.ModelPrecache {
		m, ok := models[mn]
		if !ok {
			loadModel(mn)
			m, ok = models[mn]
			if !ok {
				return fmt.Errorf("Model %s not found", mn)
			}
		}
		cl.modelPrecache = append(cl.modelPrecache, m)
		if err := CL_KeepaliveMessage(); err != nil {
			return err
		}
	}

	for _, s := range si.SoundPrecache {
		sfx := snd.PrecacheSound(s)
		cl.soundPrecache = append(cl.soundPrecache, sfx)
		if err := CL_KeepaliveMessage(); err != nil {
			return err
		}
	}

	// TODO: clean this stuff up
	cl.worldModel, _ = cl.modelPrecache[0].(*bsp.Model)
	for _, t := range cl.worldModel.Textures {
		if t != nil && strings.HasPrefix(t.Name(), "sky") {
			sky.LoadTexture(t)
		}
	}
	if err := newMap(cl.worldModel); err != nil {
		return err
	}

	// we don't consider identical messages to be duplicates if the map has changed in between
	console.lastCenter = ""
	return nil
}

func newMap(m *bsp.Model) error {
	for i := range lightStyleValues {
		lightStyleValues[i] = 264
	}

	// clean up in case of reuse
	for _, l := range m.Leafs {
		l.Temporary = nil
	}
	// r_viewleaf = NULL
	particlesClear()

	// GL_BuildLightmaps
	brushDrawer.buildVertexBuffer() // should get the model

	renderer.frameCount = 0
	renderer.visFrameCount = 0

	for _, e := range m.Entities {
		if n, ok := e.Name(); !ok || n != "worldspawn" {
			continue
		}
		sky.newMap(e)
		fog.parseWorldspawn(e)
		handleMapAlphas(e)
	}

	// load_subdivide_size = Cvar_GetValue(&gl_subdivide_size)

	return nil
}

// ParseEntityUpdate parses an entity update message from the server
// If an entities model or origin changes from frame to frame, it must be
// relinked. Other attributes can change without relinking.
func (c *Client) ParseEntityUpdate(eu *protos.EntityUpdate) {
	if cls.signon == 3 {
		// first update is the final signon stage
		cls.signon = 4
		CL_SignonReply()
	}
	num := int(eu.Entity)
	e := c.GetOrCreateEntity(num)
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
		modNum = int(*eu.Model)
	}
	if modNum >= model.MAX_MODELS {
		Error("CL_ParseModel: mad modnum")
	}
	if eu.Frame != nil {
		e.Frame = int(*eu.Frame)
	}
	if eu.Skin != nil {
		e.SkinNum = int(*eu.Skin)
	}
	if e.SkinNum != oldSkinNum {
		if num > 0 && num <= cl.maxClients {
			createPlayerSkin(num, e)
		}
	}
	e.Effects = int(eu.Effects)
	if eu.OriginX != nil {
		e.MsgOrigin[0][0] = *eu.OriginX
	}
	if eu.OriginY != nil {
		e.MsgOrigin[0][1] = *eu.OriginY
	}
	if eu.OriginZ != nil {
		e.MsgOrigin[0][2] = *eu.OriginZ
	}
	if eu.AngleX != nil {
		e.MsgAngles[0][0] = *eu.AngleX
	}
	if eu.AngleY != nil {
		e.MsgAngles[0][1] = *eu.AngleY
	}
	if eu.AngleZ != nil {
		e.MsgAngles[0][2] = *eu.AngleZ
	}

	if eu.Alpha != nil {
		e.Alpha = byte(*eu.Alpha)
	}
	if eu.LerpFinish != nil {
		e.LerpFinish = e.MsgTime + float64(*eu.LerpFinish)/255
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
				createPlayerSkin(num, e)
			}
			// do not lerp animation across model changes
			e.LerpFlags |= lerpResetAnim
		}
	} else {
		if modNum != 0 {
			conlog.Printf("len(modelPrecache): %v, modNum: %v", len(cl.modelPrecache), modNum)
		}
		if e.Model != nil {
			forceLink = true
			e.LerpFlags |= lerpResetAnim
		}
		e.Model = nil
	}

	if forceLink {
		e.MsgOrigin[1] = e.MsgOrigin[0]
		e.Origin = e.MsgOrigin[0]
		e.MsgAngles[1] = e.MsgAngles[0]
		e.Angles = e.MsgAngles[0]
		e.ForceLink = true
	}
}

func handleServerDisconnected(msg string) error {
	conlog.DPrintf("Host_EndGame: %s\n", msg)

	if ServerActive() {
		if err := hostShutdownServer(false); err != nil {
			return err
		}
	}

	if cmdl.Dedicated() {
		// dedicated servers exit
		Error("Host_EndGame: %s\n", msg)
	}

	if cls.demoNum != -1 {
		if err := CL_NextDemo(); err != nil {
			return err
		}
	} else {
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) parseStatic(pb *protos.Baseline) {
	ent := c.CreateStaticEntity()
	parseBaseline(pb, ent)
	// copy it to the current state

	ent.Model = c.modelPrecache[ent.Baseline.ModelIndex-1]
	ent.LerpFlags |= lerpResetAnim // TODO(therjak): shouldn't this be an override instead of an OR?
	ent.Frame = ent.Baseline.Frame
	ent.SkinNum = ent.Baseline.Skin
	ent.Effects = 0
	ent.Alpha = ent.Baseline.Alpha
	ent.Origin = ent.Baseline.Origin
	ent.Angles = ent.Baseline.Angles

	ent.R_AddEfrags() // clean up after removal of c-efrags
}
