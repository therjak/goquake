// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"goquake/cbuf"
)

type QFunc func(args cbuf.Arguments) error

type Commands map[string]QFunc

func New() *Commands {
	c := make(Commands)
	c.Add("cmdlist", c.printCmdList())
	return &c
}

func (c *Commands) Add(name string, f QFunc) error {
	ln := strings.ToLower(name)
	if _, ok := (*c)[ln]; ok {
		return fmt.Errorf("Cmd_AddCommand: %s already defined\n", ln)
	}
	(*c)[ln] = f
	return nil
}

func (c *Commands) Exists(cmdName string) bool {
	name := strings.ToLower(cmdName)
	_, ok := (*c)[name]
	return ok
}

func (c *Commands) Execute() func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
	return func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
		n := a.Args()
		if len(n) == 0 {
			return false, nil
		}
		name := strings.ToLower(n[0].String())
		if cmd, ok := (*c)[name]; ok {
			if err := cmd(a); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
}
