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

// args, player, source
type Efunc func(cmd.Arguments, int, int) (bool, error)

type executors []Efunc

var (
	clientExecutors  executors
	commandExecutors executors
)

func SetClientExecutors(e []Efunc) {
	clientExecutors = e
}

func SetCommandExecutors(e []Efunc) {
	commandExecutors = e
}

func ExecuteClient(s string, player int) error {
	return clientExecutors.execute(s, Client, player)
}

func ExecuteCommand(s string, player int) error {
	return commandExecutors.execute(s, Command, player)
}

func (ex *executors) execute(s string, source int, player int) error {
	a := cmd.Parse(s)
	args := a.Args()
	if len(args) == 0 {
		return nil // no tokens
	}
	for _, e := range *ex {
		if ok, err := e(a, player, source); err != nil {
			return err
		} else if ok {
			return nil
		}
	}

	name := args[0].String()
	log.Printf("Unknown command \"%s\"", name)
	conlog.Printf("Unknown command \"%s\"\n", name)
	return nil
}
