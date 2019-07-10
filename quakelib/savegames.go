package quakelib

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/execute"
	"quake/protos"
	"strings"

	"github.com/golang/protobuf/proto"
)

func init() {
	cmd.AddCommand("save", saveGame)
	cmd.AddCommand("load", loadGame)
}

func saveGameComment() string {
	ln := sv.worldModel.Name // cl.levelname
	km := cl.stats.monsters
	tm := cl.stats.totalMonsters
	// somehow nobody can count?
	// we should have 39 chars total available, why clip at 22 for the map?
	return fmt.Sprintf("%-22s kills:%3d/%3d", ln, km, tm)
}

func saveGame(args []cmd.QArg, _ int) {
	if !execute.IsSrcCommand() {
		return
	}
	if !sv.active {
		conlog.Printf("Not playing a local game.\n")
		return
	}

	if cl.intermission != 0 {
		conlog.Printf("Can't save in intermission.\n")
		return
	}

	if svs.maxClients != 1 {
		conlog.Printf("Can't save multiplayer games.\n")
		return
	}

	if len(args) != 1 {
		conlog.Printf("save <savename> : save a game\n")
		return
	}

	if EntVars(sv_clients[0].edictId).Health <= 0 {
		conlog.Printf("Can't savegame with a dead player\n")
		return
	}

	filename := args[0].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		conlog.Printf("Relative pathnames are not allowed.\n")
		return
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
		Globals:      vm.saveGameGlobals(), // protos.Globals
		Edicts:       sv.saveGameEdicts(),  // []protos.Edict
	}

	out, err := proto.Marshal(data)
	if err != nil {
		conlog.Printf("failed to encode savegame.\n")
		return
	}

	if err := ioutil.WriteFile(fullname, out, 0660); err != nil {
		conlog.Printf("ERROR: couldn't write file.\n")
		return
	}
	conlog.Printf("done.\n")
}

func loadGame(args []cmd.QArg, _ int) {
	if !execute.IsSrcCommand() {
		return
	}

	if len(args) != 1 {
		conlog.Printf("load <savename> : load a game\n")
		return
	}

	filename := args[0].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		conlog.Printf("Relative pathnames are not allowed.\n")
		return
	}

	// stop demo loop in case this fails
	cls.demoNum = -1

	fullname := filepath.Join(GameDirectory(), filename)
	if filepath.Ext(fullname) != ".sav" {
		fullname = fullname + ".sav"
	}

	// we can't call SCR_BeginLoadingPlaque, because too much stack space has
	// been used.  The menu calls it before stuffing loadgame command
	//	SCR_BeginLoadingPlaque ();

	conlog.Printf("Loading game from %s...\n", fullname)

	in, err := ioutil.ReadFile(fullname)
	if err != nil {
		conlog.Printf("ERROR: couldn't read file.\n")
		return
	}

	data := &protos.SaveGame{}
	if err := proto.Unmarshal(in, data); err != nil {
		conlog.Printf("failed to decode savegame.\n")
		return
	}

	cvars.Skill.SetValue(float32(data.GetCurrentSkill()))

	clientDisconnect()

	sv.SpawnServer(data.GetMapName())
	if !sv.active {
		conlog.Printf("Couldn't load map\n")
		return
	}
	// pause until all clients connect
	sv.paused = true
	sv.loadGame = true

	// load the light styles
	copy(sv.lightStyles[:], data.GetLightStyles())

	vm.loadGameGlobals(data.GetGlobals())
	sv.loadGameEdicts(data.GetEdicts())

	sv.time = data.GetMapTime()

	copy(sv_clients[0].spawnParams[:], data.GetSpawnParams())

	if cls.state != ca_dedicated {
		clEstablishConnection("local")
		clientReconnect()
	}
}
