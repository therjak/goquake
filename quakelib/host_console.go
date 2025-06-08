// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"bufio"
	"os"

	"goquake/cbuf"
)

var (
	conReader *consoleReader
)

type consoleReader struct {
	textChan chan string
}

func newConsoleReader() *consoleReader {
	cr := &consoleReader{
		textChan: make(chan string, 1),
	}
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			cr.textChan <- scanner.Text()
		}
	}()
	return cr
}

// Add them exactly as if they had been typed at the console
func hostGetConsoleCommands() {
	if conReader == nil {
		conReader = newConsoleReader()
	}
	for {
		select {
		case s := <-conReader.textChan:
			cbuf.AddText(s)
		default:
			return
		}
	}
}
