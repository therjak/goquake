// SPDX-License-Identifier: GPL-2.0-or-later

package cbuf

import (
	"log"

	"goquake/cmd"
	"goquake/conlog"
)

// args, player, source
type Efunc func(cmd.Arguments) (bool, error)

type executors []Efunc

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
