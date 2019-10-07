package quakelib

//#include <stdlib.h> // free
//float GetScreenConsoleCurrentHeight(void);
//void ConCheckResize(void);
//void ConInit(void);
//void ConDrawConsole(int lines);
//void ConDrawNotify(void);
//void ConClearNotify(void);
//void ConToggleConsole(void);
//void ConTabComplete(void);
//void ConPrint(const char* text);
//void ConBackscrollHome(void);
//void ConBackscrollEnd(void);
//void ConBackscrollUp(int page);
//void ConBackscrollDown(int page);
import "C"

import (
	"fmt"
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
	"quake/keys"
	"quake/math"
	svc "quake/protocol/server"
	"strings"
	"time"
	"unsafe"
)

const (
	CON_TEXTSIZE = 65536
	CON_MINSIZE  = 16384
)

type qconsole struct {
	text             []byte
	origText         []string
	backScroll       int // lines up from bottom to display
	totalLines       int // total lines in console scrollback
	current          int // where next message will be printed
	x                int // offset in current line for next print
	times            [4]time.Time
	chatTeam         bool
	width            int // pixels
	height           int // pixels
	lineWidth        int // characters
	initialized      bool
	forceDuplication bool   // because no entities to refresh
	lastCenter       string // just a temporary to prevent double print
	visibleLines     int    // con_vislines
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
func Con_DrawConsole(lines int) {
	console.Draw(lines)
	// C.ConDrawConsole(C.int(lines))
}

//export Con_DrawNotify
func Con_DrawNotify() {
	C.ConDrawNotify()
}

//export Con_ClearNotify
func Con_ClearNotify() {
	C.ConClearNotify()
	console.ClearNotify()
}

//export Con_ToggleConsole_f
func Con_ToggleConsole_f() {
	console.Toggle()
}

func (c *qconsole) Toggle() {
	C.ConToggleConsole()
}

//export Con_TabComplete
func Con_TabComplete() {
	C.ConTabComplete()
}

//export Con_LogCenterPrint
func Con_LogCenterPrint(str *C.char) {
	//TODO(therjak): we will need conlog.CenterPrint
	s := C.GoString(str)
	console.CenterPrint(s)
}

//export Con_PrintStr
func Con_PrintStr(str *C.char) {
	console.Print(C.GoString(str))
}

//export Con_Print
func Con_Print(str *C.char) {
	C.ConPrint(str)
}

//export Con_BackscrollHome
func Con_BackscrollHome() {
	console.BackScrollHome()
	C.ConBackscrollEnd()
}

//export Con_BackscrollEnd
func Con_BackscrollEnd() {
	console.BackScrollEnd()
	C.ConBackscrollEnd()
}

//export Con_BackscrollUp
func Con_BackscrollUp(page bool) {
	C.ConBackscrollUp(b2i(page))
	console.BackScrollUp(page)
}

func (c *qconsole) BackScrollEnd() {
	c.backScroll = 0
}

func (c *qconsole) BackScrollHome() {
	c.backScroll = c.maxBackScroll()
}

func (c *qconsole) scrollStep(page bool) int {
	if page {
		return (c.visibleLines / 8) - 4
	}
	return 1
}

func (c *qconsole) maxBackScroll() int {
	// TODO(therjak): this should not be origText
	max := len(c.origText) - (c.visibleLines / 8) + 2
	if max < 0 {
		return 0
	}
	return max
}

func (c *qconsole) clampBackScroll() {
	c.backScroll = math.ClampI(0, c.backScroll, c.maxBackScroll())
}

func (c *qconsole) BackScrollUp(page bool) {
	c.backScroll += c.scrollStep(page)
	c.clampBackScroll()
}

func (c *qconsole) BackScrollDown(page bool) {
	c.backScroll -= c.scrollStep(page)
	c.clampBackScroll()
}

//export Con_BackscrollDown
func Con_BackscrollDown(page bool) {
	C.ConBackscrollDown(b2i(page))
	console.BackScrollDown(page)
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
	cmd.AddCommand("toggleconsole", func([]cmd.QArg, int) { console.Toggle() })
	cmd.AddCommand("clear", func([]cmd.QArg, int) { console.Clear() })
	cmd.AddCommand("messagemode", func([]cmd.QArg, int) { console.messageMode(false) })
	cmd.AddCommand("messagemode2", func([]cmd.QArg, int) { console.messageMode(true) })
}

func (c *qconsole) Clear() {
	c.text = []byte{}
	c.origText = []string{}
	c.backScroll = 0
}

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
	if q.backScroll != 0 {
		q.backScroll++
	}
	if q.backScroll > q.totalLines-int(viewport.height/8)-1 {
		q.backScroll = q.totalLines - int(viewport.height/8) - 1
	}
	q.x = 0
	q.current++
	// memset(q.text[(q.current % q.totalLines) * q.width, ' ', q.width)
}
*/

var (
	printRecursionProtection = false
)

func (c *qconsole) Printf(format string, v ...interface{}) {
	c.Print(fmt.Sprintf(format, v...))
}

func (c *qconsole) CenterPrint(txt string) {
	// TODO(therjak): this is the function to pass to conlog.CenterPrint
	if cl.gameType == svc.GameDeathmatch &&
		cvars.ConsoleLogCenterPrint.Value() != 2 {
		return
	}
	if txt == c.lastCenter {
		return
	}
	c.lastCenter = txt
	if cvars.ConsoleLogCenterPrint.Bool() {
		c.printBar()
		c.centerPrint(txt) // '\n' is not needed as centerPrint adds it
		c.printBar()
		// clear the notify times to make sure the txt is not shown
		// twice: in the center and in the notification line at the
		// top of the screen.
		c.ClearNotify()
		C.ConClearNotify()
	}
}

const (
	centerPrintWhitespace = "                    " // 20x 'x20'
)

func (c *qconsole) centerPrint(txt string) {
	w := 40
	if w > c.lineWidth {
		w = c.lineWidth
	}
	parts := strings.Split(txt, "\n")
	// Split removes the '\n' so we can not forget to add it again.
	// Its probably ok to use Split and create new strings afterwards
	// as we add whitespace in most cases. The special case where we
	// could avoid a new string should be rare.
	for _, p := range parts {
		l := len(p)
		if l < w {
			wl := (w - l) / 2
			c.Print(centerPrintWhitespace[:wl] + p + "\n")
		} else {
			c.Print(txt + "\n")
		}
	}
}

func (c *qconsole) Print(txt string) {
	if !c.initialized {
		return
	}
	if cls.state == ca_dedicated {
		// no graphics mode
		return
	}

	c.print(txt)

	cstr := C.CString(txt)
	Con_Print(cstr)
	C.free(unsafe.Pointer(cstr))

	if cls.signon != 4 && !screen.disabled {
		if !printRecursionProtection {
			printRecursionProtection = true
			SCR_UpdateScreen()
			printRecursionProtection = false
		}
	}
}

func (c *qconsole) Draw(lines int) {
	// TODO(therjak): add line break functionality and respect '\n'
	// i.a. do not draw origText but a derived version
	if lines <= 0 {
		return
	}
	c.visibleLines = lines * c.height / int(viewport.height)
	SetCanvas(CANVAS_CONSOLE)

	DrawConsoleBackground()

	rows := (c.visibleLines + 7) / 8
	y := c.height - rows*8
	rows -= 2 // for intput and version line
	sb := 0
	if c.backScroll != 0 {
		sb = 2
	}
	//for i := c.current - rows + 1; i <= c.current-sb; i++ {
	for i := len(c.origText) - rows; i < len(c.origText)-sb; i++ {
		j := i - c.backScroll
		if j < 0 {
			y += 8
			continue
			//j = 0
		}
		// draw the actual text
		DrawStringWhite(8, y, c.origText[j])
		y += 8
	}
	if c.backScroll != 0 {
		y += 8 // blank line
		nx := 8
		for i := 0; i < c.lineWidth; i++ {
			DrawCharacterWhite(nx, y, int('^'))
			nx += 8
		}
	}

	// c.DrawInput()
	version := fmt.Sprintf("QuakeSpasm %1.2f.%d", QUAKESPASM_VERSION, QUAKESPASM_VER_PATCH)
	y += 8
	x := (c.lineWidth - len(version) + 2) * 8
	DrawStringWhite(x, y, version)
}

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
		// make the string copper color
		b := []byte(txt[1:])
		for i := 0; i < len(b); i++ {
			b[i] = b[i] | 128
		}
		txt = string(b)
	}

	var a []string
	for {
		// TODO(therjak): why do we need to check for \r?
		// if yes, change to IndexAny(txt, "\n\r")
		m := strings.Index(txt, "\n")
		if m < 0 {
			break
		}
		a = append(a, txt[:m+1]) // Dropping the '\n' would break the code below
		txt = txt[m+1:]
	}
	if len(txt) > 0 {
		a = append(a, txt)
	}

	// FIXME(therjak): We are only allowed to make a line break if we find
	// a '\n'. Otherwise we break text output originating from the vm.
	// This also means we need to verify we did not remove a '\n' in
	// conlog.Print
	ol := len(q.origText)
	if ol == 0 || strings.HasSuffix(q.origText[ol-1], "\n") {
		q.origText = append(q.origText, a...)
	} else {
		q.origText[ol-1] = q.origText[ol-1] + a[0]
		q.origText = append(q.origText, a[1:]...)
	}

	t := time.Now()
	newTimes := q.times[:]
	for i := 0; i < len(a); i++ {
		newTimes = append(newTimes, t)
	}
	copy(q.times[:], newTimes[len(newTimes)-4:])
}

//do not use. use conlog.Printf
func conPrintf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	log.Print(s)
	console.Print(s)
}

//do not use. use conlog.Printf
func conPrintStr(format string, v ...interface{}) {
	console.Printf(format, v...)
}

//do not use. use conlog.SafePrintf
func conSafePrintf(format string, v ...interface{}) {
	tmp := screen.disabled
	screen.disabled = true
	console.Printf(format, v...)
	screen.disabled = tmp
}

func init() {
	conlog.SetPrintf(conPrintf)
	conlog.SetSafePrintf(conSafePrintf)
}

const (
	// 40 chars, starts with 1d, ends with 1f, 1e between
	quakeBar = "\x1d\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1f\n"
)

//export ConPrintBar
func ConPrintBar() {
	console.printBar()
}

func (c *qconsole) printBar() {
	// TODO(therjak): we need a conlog.PrintBar
	if c.lineWidth >= len(quakeBar) {
		conlog.Printf(quakeBar)
	} else {
		var b strings.Builder
		b.WriteByte('\x1d')
		for i := 2; i < console.lineWidth; i++ {
			b.WriteByte('\x1e')
		}
		b.WriteByte('\x1f')
		b.WriteByte('\n')
		conlog.Printf(b.String())
	}
}
