// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/keys"
	"github.com/therjak/goquake/math"
	svc "github.com/therjak/goquake/protocol/server"
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

func init() {
	f := func(_ *cvar.Cvar) {
		screen.RecalcViewRect()
		updateConsoleSize()
	}
	cvars.ScreenConsoleWidth.SetCallback(f)
	cvars.ScreenConsoleScale.SetCallback(f)
}

func updateConsoleSize() {
	w := func() int {
		if cvars.ScreenConsoleWidth.Value() > 0 {
			return int(cvars.ScreenConsoleWidth.Value())
		}
		if cvars.ScreenConsoleScale.Value() > 0 {
			return int(float32(screen.Width) / cvars.ScreenConsoleScale.Value())
		}
		return screen.Width
	}()
	w = math.ClampI(320, w, screen.Width)
	w &= 0xFFFFFFF8

	console.width = int(w)
	console.height = console.width * screen.Height / screen.Width
}

//export Con_ResetLastCenterString
func Con_ResetLastCenterString() {
	console.lastCenter = ""
}

// produce new line breaks in case of a new width
func (c *qconsole) CheckResize() {
	w := (c.width / 8) - 2
	if w == c.lineWidth { // ConsoleWidth
		return
	}
	c.lineWidth = w
	// do the reflow
	// TODO

	c.ClearNotify()
	c.backScroll = 0
}

//export Con_Init
func Con_Init() {
	console.lineWidth = 38
	conlog.Printf("Console initialized.\n")

	console.initialized = true
}

func (c *qconsole) Toggle() {
	if keyDestination == keys.Console {
		// TODO(therjak): clear typing area
		c.backScroll = 0
		// TODO(therjak): return to the bottom of the command history

		if cls.state == ca_connected {
			inputActivate()
			keyDestination = keys.Game
		} else {
			enterMenuMain()
		}
	} else {
		IN_Deactivate()
		keyDestination = keys.Console
	}

	screen.EndLoadingPlaque()
	c.ClearNotify()
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

var (
	console = qconsole{
		lineWidth: 38,
	}
)

func (c *qconsole) currentHeight() int {
	return screen.consoleLines
}

func init() {
	cmd.AddCommand("toggleconsole", func([]cmd.QArg, int) { console.Toggle() })
	cmd.AddCommand("clear", func([]cmd.QArg, int) { console.Clear() })
	cmd.AddCommand("messagemode", func([]cmd.QArg, int) { console.messageMode(false) })
	cmd.AddCommand("messagemode2", func([]cmd.QArg, int) { console.messageMode(true) })

	cmd.AddCommand("condump", func([]cmd.QArg, int) { console.dump() })
}

func (c *qconsole) dump() {
	fn := path.Join(gameDirectory, "condump.txt")
	err := os.MkdirAll(gameDirectory, os.ModePerm)
	if err != nil {
		conlog.Printf("Could not create directory")
		return
	}
	s := strings.Join(c.origText, "")
	b := []byte(s)
	for i := 0; i < len(b); i++ {
		b[i] &= 0x7f
	}
	err = ioutil.WriteFile(fn, b, os.ModePerm)
	if err != nil {
		conlog.Printf("ERROR: couln't write file %s.\n", fn)
		return
	}
	conlog.Printf("Dumped console text to %s.\n", fn)
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
	keyDestination = keys.Message
}

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
	parts := strings.Split(txt, "\\n")
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

	if cls.signon != 4 && !screen.disabled {
		if !printRecursionProtection {
			printRecursionProtection = true
			screen.Update()
			printRecursionProtection = false
		}
	}
}

func (c *qconsole) DrawNotify() {
	qCanvas.Set(CANVAS_CONSOLE)
	y := c.height
	lines := 0
	delta := float64(cvars.ConsoleNotifyTime.Value()) // #sec to display
	for _, t := range c.times {
		diff := time.Since(t).Seconds()
		if diff < delta {
			lines++
		}
	}

	l := len(c.origText)
	for lines > 0 {
		DrawStringWhite(8, y, c.origText[l-lines])
		y += 8
		lines--
		screen.tileClearUpdates = 0
	}

	// TODO(therjak): add missing chat functionality again
}

func (c *qconsole) Draw(lines int) {
	// TODO(therjak): add line break functionality and respect '\n'
	// i.a. do not draw origText but a derived version
	if lines <= 0 {
		return
	}
	c.visibleLines = lines * c.height / screen.Height
	qCanvas.Set(CANVAS_CONSOLE)

	DrawConsoleBackground()

	rows := (c.visibleLines + 7) / 8
	y := c.height - rows*8
	rows -= 2 // for input and version line
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

	c.DrawInput()
	version := fmt.Sprintf("QuakeSpasm %1.2f.%d", QUAKESPASM_VERSION, QUAKESPASM_VER_PATCH)
	y += 8
	x := (c.lineWidth - len(version) + 2) * 8
	DrawStringWhite(x, y, version)
}

func (c *qconsole) DrawInput() {
	if keyDestination != keys.Console && !c.forceDuplication {
		return
	}
	// TODO(therjak): some kind of scrolling in case of len(keyInput.text > lineWidth
	DrawStringWhite(8, c.height-16, keyInput.String())

	// TODO(therjak): cursor blinking
	// depending on con_cursorspeed and key_blinktime
	DrawPicture(8+keyInput.cursorXPos*8, c.height-16, keyInput.Cursor())
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
			if b[i] != '\n' {
				b[i] = b[i] | 128
			}
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

	times := 0
	// FIXME(therjak): We are only allowed to make a line break if we find
	// a '\n'. Otherwise we break text output originating from the vm.
	// This also means we need to verify we did not remove a '\n' in
	// conlog.Print
	ol := len(q.origText)
	if ol == 0 || strings.HasSuffix(q.origText[ol-1], "\n") {
		q.origText = append(q.origText, a...)
		times = len(a)
	} else {
		q.origText[ol-1] = q.origText[ol-1] + a[0]
		q.origText = append(q.origText, a[1:]...)
		times = len(a[1:]) // only add a time if actually a new line was added
	}

	t := time.Now()
	newTimes := q.times[:]
	for i := 0; i < times; i++ {
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

//do not use. use conlog.SafePrintf
func conSafePrintf(format string, v ...interface{}) {
	tmp := screen.disabled
	screen.disabled = true
	s := fmt.Sprintf(format, v...)
	log.Print(s)
	console.Print(s)
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
