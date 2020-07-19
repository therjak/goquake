package quakelib

//void CL_ParseUpdate(int bits);
//void CL_ParseServerInfo(void);
//void CL_NewTranslation(int slot);
//void CL_ParseStatic(int version);
//void R_CheckEfrags(void);
//void Fog_Update(float density, float red, float green, float blue, float time);
import "C"

import (
	"fmt"

	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/execute"
	"github.com/therjak/goquake/math/vec"
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

//export CL_ParseBaseline
func CL_ParseBaseline(i, version int) {
	var err error
	// must use CL_EntityNum() to force cl.num_entities up
	e := cl.EntityNum(i)
	es := &EntityState{
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

	e.SetBaseline(es)
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

//export CL_ParseServerMessage
func CL_ParseServerMessage() {
	//  int cmd;
	//  int i;
	//  const char *str;        // johnfitz
	//  int total, j, lastcmd;  // johnfitz

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
			C.CL_ParseUpdate(C.int(cmd & 127))
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
			case protocol.NetQuake, protocol.FitzQuake, protocol.RMQ:
			default:
				HostError("Server returned version %i, not %i or %i or %i", i,
					protocol.NetQuake, protocol.FitzQuake, protocol.RMQ)
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
			C.CL_ParseServerInfo()
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
			Sbar_Changed()
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
			Sbar_Changed()
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
			Sbar_Changed()
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
			C.CL_NewTranslation(C.int(i))

		case svc.Particle:
			var dir vec.Vec3
			//int i, count, msgcount, color;
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
			CL_ParseBaseline(int(i), 1)

		case svc.SpawnStatic:
			C.CL_ParseStatic(1)

		case svc.TempEntity:
			CL_ParseTEnt()

		case svc.SetPause:
			// therjak: this byte was used to pause cd audio
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
			// check for excessive static ents and efrags
			if i == 2 {
				if cl.numStatics > 128 {
					conlog.DWarning("%i static entities exceeds standard limit of 128.\n", cl.numStatics)
				}
				C.R_CheckEfrags()
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
				density := C.float(data.Density) / 255.0
				red := C.float(data.Red) / 255.0
				green := C.float(data.Green) / 255.0
				blue := C.float(data.Blue) / 255.0
				time := C.float(data.Time) / 100.0
				if time < 0 {
					time = 0
				}
				C.Fog_Update(density, red, green, blue, time)
			}
		case svc.SpawnBaseline2:
			i, err := cls.inMessage.ReadInt16()
			if err != nil {
				cls.msgBadRead = true
				continue
			}
			CL_ParseBaseline(int(i), 2)

		case svc.SpawnStatic2:
			C.CL_ParseStatic(2)

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
