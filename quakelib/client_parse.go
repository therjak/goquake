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
	"github.com/therjak/goquake/math"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/protocol"
	svc "github.com/therjak/goquake/protocol/server"
	"github.com/therjak/goquake/snd"
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

func CL_ParseBaseline(e *Entity, version int) {
	var err error
	es := EntityState{
		Alpha: svc.EntityAlphaDefault,
	}
	bits := byte(0)
	if version == 2 {
		bits, err = cls.inMessage.ReadByte()
		if err != nil {
			cls.msgBadRead = true
			return
		}
	}
	if bits&svc.EntityBaselineLargeModel != 0 {
		i, err := cls.inMessage.ReadUint16()
		if err != nil {
			cls.msgBadRead = true
			return
		}
		es.ModelIndex = i
	} else {
		i, err := cls.inMessage.ReadByte()
		if err != nil {
			cls.msgBadRead = true
			return
		}
		es.ModelIndex = uint16(i)
	}
	if bits&svc.EntityBaselineLargeFrame != 0 {
		f, err := cls.inMessage.ReadUint16()
		if err != nil {
			cls.msgBadRead = true
			return
		}
		es.Frame = f
	} else {
		f, err := cls.inMessage.ReadByte()
		if err != nil {
			cls.msgBadRead = true
			return
		}
		es.Frame = uint16(f)
	}

	// colormap: no idea what this is good for. Is not really used.
	es.ColorMap, err = cls.inMessage.ReadByte()
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Skin, err = cls.inMessage.ReadByte()
	if err != nil {
		cls.msgBadRead = true
		return
	}

	es.Origin[0], err = cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Angles[0], err = cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Origin[1], err = cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Angles[1], err = cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Origin[2], err = cls.inMessage.ReadCoord(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}
	es.Angles[2], err = cls.inMessage.ReadAngle(cl.protocolFlags)
	if err != nil {
		cls.msgBadRead = true
		return
	}

	if bits&svc.EntityBaselineAlpha != 0 {
		es.Alpha, err = cls.inMessage.ReadByte()
		if err != nil {
			cls.msgBadRead = true
			return
		}
	}

	e.Baseline = es
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
			cl.ParseEntityUpdate(cmd & 127)
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
			err := cl.parseClientData()
			if err != nil {
				cls.msgBadRead = true
				continue
			}

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
			err := CL_ParseServerInfo()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
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
			ReadLightStyle() // ReadByte + ReadString

		case svc.Sound:
			CL_ParseStartSoundPacket()

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
			CL_ParseBaseline(e, 1)

		case svc.SpawnStatic:
			cl.ParseStatic(1)

		case svc.TempEntity:
			if err := cls.parseTEnt(); err != nil {
				cls.msgBadRead = true
			}

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
			CL_ParseBaseline(e, 2)

		case svc.SpawnStatic2:
			cl.ParseStatic(2)

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

func CL_ParseServerInfo() error {
	// protocol uint32
	// if protocol RMQ protocolFlags uint32
	// maxClients byte
	// gameMode (coop/dethmatch) byte
	// levelname string (EntVars(0).Message)
	// []string modelPrecache
	// 0 byte
	// []string soundPrecache
	// 0 byte

	conlog.DPrintf("Serverinfo packet received.\n")

	// bring up loading plaque for map changes within a demo.
	// it will be hidden in CL_SignonReply.
	if cls.demoPlayback {
		screen.BeginLoadingPlaque()
	}

	C.CL_ClearState()

	// parse protocol version number
	ptl, err := cls.inMessage.ReadInt32()
	if err != nil {
		return err
	}
	switch ptl {
	case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake:
	default:
		conlog.Printf("\n") // because there's no newline after serverinfo print
		HostError("Server returned version %d, not %d or %d or %d or %d", ptl,
			protocol.NetQuake, protocol.FitzQuake, protocol.RMQ, protocol.GoQuake)
	}
	cl.protocol = int(ptl)

	if cl.protocol == protocol.RMQ {
		const supportedflags uint32 = protocol.PRFL_SHORTANGLE |
			protocol.PRFL_FLOATANGLE |
			protocol.PRFL_24BITCOORD |
			protocol.PRFL_FLOATCOORD |
			protocol.PRFL_EDICTSCALE |
			protocol.PRFL_INT32COORD

		flags, err := cls.inMessage.ReadUint32()
		if err != nil {
			return err
		}
		cl.protocolFlags = flags

		if cl.protocolFlags&^supportedflags != 0 {
			conlog.Warning("PROTOCOL_RMQ protocolflags %d contains unsupported flags\n", cl.protocolFlags)
		}
	} else {
		cl.protocolFlags = 0
	}

	maxClients, err := cls.inMessage.ReadByte()
	if err != nil {
		return err
	}
	if maxClients < 1 || maxClients > 16 {
		HostError("Bad maxclients (%d) from server", maxClients)
	}
	cl.maxClients = int(maxClients)
	cl.scores = make([]score, maxClients)

	gameType, err := cls.inMessage.ReadByte()
	if err != nil {
		return err
	}
	cl.gameType = int(gameType)

	levelName, err := cls.inMessage.ReadString()
	if err != nil {
		return err
	}
	cl.levelName = levelName

	// seperate the printfs so the server message can have a color
	console.printBar()
	conlog.Printf("%c%s\n", 2, cl.levelName)

	conlog.Printf("Using protocol %d\n", cl.protocol)

	cl.modelPrecache = cl.modelPrecache[:]
	var modelNames []string
	for {
		m, err := cls.inMessage.ReadString()
		if err != nil {
			return err
		}
		if m == "" {
			break
		}
		if len(modelNames) == 2048 {
			HostError("Server sent too many model precaches")
		}
		modelNames = append(modelNames, m)
	}

	if len(modelNames) >= 256 {
		conlog.DWarning("%d models exceeds standard limit of 256.\n", len(modelNames))
	}

	cl.soundPrecache = cl.soundPrecache[:0]
	var sounds []string
	for {
		s, err := cls.inMessage.ReadString()
		if err != nil {
			return err
		}
		if s == "" {
			break
		}
		if len(sounds) == 2048 {
			HostError("Server sent too many sound precaches")
		}
		sounds = append(sounds, s)
	}

	if len(sounds) >= 256 {
		conlog.DWarning("%d sounds exceeds standard limit of 256.\n", len(sounds))
	}

	// now we try to load everything else until a cache allocation fails
	cl.mapName = strings.TrimSuffix(filepath.Base(modelNames[0]), filepath.Ext(modelNames[0]))

	C.CLPrecacheModelClear()
	for i, mn := range modelNames {
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

	for _, s := range sounds {
		sfx := snd.PrecacheSound(s)
		cl.soundPrecache = append(cl.soundPrecache, sfx)
		CL_KeepaliveMessage()
	}

	// TODO: clean this stuff up
	cl.worldModel, _ = cl.modelPrecache[0].(*bsp.Model)
	for _, t := range cl.worldModel.Textures {
		if strings.HasPrefix(t.Name(), "sky") {
			sky.LoadTexture(t.Data, t.Name(), cl.mapName)
		}
	}

	C.FinishCL_ParseServerInfo()

	// we don't consider identical messages to be duplicates if the map has changed in between
	console.lastCenter = ""
	return nil
}

//ParseEntityUpdate parses an entity update message from the server
//If an entities model or origin changes from frame to frame, it must be
//relinked. Other attributes can change without relinking.
func (c *Client) ParseEntityUpdate(cmd byte) error {
	if cls.signon == 3 {
		// first update is the final signon stage
		cls.signon = 4
		CL_SignonReply()
	}
	bits := uint32(cmd)
	if bits&svc.U_MOREBITS != 0 {
		b, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		bits |= uint32(b) << 8
	}
	switch c.protocol {
	case protocol.FitzQuake, protocol.RMQ:
		if bits&svc.U_EXTEND1 != 0 {
			b, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			bits |= uint32(b) << 16
		}
		if bits&svc.U_EXTEND2 != 0 {
			b, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			bits |= uint32(b) << 24
		}
	}
	num, err := func() (int, error) {
		if bits&svc.U_LONGENTITY != 0 {
			s, err := cls.inMessage.ReadInt16()
			return int(s), err
		}
		b, err := cls.inMessage.ReadByte()
		return int(b), err
	}()
	if err != nil {
		return err
	}
	e := c.GetOrCreateEntity(num)
	e.SyncC()
	forceLink := e.MsgTime != c.messageTimeOld

	if e.MsgTime+0.2 < c.messageTime {
		// most entities think every 0.1s, if we missed one we would be lerping from the wrong frame
		e.LerpFlags |= lerpResetAnim
	}
	if bits&svc.U_STEP != 0 {
		e.ForceLink = true
		e.LerpFlags |= lerpMoveStep
	} else {
		e.LerpFlags &^= lerpMoveStep
	}

	e.MsgTime = c.messageTime
	e.Frame = int(e.Baseline.Frame)
	oldSkinNum := e.SkinNum
	e.SkinNum = int(e.Baseline.Skin)
	e.Effects = 0
	// shift known values for interpolation
	e.MsgOrigin[1] = e.MsgOrigin[0]
	e.MsgAngles[1] = e.MsgAngles[0]
	e.MsgOrigin[0] = e.Baseline.Origin
	e.MsgAngles[0] = e.Baseline.Angles
	e.Alpha = e.Baseline.Alpha
	e.SyncBase = 0

	modNum := int(e.Baseline.ModelIndex)
	if bits&svc.U_MODEL != 0 {
		v, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		modNum = int(v)
	}
	if modNum >= model.MAX_MODELS {
		Error("CL_ParseModel: mad modnum")
	}
	if bits&svc.U_FRAME != 0 {
		v, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		e.Frame = int(v)
	}
	if bits&svc.U_COLORMAP != 0 {
		// ColorMap -- no idea what this was good for. It was not read.
		if _, err := cls.inMessage.ReadByte(); err != nil {
			return err
		}
	}
	if bits&svc.U_SKIN != 0 {
		v, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		e.SkinNum = int(v)
	}
	if e.SkinNum != oldSkinNum {
		if num > 0 && num <= cl.maxClients {
			// C.R_TranslateNewPlaykerSkin(num - 1)
		}
	}
	if bits&svc.U_EFFECTS != 0 {
		v, err := cls.inMessage.ReadByte()
		if err != nil {
			return err
		}
		e.Effects = int(v)
	}
	if bits&svc.U_ORIGIN1 != 0 {
		v, err := cls.inMessage.ReadCoord(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgOrigin[0][0] = v
	}
	if bits&svc.U_ANGLE1 != 0 {
		v, err := cls.inMessage.ReadAngle(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgAngles[0][0] = v
	}
	if bits&svc.U_ORIGIN2 != 0 {
		v, err := cls.inMessage.ReadCoord(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgOrigin[0][1] = v
	}
	if bits&svc.U_ANGLE2 != 0 {
		v, err := cls.inMessage.ReadAngle(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgAngles[0][1] = v
	}
	if bits&svc.U_ORIGIN3 != 0 {
		v, err := cls.inMessage.ReadCoord(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgOrigin[0][2] = v
	}
	if bits&svc.U_ANGLE3 != 0 {
		v, err := cls.inMessage.ReadAngle(cl.protocolFlags)
		if err != nil {
			return err
		}
		e.MsgAngles[0][2] = v
	}

	switch cl.protocol {
	case protocol.FitzQuake, protocol.RMQ:
		e.LerpFlags &^= lerpFinish
		if bits&svc.U_ALPHA != 0 {
			v, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			e.Alpha = v
		}
		if bits&svc.U_SCALE != 0 {
			// RMQ, currenty ignored
			_, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
		}
		if bits&svc.U_FRAME2 != 0 {
			v, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			e.Frame |= int(v) << 8
		}
		if bits&svc.U_MODEL2 != 0 {
			v, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			modNum |= int(v) << 8
		}
		if bits&svc.U_LERPFINISH != 0 {
			v, err := cls.inMessage.ReadByte()
			if err != nil {
				return err
			}
			e.LerpFinish = e.MsgTime + float64(v)/255
			e.LerpFlags |= lerpFinish
		}
	case protocol.NetQuake:
		if bits&svc.U_TRANS != 0 {
			// HACK: if this bit is set, assume this is protocol NEHAHRA
			a, err := cls.inMessage.ReadFloat32()
			if err != nil {
				return err
			}
			b, err := cls.inMessage.ReadFloat32() // alpha
			if err != nil {
				return err
			}
			if a == 2 {
				// fullbright (not using this yet)
				_, err := cls.inMessage.ReadFloat32()
				if err != nil {
					return err
				}
			}
			if b == 0 {
				e.Alpha = 0
			} else {
				e.Alpha = byte(math.Round(math.Clamp32(1, b*254+1, 255)))
			}
		}
	}

	if modNum > 0 && modNum <= len(cl.modelPrecache) {
		model := cl.modelPrecache[modNum-1] // server sends this 1 based, modelPrecache is 0 based
		if model != e.Model {
			e.Model = model
			// automatic animation (torches, etc) can be either all together or randomized
			if model != nil {
				// TODO(therjak):
				// synctype only set for alias and sprite models
				//			if model.SyncType == ST_RAND {
				//				e.SyncBase = float32(rand()&0x7fff / 0x7fff)
				//			}
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
	return nil
}

func (c *Client) ParseStatic(version int) {
	ent := c.CreateStaticEntity()
	CL_ParseBaseline(ent, version)
	// copy it to the current state

	ent.Model = c.modelPrecache[ent.Baseline.ModelIndex-1]
	ent.LerpFlags |= lerpResetAnim // TODO(therjak): shouldn't this be an override instead of an OR?
	ent.Frame = int(ent.Baseline.Frame)
	ent.SkinNum = int(ent.Baseline.Skin)
	ent.Effects = 0
	ent.Alpha = ent.Baseline.Alpha
	ent.Origin = ent.Baseline.Origin
	ent.Angles = ent.Baseline.Angles
	ent.ParseStaticC(int(ent.Baseline.ModelIndex))
	ent.Sync()

	ent.R_AddEfrags() // clean up after removal of c-efrags
}
