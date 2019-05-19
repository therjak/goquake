package quakelib

import (
	"fmt"
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/net"
	"quake/progs"
	"quake/protocol"
	"quake/protocol/server"
	"quake/qtime"
	"strings"
	"time"
)

func init() {
	cmd.AddCommand("begin", hostBegin)
	cmd.AddCommand("color", hostColor)
	cmd.AddCommand("fly", hostFly)
	cmd.AddCommand("give", hostGive)
	cmd.AddCommand("god", hostGod)
	cmd.AddCommand("kill", hostKill)
	cmd.AddCommand("name", hostName)
	cmd.AddCommand("noclip", hostNoClip)
	cmd.AddCommand("notarget", hostNoTarget)
	cmd.AddCommand("pause", hostPause)
	cmd.AddCommand("ping", hostPing)
	cmd.AddCommand("status", hostStatus)
	cmd.AddCommand("say", hostSayAll)
	cmd.AddCommand("say_team", hostSayTeam)
	cmd.AddCommand("setpos", hostSetPos)
	cmd.AddCommand("spawn", hostSpawn)
	cmd.AddCommand("tell", hostTell)
	cmd.AddCommand("mapname", hostMapName)
	cmd.AddCommand("prespawn", hostPreSpawn)
}

func qFormatI(b int32) string {
	if b == 0 {
		return "OFF"
	}
	return "ON"
}

func hostPreSpawn(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		conlog.Printf("prespawn is not valid from the console\n")
		return
	}
	c := HostClient()
	if c.spawned {
		conlog.Printf("prespawn not valid -- already spawned\n")
		return
	}
	c.msg.WriteBytes(sv.signon.Bytes())
	c.msg.WriteByte(server.SignonNum)
	c.msg.WriteByte(2)
	c.sendSignon = true
}

func hostGod(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("god", args)
		return
	}
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := EntVars(sv_player)
	f := int32(ev.Flags)
	const flag = progs.FlagGodMode
	switch len(args) {
	default:
		conlog.Printf("god [value] : toggle god mode. values: 0 = off, 1 = on\n")
		return
	case 0:
		f = f ^ flag
	case 1:
		if args[0].Bool() {
			f = f | flag
		} else {
			f = f &^ flag
		}
	}
	ev.Flags = float32(f)
	HostClient().Printf("godmode %v\n", qFormatI(f&flag))
}

func hostNoTarget(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("notarget", args)
		return
	}
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := EntVars(sv_player)
	f := int32(ev.Flags)
	const flag = progs.FlagNoTarget
	switch len(args) {
	default:
		conlog.Printf("notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n")
		return
	case 0:
		f = f ^ flag
	case 1:
		if args[0].Bool() {
			f = f | flag
		} else {
			f = f &^ flag
		}
	}
	ev.Flags = float32(f)
	HostClient().Printf("notarget %v\n", qFormatI(f&flag))
}

func hostFly(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("fly", args)
		return
	}
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := EntVars(sv_player)
	m := int32(ev.MoveType)
	switch len(args) {
	default:
		conlog.Printf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n")
		return
	case 0:
		if m != progs.MoveTypeFly {
			m = progs.MoveTypeFly
		} else {
			m = progs.MoveTypeWalk
		}
	case 1:
		if args[0].Bool() {
			m = progs.MoveTypeFly
		} else {
			m = progs.MoveTypeWalk
		}
	}
	ev.MoveType = float32(m)
	HostClient().Printf("flymode %v\n", qFormatI(m&progs.MoveTypeFly))
}

func hostColor(args []cmd.QArg) {
	c := int(cvars.ClientColor.Value())
	t := c >> 4
	b := c & 0x0f
	if len(args) == 0 {
		conlog.Printf("\"color\" is \"%d %d\"\n", t, b)
		conlog.Printf("color <0-13> [0-13]\n")
		return
	}
	t = args[0].Int()
	b = t
	if len(args) > 1 {
		b = args[1].Int()
	}

	t &= 0x0f
	if t > 13 {
		t = 13
	}
	b &= 0x0f
	if b > 13 {
		b = 13
	}
	c = t*16 + b
	if execute.IsSrcCommand() {
		cvars.ClientColor.SetValue(float32(c))
		if cls.state == ca_connected {
			forwardToServer("color", args)
		}
		return
	}
	cID := HostClientID()
	client := HostClient()
	client.colors = c
	EntVars(client.edictId).Team = float32(b + 1)
	sv.reliableDatagram.WriteByte(server.UpdateColors)
	sv.reliableDatagram.WriteByte(cID)
	sv.reliableDatagram.WriteByte(c)
}

func hostPause(args []cmd.QArg) {
	if cls.demoPlayback {
		cls.demoPaused = !cls.demoPaused
		cl.paused = cls.demoPaused
		return
	}
	if execute.IsSrcCommand() {
		forwardToServer("pause", args)
		return
	}
	if cvars.Pausable.String() != "1" {
		HostClient().Printf("Pause not allowed.\n")
		return
	}
	sv.paused = !sv.paused

	ev := EntVars(sv_player)
	playerName := *PRGetString(int(ev.NetName))
	SV_BroadcastPrintf("%s %s the game\n", playerName, func() string {
		if sv.paused {
			return "paused"
		}
		return "unpaused"
	}())

	sv.reliableDatagram.WriteByte(server.SetPause)
	sv.reliableDatagram.WriteByte(func() int {
		if sv.paused {
			return 1
		}
		return 0
	}())
}

func hostBegin(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		conlog.Printf("begin is not valid from the console\n")
		return
	}
	HostClient().spawned = true
}

func hostGive(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("give", args)
		return
	}
	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := EntVars(sv_player)

	if len(args) == 0 {
		return
	}

	t := args[0].String()

	v := float32(0)
	if len(args) > 1 {
		v = float32(args[1].Int())
	}

	switch t[0] {
	case byte('0'):
	case byte('1'):
	case byte('2'):
		ev.Items = float32(int32(ev.Items) | progs.ItemShotgun)
	case byte('3'):
		ev.Items = float32(int32(ev.Items) | progs.ItemSuperShotgun)
	case byte('4'):
		ev.Items = float32(int32(ev.Items) | progs.ItemNailgun)
	case byte('5'):
		ev.Items = float32(int32(ev.Items) | progs.ItemSuperNailgun)
	case byte('6'):
		ev.Items = float32(int32(ev.Items) | progs.ItemGrenadeLauncher)
	case byte('7'):
		ev.Items = float32(int32(ev.Items) | progs.ItemRocketLauncher)
	case byte('8'):
		ev.Items = float32(int32(ev.Items) | progs.ItemLightning)
	case byte('9'):
	case byte('s'):
		ev.AmmoShells = v
	case byte('n'):
		ev.AmmoNails = v
	case byte('r'):
		ev.AmmoRockets = v
	case byte('h'):
		ev.Health = v
	case byte('c'):
		ev.AmmoCells = v
	case byte('a'):
		if v > 150 {
			ev.ArmorType = 0.8
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor1 | progs.ItemArmor2)) | progs.ItemArmor3)
		} else if v > 100 {
			ev.ArmorType = 0.6
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor1 | progs.ItemArmor3)) | progs.ItemArmor2)
		} else if v >= 0 {
			ev.ArmorType = 0.3
			ev.ArmorValue = v
			ev.Items = float32((int32(ev.Items) &^ (progs.ItemArmor2 | progs.ItemArmor3)) | progs.ItemArmor1)
		}

	}
	/*
	  switch (t[0]) {
	    case '0':
	    case '1':
	    case '2':
	    case '3':
	    case '4':
	    case '5':
	    case '6':
	    case '7':
	    case '8':
	    case '9':
	      // MED 01/04/97 added hipnotic give stuff
	      if (CMLHipnotic()) {
	        if (t[0] == '6') {
	          if (t[1] == 'a')
	            pent->items = (int)pent->items | HIT_PROXIMITY_GUN;
	          else
	            pent->items = (int)pent->items | IT_GRENADE_LAUNCHER;
	        } else if (t[0] == '9')
	          pent->items = (int)pent->items | HIT_LASER_CANNON;
	        else if (t[0] == '0')
	          pent->items = (int)pent->items | HIT_MJOLNIR;
	        else if (t[0] >= '2')
	          pent->items = (int)pent->items | (IT_SHOTGUN << (t[0] - '2'));
	      } else {
	        if (t[0] >= '2')
	          pent->items = (int)pent->items | (IT_SHOTGUN << (t[0] - '2'));
	      }
	      break;

	    case 's':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_shells1");
	        if (val) val->_float = v;
	      }
	      pent->ammo_shells = v;
	      break;

	    case 'n':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_nails1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_nails = v;
	        }
	      } else {
	        pent->ammo_nails = v;
	      }
	      break;

	    case 'l':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_lava_nails");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_nails = v;
	        }
	      }
	      break;

	    case 'r':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_rockets1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_rockets = v;
	        }
	      } else {
	        pent->ammo_rockets = v;
	      }
	      break;

	    case 'm':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_multi_rockets");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_rockets = v;
	        }
	      }
	      break;

	    case 'h':
	      pent->health = v;
	      break;

	    case 'c':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_cells1");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon <= IT_LIGHTNING) pent->ammo_cells = v;
	        }
	      } else {
	        pent->ammo_cells = v;
	      }
	      break;

	    case 'p':
	      if (CMLRogue()) {
	        val = GetEdictFieldValue(pent, "ammo_plasma");
	        if (val) {
	          val->_float = v;
	          if (pent->weapon > IT_LIGHTNING) pent->ammo_cells = v;
	        }
	      }
	      break;

	    // johnfitz -- give armour
	    case 'a':
	      if (v > 150) {
	        pent->armortype = 0.8;
	        pent->armorvalue = v;
	        pent->items =
	            pent->items -
	            ((int)(pent->items) & (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
	            IT_ARMOR3;
	      } else if (v > 100) {
	        pent->armortype = 0.6;
	        pent->armorvalue = v;
	        pent->items =
	            pent->items -
	            ((int)(pent->items) & (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
	            IT_ARMOR2;
	      } else if (v >= 0) {
	        pent->armortype = 0.3;
	        pent->armorvalue = v;
	        pent->items =
	            pent->items -
	            ((int)(pent->items) & (int)(IT_ARMOR1 | IT_ARMOR2 | IT_ARMOR3)) +
	            IT_ARMOR1;
	      }
	      break;
	      // johnfitz
	  }

	  // johnfitz -- update currentammo to match new ammo (so statusbar updates
	  // correctly)
	  switch ((int)(pent->weapon)) {
	    case IT_SHOTGUN:
	    case IT_SUPER_SHOTGUN:
	      pent->currentammo = pent->ammo_shells;
	      break;
	    case IT_NAILGUN:
	    case IT_SUPER_NAILGUN:
	    case RIT_LAVA_SUPER_NAILGUN:
	      pent->currentammo = pent->ammo_nails;
	      break;
	    case IT_GRENADE_LAUNCHER:
	    case IT_ROCKET_LAUNCHER:
	    case RIT_MULTI_GRENADE:
	    case RIT_MULTI_ROCKET:
	      pent->currentammo = pent->ammo_rockets;
	      break;
	    case IT_LIGHTNING:
	    case HIT_LASER_CANNON:
	    case HIT_MJOLNIR:
	      pent->currentammo = pent->ammo_cells;
	      break;
	    case RIT_LAVA_NAILGUN:  // same as IT_AXE
	      if (CMLRogue()) pent->currentammo = pent->ammo_nails;
	      break;
	    case RIT_PLASMA_GUN:  // same as HIT_PROXIMITY_GUN
	      if (CMLRogue()) pent->currentammo = pent->ammo_cells;
	      if (CMLHipnotic()) pent->currentammo = pent->ammo_rockets;
	      break;
	  }
	  // johnfitz
	*/

	// Update currentammo to update statusbar correctly
	switch int(ev.Weapon) {
	case progs.ItemShotgun, progs.ItemSuperShotgun:
		ev.CurrentAmmo = ev.AmmoShells
	case progs.ItemNailgun, progs.ItemSuperNailgun:
		ev.CurrentAmmo = ev.AmmoNails
	case progs.ItemGrenadeLauncher, progs.ItemRocketLauncher:
		ev.CurrentAmmo = ev.AmmoRockets
	case progs.ItemLightning:
		ev.CurrentAmmo = ev.AmmoCells
	}
}

func concatArgs(args []cmd.QArg) string {
	n := len(args)
	for i := 0; i < len(args); i++ {
		n += len(args[i].String())
	}

	b := make([]byte, n)
	bp := copy(b, args[0].String())
	for _, s := range args[1:] {
		bp += copy(b[bp:], " ")
		bp += copy(b[bp:], s.String())
	}
	bp += copy(b[bp:], "\n")
	return string(b)
}

func hostTell(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("tell", args)
		return
	}

	if len(args) < 2 {
		// need at least destination and message
		return
	}

	cn := HostClient().name
	// TODO: should we realy concat or use cmd.CmdArgs?
	ms := concatArgs(args[1:])
	text := fmt.Sprintf("%s: %s", cn, ms)

	for _, c := range sv_clients {
		if !c.active || !c.spawned {
			continue
		}
		if strings.ToLower(c.name) != strings.ToLower(args[0].String()) {
			continue
		}
		// TODO: We check without case check. Are names unique ignoring the case?
		c.Printf(text)
	}
}

func hostSay(team bool, args []cmd.QArg) {
	// we know len(args) >= 1
	fromServer := false
	if execute.IsSrcCommand() {
		team = false
		fromServer = true
	}
	ms := concatArgs(args)
	text := func() string {
		if fromServer {
			return fmt.Sprintf("\001<%s> %s", cvars.HostName.String(), ms)
		} else {
			return fmt.Sprintf("\001%s: %s", HostClient().name, ms)
		}
	}()
	for _, c := range sv_clients {
		if !c.active || !c.spawned {
			continue
		}
		if team && cvars.TeamPlay.Bool() &&
			EntVars(c.edictId).Team != EntVars(HostClient().edictId).Team {
			continue
		}
		c.Printf(text)
	}
	if cls.state == ca_dedicated {
		log.Printf(text)
	}
}

func hostSayAll(args []cmd.QArg) {
	// say
	if len(args) < 1 {
		return
	}
	if execute.IsSrcCommand() {
		if cls.state != ca_dedicated {
			forwardToServer("say", args)
			return
		}
	}
	hostSay(false, args)
}

func hostSayTeam(args []cmd.QArg) {
	// say_team
	if len(args) < 1 {
		return
	}
	if execute.IsSrcCommand() {
		if cls.state != ca_dedicated {
			forwardToServer("say_team", args)
			return
		}
	}
	hostSay(true, args)
}

func hostPing(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("ping", args)
		return
	}
	HostClient().Printf("Client ping times:\n")
	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		HostClient().Printf("%4d %s\n", int(c.PingTime()*1000), c.name)
	}
}

func findViewThingEV() *progs.EntVars {
	for i := 0; i < sv.numEdicts; i++ {
		ev := EntVars(i)
		name := PRGetString(int(ev.ClassName))
		if name != nil && *name == "viewthing" {
			return ev
		}
	}
	conlog.Printf("No viewthing on map\n")
	return nil
}

func hostSpawn(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		conlog.Printf("spawn is not valid from the console\n")
		return
	}
	c := HostClient()
	if c.spawned {
		conlog.Printf("Spawn not valid -- allready spawned\n")
		return
	}
	// run the entrance script
	if sv.loadGame {
		// loaded games are fully inited allready
		// if this is the last client to be connected, unpause
		sv.paused = false
	} else {
		TTClearEntVars(c.edictId)
		ev := EntVars(c.edictId)
		ev.ColorMap = float32(c.edictId)
		ev.Team = float32((c.colors & 15) + 1)
		ev.NetName = int32(PRSetEngineString(c.name))
		progsdat.Globals.Parm = c.spawnParams
		progsdat.Globals.Time = sv.time
		progsdat.Globals.Self = int32(sv_player)
		PRExecuteProgram(progsdat.Globals.ClientConnect)
		if (qtime.QTime() - c.ConnectTime()).Seconds() <= float64(sv.time) {
			log.Printf("%v entered the game\n", c.name)
		}
		PRExecuteProgram(progsdat.Globals.PutClientInServer)
	}

	// send all current names, colors, and frag counts
	c.msg.ClearMessage()

	// send time of update
	c.msg.WriteByte(server.Time)
	c.msg.WriteFloat(sv.time)

	for i, sc := range sv_clients {
		if i >= svs.maxClients {
			// TODO: figure out why it ever makes sense to have len(sv_clients) svs.maxClients
			break
		}
		c.msg.WriteByte(server.UpdateName)
		c.msg.WriteByte(i)
		c.msg.WriteString(sc.name)
		c.msg.WriteByte(server.UpdateFrags)
		c.msg.WriteByte(i)
		c.msg.WriteShort(sc.oldFrags)
		c.msg.WriteByte(server.UpdateColors)
		c.msg.WriteByte(i)
		c.msg.WriteByte(sc.colors)
	}

	// send all current light styles
	for i, ls := range sv.lightStyles {
		c.msg.WriteByte(server.LightStyle)
		c.msg.WriteByte(i)
		c.msg.WriteString(ls)
	}

	c.msg.WriteByte(server.UpdateStat)
	c.msg.WriteByte(server.StatTotalSecrets)
	c.msg.WriteLong(int(progsdat.Globals.TotalSecrets))

	c.msg.WriteByte(server.UpdateStat)
	c.msg.WriteByte(server.StatTotalMonsters)
	c.msg.WriteLong(int(progsdat.Globals.TotalMonsters))

	c.msg.WriteByte(server.UpdateStat)
	c.msg.WriteByte(server.StatSecrets)
	c.msg.WriteLong(int(progsdat.Globals.FoundSecrets))

	c.msg.WriteByte(server.UpdateStat)
	c.msg.WriteByte(server.StatMonsters)
	c.msg.WriteLong(int(progsdat.Globals.KilledMonsters))

	// send a fixangle
	// Never send a roll angle, because savegames can catch the server
	// in a state where it is expecting the client to correct the angle
	// and it won't happen if the game was just loaded, so you wind up
	// with a permanent head tilt
	cid := HostClientID() + 1
	c.msg.WriteByte(server.SetAngle)
	c.msg.WriteAngle(EntVars(cid).Angles[0], int(sv.protocolFlags))
	c.msg.WriteAngle(EntVars(cid).Angles[1], int(sv.protocolFlags))
	c.msg.WriteAngle(0, int(sv.protocolFlags))

	msgBuf.ClearMessage()
	msgBufMaxLen = protocol.MaxDatagram
	sv.WriteClientdataToMessage(EntVars(sv_player), EntityAlpha(sv_player))
	c.msg.WriteBytes(msgBuf.Bytes())

	c.msg.WriteByte(server.SignonNum)
	c.msg.WriteByte(3)
	c.sendSignon = true
}

func hostNoClip(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("noclip", args)
		return
	}

	if progsdat.Globals.DeathMatch != 0 {
		return
	}
	ev := EntVars(sv_player)
	m := int32(ev.MoveType)
	switch len(args) {
	default:
		conlog.Printf("noclip [value] : toggle noclip mode. values: 0 = off, 1 = on\n")
		return
	case 0:
		if m != progs.MoveTypeNoClip {
			m = progs.MoveTypeNoClip
		} else {
			m = progs.MoveTypeWalk
		}
	case 1:
		if args[0].Bool() {
			m = progs.MoveTypeNoClip
		} else {
			m = progs.MoveTypeWalk
		}
	}
	ev.MoveType = float32(m)
	HostClient().Printf("noclip %v\n", qFormatI(m&progs.MoveTypeNoClip))
}

func hostSetPos(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("setpos", args)
		return
	}

	if progsdat.Globals.DeathMatch != 0 {
		return
	}

	ev := EntVars(sv_player)
	if len(args) != 6 && len(args) != 3 {
		c := HostClient()
		c.Printf("usage:\n")
		c.Printf("   setpos <x> <y> <z>\n")
		c.Printf("   setpos <x> <y> <z> <pitch> <yaw> <roll>\n")
		c.Printf("current values:\n")
		c.Printf("   %d %d %d %d %d %d\n",
			int(ev.Origin[0]), int(ev.Origin[1]),
			int(ev.Origin[2]), int(ev.VAngle[0]),
			int(ev.VAngle[1]), int(ev.VAngle[2]))
		return
	}

	m := int32(ev.MoveType)
	if m != progs.MoveTypeNoClip {
		ev.MoveType = float32(progs.MoveTypeNoClip)
		HostClient().Printf("noclip ON\n")
	}
	// make sure they're not going to whizz away from it
	ev.Velocity = [3]float32{0, 0, 0}

	ev.Origin = [3]float32{
		args[0].Float32(),
		args[1].Float32(),
		args[2].Float32(),
	}

	if len(args) == 6 {
		ev.Angles = [3]float32{
			args[3].Float32(),
			args[4].Float32(),
			args[5].Float32(),
		}
		ev.FixAngle = 1
	}

	LinkEdict(sv_player, false)
}

func hostName(args []cmd.QArg) {
	if len(args) == 0 {
		conlog.Printf("\"name\" is %q\n", cvars.ClientName.String())
		return
	}
	newName := func() string {
		if len(args) == 1 {
			return args[0].String()
		}
		b := strings.Builder{}
		b.WriteString(args[0].String())
		for _, a := range args[1:] {
			b.WriteRune(' ')
			b.WriteString(a.String())
		}
		return b.String()
	}()
	// client_t structure says name[32]
	if len(newName) > 15 {
		newName = newName[:15]
	}

	if execute.IsSrcCommand() {
		if cvars.ClientName.String() == newName {
			return
		}
		cvars.ClientName.SetByString(newName)
		if cls.state == ca_connected {
			forwardToServer("name", args)
		}
		return
	}

	c := HostClient()
	if len(c.name) != 0 && c.name != "unconnected" && c.name != newName {
		conlog.Printf("%s renamed to %s\n", c.name, newName)
	}
	c.name = newName
	EntVars(c.edictId).NetName = int32(PRSetEngineString(newName))

	// send notification to all clients
	rd := &sv.reliableDatagram
	rd.WriteByte(server.UpdateName)
	rd.WriteByte(HostClientID())
	rd.WriteString(newName)
}

func hostKill(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("kill", args)
		return
	}

	ev := EntVars(sv_player)

	if ev.Health <= 0 {
		HostClient().Printf("Can't suicide -- allready dead!\n")
		return
	}

	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = int32(sv_player)
	PRExecuteProgram(progsdat.Globals.ClientKill)
}

func hostStatus(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		if !sv.active {
			forwardToServer("status", args)
			return
		}

	}
	printf := func() func(format string, v ...interface{}) {
		if execute.IsSrcCommand() {
			return conlog.Printf
		}
		return HostClient().Printf
	}()

	printf("host:    %s\n", cvars.HostName.String())
	printf("version: %4.2f\n", VERSION)
	printf("tcp/ip:  %s\n", net.Address())
	printf("map:     %s\n", sv.name)
	active := 0
	for _, c := range sv_clients {
		if c.active {
			active++
		}
	}
	printf("players: %d active (%d max)\n\n", active, svs.maxClients)
	ntime := net.Time()
	for i, c := range sv_clients {
		if !c.active {
			continue
		}
		d := ntime - c.ConnectTime()
		d = d.Truncate(time.Second)
		ev := EntVars(c.edictId)
		printf("#%-2d %-16.16s  %3d  %9s\n", i+1, c.name, int(ev.Frags), d.String())
		printf("   %s\n", c.Address())
	}
}

func hostMapName(args []cmd.QArg) {
	if sv.active {
		conlog.Printf("%q is %q\n", "mapname", sv.name)
		return
	}
	if cls.state == ca_connected {
		// TODO
		// conlog.Printf("%q is %q\n", "mapname", cl.mapname)
		return
	}
	conlog.Printf("no map loaded\n")
}
