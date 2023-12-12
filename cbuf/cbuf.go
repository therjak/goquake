// SPDX-License-Identifier: GPL-2.0-or-later

package cbuf

import (
	"goquake/cmd"
	"strings"
)

var (
	cbuf CommandBuffer
)

type CommandBuffer struct {
	// original: buffer of 8192 byte size
	buf string
	// toogle to add a wait to Cbuf_Execute,
	// causing the following commands to be executed one frame later
	wait bool

	ex executors
}

func (c *CommandBuffer) Execute() {
	for len(c.buf) != 0 {
		i := 0
		quote := false
	LineLoop:
		for i = 0; i < len(c.buf); i++ {
			switch c.buf[i] {
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
		line := c.buf[:i]
		// but remove this char as well
		if i < len(c.buf) {
			i++
		}
		c.buf = c.buf[i:]
		c.ex.execute(line)
		if c.wait {
			// wait for the next frame to continue executing
			c.wait = false
			return
		}
	}
}

func (c *CommandBuffer) AddText(text string) {
	c.buf = c.buf + text
}

func (c *CommandBuffer) InsertText(text string) {
	c.buf = text + "\n" + c.buf
}

func (c *CommandBuffer) SetCommandExecutors(e []Efunc) {
	var wait Efunc = func(a cmd.Arguments) (bool, error) {
		n := a.Args()
		if len(n) == 0 {
			return false, nil
		}
		name := strings.ToLower(n[0].String())
		if name == "wait" {
			c.wait = true
			return true, nil
		}
		return false, nil
	}
	c.ex = append([]Efunc{wait}, e...)
}

// TODO(therjak): the following functions are deprecated and should be removed

func Execute() {
	cbuf.Execute()
}

func InsertText(text string) {
	cbuf.InsertText(text)
}

func AddText(text string) {
	cbuf.AddText(text)
}

func SetCommandExecutors(e []Efunc) {
	cbuf.SetCommandExecutors(e)
}

func ExecuteCommand(s string) error {
	return cbuf.ex.execute(s)
}
