package quakelib

//float GetScreenConsoleCurrentHeight(void);
import "C"

import (
	"quake/cmd"
	"quake/keys"
	"strings"
	"time"
)

const (
	CON_TEXTSIZE = 65536
	CON_MINSIZE  = 16384
)

type qconsole struct {
	text       []byte
	backscroll int // lines up from bottom to display
	totalLines int // total lines in console scrollback
	current    int // where next message will be printed
	x          int // offset in current line for next print
	times      [4]time.Time
	chatTeam   bool
	width      int // pixels
	height     int // pixels
	lineWidth  int // characters
}

var (
	console = qconsole{
		lineWidth: 38,
	}
)

func (c *qconsole) currentHeight() int {
	return int(C.GetScreenConsoleCurrentHeight())
}

func init() {
	// cmd.AddCommand("clear", func([]cmd.QArg, int) { console.clear() })
	cmd.AddCommand("messagemode", func([]cmd.QArg, int) { console.messageMode(false) })
	cmd.AddCommand("messagemode2", func([]cmd.QArg, int) { console.messageMode(true) })
}

// for cmd.AddCommand("clear", ...
func (q *qconsole) clear() {
	q.text = []byte{}
	q.backscroll = 0
}

// Con_ClearNotify
func (q *qconsole) ClearNotify() {
	q.times = [4]time.Time{}
}

// Con_MessageMode_f and Con_MessageMode2_f
func (q *qconsole) messageMode(team bool) {
	if cls.state != ca_connected || cls.demoPlayback {
		return
	}
	q.chatTeam = team
	keyDestination = keys.Menu
}

/*
// Con_Linefeed
func (q *qconsole) lineFeed() {
	if q.backscroll != 0 {
		q.backscroll++
	}
	if q.backscroll > q.totalLines-int(viewport.height/8)-1 {
		q.backscroll = q.totalLines - int(viewport.height/8) - 1
	}
	q.x = 0
	q.current++
	// memset(q.text[(q.current % q.totalLines) * q.width, ' ', q.width)
}
*/

// Con_Print
func (q *qconsole) print(txt string) {
	if len(txt) < 1 {
		return
	}
	switch txt[0] {
	case 1:
		localSound("misc/talk.wav")
		fallthrough
	case 2:
		txt = txt[1:]
		txt = strings.Map(func(r rune) rune {
			// colored text
			return r | 128
		}, txt)
	}
	q.text = append(q.text, txt...)
	copy(q.times[:], append(q.times[1:], time.Now()))
}
