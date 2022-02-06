// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"
	"os"
	"path/filepath"

	"goquake/cmd"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/filesystem"
)

var (
	baseDirectory string
	gameDirectory string
	searchPaths   []qPath
)

type qPath struct {
	path string
	paks []string
}

func init() {
	Must(cmd.AddCommand("path", CmdPath))
	Must(cmd.AddCommand("game", CmdGame))

	cvars.Developer.SetCallback(func(cv *cvar.Cvar) {
		conlog.SetDeveloper(cv.Value())
	})
}

func CmdPath(args []cmd.QArg, _ int) {
	// TODO
	log.Printf("path called")
}
func CmdGame(args []cmd.QArg, _ int) {
	// TODO
}

func filesystemInit() {
	bd := cmdl.BaseDirectory()
	if bd != "" {
		baseDirectory = filepath.Clean(bd)
	} else {
		var err error
		baseDirectory, err = os.Getwd()
		if err != nil {
			log.Fatalf("Could not get current working dir: %v", err)
		}
	}

	addGameDirectory(baseDirectory, "id1")

	// g := cmdl.Game()
	if cmdl.Rogue() /*|| game == "rogue"*/ {
		addGameDirectory(baseDirectory, "rogue")
	} else if cmdl.Hipnotic() /*|| game == "hipnotic"*/ {
		addGameDirectory(baseDirectory, "hipnotic")
	} else if cmdl.Quoth() /*|| game == "quoth"*/ {
		addGameDirectory(baseDirectory, "quoth")
	}
}

func GameDirectory() string {
	return gameDirectory
}

func BaseDirectory() string {
	return baseDirectory
}

func addGameDirectory(base, dir string) {
	gameDirectory = filepath.Join(base, dir)
	filesystem.AddGameDir(gameDirectory)
}
