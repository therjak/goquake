// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"log"

	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/net"
)

//export NET_Init
func NET_Init() {
	net.SetPort(cmdl.Port())

	clients := svs.maxClientsLimit
	if cls.state != ca_dedicated {
		clients++
	}
	if cmdl.Listen() || cls.state == ca_dedicated {
		log.Printf("Listening to network")
		net.Listen(clients)
	}

	net.SetTime()

	// Cmd_AddCommand("slist", NET_Slist_f);

	//if *my_tcpip_address {
	//	conlog.DPrintf("TCP/IP address %s\n", my_tcpip_address)
	//}
}
