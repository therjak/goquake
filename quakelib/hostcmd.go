// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"log"
	"strings"
	"time"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/execute"
	"goquake/filesystem"
	"goquake/keys"
	"goquake/net"
	"goquake/progs"
	"goquake/protocol"
	svc "goquake/protocol/server"
	"goquake/protos"
	"goquake/qtime"
)

func init() {
	addClientCommand("begin", hostBegin)
	addClientCommand("color", hostColor)
	addClientCommand("fly", hostFly)
	addClientCommand("give", hostGive)
	addClientCommand("god", hostGod)
	addClientCommand("kick", hostKick)
	addClientCommand("kill", hostKill)
	addClientCommand("name", hostName)
	addClientCommand("noclip", hostNoClip)
	addClientCommand("notarget", hostNoTarget)
	addClientCommand("pause", hostPause)
	addClientCommand("ping", hostPing)
	addClientCommand("prespawn", hostPreSpawn)
	addClientCommand("say", hostSayAll)
	addClientCommand("say_team", hostSayTeam)
	addClientCommand("setpos", hostSetPos)
	addClientCommand("spawn", hostSpawn)
	addClientCommand("status", hostStatus)
	addClientCommand("tell", hostTell)
	addCommand("changelevel", hostChangelevel)
	addCommand("connect", hostConnect)
	addCommand("map", hostMap)
	addCommand("mapname", hostMapName)
	addCommand("quit", func(a []cmd.QArg, p, s int) error { return hostQuit() })
	addCommand("restart", hostRestart)
	addCommand("version", hostVersion)
}

func hostQuit() error {
	if keyDestination != keys.Console && !cmdl.Dedicated() {
		enterQuitMenu()
		return nil
	}
	if err := cls.Disconnect(); err != nil {
		return err
	}

	if err := hostShutdownServer(false); err != nil {
		return err
	}

	Sys_Quit()
	return nil
}

func qFormatI(b int32) string {
	if b == 0 {
		return "OFF"
	}
	return "ON"
}

func hostVersion(args []cmd.QArg, p, s int) error {
	conlog.Printf("GoQuake Version %1.2f.%d\n", GoQuakeVersion, GoQuakePatch)
	return nil
}

func hostPreSpawn(args []cmd.QArg, p, s int) error {
	if s == execute.Command {
		conlog.Printf("prespawn is not valid from the console\n")
		return nil
	}
	c := HostClient()
	if c.spawned {
		conlog.Printf("prespawn not valid -- already spawned\n")
		return nil
	}
	c.msg.WriteBytes(sv.signon.Bytes())
	c.msg.WriteByte(svc.SignonNum)
	c.msg.WriteByte(2)
	c.sendSignon = true
	return nil
}

func hostGod(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("god", args)
		return nil
	}
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(playerEdictId)
	f := int32(ev.Flags)
	const flag = progs.FlagGodMode
	switch len(args) {
	default:
		conlog.Printf("god [value] : toggle god mode. values: 0 = off, 1 = on\n")
		return nil
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
	return nil
}

func hostNoTarget(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("notarget", args)
		return nil
	}
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(playerEdictId)
	f := int32(ev.Flags)
	const flag = progs.FlagNoTarget
	switch len(args) {
	default:
		conlog.Printf("notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n")
		return nil
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
	return nil
}

func hostFly(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("fly", args)
		return nil
	}
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(playerEdictId)
	m := int32(ev.MoveType)
	switch len(args) {
	default:
		conlog.Printf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n")
		return nil
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
	if m == progs.MoveTypeFly {
		HostClient().Printf("flymode %v\n", qFormatI(1))
	} else {
		HostClient().Printf("flymode %v\n", qFormatI(0))
	}
	return nil
}

func hostColor(args []cmd.QArg, p, s int) error {
	c := int(cvars.ClientColor.Value())
	t := c >> 4
	b := c & 0x0f
	if len(args) == 0 {
		conlog.Printf("\"color\" is \"%d %d\"\n", t, b)
		conlog.Printf("color <0-13> [0-13]\n")
		return nil
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
	if s == execute.Command {
		cvars.ClientColor.SetValue(float32(c))
		if cls.state == ca_connected {
			forwardToServer("color", args)
		}
		return nil
	}
	client := HostClient()
	client.colors = c
	entvars.Get(client.edictId).Team = float32(b + 1)
	uc := &protos.UpdateColors{
		Player:   int32(client.id),
		NewColor: int32(c),
	}
	svc.WriteUpdateColors(uc, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
	return nil
}

func hostPause(args []cmd.QArg, playerEdictId, s int) error {
	if cls.demoPlayback {
		cls.demoPaused = !cls.demoPaused
		cl.paused = cls.demoPaused
		return nil
	}
	if s == execute.Command {
		forwardToServer("pause", args)
		return nil
	}
	if cvars.Pausable.String() != "1" {
		HostClient().Printf("Pause not allowed.\n")
		return nil
	}
	sv.paused = !sv.paused

	ev := entvars.Get(playerEdictId)
	playerName, _ := progsdat.String(ev.NetName)
	SV_BroadcastPrintf("%s %s the game\n", playerName, func() string {
		if sv.paused {
			return "paused"
		}
		return "unpaused"
	}())

	svc.WriteSetPause(sv.paused, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
	return nil
}

func hostBegin(args []cmd.QArg, p, s int) error {
	if s == execute.Command {
		conlog.Printf("begin is not valid from the console\n")
		return nil
	}
	HostClient().spawned = true
	return nil
}

func hostGive(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("give", args)
		return nil
	}
	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(playerEdictId)

	if len(args) == 0 {
		return nil
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
	  }
	*/

	// Update currentammo to update statusbar correctly
	switch ev.Weapon {
	case progs.ItemShotgun,
		progs.ItemSuperShotgun:
		ev.CurrentAmmo = ev.AmmoShells
	case progs.ItemNailgun,
		progs.ItemSuperNailgun,
		progs.RogueItemLavaSuperNailgun:
		ev.CurrentAmmo = ev.AmmoNails
	case progs.ItemGrenadeLauncher,
		progs.ItemRocketLauncher,
		progs.RogueItemMultiGrenade,
		progs.RogueItemMultiRocket:
		ev.CurrentAmmo = ev.AmmoRockets
	case progs.ItemLightning,
		progs.HipnoticItemLaserCannon,
		progs.HipnoticItemMjolnir:
		ev.CurrentAmmo = ev.AmmoCells
	case progs.RogueItemLavaNailgun:
		// This is the same as ItemAxe so we need to be more careful
		if cmdl.Rogue() {
			ev.CurrentAmmo = ev.AmmoNails
		}
	case progs.RogueItemPlasmaGun:
		// This is the same as HipnoticItemProximityGun, so be more careful
		if cmdl.Rogue() {
			ev.CurrentAmmo = ev.AmmoCells
		} else if cmdl.Hipnotic() {
			ev.CurrentAmmo = ev.AmmoRockets
		}
	}
	return nil
}

func concatArgs(args []cmd.QArg) string {
	n := len(args)
	for i := 0; i < len(args); i++ {
		n += len(args[i].String())
	}
	var b strings.Builder
	b.Grow(n)
	b.WriteString(args[0].String())
	for _, s := range args[1:] {
		b.WriteString(" ")
		b.WriteString(s.String())
	}
	b.WriteString("\n")
	return b.String()
}

func hostTell(args []cmd.QArg, p, s int) error {
	if s == execute.Command {
		forwardToServer("tell", args)
		return nil
	}

	if len(args) < 2 {
		// need at least destination and message
		return nil
	}

	cn := HostClient().name
	// TODO: should we really concat or use cmd.Full?
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
	return nil
}

func hostSay(team bool, args []cmd.QArg, s int) {
	// we know len(args) >= 1
	fromServer := false
	if s == execute.Command {
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
			entvars.Get(c.edictId).Team != entvars.Get(HostClient().edictId).Team {
			continue
		}
		c.Printf(text)
	}
	if cmdl.Dedicated() {
		log.Printf(text)
	}
}

func hostSayAll(args []cmd.QArg, p, s int) error {
	// say
	if len(args) < 1 {
		return nil
	}
	if s == execute.Command {
		if !cmdl.Dedicated() {
			forwardToServer("say", args)
			return nil
		}
	}
	hostSay(false, args, s)
	return nil
}

func hostSayTeam(args []cmd.QArg, p, s int) error {
	// say_team
	if len(args) < 1 {
		return nil
	}
	if s == execute.Command {
		if !cmdl.Dedicated() {
			forwardToServer("say_team", args)
			return nil
		}
	}
	hostSay(true, args, s)
	return nil
}

func hostPing(args []cmd.QArg, p, s int) error {
	if s == execute.Command {
		forwardToServer("ping", args)
		return nil
	}
	HostClient().Printf("Client ping times:\n")
	for _, c := range sv_clients {
		if !c.active {
			continue
		}
		HostClient().Printf("%4d %s\n", int(c.PingTime()*1000), c.name)
	}
	return nil
}

func hostSpawn(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		conlog.Printf("spawn is not valid from the console\n")
		return nil
	}
	c := HostClient()
	if c.spawned {
		conlog.Printf("Spawn not valid -- already spawned\n")
		return nil
	}
	// run the entrance script
	if sv.loadGame {
		// loaded games are fully inited already
		// if this is the last client to be connected, unpause
		sv.paused = false
	} else {
		entvars.Clear(c.edictId)
		ev := entvars.Get(c.edictId)
		ev.ColorMap = float32(c.edictId)
		ev.Team = float32((c.colors & 15) + 1)
		ev.NetName = progsdat.AddString(c.name)
		progsdat.Globals.Parm = c.spawnParams
		progsdat.Globals.Time = sv.time
		progsdat.Globals.Self = int32(playerEdictId)
		if err := vm.ExecuteProgram(progsdat.Globals.ClientConnect); err != nil {
			return err
		}
		if (qtime.QTime() - c.ConnectTime()).Seconds() <= float64(sv.time) {
			log.Printf("%v entered the game\n", c.name)
		}
		if err := vm.ExecuteProgram(progsdat.Globals.PutClientInServer); err != nil {
			return err
		}
	}

	// send all current names, colors, and frag counts
	c.msg.ClearMessage()

	// send time of update
	svc.WriteTime(sv.time, sv.protocol, sv.protocolFlags, &c.msg)

	for i, sc := range sv_clients {
		if i >= svs.maxClients {
			// TODO: figure out why it ever makes sense to have len(sv_clients) svs.maxClients
			break
		}
		un := &protos.UpdateName{
			Player:  int32(i),
			NewName: sc.name,
		}
		svc.WriteUpdateName(un, sv.protocol, sv.protocolFlags, &c.msg)
		uf := &protos.UpdateFrags{
			Player:   int32(i),
			NewFrags: int32(sc.oldFrags),
		}
		svc.WriteUpdateFrags(uf, sv.protocol, sv.protocolFlags, &c.msg)
		uc := &protos.UpdateColors{
			Player:   int32(i),
			NewColor: int32(sc.colors),
		}
		svc.WriteUpdateColors(uc, sv.protocol, sv.protocolFlags, &c.msg)
	}

	// send all current light styles
	for i, ls := range sv.lightStyles {
		c.msg.WriteByte(svc.LightStyle)
		c.msg.WriteByte(i)
		c.msg.WriteString(ls)
	}

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatTotalSecrets)
	c.msg.WriteLong(int(progsdat.Globals.TotalSecrets))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatTotalMonsters)
	c.msg.WriteLong(int(progsdat.Globals.TotalMonsters))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatSecrets)
	c.msg.WriteLong(int(progsdat.Globals.FoundSecrets))

	c.msg.WriteByte(svc.UpdateStat)
	c.msg.WriteByte(svc.StatMonsters)
	c.msg.WriteLong(int(progsdat.Globals.KilledMonsters))

	// send a fixangle
	// Never send a roll angle, because savegames can catch the server
	// in a state where it is expecting the client to correct the angle
	// and it won't happen if the game was just loaded, so you wind up
	// with a permanent head tilt
	sa := &protos.Coord{
		X: entvars.Get(c.edictId).Angles[0],
		Y: entvars.Get(c.edictId).Angles[1],
		Z: 0,
	}
	svc.WriteSetAngle(sa, sv.protocol, sv.protocolFlags, &c.msg)

	msgBuf.ClearMessage()
	msgBufMaxLen = protocol.MaxDatagram
	sv.WriteClientdataToMessage(playerEdictId)
	c.msg.WriteBytes(msgBuf.Bytes())

	c.msg.WriteByte(svc.SignonNum)
	c.msg.WriteByte(3)
	c.sendSignon = true
	return nil
}

func hostNoClip(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("noclip", args)
		return nil
	}

	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}
	ev := entvars.Get(playerEdictId)
	m := int32(ev.MoveType)
	switch len(args) {
	default:
		conlog.Printf("noclip [value] : toggle noclip mode. values: 0 = off, 1 = on\n")
		return nil
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
	return nil
}

func hostSetPos(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("setpos", args)
		return nil
	}

	if progsdat.Globals.DeathMatch != 0 {
		return nil
	}

	ev := entvars.Get(playerEdictId)
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
		return nil
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

	if err := vm.LinkEdict(playerEdictId, false); err != nil {
		return err
	}
	return nil
}

func hostName(args []cmd.QArg, p, s int) error {
	if len(args) == 0 {
		conlog.Printf("\"name\" is %q\n", cvars.ClientName.String())
		return nil
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

	if s == execute.Command {
		if cvars.ClientName.String() == newName {
			return nil
		}
		cvars.ClientName.SetByString(newName)
		if cls.state == ca_connected {
			forwardToServer("name", args)
		}
		return nil
	}

	c := HostClient()
	if len(c.name) != 0 && c.name != "unconnected" && c.name != newName {
		conlog.Printf("%s renamed to %s\n", c.name, newName)
	}
	c.name = newName
	entvars.Get(c.edictId).NetName = progsdat.AddString(newName)

	// send notification to all clients
	un := &protos.UpdateName{
		Player:  int32(c.id),
		NewName: newName,
	}
	svc.WriteUpdateName(un, sv.protocol, sv.protocolFlags, &sv.reliableDatagram)
	return nil
}

func hostKill(args []cmd.QArg, playerEdictId, s int) error {
	if s == execute.Command {
		forwardToServer("kill", args)
		return nil
	}

	ev := entvars.Get(playerEdictId)

	if ev.Health <= 0 {
		HostClient().Printf("Can't suicide -- already dead!\n")
		return nil
	}

	progsdat.Globals.Time = sv.time
	progsdat.Globals.Self = int32(playerEdictId)
	if err := vm.ExecuteProgram(progsdat.Globals.ClientKill); err != nil {
		return err
	}
	return nil
}

func hostStatus(args []cmd.QArg, p, s int) error {
	const baseVersion = 1.09
	if s == execute.Command {
		if !sv.active {
			forwardToServer("status", args)
			return nil
		}

	}
	printf := func() func(format string, v ...interface{}) {
		if s == execute.Command {
			return conlog.Printf
		}
		return HostClient().Printf
	}()

	printf("host:    %s\n", cvars.HostName.String())
	printf("version: %4.2f\n", baseVersion)
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
		ev := entvars.Get(c.edictId)
		printf("#%-2d %-16.16s  %3d  %9s\n", i+1, c.name, int(ev.Frags), d.String())
		printf("   %s\n", c.Address())
	}
	return nil
}

func hostMapName(args []cmd.QArg, p, s int) error {
	if sv.active {
		conlog.Printf("%q is %q\n", "mapname", sv.name)
		return nil
	}
	if cls.state == ca_connected {
		// TODO
		// conlog.Printf("%q is %q\n", "mapname", cl.mapname)
		return nil
	}
	conlog.Printf("no map loaded\n")
	return nil
}

// This only happens at the end of a game, not between levels
func hostShutdownServer(crash bool) error {
	if !sv.active {
		return nil
	}

	sv.active = false

	// stop all client sounds immediately
	if cls.state == ca_connected {
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}

	// flush any pending messages - like the score!!!
	start := qtime.QTime()
	count := 1
	for count != 0 {
		count = 0
		for _, c := range sv_clients {
			if c.active && c.msg.HasMessage() {
				if c.CanSendMessage() {
					c.SendMessage()
					c.msg.ClearMessage()
				} else {
					if err := c.GetMessage(); err != nil {
						return err
					}
					count++
				}
			}
		}
		if (qtime.QTime() - start).Seconds() > 3.0 {
			break
		}
	}

	// make sure all the clients know we're disconnecting
	SendToAll([]byte{svc.Disconnect})

	for _, c := range sv_clients {
		if c.active {
			if err := c.Drop(crash); err != nil {
				return nil
			}
		}
	}

	sv.worldModel = nil

	CreateSVClients()
	return nil
}

// Kicks a user off of the server
func hostKick(args []cmd.QArg, playerEdictId, s int) error {
	if len(args) == 0 {
		return nil
	}
	if s == execute.Command {
		if !sv.active {
			forwardToServer("kick", args)
			return nil
		}
	} else if progsdat.Globals.DeathMatch != 0 {
		return nil
	}

	var toKick *SVClient
	var message string

	if len(args) > 1 && args[0].String() == "#" {
		i := args[1].Int() - 1
		if i < 0 || i >= svs.maxClients {
			return nil
		}
		toKick = sv_clients[i]
		if !toKick.active {
			return nil
		}
		if len(args) > 2 {
			// skip # and number
			message = concatArgs(args[2:])
		}
	} else {
		for _, c := range sv_clients {
			if !c.active {
				continue
			}
			if c.name == args[0].String() {
				toKick = c
				if len(args) > 1 {
					// skip name
					message = concatArgs(args[1:])
				}
				break
			}
		}
	}
	if toKick == nil {
		return nil
	}
	if playerEdictId == toKick.edictId {
		// can't kick yourself!
		return nil
	}
	who := func() string {
		if s == execute.Command {
			if cmdl.Dedicated() {
				return "Console"
			} else {
				return cvars.ClientName.String()
			}
		}
		return HostClient().name
	}()

	if message != "" {
		toKick.Printf("Kicked by %s: %s\n", who, message)
	} else {
		toKick.Printf("Kicked by %s\n", who)
	}
	if err := toKick.Drop(false); err != nil {
		return err
	}
	return nil
}

// User command to connect to server
func hostConnect(args []cmd.QArg, p, s int) error {
	if len(args) == 0 {
		return nil
	}
	// stop demo loop in case this fails
	cls.demoNum = -1
	if cls.demoPlayback {
		cls.demoPlayback = false
		if err := cls.Disconnect(); err != nil {
			return err
		}
	}
	if err := clEstablishConnection(args[0].String()); err != nil {
		return err
	}
	clientReconnect()
	return nil
}

// handle a
// map <servername>
// command from the console.  Active clients are kicked off.
func hostMap(args []cmd.QArg, p, s int) error {
	if len(args) == 0 {
		// no map name given
		if cmdl.Dedicated() {
			if sv.active {
				conlog.Printf("Current map: %s\n", sv.name)
			} else {
				conlog.Printf("Server not active\n")
			}
		} else if cls.state == ca_connected {
			// conlog.Printf("Current map: %s ( %s )\n", cl.levelname, cl.mapname);
		} else {
			conlog.Printf("map <levelname>: start a new server\n")
		}
		return nil
	}

	if s != execute.Command {
		return nil
	}

	// stop demo loop in case this fails
	cls.demoNum = -1

	if err := cls.Disconnect(); err != nil {
		return err
	}
	if err := hostShutdownServer(false); err != nil {
		return err
	}

	if !cmdl.Dedicated() {
		inputActivate()
	}

	keyDestination = keys.Game // remove console or menu
	screen.BeginLoadingPlaque()

	svs.serverFlags = 0 // haven't completed an episode yet

	mapName := args[0].String()
	mapName = strings.TrimSuffix(mapName, ".bsp")

	if err := sv.SpawnServer(mapName); err != nil {
		return err
	}
	if !sv.active {
		return nil
	}

	if !cmdl.Dedicated() {
		var b strings.Builder
		for _, a := range args[1:] {
			b.WriteString(a.String())
			b.WriteRune(' ')
		}
		cls.spawnParms = b.String()

		if err := clEstablishConnection("local"); err != nil {
			return err
		}
		clientReconnect()
	}
	return nil
}

// Goes to a new map, taking all clients along
func hostChangelevel(args []cmd.QArg, p, s int) error {
	if len(args) != 1 {
		conlog.Printf("changelevel <levelname> : continue game on a new level\n")
		return nil
	}

	if cls.demoPlayback || !sv.active {
		conlog.Printf("Only the server may changelevel\n")
		return nil
	}
	level := args[0].String()
	if _, err := filesystem.GetFile(fmt.Sprintf("maps/%s.bsp", level)); err != nil {
		return fmt.Errorf("cannot find map %s", level)
	}
	if !cmdl.Dedicated() {
		inputActivate()
	}

	// remove console or menu
	keyDestination = keys.Game
	if err := SV_SaveSpawnparms(); err != nil {
		return err
	}
	if err := sv.SpawnServer(level); err != nil {
		return err
	}
	// also issue an error if spawn failed -- O.S.
	if !sv.active {
		return fmt.Errorf("cannot run map %s", level)
	}
	return nil
}

// Restarts the current server for a dead player
func hostRestart(args []cmd.QArg, p, s int) error {
	if cls.demoPlayback || !sv.active {
		return nil
	}
	if s != execute.Command {
		return nil
	}
	mapname := sv.name // sv.name gets cleared in spawnserver
	if err := sv.SpawnServer(mapname); err != nil {
		return err
	}

	if !sv.active {
		return fmt.Errorf("cannot restart map %s", mapname)
	}
	return nil
}
