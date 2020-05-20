// extention to cmd.go
// adds some explicit cmds
package quakelib

import (
	"github.com/therjak/goquake/cbuf"
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/filesystem"
	"os"
	"strings"
)

func echo(args []cmd.QArg, _ int) {
	for _, a := range args {
		conlog.Printf("%s ", a)
	}
	conlog.Printf("\n")
}

func printCmdList(args []cmd.QArg, _ int) {
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
func executeCommandLineScripts(_ []cmd.QArg, _ int) {
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

func execFile(args []cmd.QArg, _ int) {
	if len(args) != 1 {
		conlog.Printf("exec <filename> : execute a script file\n")
		return
	}
	b, err := filesystem.GetFileContents(args[0].String())
	if err != nil {
		conlog.Printf("couldn't exec %v\n", args[0])
		return
	}
	conlog.Printf("execing %v\n", args[0])
	cbuf.InsertText(string(b))
}

func init() {
	cmd.AddCommand("echo", echo)
	cmd.AddCommand("cmdlist", printCmdList)
	cmd.AddCommand("stuffcmds", executeCommandLineScripts)
	cmd.AddCommand("exec", execFile)
}
