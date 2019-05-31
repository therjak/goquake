package quakelib

import "C"

import (
	"log"
	"os"
	"path/filepath"
	"quake/cmd"
	cmdl "quake/commandline"
	"quake/conlog"
	"quake/cvars"
	"quake/filesystem"
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

	cvars.Developer.SetCallback(func() {
		conlog.SetDeveloper(cvars.Developer.Value())
	})
}

func CmdPath(args []cmd.QArg) {
	// TODO
	log.Printf("path called")
}
func CmdGame(args []cmd.QArg) {
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

//export COM_GameDir
func COM_GameDir() *C.char {
	return C.CString(gameDirectory)
}

//export COM_BaseDir
func COM_BaseDir() *C.char {
	return C.CString(baseDirectory)
}

func addGameDirectory(base, dir string) {
	gameDirectory = filepath.Join(base, dir)
	filesystem.AddGameDir(gameDirectory)
}
