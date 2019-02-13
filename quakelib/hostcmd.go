package quakelib

//int HostClient(void);
import "C"

import (
	"fmt"
	"quake/cmd"
	"quake/execute"
	"quake/progs"
)

func init() {
	cmd.AddCommand("god", hostGod)
	cmd.AddCommand("notarget", hostNoTarget)
	// cmd.AddCommand("fly", hostFly)
	// cmd.AddCommand("noclip", hostNoClip)
	// cmd.AddCommand("give", hostGive)
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
