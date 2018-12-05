// extention to cmd.go
// adds some explicit cmds
package quakelib

import (
	"os"
	"quake/cbuf"
	"quake/cmd"
	"quake/conlog"
	"quake/filesystem"
	"strings"
)

func echo(args []cmd.QArg) {
	for _, a := range args {
		ConPrintStr("%s ", a)
	}
	ConPrintStr("\n")
}

func printCmdList(args []cmd.QArg) {
	//TODO(therjak):
	// this should probably print the syntax of cmdlist if len(args) > 1
	switch len(args) {
	default:
		printPartialCmdList(args[0].String())
		return
	case 0:
		printFullCmdList()
		break
	}
}

func printFullCmdList() {
	cmds := cmd.Cmd_List()
	for _, c := range cmds {
		conlog.SafePrintf("  %s\n", c)
	}
	conlog.SafePrintf("%v commands\n", len(cmds))
}

func printPartialCmdList(part string) {
	cmds := cmd.Cmd_List()
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
func executeCommandLineScripts(_ []cmd.QArg) {
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
}

func execFile(args []cmd.QArg) {
	if len(args) != 1 {
		ConPrintStr("exec <filename> : execute a script file\n")
		return
	}
	b, err := filesystem.GetFileContents(args[0].String())
	if err != nil {
		ConPrintStr("couldn't exec %v\n", args[0])
		return
	}
	ConPrintStr("execing %v\n", args[0])
	cbuf.InsertText(string(b))
}

func init() {
	cmd.AddCommand("echo", echo)
	cmd.AddCommand("cmdlist", printCmdList)
	cmd.AddCommand("stuffcmds", executeCommandLineScripts)
	cmd.AddCommand("exec", execFile)
}
