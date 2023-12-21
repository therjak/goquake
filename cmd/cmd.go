// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"fmt"
	"goquake/cbuf"
	"sort"
	"strings"
)

type QFunc func(args cbuf.Arguments) error

type Commands map[string]QFunc

func New() *Commands {
	c := make(Commands)
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

func (c *Commands) List() []string {
	cmds := make([]string, 0, len(*c))
	for cmd := range *c {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return cmds
}

func (c *Commands) Execute(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
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

var (
	commands = make(Commands)
)

func Must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func AddCommand(name string, f QFunc) error {
	return commands.Add(name, f)
}

func Exists(cmdName string) bool {
	return commands.Exists(cmdName)
}

func Execute(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
	return commands.Execute(cb, a)
}

func List() []string {
	return commands.List()
}
