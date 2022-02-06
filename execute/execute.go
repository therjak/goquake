// SPDX-License-Identifier: GPL-2.0-or-later

package execute

import (
	"log"

	"goquake/cmd"
	"goquake/conlog"
)

const (
	Client  = 0
	Command = 1
)

type Efunc func([]cmd.QArg, int) (bool, error)

var (
	cmdSource = Client
	executors []Efunc
)

func SetExecutors(e []Efunc) {
	executors = e
}

func Execute(s string, source int, player int) error {
	cmdSource = source
	cmd.Parse(s)

	args := cmd.Args()
	if len(args) == 0 {
		return nil // no tokens
	}
	name := args[0].String()
	for _, e := range executors {
		if ok, err := e(args, player); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	log.Printf("Unknown command \"%s\"", name)
	conlog.Printf("Unknown command \"%s\"\n", name)
	return nil
}

func IsSrcCommand() bool {
	return cmdSource == Command
}
