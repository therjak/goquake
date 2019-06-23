package cbuf

import (
	"quake/cmd"
	"quake/execute"
)

var (
	// original: buffer of 8192 byte size
	cbuf string
	// toogle to add a wait to Cbuf_Execute,
	// causing the following commands to be executed one frame later
	wait bool
)

func init() {
	cmd.AddCommand("wait", waitCmd)
}

func waitCmd(_ []cmd.QArg, _ int) {
	wait = true
}

func Execute(player int) {
	for len(cbuf) != 0 {
		i := 0
		quote := false
	LineLoop:
		for i = 0; i < len(cbuf); i++ {
			switch cbuf[i] {
			case '"':
				quote = !quote
				continue LineLoop
			case ';':
				if quote {
					continue LineLoop
				}
				break LineLoop
			case '\n':
				break LineLoop
			}
		}
		// do not put ';' or '\n' in line
		line := cbuf[:i]
		// but remove this char as well
		if i < len(cbuf) {
			i++
		}
		cbuf = cbuf[i:]
		execute.Execute(line, execute.Command, player)
		if wait {
			// wait for the next frame to continue executing
			wait = false
			return
		}
	}
}

func AddText(text string) {
	cbuf = cbuf + text
}

func InsertText(text string) {
	cbuf = text + "\n" + cbuf
}
