package quakelib

import (
	"bufio"
	"github.com/therjak/goquake/cbuf"
	cmdl "github.com/therjak/goquake/commandline"
	"os"
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
	if !cmdl.Dedicated() {
		// no stdin necessary in graphical mode
		return
	}
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
