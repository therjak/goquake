// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"log"

	cmdl "goquake/commandline"
	"goquake/net"
)

func networkInit() {
	net.SetPort(cmdl.Port())

	clients := svs.maxClientsLimit
	if !cmdl.Dedicated() {
		clients++
	}
	if cmdl.Listen() || cmdl.Dedicated() {
		log.Printf("Listening to network")
		net.Listen(clients)
	}

	net.SetTime()

	// cmd.AddCommand("slist", NET_Slist_f);

	//if *my_tcpip_address {
	//	conlog.DPrintf("TCP/IP address %s\n", my_tcpip_address)
	//}
}
