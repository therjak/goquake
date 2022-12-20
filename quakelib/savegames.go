// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/execute"
	"goquake/net"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

func init() {
	addCommand("save", saveGame)
	addCommand("load", loadGame)
}

func saveGameComment() string {
	ln := cl.levelName
	km := cl.stats.monsters
	tm := cl.stats.totalMonsters
	// somehow nobody can count?
	// we should have 39 chars total available, why clip at 22 for the map?
	return fmt.Sprintf("%-22s kills:%3d/%3d", ln, km, tm)
}

func saveGame(a cmd.Arguments, p, s int) error {
	if s != execute.Command {
		return nil
	}
	if !sv.active {
		conlog.Printf("Not playing a local game.\n")
		return nil
	}

	if cl.intermission != 0 {
		conlog.Printf("Can't save in intermission.\n")
		return nil
	}

	if svs.maxClients != 1 {
		conlog.Printf("Can't save multiplayer games.\n")
		return nil
	}
	args := a.Args()[1:]
	if len(args) != 1 {
		conlog.Printf("save <savename> : save a game\n")
		return nil
	}

	if entvars.Get(sv_clients[0].edictId).Health <= 0 {
		conlog.Printf("Can't savegame with a dead player\n")
		return nil
	}

	filename := args[0].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		conlog.Printf("Relative pathnames are not allowed.\n")
		return nil
	}

	fullname := filepath.Join(GameDirectory(), filename)
	if filepath.Ext(fullname) != ".sav" {
		fullname = fullname + ".sav"
	}

	conlog.Printf("Saving game to %s...\n", fullname)

	data := &protos.SaveGame{
		Comment:      saveGameComment(),
		SpawnParams:  sv_clients[0].spawnParams[:], //[]float32
		CurrentSkill: int32(cvars.Skill.Value()),
		MapName:      sv.name,
		MapTime:      sv.time,
		LightStyles:  sv.lightStyles[:],    //[]string
		Globals:      vm.SaveGameGlobals(), // protos.Globals
		Edicts:       sv.saveGameEdicts(),  // []protos.Edict
	}

	out, err := proto.Marshal(data)
	if err != nil {
		conlog.Printf("failed to encode savegame.\n")
		return nil
	}

	if err := ioutil.WriteFile(fullname, out, 0660); err != nil {
		conlog.Printf("ERROR: couldn't write file.\n")
		return nil
	}
	conlog.Printf("done.\n")
	return nil
}

func loadGame(a cmd.Arguments, p, s int) error {
	if s != execute.Command {
		return nil
	}

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

	fullname := filepath.Join(GameDirectory(), filename)
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

	cvars.Skill.SetValue(float32(data.GetCurrentSkill()))

	if err := clientDisconnect(); err != nil {
		return err
	}

	if err := sv.SpawnServer(data.GetMapName(), sv_protocol); err != nil {
		return err
	}
	if !sv.active {
		conlog.Printf("Couldn't load map\n")
		return nil
	}
	// pause until all clients connect
	sv.paused = true
	sv.loadGame = true

	// load the light styles
	copy(sv.lightStyles[:], data.GetLightStyles())

	vm.LoadGameGlobals(data.GetGlobals())
	if err := sv.loadGameEdicts(data.GetEdicts()); err != nil {
		return err
	}

	sv.time = data.GetMapTime()

	copy(sv_clients[0].spawnParams[:], data.GetSpawnParams())

	if !cmdl.Dedicated() {
		if err := clEstablishConnection(net.LocalAddress); err != nil {
			return err
		}
		clientReconnect()
	}
	return nil
}
