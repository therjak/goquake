// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"
	"os"
	"path/filepath"

	"goquake/cbuf"
	cmdl "goquake/commandline"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/filesystem"
)

func init() {
	addCommand("path", CmdPath)
	addCommand("game", CmdGame)

	cvars.Developer.SetCallback(func(cv *cvar.Cvar) {
		conlog.SetDeveloper(cv.Value())
	})
}

func CmdPath(a cbuf.Arguments) error {
	// TODO
	log.Printf("path called")
	return nil
}
func CmdGame(args cbuf.Arguments) error {
	// TODO
	return nil
}

func filesystemInit() {
	bd := cmdl.BaseDirectory()
	var baseDirectory string
	if bd != "" {
		baseDirectory = filepath.Clean(bd)
	} else {
		var err error
		baseDirectory, err = os.Getwd()
		if err != nil {
			log.Fatalf("Could not get current working dir: %v", err)
		}
	}

	filesystem.UseBaseDir(baseDirectory)

	// g := cmdl.Game()
	if cmdl.Rogue() /*|| game == "rogue"*/ {
		filesystem.UseGameDir("rogue")
	} else if cmdl.Hipnotic() /*|| game == "hipnotic"*/ {
		filesystem.UseGameDir("hipnotic")
	} else if cmdl.Quoth() /*|| game == "quoth"*/ {
		filesystem.UseGameDir("quoth")
	}
}
