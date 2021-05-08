// SPDX-License-Identifier: GPL-2.0-or-later

package execute

import (
	"log"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
)

const (
	Client  = 0
	Command = 1
)

var (
	cmdSource = Client
	executors []func([]cmd.QArg, int) bool
)

func SetExecutors(e []func([]cmd.QArg, int) bool) {
	executors = e
}

func Execute(s string, source int, player int) {
	cmdSource = source
	cmd.Parse(s)

	args := cmd.Args()
	if len(args) == 0 {
		return // no tokens
	}
	name := args[0].String()
	for _, e := range executors {
		if e(args, player) {
			return
		}
	}

	log.Printf("Unknown command \"%s\"", name)
	conlog.Printf("Unknown command \"%s\"\n", name)
}

func IsSrcCommand() bool {
	return cmdSource == Command
}
