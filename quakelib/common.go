package quakelib

import "C"

import (
	"log"
	"os"
	"path/filepath"

	"github.com/therjak/goquake/cmd"
	cmdl "github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/filesystem"
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
	cmd.AddCommand("path", CmdPath)
	cmd.AddCommand("game", CmdGame)

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

//export COM_InitFilesystem
func COM_InitFilesystem() {

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
