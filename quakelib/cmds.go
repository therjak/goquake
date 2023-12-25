// SPDX-License-Identifier: GPL-2.0-or-later
// extension to cmd.go
// adds some explicit cmds
package quakelib

import (
	"os"
	"strings"

	"goquake/alias"
	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvars"
	"goquake/filesystem"
	"goquake/input"
)

var (
	commands    = cmd.New()
	aliases     = alias.New()
	commandVars = cvars.New()
)

func must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func init() {
	must(aliases.Commands(commands))
	must(commandVars.Commands(commands))
	must(input.Commands(commands))
	must(cvars.Register(commandVars))
	cbuf.SetCommandExecutors([]cbuf.Efunc{
		commands.Execute(),
		aliases.Execute(),
		commandVars.Execute(),
	})
}

func addCommand(name string, f cmd.QFunc) {
	must(commands.Add(name, f))
}

func echo(a cbuf.Arguments) error {
	for _, arg := range a.Args()[1:] {
		conlog.Printf("%s ", arg)
	}
	conlog.Printf("\n")
	return nil
}

func printCmdList(a cbuf.Arguments) error {
	//TODO(therjak):
	// this should probably print the syntax of cmdlist if len(args) > 1
	args := a.Args()
	switch len(args) {
	default:
		printPartialCmdList(args[1].String())
	case 0, 1:
		printFullCmdList()
	}
	return nil
}

func printFullCmdList() {
	cmds := commands.List()
	for _, c := range cmds {
		conlog.SafePrintf("  %s\n", c)
	}
	conlog.SafePrintf("%v commands\n", len(cmds))
}

func printPartialCmdList(part string) {
	cmds := commands.List()
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
func executeCommandLineScripts(_ cbuf.Arguments) error {
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
		case '-':
			plus = false
		default:
			if plus {
				cmd = cmd + " " + a
			}
		}
	}
	if len(cmd) > 0 {
		cbuf.InsertText(cmd)
	}
	return nil
}

func execFile(a cbuf.Arguments) error {
	args := a.Args()
	if len(args) != 2 {
		conlog.Printf("exec <filename> : execute a script file\n")
		return nil
	}
	b, err := filesystem.GetFileContents(args[1].String())
	if err != nil {
		if args[1].String() == "default.cfg" {
			conlog.Printf("execing %v\n", args[1])
			cbuf.InsertText(defaultCfg)
		} else {
			conlog.Printf("couldn't exec %v\n", args[1])
		}
		return nil
	}
	conlog.Printf("execing %v\n", args[1])
	cbuf.InsertText(string(b))
	return nil
}

func init() {
	addCommand("echo", echo)
	addCommand("cmdlist", printCmdList)
	addCommand("stuffcmds", executeCommandLineScripts)
	addCommand("exec", execFile)
}
