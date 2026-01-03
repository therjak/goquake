// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/cbuf"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/net"
)

var (
	tcpipAvailable = true // TODO: better start with false
)

func init() {
	addCommand("listen", listenCmd)
	addCommand("port", portCmd)
	addCommand("maxplayers", maxPlayersCmd)
}

func listenCmd(a cbuf.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	case 1:
		arg := args[0].Bool()
		if arg {
			svTODO.Listen()
		} else {
			svTODO.StopListen()
		}
	default:
		conlog.Printf("listen is %t", svTODO.Listening())
	}
	return nil
}

func portCmd(a cbuf.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	default:
		conlog.Printf("port is %d", net.Port())
	case 1:
		arg := args[0].Int()
		if arg < 1 || 65534 < arg {
			conlog.Printf("Bad value, must be between 1 and 65534")
			return nil
		}
		net.SetPort(arg)
		if svTODO.Listening() {
			// Force a change to the new port
			cbuf.AddText("listen false\n")
			cbuf.AddText("listen true\n")
		}
	}
	return nil
}

func maxPlayersCmd(a cbuf.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	default:
		conlog.Printf("maxplayers is %d", svTODO.MaxClients())
	case 1:
		if ServerActive() {
			conlog.Printf("maxplayers can not be changed while a server is running")
			return nil
		}
		arg := args[0].Int()
		if arg < 1 {
			arg = 1
		}
		if svTODO.MaxClientsLimit() < arg {
			arg = svTODO.MaxClientsLimit()
			conlog.Printf("maxplayers set to %d", arg)
		}
		if arg == 1 && svTODO.Listening() {
			cbuf.AddText("listen false\n")
		}
		if arg > 1 && !svTODO.Listening() {
			cbuf.AddText("listen true\n")
		}
		svTODO.SetMaxClients(arg)
		if arg == 1 {
			cvars.DeathMatch.SetByString("0")
		} else if !cvars.Coop.Bool() {
			cvars.DeathMatch.SetByString("1")
		}
	}
	return nil
}
