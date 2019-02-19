package quakelib

//int HostClient(void);
import "C"

import (
	"fmt"
	"quake/cmd"
	"quake/cvars"
	"quake/execute"
	"quake/progs"
	"quake/protocol/server"
	"strings"
)

func init() {
	cmd.AddCommand("god", hostGod)
	cmd.AddCommand("notarget", hostNoTarget)
	cmd.AddCommand("fly", hostFly)
	// cmd.AddCommand("noclip", hostNoClip) -- anglehack
	cmd.AddCommand("give", hostGive)
	cmd.AddCommand("color", hostColor)
	// cmd.AddCommand("ping", hostPing)
	cmd.AddCommand("say", hostSayAll)
	cmd.AddCommand("say_team", hostSayTeam)
	cmd.AddCommand("tell", hostTell)
	cmd.AddCommand("pause", hostPause)
	cmd.AddCommand("begin", hostBegin)
}

func qFormatI(b int32) string {
	if b == 0 {
		return "OFF"
	}
	return "ON"
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
		conPrintf("god [value] : toggle god mode. values: 0 = off, 1 = on\n")
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
	SV_ClientPrint(int(C.HostClient()),
		fmt.Sprintf("godmode %v\n", qFormatI(f&flag)))
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
		conPrintf("notarget [value] : toggle notarget mode. values: 0 = off, 1 = on\n")
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
	SV_ClientPrint(int(C.HostClient()),
		fmt.Sprintf("notarget %v\n", qFormatI(f&flag)))
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
		conPrintf("fly [value] : toggle fly mode. values: 0 = off, 1 = on\n")
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
	SV_ClientPrint(int(C.HostClient()),
		fmt.Sprintf("flymode %v\n", qFormatI(m&progs.MoveTypeFly)))
}

func hostColor(args []cmd.QArg) {
	c := int(cvars.ClientColor.Value())
	t := c >> 4
	b := c & 0x0f
	if len(args) == 0 {
		conPrintf("\"color\" is \"%d %d\"\n", t, b)
		conPrintf("color <0-13> [0-13]\n")
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
	cID := int(C.HostClient())
	client := sv_clients[cID]
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
		SV_ClientPrint(int(C.HostClient()), "Pause not allowed.\n")
		return
	}
	sv.paused = !sv.paused
	/*
		ev := EntVars(sv_player)
		if sv.paused {
		  SV_BroadcastPrintf("%s paused the game\n", PR_GetString(ev.netname))
		} else {
		  SV_BroadcastPrintf("%s unpaused the game\n", PR_GetString(ev.netname))
		}
	*/
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
		conPrintf("begin is not valid from the console\n")
		return
	}
	sv_clients[C.HostClient()].spawned = true
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

func hostTell(args []cmd.QArg) {
	if execute.IsSrcCommand() {
		forwardToServer("tell", args)
		return
	}

	if len(args) < 2 {
		// need at least destination and message
		return
	}

	cn := sv_clients[int(C.HostClient())].name
	// TODO: should we realy concat or use cmd.CmdArgs?
	ms := func() string {
		a := args[1:]
		n := (len(a) - 1)
		for i := 0; i < len(a); i++ {
			n += len(a[i].String())
		}

		b := make([]byte, n)
		bp := copy(b, a[0].String())
		for _, s := range a[1:] {
			bp += copy(b[bp:], " ")
			bp += copy(b[bp:], s.String())
		}
		return string(b)
	}()
	text := fmt.Sprintf("%s: %s", cn, ms)

	for i, c := range sv_clients {
		if !c.active || !c.spawned {
			continue
		}
		if strings.ToLower(c.name) != strings.ToLower(args[0].String()) {
			continue
		}
		// TODO: We check without case check. Are names unique ignoring the case?
		SV_ClientPrint(i, text)
	}
}

func hostSayAll(args []cmd.QArg) {
	// say
}
func hostSayTeam(args []cmd.QArg) {
	// say_team
}
