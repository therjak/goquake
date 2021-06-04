// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"log"

	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/net"
)

func networkInit() {
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

	// cmd.AddCommand("slist", NET_Slist_f);

	//if *my_tcpip_address {
	//	conlog.DPrintf("TCP/IP address %s\n", my_tcpip_address)
	//}
}
