// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"log/slog"
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

func (c *Client) ParseServerMessage(pb *protos.ServerMessage) (serverState, error) {
	switch cvars.ClientShowNet.String() {
	case "1", "2":
		conlog.Printf("------------------\n")
	}

	for _, scmd := range pb.GetCmds() {
		switch scmd.WhichUnion() {
		default:
			// nop
		case protos.SCmd_EntityUpdate_case:
			if err := c.ParseEntityUpdate(scmd.GetEntityUpdate()); err != nil {
				return serverDisconnected, err
			}
		case protos.SCmd_Time_case:
			c.messageTimeOld = c.messageTime
			c.messageTime = float64(scmd.GetTime())
		case protos.SCmd_ClientData_case:
			c.parseClientData(scmd.GetClientData())
		case protos.SCmd_Version_case:
			switch scmd.GetVersion() {
			case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
				c.protocol = int(scmd.GetVersion())
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
			// THERJAK: console color part 1
			conlog.Printf("%s", scmd.GetPrint())
		case protos.SCmd_CenterPrint_case:
			screen.CenterPrint(scmd.GetCenterPrint())
			console.CenterPrint(scmd.GetCenterPrint())
		case protos.SCmd_StuffText_case:
			cbuf.AddText(scmd.GetStuffText())
		case protos.SCmd_Damage_case:
			d := scmd.GetDamage()
			pos := d.GetPosition()
			c.parseDamage(int(d.GetArmor()), int(d.GetBlood()), vec.Vec3{
				pos.GetX(), pos.GetY(), pos.GetZ(),
			})
		case protos.SCmd_ServerInfo_case:
			if err := c.ParseServerInfo(scmd.GetServerInfo()); err != nil {
				return serverRunning, err
			}
			screen.recalcViewRect = true // leave intermission full screen
		case protos.SCmd_SetAngle_case:
			c.pitch = scmd.GetSetAngle().GetX()
			c.yaw = scmd.GetSetAngle().GetY()
			c.roll = scmd.GetSetAngle().GetZ()
		case protos.SCmd_SetViewEntity_case:
			c.viewentity = int(scmd.GetSetViewEntity())
		case protos.SCmd_LightStyle_case:
			if err := readLightStyle(scmd.GetLightStyle().GetIdx(), scmd.GetLightStyle().GetNewStyle()); err != nil {
				return serverDisconnected, err
			}
		case protos.SCmd_Sound_case:
			if err := CL_ParseStartSoundPacket(scmd.GetSound()); err != nil {
				return serverRunning, err
			}
		case protos.SCmd_StopSound_case:
			snd.Stop(int(scmd.GetStopSound())>>3, int(scmd.GetStopSound())&7)
		case protos.SCmd_UpdateName_case:
			player := int(scmd.GetUpdateName().GetPlayer())
			if player >= c.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatename > MAX_SCOREBOARD")
			}
			c.scores[player].name = scmd.GetUpdateName().GetNewName()
		case protos.SCmd_UpdateFrags_case:
			player := int(scmd.GetUpdateFrags().GetPlayer())
			if player >= c.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatefrags > MAX_SCOREBOARD")
			}
			c.scores[player].frags = int(scmd.GetUpdateFrags().GetNewFrags())
		case protos.SCmd_UpdateColors_case:
			player := int(scmd.GetUpdateColors().GetPlayer())
			if player < 0 || player >= c.maxClients {
				return serverRunning, fmt.Errorf("CL_ParseServerMessage: svc_updatecolors > MAX_SCOREBOARD")
			}
			color := scmd.GetUpdateColors().GetNewColor()
			c.scores[player].topColor = int((color & 0xf0) >> 4)
			c.scores[player].bottomColor = int(color & 0x0f)
			e := c.Entities(player + 1)
			translatePlayerSkin(e)
		case protos.SCmd_Particle_case:
			org := scmd.GetParticle().GetOrigin()
			dir := scmd.GetParticle().GetDirection()
			particlesRunEffect(
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				vec.Vec3{dir.GetX(), dir.GetY(), dir.GetZ()},
				int(scmd.GetParticle().GetColor()), int(scmd.GetParticle().GetCount()), c.time)
		case protos.SCmd_SpawnBaseline_case:
			i := scmd.GetSpawnBaseline().GetIndex()
			// force c.num_entities up
			e, err := c.GetOrCreateEntity(int(i))
			if err != nil {
				return serverDisconnected, err
			}
			parseBaseline(scmd.GetSpawnBaseline().GetBaseline(), e)
		case protos.SCmd_SpawnStatic_case:
			err := c.parseStatic(scmd.GetSpawnStatic())
			if err != nil {
				return serverDisconnected, err
			}
		case protos.SCmd_TempEntity_case:
			err := c.parseTempEntity(scmd.GetTempEntity())
			if err != nil {
				return serverDisconnected, err
			}
		case protos.SCmd_SetPause_case:
			// this was used to pause cd audio, other pause as well?
			c.paused = scmd.GetSetPause()
		case protos.SCmd_SignonNum_case:
			i := int(scmd.GetSignonNum())
			if i <= cls.signon {
				return serverRunning, fmt.Errorf("Received signon %d when at %d", i, cls.signon)
			}
			cls.signon = i
			// if signonnum==2, signon packet has been fully parsed, so
			// check for excessive static entities and entity fragments
			if i == 2 {
				if len(c.staticEntities) > 128 {
					slog.Debug("static entities exceeds standard limit of 128.", slog.Int("Count",
						len(c.staticEntities)))
				}
			}
			CL_SignonReply()
		case protos.SCmd_KilledMonster_case:
			c.stats.monsters++
		case protos.SCmd_FoundSecret_case:
			c.stats.secrets++
		case protos.SCmd_UpdateStat_case:
			// Only used for STAT_TOTALSECRETS, STAT_TOTALMONSTERS, STAT_SECRETS,
			// STAT_MONSTERS
			cl_setStats(int(scmd.GetUpdateStat().GetStat()), int(scmd.GetUpdateStat().GetValue()))
		case protos.SCmd_SpawnStaticSound_case:
			s := scmd.GetSpawnStaticSound()
			org := s.GetOrigin()
			c.sound.StartAmbient(int(s.GetIndex()-1),
				vec.Vec3{org.GetX(), org.GetY(), org.GetZ()},
				float32(s.GetVolume())/255, float32(s.GetAttenuation())/64)
		case protos.SCmd_CdTrack_case:
			// We do not play cds
		case protos.SCmd_Intermission_case:
			c.intermission = 1
			c.intermissionTime = int(c.time)
			screen.recalcViewRect = true // go to full screen
			restoreViewAngles()
		case protos.SCmd_Finale_case:
			c.intermission = 2
			c.intermissionTime = int(c.time)
			screen.recalcViewRect = true // go to full screen
			screen.CenterPrint(scmd.GetFinale())
			console.CenterPrint(scmd.GetFinale())
			restoreViewAngles()
		case protos.SCmd_Cutscene_case:
			c.intermission = 3
			c.intermissionTime = int(c.time)
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
			c.bonusFlash()
		case protos.SCmd_Fog_case:
			f := scmd.GetFog()
			fog.Update(f.GetDensity(), f.GetRed(), f.GetGreen(), f.GetBlue(), float64(f.GetTime()))
		case protos.SCmd_Achievement_case:
			slog.Debug("Ignoring svc_achievement", slog.String("Archievement", scmd.GetAchievement()))
		}
	}
	return serverRunning, nil
}

func restoreViewAngles() {
	e := cl.Entities(cl.viewentity)
	e.Angles = e.MsgAngles[0]
}

func (c *Client) ParseServerInfo(si *protos.ServerInfo) error {
	slog.Debug("Serverinfo packet received.")

	// bring up loading plaque for map changes within a demo.
	// it will be hidden in CL_SignonReply.
	if cls.demoPlayback {
		screen.BeginLoadingPlaque()
	}

	if err := cl.ClearState(); err != nil {
		return err
	}

	c.protocol = int(si.GetProtocol())
	c.protocolFlags = uint32(si.GetFlags())

	if c.protocol == protocol.RMQ {
		const supportedflags uint32 = protocol.PRFL_SHORTANGLE |
			protocol.PRFL_FLOATANGLE |
			protocol.PRFL_24BITCOORD |
			protocol.PRFL_FLOATCOORD |
			protocol.PRFL_EDICTSCALE |
			protocol.PRFL_INT32COORD

		if c.protocolFlags&^supportedflags != 0 {
			conlog.Warning("PROTOCOL_RMQ protocolflags %d contains unsupported flags\n", c.protocolFlags)
		}
	}

	if si.GetMaxClients() < 1 || si.GetMaxClients() > 16 {
		return fmt.Errorf("Bad maxclients (%d) from server", si.GetMaxClients())
	}
	c.maxClients = int(si.GetMaxClients())
	c.scores = make([]score, cl.maxClients)
	c.gameType = int(si.GetGameType())
	c.levelName = si.GetLevelName()

	// separate the printfs so the server message can have a color
	console.printBar()
	// TODO: color print part 2
	conlog.Printf("%c%s\n", 2, c.levelName)

	conlog.Printf("Using protocol %d\n", c.protocol)

	c.modelPrecache = c.modelPrecache[:0]
	if len(si.GetModelPrecache()) >= 2048 {
		return fmt.Errorf("Server sent too many model precaches")
	}
	if len(si.GetModelPrecache()) >= 256 {
		slog.Debug("models exceeds standard limit of 256.", slog.Int("Count", len(si.GetModelPrecache())))
	}

	if len(si.GetSoundPrecache()) >= 2048 {
		return fmt.Errorf("Server sent too many sound precaches")
	}
	if len(si.GetSoundPrecache()) >= 256 {
		slog.Debug("sounds exceeds standard limit of 256.", slog.Int("Count", len(si.GetSoundPrecache())))
	}

	mapName := si.GetModelPrecache()[0]
	// now we try to load everything else until a cache allocation fails
	c.mapName = strings.TrimSuffix(filepath.Base(mapName), filepath.Ext(mapName))

	for _, mn := range si.GetModelPrecache() {
		m, ok := models[mn]
		if !ok {
			if _, err := loadModel(mn); err != nil {
				return fmt.Errorf("Model %s not found: %v", mn, err)
			}
			m, ok = models[mn]
			if !ok {
				return fmt.Errorf("Model %s not found", mn)
			}
		}
		c.modelPrecache = append(c.modelPrecache, m)
		if err := CL_KeepaliveMessage(); err != nil {
			return err
		}
	}

	var snds []qsnd.Sound
	for i, s := range si.GetSoundPrecache() {
		snds = append(snds, qsnd.Sound{i, s})
	}
	c.sound = snd.NewPrecache(snds...)

	// TODO: clean this stuff up
	c.worldModel, _ = c.modelPrecache[0].(*bsp.Model)
	for _, t := range c.worldModel.Textures {
		if t != nil && strings.HasPrefix(t.Name(), "sky") {
			sky.LoadTexture(t)
		}
	}
	if err := newMap(c.worldModel); err != nil {
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
	brushDrawer.buildVertexBuffer(cl.modelPrecache) // should get the model

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
func (c *Client) ParseEntityUpdate(eu *protos.EntityUpdate) error {
	if cls.signon == 3 {
		// first update is the final signon stage
		cls.signon = 4
		CL_SignonReply()
	}
	num := int(eu.GetEntity())
	e, err := c.GetOrCreateEntity(num)
	if err != nil {
		return err
	}
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
		return fmt.Errorf("CL_ParseModel: mad modnum")
	}
	if eu.HasFrame() {
		e.Frame = int(eu.GetFrame())
	}
	if eu.HasSkin() {
		e.SkinNum = int(eu.GetSkin())
	}
	if e.SkinNum != oldSkinNum {
		if num > 0 && num <= c.maxClients {
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

	if modNum > 0 && modNum <= len(c.modelPrecache) {
		model := c.modelPrecache[modNum-1] // server sends this 1 based, modelPrecache is 0 based
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
			if num > 0 && num <= c.maxClients {
				createPlayerSkin(num, e)
			}
			// do not lerp animation across model changes
			e.LerpFlags |= lerpResetAnim
		}
	} else {
		if modNum != 0 {
			conlog.Printf("len(modelPrecache): %v, modNum: %v", len(c.modelPrecache), modNum)
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
	return nil
}

func handleServerDisconnected(msg string) error {
	slog.Debug("Host_EndGame", slog.String("msg", msg))

	if ServerActive() {
		if err := hostShutdownServer(false); err != nil {
			return err
		}
	}

	if cmdl.Dedicated() {
		// dedicated servers exit
		QError("Host_EndGame: %s\n", msg)
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

func (c *Client) parseStatic(pb *protos.Baseline) error {
	ent, err := c.CreateStaticEntity()
	if err != nil {
		return err
	}
	parseBaseline(pb, ent)
	// copy it to the current state

	ent.Model = c.modelPrecache[ent.Baseline.ModelIndex-1]
	ent.LerpFlags |= lerpResetAnim
	ent.Frame = ent.Baseline.Frame
	ent.SkinNum = ent.Baseline.Skin
	ent.Effects = 0
	ent.Alpha = ent.Baseline.Alpha
	ent.Origin = ent.Baseline.Origin
	ent.Angles = ent.Baseline.Angles

	adder := EntityFragmentAdder{entity: ent, world: c.worldModel}
	adder.Do()

	return nil
}
