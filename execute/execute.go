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
type Efunc func(cmd.Arguments) (bool, error)

type executors []Efunc

var (
	commandExecutors executors
)

func SetCommandExecutors(e []Efunc) {
	commandExecutors = e
}

func ExecuteCommand(s string) error {
	return commandExecutors.execute(s)
}

func (ex *executors) execute(s string) error {
	a := cmd.Parse(s)
	args := a.Args()
	if len(args) == 0 {
		return nil // no tokens
	}
	for _, e := range *ex {
		if ok, err := e(a); err != nil {
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
