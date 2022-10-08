// SPDX-License-Identifier: GPL-2.0-or-later

package alias

import (
	"strings"

	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
)

var (
	aliases = make(map[string]string)
)

func alias(args []cmd.QArg, p, s int) error {
	switch c := len(args); c {
	case 0:
		listAliases()
		break
	case 1:
		printAlias(args[0])
		break
	default:
		setAlias(args)
		break
	}
	return nil
}

func listAliases() {
	if len(aliases) == 0 {
		conlog.SafePrintf("no alias commands found\n")
		return
	}
	for k, v := range aliases {
		// each alias value ends with a '\n'
		conlog.SafePrintf("  %s: %s", k, v)
	}
	conlog.SafePrintf("%v alias command(s)\n", len(aliases))
}

func printAlias(arg cmd.QArg) {
	name := arg.String()
	if v, ok := aliases[name]; ok {
		conlog.Printf("  %s: %s", name, v)
	}
}

func join(a []cmd.QArg, sep string) string {
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

func setAlias(args []cmd.QArg) {
	// join the parts, the parts have '"' already removed
	// len(args) > 1,
	name := args[0]
	command := join(args[1:], " ")
	aliases[name.String()] = strings.TrimSpace(command) + "\n"
}

func unalias(args []cmd.QArg, p, s int) error {
	switch c := len(args); c {
	case 1:
		name := args[0].String()
		if _, ok := aliases[name]; ok {
			delete(aliases, name)
		} else {
			conlog.Printf("No alias named %s\n", name)
		}
		break
	default:
		conlog.Printf("unalias <name> : delete alias\n")
		break
	}
	return nil
}

func unaliasAll(args []cmd.QArg, p, s int) error {
	aliases = make(map[string]string)
	return nil
}

func Get(name string) (string, bool) {
	a, ok := aliases[name]
	return a, ok
}

func Execute(args []cmd.QArg, player int, source int) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	name := args[0].String()
	if v, ok := Get(name); ok {
		cbuf.InsertText(v)
		return true, nil
	}
	return false, nil
}

func init() {
	cmd.Must(cmd.AddCommand("alias", alias))
	cmd.Must(cmd.AddCommand("unalias", unalias))
	cmd.Must(cmd.AddCommand("unaliasall", unaliasAll))
}
