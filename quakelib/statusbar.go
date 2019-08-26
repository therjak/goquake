package quakelib

//void Sbar_Changed();
import "C"

import (
	"quake/cvars"
	"quake/math"
	svc "quake/protocol/server"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type Statusbar struct{}

var (
	statusbar Statusbar
)

func (s *Statusbar) MarkChanged() {
	C.Sbar_Changed()
}

func StatusbarChanged() {
	C.Sbar_Changed()
}

//export Sbar_DrawScrollString
func Sbar_DrawScrollString(x int, y int, width int, str *C.char) {
	statusbar.DrawScrollString(x, y, width, C.GoString(str))
}

// scroll the string inside a glscissor region
func (s *Statusbar) DrawScrollString(x, y, width int, str string) {

	scale := cvars.ScreenStatusbarScale.Value()
	scale = math.Clamp32(1.0, scale, float32(viewport.width)/320.0)
	left := float32(x) * scale
	if cl.gameType != svc.GameDeathmatch {
		left += (float32(viewport.width) - 320.0*scale) / 2
	}

	gl.Enable(gl.SCISSOR_TEST)
	gl.Scissor(int32(left), 0, int32(float32(width)*scale), int32(viewport.height))

	len := len(str)*8 + 40
	ofs := int(host.time*30) % len
	drawString(x-ofs, y+24, str)
	drawSymbol(x-ofs+len-32, y+24, '/')
	drawSymbol(x-ofs+len-24, y+24, '/')
	drawSymbol(x-ofs+len-16, y+24, '/')
	drawString(x-ofs+len, y+24, str)

	gl.Disable(gl.SCISSOR_TEST)
}
