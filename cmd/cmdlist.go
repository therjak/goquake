// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"sort"
	"strings"

	"goquake/cbuf"
	"goquake/conlog"
)

type cmdList []string

func (c *Commands) list() cmdList {
	cmds := make(cmdList, 0, len(*c))
	for cmd := range *c {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return cmds
}

func (c *Commands) printCmdList() QFunc {
	return func(a cbuf.Arguments) error {
		//TODO(therjak):
		// this should probably print the syntax of cmdlist if len(args) > 1
		args := a.Args()
		cl := c.list()
		switch len(args) {
		default:
			cl.printPartialCmdList(args[1].String())
		case 0, 1:
			cl.printFullCmdList()
		}
		return nil
	}
}

func (cl *cmdList) printFullCmdList() {
	for _, c := range *cl {
		conlog.SafePrintf("  %s\n", c)
	}
	conlog.SafePrintf("%v commands\n", len(*cl))
}

func (cl *cmdList) printPartialCmdList(part string) {
	count := 0
	for _, c := range *cl {
		if strings.HasPrefix(c, part) {
			conlog.SafePrintf("  %s\n", c)
			count++
		}
	}
	conlog.SafePrintf("%v commands beginning with \"%v\"\n", count, part)
}
