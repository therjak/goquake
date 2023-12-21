// SPDX-License-Identifier: GPL-2.0-or-later

package alias

import (
	"strings"

	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
)

type Aliases map[string]string

func (al *Aliases) Alias() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch c := len(args); c {
		case 0:
			al.listAliases()
		case 1:
			al.printAlias(args[0])
		default:
			al.setAlias(args)
		}
		return nil
	}
}

func (al *Aliases) listAliases() {
	if len(*al) == 0 {
		conlog.SafePrintf("no alias commands found\n")
		return
	}
	for k, v := range *al {
		// each alias value ends with a '\n'
		conlog.SafePrintf("  %s: %s", k, v)
	}
	conlog.SafePrintf("%v alias command(s)\n", len(*al))
}

func (al *Aliases) printAlias(arg cbuf.QArg) {
	name := arg.String()
	if v, ok := (*al)[name]; ok {
		conlog.Printf("  %s: %s", name, v)
	}
}

func join(a []cbuf.QArg, sep string) string {
	switch len(a) {
	case 0:
		return ""
	case 1:
		return a[0].String()
	}
	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(a[i].String())
	}

	b := make([]byte, n)
	bp := copy(b, a[0].String())
	for _, s := range a[1:] {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], s.String())
	}
	return string(b)
}

func (al *Aliases) setAlias(args []cbuf.QArg) {
	// join the parts, the parts have '"' already removed
	// len(args) > 1,
	name := args[0]
	command := join(args[1:], " ")
	(*al)[name.String()] = strings.TrimSpace(command) + "\n"
}

func (al *Aliases) Unalias() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch c := len(args); c {
		case 1:
			name := args[0].String()
			if _, ok := (*al)[name]; ok {
				delete(*al, name)
			} else {
				conlog.Printf("No alias named %s\n", name)
			}
		default:
			conlog.Printf("unalias <name> : delete alias\n")
		}
		return nil
	}
}

func (al *Aliases) UnaliasAll() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		*al = make(map[string]string)
		return nil
	}
}

func (al *Aliases) Execute() func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
	return func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
		args := a.Args()
		if len(args) < 1 {
			return false, nil
		}
		name := args[0].String()
		if v, ok := (*al)[name]; ok {
			cb.InsertText(v)
			return true, nil
		}
		return false, nil
	}
}

func (al *Aliases) Register(c *cmd.Commands) error {
	if err := c.Add("alias", al.Alias()); err != nil {
		return err
	}
	if err := c.Add("unalias", al.Unalias()); err != nil {
		return err
	}
	if err := c.Add("unaliasall", al.UnaliasAll()); err != nil {
		return err
	}
	return nil
}

func New() *Aliases {
	a := make(Aliases)
	return &a
}

var (
	Execute func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error)
)

func init() {
	al := New()
	Execute = al.Execute()
	cmd.Must(cmd.AddCommand("alias", al.Alias()))
	cmd.Must(cmd.AddCommand("unalias", al.Unalias()))
	cmd.Must(cmd.AddCommand("unaliasall", al.UnaliasAll()))
}
