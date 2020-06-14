package quakelib

//#include "stdlib.h"
//#include "draw.h"
//void GL_SetCanvas(canvastype newCanvas);
//void GL_CanvasEnd(void);
import "C"

import (
	"encoding/binary"
	"fmt"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/filesystem"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/texture"
	"github.com/therjak/goquake/wad"
)

type canvas int

const (
	CANVAS_NONE        canvas = C.CANVAS_NONE
	CANVAS_DEFAULT     canvas = C.CANVAS_DEFAULT
	CANVAS_CONSOLE     canvas = C.CANVAS_CONSOLE
	CANVAS_MENU        canvas = C.CANVAS_MENU
	CANVAS_STATUSBAR   canvas = C.CANVAS_SBAR
	CANVAS_WARPIMAGE   canvas = C.CANVAS_WARPIMAGE
	CANVAS_BOTTOMRIGHT canvas = C.CANVAS_BOTTOMRIGHT
)

const (
	vertexSourceDrawer = `
#version 410
in vec3 position;
in vec2 texcoord;
out vec2 Texcoord;

void main() {
	Texcoord = texcoord;
	gl_Position = vec4(position, 1.0);
}
` + "\x00"
	fragmentSourceDrawer = `
#version 410
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
  vec4 color = texture(tex, Texcoord);
	if (color.a < 0.666)
	  discard;
  frag_color = color;
}
` + "\x00"
	fragmentSourceColorRecDrawer = `
#version 410
in vec2 Texcoord;
out vec4 frag_color;
uniform vec4 in_color;

void main() {
  frag_color = in_color;
}
` + "\x00"
)

func newRecDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceDrawer, fragmentSourceColorRecDrawer)
}

func newDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceDrawer, fragmentSourceDrawer)
}

type recDrawer struct {
	vao      *glh.VertexArray
	vbo      *glh.Buffer
	ebo      *glh.Buffer
	prog     *glh.Program
	position uint32
	color    int32
}

func NewRecDrawer() *recDrawer {
	d := &recDrawer{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer()
	d.ebo = glh.NewBuffer()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(elements), gl.Ptr(elements), gl.STATIC_DRAW)
	var err error
	d.prog, err = newRecDrawProgram()
	if err != nil {
		Error(err.Error())
	}
	d.position = d.prog.GetAttribLocation("position")
	d.color = d.prog.GetUniformLocation("in_color")

	return d
}

func (d *recDrawer) Draw(x, y, w, h float32, c Color) {
	sx, sy := applyCanvas()
	x1, x2 := x, x+w
	y1, y2 := y+h, y
	x1 = x1*sx - 1
	x2 = x2*sx - 1
	ys := yShift()
	y1 = -y1*sy + ys
	y2 = -y2*sy + ys
	vertices := []float32{
		x1, y2, 0,
		x2, y2, 0,
		x2, y1, 0,
		x1, y1, 0,
	}

	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)

	d.prog.Use()
	d.vao.Bind()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	d.vbo.Bind(gl.ARRAY_BUFFER)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 3, gl.FLOAT, false, 4*3, gl.PtrOffset(0))

	gl.Uniform4f(d.color, c.R, c.G, c.B, c.A)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))

	gl.DisableVertexAttribArray(d.position)
	gl.Disable(gl.BLEND)
}

type drawer struct {
	vao      *glh.VertexArray
	vbo      *glh.Buffer
	ebo      *glh.Buffer
	prog     *glh.Program
	position uint32
	texcoord uint32
}

func NewDrawer() *drawer {
	d := &drawer{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer()
	d.ebo = glh.NewBuffer()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(elements), gl.Ptr(elements), gl.STATIC_DRAW)
	var err error
	d.prog, err = newDrawProgram()
	if err != nil {
		Error(err.Error())
	}
	d.position = d.prog.GetAttribLocation("position")
	d.texcoord = d.prog.GetAttribLocation("texcoord")

	return d
}

func yShift() float32 {
	if qCanvas != CANVAS_CONSOLE {
		return 1
	}
	sh := float32(screen.consoleLines)
	vh := float32(viewport.height)
	l := (sh / vh)
	return 3 - (2 * l)
}

func (d *drawer) Draw(x, y, w, h float32, t *texture.Texture) {
	sx, sy := applyCanvas()
	x1, x2 := x, x+w
	y1, y2 := y+h, y
	x1 = x1*sx - 1
	x2 = x2*sx - 1
	ys := yShift()
	y1 = -y1*sy + ys
	y2 = -y2*sy + ys
	vertices := []float32{
		x1, y2, 0, 0, 0,
		x2, y2, 0, 1, 0,
		x2, y1, 0, 1, 1,
		x1, y1, 0, 0, 1,
	}

	d.prog.Use()
	d.vao.Bind()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	d.vbo.Bind(gl.ARRAY_BUFFER)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 3, gl.FLOAT, false, 4*5, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(d.texcoord)
	gl.VertexAttribPointer(d.texcoord, 2, gl.FLOAT, false, 4*5, gl.PtrOffset(3*4))

	textureManager.Bind(t)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
	gl.DisableVertexAttribArray(d.texcoord)
	gl.DisableVertexAttribArray(d.position)
}

func (d *drawer) DrawQuad(x, y float32, num byte) {
	size := float32(0.0625)
	row := float32(num>>4) * size
	col := float32(num&15) * size

	sx, sy := applyCanvas()
	x1, x2 := x, x+8
	y1, y2 := y+8, y
	x1 = x1*sx - 1
	x2 = x2*sx - 1
	ys := yShift()
	y1 = -y1*sy + ys
	y2 = -y2*sy + ys
	vertices := []float32{
		x1, y2, 0, col, row,
		x2, y2, 0, col + size, row,
		x2, y1, 0, col + size, row + size,
		x1, y1, 0, col, row + size,
	}

	d.prog.Use()
	d.vao.Bind()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	d.vbo.Bind(gl.ARRAY_BUFFER)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 3, gl.FLOAT, false, 4*5, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(d.texcoord)
	gl.VertexAttribPointer(d.texcoord, 2, gl.FLOAT, false, 4*5, gl.PtrOffset(3*4))

	textureManager.Bind(consoleTexture)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))

	gl.DisableVertexAttribArray(d.texcoord)
	gl.DisableVertexAttribArray(d.position)
}

var (
	qDrawer         *drawer
	qRecDrawer      *recDrawer
	consoleTexture  *texture.Texture
	backtileTexture *texture.Texture
)

//export Draw_Delete
func Draw_Delete() {
	qDrawer = nil
	qRecDrawer = nil
}

//export Draw_Init
func Draw_Init() {
	qDrawer = NewDrawer()
	qRecDrawer = NewRecDrawer()
	consoleTexture = textureManager.LoadConsoleChars()
	backtileTexture = textureManager.LoadBacktile()
}

var (
	qCanvas canvas
)

//export GLSetCanvas
func GLSetCanvas(c C.canvastype) {
	SetCanvas(canvas(c))
}

func SetCanvas(c canvas) {
	if qCanvas == c {
		return
	}
	qCanvas = c
	switch c {
	case CANVAS_BOTTOMRIGHT, CANVAS_CONSOLE, CANVAS_MENU, CANVAS_STATUSBAR:
	default:
		C.GL_SetCanvas(C.canvastype(c))
	}
	C.GL_CanvasEnd()
}

func drawSet2D() {
	qCanvas = CANVAS_NONE
	SetCanvas(CANVAS_DEFAULT)
	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.BLEND)
}

func applyCanvas() (float32, float32) {
	switch qCanvas {
	case CANVAS_DEFAULT:
		gl.Viewport(0, 0, viewport.width, viewport.height)
		return 2 / float32(viewport.width), 2 / float32(viewport.height)
	case CANVAS_CONSOLE:
		gl.Viewport(0, 0, viewport.width, viewport.height)
		h := float32(console.height)
		w := float32(console.width)
		return 2 / w, 2 / h
	case CANVAS_MENU:
		s := cvars.ScreenMenuScale.Value()
		if s < 1 {
			s = 1
		}
		dw := float32(viewport.width) / 320
		if s > dw {
			s = dw
		}
		dh := float32(viewport.height) / 200
		if s > dh {
			s = dh
		}
		gl.Viewport(
			int32((float32(viewport.width)-320*s)/2),
			int32((float32(viewport.height)-200*s)/2),
			int32(640*s), int32(200*s))
		return float32(2) / 640, float32(2) / 200
	case CANVAS_STATUSBAR:
		w := float32(viewport.width)
		s := cvars.ScreenStatusbarScale.Value()
		if s < 1 {
			s = 1
		}
		dw := w / 320
		if s > dw {
			s = dw
		}
		if cl.DeathMatch() {
			gl.Viewport(0, 0, viewport.width, int32(48*s))
			return 2 * s / w, float32(2) / 48
		}
		gl.Viewport(
			int32((float32(viewport.width)-320*s)/2),
			0,
			int32(320*s),
			int32(48*s))
		return float32(2) / 320, float32(2) / 48
	case CANVAS_WARPIMAGE:
		gl.Viewport(0, viewport.height-glWarpImageSize, glWarpImageSize, glWarpImageSize)
		return float32(2) / 128, float32(2) / 128
	case CANVAS_BOTTOMRIGHT:
		s := float32(viewport.width) / float32(console.width)
		gl.Viewport(
			int32(float32(viewport.width)-320*s),
			0,
			int32(320*s),
			int32(200*s))
		return float32(2) / 320, float32(2) / 200
	default:
		// case CANVAS_NONE:
		Error("SetCanvas: bad canvas type")
		return 0, 0
	}
}

func DrawCrosshair() {
	// s := cvars.ScreenCrosshairScale.Value()
	// if s < 1 { s = 1 } else if (s > 10) { s = 10 }
	// 2s/screen.vrect.width 0 0 0
	// 0 2s/screen.vrect.height 0 0
	// 0 0 0 0
	// 0 0 0 1
	//gl.Viewport(
	// screen.vrect.x,
	// viewport.height - screen.vrect.y - screen.vrect.height,
	// screen.vrect.width &^1,
	// screen.vrect.height &^1)

	// DrawCharacterWhite(-4, -4, '+')
}

type Color struct {
	R, G, B, A float32
}

type QPic struct {
	Width   int
	Height  int
	Texture *texture.Texture
}

func DrawCharacterWhite(x, y int, num int) {
	if y <= -8 {
		// Off screen
		return
	}
	num &= 255
	if num == 32 {
		return
	}
	qDrawer.DrawQuad(float32(x), float32(y), byte(num))
}

func DrawCharacterCopper(x, y int, num int) {
	if y <= -8 {
		// Off screen
		return
	}
	num += 128
	num &= 255
	qDrawer.DrawQuad(float32(x), float32(y), byte(num))
}

func DrawPicture(x, y int, p *QPic) {
	qDrawer.Draw(float32(x), float32(y), float32(p.Width), float32(p.Height), p.Texture)
}

func DrawPictureAlpha(x, y int, p *QPic, alpha float32) {
	gl.BlendColor(0, 0, 0, alpha)
	gl.BlendFunc(gl.CONSTANT_ALPHA, gl.ONE_MINUS_CONSTANT_ALPHA)
	gl.Enable(gl.BLEND)

	qDrawer.Draw(float32(x), float32(y), float32(p.Width), float32(p.Height), p.Texture)

	gl.Disable(gl.BLEND)

	// TODO(therjak): why reset the blend func? who misses to set it?
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}

var (
	drawTop    = -2
	drawBottom = -2
)

func DrawPictureTranslate(x, y int, p *QPic, top, bottom int) {
	if top != drawTop || bottom != drawBottom {
		drawTop = top
		drawBottom = bottom
		// TODO(therjak): do the mapping
		textureManager.ReloadImage(p.Texture)
	}

	qDrawer.Draw(float32(x), float32(y), float32(p.Width), float32(p.Height), p.Texture)
}

func DrawConsoleBackground() {
	pic := GetCachedPicture("gfx/conback.lmp")
	pic.Width = console.width
	pic.Height = console.height

	alpha := float32(1.0)
	if !console.forceDuplication {
		alpha = cvars.ScreenConsoleAlpha.Value()
	}
	if alpha <= 0 {
		return
	}

	SetCanvas(CANVAS_CONSOLE)

	if alpha < 1 {
		gl.BlendColor(0, 0, 0, alpha)
		gl.BlendFunc(gl.CONSTANT_ALPHA, gl.ONE_MINUS_CONSTANT_ALPHA)
		gl.Enable(gl.BLEND)
	}

	//TODO(therjak): this should be without alpha test
	qDrawer.Draw(0, 0, float32(pic.Width), float32(pic.Height), pic.Texture)

	if alpha < 1 {
		gl.Disable(gl.BLEND)
		// TODO(therjak): why do we need to reset the blend func?
		// check each gl.Enable(BLEND) to set it before
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}
}

func DrawFadeScreen() {
	SetCanvas(CANVAS_DEFAULT)
	c := Color{0, 0, 0, 0.5}
	qRecDrawer.Draw(0, 0, float32(viewport.width), float32(viewport.height), c)
	statusbar.MarkChanged()
}

func DrawFill(x, y, w, h int, c int, alpha float32) {
	col := Color{
		R: float32(palette.table[c*4]) / 255,
		G: float32(palette.table[c*4+1]) / 255,
		B: float32(palette.table[c*4+2]) / 255,
		A: alpha,
	}
	qRecDrawer.Draw(float32(x), float32(y), float32(w), float32(h), col)
}

func DrawTileClear(xi, yi, wi, hi int) {
	x, y, w, h := float32(xi), float32(yi), float32(wi), float32(hi)
	qDrawer.TileClear(x, y, w, h)
}

func (d *drawer) TileClear(x, y, w, h float32) {
	sx, sy := applyCanvas()
	x1, x2 := x, x+w
	y1, y2 := y+h, y
	x1 = x1*sx - 1
	x2 = x2*sx - 1
	ys := yShift()
	y1 = -y1*sy + ys
	y2 = -y2*sy + ys
	vertices := []float32{
		x1, y2, 0, x / 64, y / 64,
		x2, y2, 0, (x + w) / 64, y / 64,
		x2, y1, 0, (x + w) / 64, (y + h) / 64,
		x1, y1, 0, x / 64, (y + h) / 64,
	}

	d.prog.Use()
	d.vao.Bind()
	d.ebo.Bind(gl.ELEMENT_ARRAY_BUFFER)
	d.vbo.Bind(gl.ARRAY_BUFFER)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 3, gl.FLOAT, false, 4*5, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(d.texcoord)
	gl.VertexAttribPointer(d.texcoord, 2, gl.FLOAT, false, 4*5, gl.PtrOffset(3*4))

	color := []float32{1.0, 1.0, 1.0, 1.0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &(color[0]))

	textureManager.Bind(backtileTexture)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))

	gl.DisableVertexAttribArray(d.texcoord)
	gl.DisableVertexAttribArray(d.position)
}

var (
	cachePics map[string]*QPic
)

func init() {
	cachePics = make(map[string]*QPic)
}

func GetCachedPicture(name string) *QPic {
	p, ok := cachePics[name]
	if ok {
		return p
	}
	p, err := loadPicFromFile(name)
	if err != nil {
		Error("GetCachedPicture: failed to load %s", name)
	}
	cachePics[name] = p
	return p
}

func loadPicFromFile(name string) (*QPic, error) {
	b, err := filesystem.GetFileContents(name)
	if err != nil {
		return nil, err
	}
	p := &QPic{
		Width:  int(binary.LittleEndian.Uint32(b[0:])),
		Height: int(binary.LittleEndian.Uint32(b[4:])),
	}
	p.Texture = textureManager.LoadWadTex(name, p.Width, p.Height, b[8:])
	return p, nil
}

var nullPic *QPic

func getNullPic() *QPic {
	if nullPic == nil {
		nullPic = GetPictureFromBytes("nul", 8, 8, []byte{
			252, 252, 252, 252, 0, 0, 0, 0,
			252, 252, 252, 252, 0, 0, 0, 0,
			252, 252, 252, 252, 0, 0, 0, 0,
			252, 252, 252, 252, 0, 0, 0, 0,
			0, 0, 0, 0, 252, 252, 252, 252,
			0, 0, 0, 0, 252, 252, 252, 252,
			0, 0, 0, 0, 252, 252, 252, 252,
			0, 0, 0, 0, 252, 252, 252, 252},
		)
	}
	return nullPic
}

func GetPictureFromWad(name string) *QPic {
	p := wad.GetPic(name)
	if p == nil {
		return getNullPic()
	}
	n := fmt.Sprintf("gfx.wad:%s", name)
	t := textureManager.LoadWadTex(n, p.Width, p.Height, p.Data)
	return &QPic{
		Width:   p.Width,
		Height:  p.Height,
		Texture: t,
	}
}

func GetPictureFromBytes(n string, w, h int, d []byte) *QPic {
	t := textureManager.LoadInternalTex(n, w, h, d)
	return &QPic{
		Width:   w,
		Height:  h,
		Texture: t,
	}
}

// 0-127 are white
// 128+ are normal
// We draw on a 320x200 screen
func DrawStringCopper(x, y int, t string) {
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterCopper(nx, y, int(t[i]))
		nx += 8
	}
}

func DrawStringWhite(x, y int, t string) {
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterWhite(nx, y, int(t[i]))
		nx += 8
	}
}
