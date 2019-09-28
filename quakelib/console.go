package quakelib

//float GetScreenConsoleCurrentHeight(void);
//void ConCheckResize(void);
//void ConInit(void);
//void ConDrawConsole(int lines, int drawinput);
//void ConDrawNotify(void);
//void ConClearNotify(void);
//void ConToggleConsole(void);
//void ConTabComplete(void);
//void ConLogCenterPrint(const char* str);
import "C"

import (
	"quake/cmd"
	"quake/cvars"
	"quake/keys"
	svc "quake/protocol/server"
	"strings"
	"time"
)

const (
	CON_TEXTSIZE = 65536
	CON_MINSIZE  = 16384
)

type qconsole struct {
	text             []byte
	backscroll       int // lines up from bottom to display
	totalLines       int // total lines in console scrollback
	current          int // where next message will be printed
	x                int // offset in current line for next print
	times            [4]time.Time
	chatTeam         bool
	width            int // pixels
	height           int // pixels
	lineWidth        int // characters
	initialized      bool
	forceDuplication bool // because no entities to refresh
	lastCenter       string
}

//export Con_ResetLastCenterString
func Con_ResetLastCenterString() {
	console.lastCenter = ""
}

//export Con_ForceDup
func Con_ForceDup() bool {
	return console.forceDuplication
}

//export Con_SetForceDup
func Con_SetForceDup(s bool) {
	console.forceDuplication = s
}

//export Con_CheckResize
func Con_CheckResize() {
	C.ConCheckResize()
}

//export Con_Init
func Con_Init() {
	C.ConInit()
	console.initialized = true
}

//export Con_Initialized
func Con_Initialized() bool {
	return console.initialized
}

//export Con_DrawConsole
func Con_DrawConsole(lines int, drawinput bool) {
	C.ConDrawConsole(C.int(lines), b2i(drawinput))
}

//export Con_DrawNotify
func Con_DrawNotify() {
	C.ConDrawNotify()
}

//export Con_ClearNotify
func Con_ClearNotify() {
	C.ConClearNotify()
}

//export Con_ToggleConsole_f
func Con_ToggleConsole_f() {
	C.ConToggleConsole()
}

//export Con_TabComplete
func Con_TabComplete() {
	C.ConTabComplete()
}

//export Con_LogCenterPrint
func Con_LogCenterPrint(str *C.char) {
	if cl.gameType == svc.GameDeathmatch && cvars.ConsoleLogCenterPrint.Value() != 2 {
		return
	}
	s := C.GoString(str)
	if s == console.lastCenter {
		return
	}
	console.lastCenter = s
	if cvars.ConsoleLogCenterPrint.Bool() {
		C.ConLogCenterPrint(str)
	}
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
