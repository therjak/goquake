// SPDX-License-Identifier: GPL-2.0-or-later
// extension to cmd.go
// adds some explicit cmds
package quakelib

import (
	"os"
	"strings"

	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
	"goquake/filesystem"
)

func addCommand(name string, f cmd.QFunc) {
	cmd.Must(cmd.AddCommand(name, f))
}
func addClientCommand(name string, f cmd.QFunc) {
	cmd.Must(cmd.AddClientCommand(name, f))
}

func echo(args []cmd.QArg, _ int) error {
	for _, a := range args {
		conlog.Printf("%s ", a)
	}
	conlog.Printf("\n")
	return nil
}

func printCmdList(args []cmd.QArg, _ int) error {
	//TODO(therjak):
	// this should probably print the syntax of cmdlist if len(args) > 1
	switch len(args) {
	default:
		printPartialCmdList(args[0].String())
		return nil
	case 0:
		printFullCmdList()
		break
	}
	return nil
}

func printFullCmdList() {
	cmds := cmd.List()
	for _, c := range cmds {
		conlog.SafePrintf("  %s\n", c)
	}
	conlog.SafePrintf("%v commands\n", len(cmds))
}

func printPartialCmdList(part string) {
	cmds := cmd.List()
	count := 0
	for _, c := range cmds {
		if strings.HasPrefix(c, part) {
			conlog.SafePrintf("  %s\n", c)
			count++
		}
	}
	conlog.SafePrintf("%v commands beginning with \"%v\"\n", count, part)
}

// Adds command line parameters as script statements
// Commands lead with a +, and continue until a - or another +
// quake +prog jctest.qp +cmd amlev1
// quake -nosound +cmd amlev1
func executeCommandLineScripts(_ []cmd.QArg, _ int) error {
	plus := false
	cmd := ""
	// args[0] is command name
	for _, a := range os.Args[1:] {
		switch a[0] {
		case '+':
			// we only care about what follows after the '+'
			if len(cmd) == 0 {
				cmd = a[1:]
			} else {
				cmd = "; " + a[1:]
			}
			plus = true
			break
		case '-':
			plus = false
			break
		default:
			if plus {
				cmd = cmd + " " + a
			}
			break
		}
	}
	if len(cmd) > 0 {
		cbuf.InsertText(cmd)
	}
	return nil
}

func execFile(args []cmd.QArg, _ int) error {
	if len(args) != 1 {
		conlog.Printf("exec <filename> : execute a script file\n")
		return nil
	}
	b, err := filesystem.GetFileContents(args[0].String())
	if err != nil {
		if args[0].String() == "default.cfg" {
			conlog.Printf("execing %v\n", args[0])
			cbuf.InsertText(defaultCfg)
		} else {
			conlog.Printf("couldn't exec %v\n", args[0])
		}
		return nil
	}
	conlog.Printf("execing %v\n", args[0])
	cbuf.InsertText(string(b))
	return nil
}

func init() {
	addCommand("echo", echo)
	addCommand("cmdlist", printCmdList)
	addCommand("stuffcmds", executeCommandLineScripts)
	addCommand("exec", execFile)
}
