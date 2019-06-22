package execute

import (
	"log"

	"quake/cmd"
	"quake/conlog"
)

const (
	Client  = 0
	Command = 1
)

var (
	cmdSource = Client
	executors []func([]cmd.QArg) bool
)

func SetExecutors(e []func([]cmd.QArg) bool) {
	executors = e
}

func Execute(s string, source int) {
	cmdSource = source
	cmd.Parse(s)

	args := cmd.Args()
	if len(args) == 0 {
		return // no tokens
	}
	name := args[0].String()
	for _, e := range executors {
		if e(args) {
			return
		}
	}

	log.Printf("Unknown command \"%s\"", name)
	conlog.Printf("Unknown command \"%s\"\n", name)
}

func IsSrcCommand() bool {
	return cmdSource == Command
}
