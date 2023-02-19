// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"goquake/cmd"
	"goquake/cvars"
	"goquake/filesystem"
	"goquake/protos"

	"google.golang.org/protobuf/proto"
)

func saveGameComment() string {
	levelName, err := progsdat.String(entvars.Get(0).Message)
	if err != nil {
		levelName = ""
	}
	km := progsdat.Globals.KilledMonsters
	tm := progsdat.Globals.TotalMonsters
	// somehow nobody can count?
	// we should have 39 chars total available, why clip at 22 for the map?
	log.Printf("%-22s kills:%3d/%3d", levelName, km, tm)
	return fmt.Sprintf("%-22s kills:%3d/%3d", levelName, km, tm)
}

func (c *SVClient) saveCmd(a cmd.Arguments) {
	args := a.Args()
	if len(args) != 2 {
		return
	}

	if svs.maxClients != 1 || !c.admin {
		c.Printf("Can't save multiplayer games.\n")
		return
	}

	if entvars.Get(c.edictId).Health <= 0 {
		c.Printf("Can't savegame with a dead player\n")
		return
	}

	filename := args[1].String()
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		// We will add filename to the gamedir so we with this we are always inside the gamedir
		c.Printf("Relative pathnames are not allowed.\n")
		return
	}

	fullname := filepath.Join(filesystem.GameDir(), filename)
	if filepath.Ext(fullname) != ".sav" {
		fullname = fullname + ".sav"
	}

	c.Printf("Saving game to %s...\n", fullname)

	data := &protos.SaveGame{
		Comment:      saveGameComment(),
		SpawnParams:  c.spawnParams[:], //[]float32
		CurrentSkill: int32(cvars.Skill.Value()),
		MapName:      sv.name,
		MapTime:      sv.time,
		LightStyles:  sv.lightStyles[:],    //[]string
		Globals:      vm.SaveGameGlobals(), // protos.Globals
		Edicts:       sv.saveGameEdicts(),  // []protos.Edict
	}

	out, err := proto.Marshal(data)
	if err != nil {
		c.Printf("failed to encode savegame.\n")
		return
	}

	if err := ioutil.WriteFile(fullname, out, 0660); err != nil {
		c.Printf("ERROR: couldn't write file.\n")
		return
	}
	c.Printf("done.\n")
}
