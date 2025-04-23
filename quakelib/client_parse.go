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
	qsnd "goquake/snd"
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
		switch scmd.WhichUnion() {
		default:
			// nop
		case protos.SCmd_EntityUpdate_case:
			cl.ParseEntityUpdate(scmd.GetEntityUpdate())
		case protos.SCmd_Time_case:
			cl.messageTimeOld = cl.messageTime
			cl.messageTime = float64(scmd.GetTime())
		case protos.SCmd_ClientData_case:
			cl.parseClientData(scmd.GetClientData())
		case protos.SCmd_Version_case:
			switch scmd.GetVersion() {
			case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
				cl.protocol = int(scmd.GetVersion())
			default:
				return serverRunning, fmt.Errorf("Server returned version %d, not %d or %d or %d or %d", scmd.GetVersion(),
					protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
			}
		case protos.SCmd_Disconnect_case:
			if err := handleServerDisconnected("Server disconnected\n"); err != nil {
				return serverDisconnected, err
			}
			return serverDisconnected, nil
		case protos.SCmd_Print_case:
			conlog.Printf("%s", scmd.GetPrint())
		case protos.SCmd_CenterPrint_case:
			screen.CenterPrint(scmd.GetCenterPrint())
			console.CenterPrint(scmd.GetCenterPrint())
		case protos.SCmd_StuffText_case:
			cbuf.AddText(scmd.GetStuffText())
		case protos.SCmd_Damage_case:
			d := scmd.GetDamage()
			pos := d.GetPosition()
			cl.parseDamage(int(d.GetArmor()), int(d.GetBlood()), vec.Vec3{
				pos.GetX(), pos.GetY(), pos.GetZ(),
			})
		case protos.SCmd_ServerInfo_case:
			if err := CL_ParseServerInfo(scmd.GetServerInfo()); err != nil {
				return serverRunning, err
			}
			screen.recalcViewRect = true // leave intermission full screen
		case protos.SCmd_SetAngle_case:
			cl.pitch = scmd.GetSetAngle().GetX()
			cl.yaw = scmd.GetSetAngle().GetY()
			cl.roll = scmd.GetSetAngle().GetZ()
		case protos.SCmd_SetViewEntity_case:
			cl.viewentity = int(scmd.GetSetViewEntity())
		case protos.SCmd_LightStyle_case:
			if err := readLightStyle(scmd.GetLightStyle().GetIdx(), scmd.GetLightStyle().GetNewStyle()); err != nil {
				Error("svc_lightstyle: %v", err)
			}
		case protos.SCmd_Sound_case:
			if err := CL_ParseStartSoundPacket(scmd.GetSound()); err != nil {
				return serverRunning, err
			}
		case protos.SCmd_StopSound_case:
			snd.Stop(int(scmd.GetStopSound())>>3, int(scmd.GetStopSound())&7)
		case protos.SCmd_UpdateName_case:
			player := int(scmd.GetUpdateName().GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatename > MAX_SCOREBOARD")
			}
			cl.scores[player].name = scmd.GetUpdateName().GetNewName()
		case protos.SCmd_UpdateFrags_case:
			player := int(scmd.GetUpdateFrags().GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatefrags > MAX_SCOREBOARD")
			}
			cl.scores[player].frags = int(scmd.GetUpdateFrags().GetNewFrags())
		case protos.SCmd_UpdateColors_case:
			player := int(scmd.GetUpdateColors().GetPlayer())
			if player >= cl.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatecolors > MAX_SCOREBOARD")
			}
			c := scmd.GetUpdateColors().GetNewColor()
			cl.scores[player].topColor = int((c & 0xf0) >> 4)
			cl.scores[player].bottomColor = int(c & 0x0f)
			updatePlayerSkin(player)
		case protos.SCmd_Particle_case:
			org := scmd.GetParticle().GetOrigin()
			dir := scmd.GetParticle().GetDirection()
			particlesRunEffect(
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				vec.Vec3{dir.GetX(), dir.GetY(), dir.GetZ()},
				int(scmd.GetParticle().GetColor()), int(scmd.GetParticle().GetCount()), float32(cl.time))
		case protos.SCmd_SpawnBaseline_case:
			i := scmd.GetSpawnBaseline().GetIndex()
			// force cl.num_entities up
			e := cl.GetOrCreateEntity(int(i))
			parseBaseline(scmd.GetSpawnBaseline().GetBaseline(), e)
		case protos.SCmd_SpawnStatic_case:
			cl.parseStatic(scmd.GetSpawnStatic())
		case protos.SCmd_TempEntity_case:
			cls.parseTempEntity(scmd.GetTempEntity())
		case protos.SCmd_SetPause_case:
			// this was used to pause cd audio, other pause as well?
			cl.paused = scmd.GetSetPause()
		case protos.SCmd_SignonNum_case:
			i := int(scmd.GetSignonNum())
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
		case protos.SCmd_KilledMonster_case:
			cl.stats.monsters++
		case protos.SCmd_FoundSecret_case:
			cl.stats.secrets++
		case protos.SCmd_UpdateStat_case:
			// Only used for STAT_TOTALSECRETS, STAT_TOTALMONSTERS, STAT_SECRETS,
			// STAT_MONSTERS
			cl_setStats(int(scmd.GetUpdateStat().GetStat()), int(scmd.GetUpdateStat().GetValue()))
		case protos.SCmd_SpawnStaticSound_case:
			s := scmd.GetSpawnStaticSound()
			org := s.GetOrigin()
			cl.sound.StartAmbient(int(s.GetIndex()-1),
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				float32(s.GetVolume())/255, float32(s.GetAttenuation())/64)
		case protos.SCmd_CdTrack_case:
			// We do not play cds
		case protos.SCmd_Intermission_case:
			cl.intermission = 1
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			restoreViewAngles()
		case protos.SCmd_Finale_case:
			cl.intermission = 2
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			screen.CenterPrint(scmd.GetFinale())
			console.CenterPrint(scmd.GetFinale())
			restoreViewAngles()
		case protos.SCmd_Cutscene_case:
			cl.intermission = 3
			cl.intermissionTime = int(cl.time)
			screen.recalcViewRect = true // go to full screen
			screen.CenterPrint(scmd.GetCutscene())
			console.CenterPrint(scmd.GetCutscene())
			restoreViewAngles()
		case protos.SCmd_SellScreen_case:
			// Origin seems to be progs.dat
			enterMenuHelp()
		case protos.SCmd_Skybox_case:
			sky.LoadBox(scmd.GetSkybox())
		case protos.SCmd_BackgroundFlash_case:
			// Origin seems to be progs.dat
			cl.bonusFlash()
		case protos.SCmd_Fog_case:
			f := scmd.GetFog()
			fog.Update(f.GetDensity(), f.GetRed(), f.GetGreen(), f.GetBlue(), float64(f.GetTime()))
		case protos.SCmd_Achievement_case:
			conlog.DPrintf("Ignoring svc_achievement (%s)\n", scmd.GetAchievement())
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

	cl.protocol = int(si.GetProtocol())
	cl.protocolFlags = uint32(si.GetFlags())

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

	if si.GetMaxClients() < 1 || si.GetMaxClients() > 16 {
		return fmt.Errorf("Bad maxclients (%d) from server", si.GetMaxClients())
	}
	cl.maxClients = int(si.GetMaxClients())
	cl.scores = make([]score, cl.maxClients)
	cl.gameType = int(si.GetGameType())
	cl.levelName = si.GetLevelName()

	// separate the printfs so the server message can have a color
	console.printBar()
	conlog.Printf("%c%s\n", 2, cl.levelName)

	conlog.Printf("Using protocol %d\n", cl.protocol)

	cl.modelPrecache = cl.modelPrecache[:0]
	if len(si.GetModelPrecache()) >= 2048 {
		return fmt.Errorf("Server sent too many model precaches")
	}
	if len(si.GetModelPrecache()) >= 256 {
		conlog.DWarning("%d models exceeds standard limit of 256.\n", len(si.GetModelPrecache()))
	}

	if len(si.GetSoundPrecache()) >= 2048 {
		return fmt.Errorf("Server sent too many sound precaches")
	}
	if len(si.GetSoundPrecache()) >= 256 {
		conlog.DWarning("%d sounds exceeds standard limit of 256.\n", len(si.GetSoundPrecache()))
	}

	mapName := si.GetModelPrecache()[0]
	// now we try to load everything else until a cache allocation fails
	cl.mapName = strings.TrimSuffix(filepath.Base(mapName), filepath.Ext(mapName))

	for _, mn := range si.GetModelPrecache() {
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

	var snds []qsnd.Sound
	for i, s := range si.GetSoundPrecache() {
		snds = append(snds, qsnd.Sound{i, s})
	}
	cl.sound = snd.NewPrecache(snds...)

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
	num := int(eu.GetEntity())
	e := c.GetOrCreateEntity(num)
	forceLink := e.MsgTime != c.messageTimeOld

	if e.MsgTime+0.2 < c.messageTime {
		// most entities think every 0.1s, if we missed one we would be lerping from the wrong frame
		e.LerpFlags |= lerpResetAnim
	}
	if eu.GetLerpMoveStep() {
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
	if eu.HasModel() {
		modNum = int(eu.GetModel())
	}
	if modNum >= model.MAX_MODELS {
		Error("CL_ParseModel: mad modnum")
	}
	if eu.HasFrame() {
		e.Frame = int(eu.GetFrame())
	}
	if eu.HasSkin() {
		e.SkinNum = int(eu.GetSkin())
	}
	if e.SkinNum != oldSkinNum {
		if num > 0 && num <= cl.maxClients {
			createPlayerSkin(num, e)
		}
	}
	e.Effects = int(eu.GetEffects())
	if eu.HasOriginX() {
		e.MsgOrigin[0][0] = eu.GetOriginX()
	}
	if eu.HasOriginY() {
		e.MsgOrigin[0][1] = eu.GetOriginY()
	}
	if eu.HasOriginZ() {
		e.MsgOrigin[0][2] = eu.GetOriginZ()
	}
	if eu.HasAngleX() {
		e.MsgAngles[0][0] = eu.GetAngleX()
	}
	if eu.HasAngleY() {
		e.MsgAngles[0][1] = eu.GetAngleY()
	}
	if eu.HasAngleZ() {
		e.MsgAngles[0][2] = eu.GetAngleZ()
	}

	if eu.HasAlpha() {
		e.Alpha = byte(eu.GetAlpha())
	}
	if eu.HasLerpFinish() {
		e.LerpFinish = e.MsgTime + float64(eu.GetLerpFinish())/255
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
