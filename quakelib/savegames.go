// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/filesystem"
	"goquake/net"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

func init() {
	addCommand("save", saveGame)
	addCommand("load", loadGame)
}

func saveGame(a cbuf.Arguments) error {
	args := a.Args()
	if len(args) != 2 {
		conlog.Printf("save <savename> : save a game\n")
		return nil
	}
	if cl.intermission != 0 {
		conlog.Printf("Can't save in intermission.\n")
		return nil
	}
	forwardToServer(a)
	return nil
}

func loadGame(a cbuf.Arguments) error {
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("load <savename> : load a game\n")
		return nil
	}

	filename := args[0].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		conlog.Printf("Relative pathnames are not allowed.\n")
		return nil
	}

	// stop demo loop in case this fails
	cls.demoNum = -1

	fullname := filepath.Join(filesystem.GameDir(), filename)
	if filepath.Ext(fullname) != ".sav" {
		fullname = fullname + ".sav"
	}

	// we can't call screen.BeginLoadingPlaque, because too much stack space has
	// been used.  The menu calls it before stuffing loadgame command
	//	screen.BeginLoadingPlaque ();

	conlog.Printf("Loading game from %s...\n", fullname)

	in, err := ioutil.ReadFile(fullname)
	if err != nil {
		conlog.Printf("ERROR: couldn't read file.\n")
		return nil
	}

	data := &protos.SaveGame{}
	if err := proto.Unmarshal(in, data); err != nil {
		conlog.Printf("failed to decode savegame.\n")
		return nil
	}

	if err := clientDisconnect(); err != nil {
		return err
	}

	if err := sv.SpawnSaveGameServer(data, sv_protocol); err != nil {
		conlog.Printf("Couldn't load map\n")
		return err
	}

	if !cmdl.Dedicated() {
		if err := clEstablishConnection(net.LocalAddress); err != nil {
			return err
		}
		clientReconnect()
	}
	return nil
}
