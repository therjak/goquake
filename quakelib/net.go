// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/net"
)

var (
	tcpipAvailable = true // TODO: better start with false
	listening      = false
)

func init() {
	addCommand("listen", listenCmd)
	addCommand("port", portCmd)
	addCommand("maxplayers", maxPlayersCmd)
}

func listenCmd(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	case 1:
		arg := args[0].Bool()
		if arg {
			listen()
		} else {
			unlisten()
		}
	default:
		conlog.Printf("listen is %t", listening)
	}
	return nil
}

func listen() {
	net.Listen(svs.maxClients)
}
func unlisten() {
	net.StopListen()
}

func portCmd(a cmd.Arguments) error {
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
		if listening {
			// Force a change to the new port
			cbuf.AddText("listen false\n")
			cbuf.AddText("listen true\n")
		}
	}
	return nil
}

func maxPlayersCmd(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	default:
		conlog.Printf("maxplayers is %d", svs.maxClients)
	case 1:
		if ServerActive() {
			conlog.Printf("maxplayers can not be changed while a server is running")
			return nil
		}
		arg := args[0].Int()
		if arg < 1 {
			arg = 1
		}
		if svs.maxClientsLimit < arg {
			arg = svs.maxClientsLimit
			conlog.Printf("maxplayers set to %d", arg)
		}
		if arg == 1 && listening {
			cbuf.AddText("listen false\n")
		}
		if arg > 1 && !listening {
			cbuf.AddText("listen true\n")
		}
		svs.maxClients = arg
		if arg == 1 {
			cvars.DeathMatch.SetByString("0")
		} else if !cvars.Coop.Bool() {
			cvars.DeathMatch.SetByString("1")
		}
	}
	return nil
}
