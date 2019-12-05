package quakelib

//#include "stdlib.h"
//#include "draw.h"
//void GL_SetCanvas(canvastype newCanvas);
//void Draw_CharacterQuad(int x, int y, char num);
//void Draw_TileClear(int x, int y, int w, int h);
//void Draw_Fill(int x, int y, int w, int h, int c, float alpha);
//void Draw_FadeScreen(void);
//void Draw_String(int x, int y, const char *str);
//void Draw_Pic2(int x, int y, QPIC pic);
//void Draw_TransPicTranslate2(int x, int y, QPIC pic, int top, int bottom);
//void Draw_LoadPics(void);
import "C"

import (
	"encoding/binary"
	"fmt"
	"quake/cvars"
	"quake/filesystem"
	"quake/wad"
	"strings"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type canvas int

const (
	CANVAS_NONE        canvas = C.CANVAS_NONE
	CANVAS_DEFAULT     canvas = C.CANVAS_DEFAULT
	CANVAS_CONSOLE     canvas = C.CANVAS_CONSOLE
	CANVAS_MENU        canvas = C.CANVAS_MENU
	CANVAS_STATUSBAR   canvas = C.CANVAS_SBAR
	CANVAS_WARPIMAGE   canvas = C.CANVAS_WARPIMAGE
	CANVAS_CROSSHAIR   canvas = C.CANVAS_CROSSHAIR
	CANVAS_BOTTOMRIGHT canvas = C.CANVAS_BOTTOMRIGHT
)

const (
	vertexSourceDrawer = `
#version 410
in vec2 position;
in vec2 texcoord;
out vec2 Texcoord;

void main() {
	Texcoord = texcoord;
	gl_Position = vec4(position, 0.0, 1.0);
}
`
	fragmentSourceDrawer = `
#version 410
in vec2 Texcoord;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
  frag_color = texture(tex, Texcoord);
}
`
)

func getShader(src string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)
	csource, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csource, nil)
	free()
	gl.CompileShader(shader)
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		Error("Failed to compile shader: %v", log)
	}
	return shader
}

func drawRect(x, y, w, h float32, c Color) {
	/*
		fx, fy, fw, fh := float32(x), float32(y), float32(w), float32(h)
		vertices := []float32{
			fx, fy, 0,
			fx + fw, fy, 0,
			fx + fw, fy + fh, 0,

			fx + fw, fy, 0,
			fx + fw, fy + fh, 0,
			fx, fy + fh, 0,
		}
		var VBO uint32
		gl.GenBuffers(1, &VBO)
		gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
		gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)
		gl.Disable(gl.TEXTURE_2D)
		gl.Enable(gl.BLEND)
		// gl.BlendColor(float32(c*4)/255,....,alpha)
		// gl.Begin(gl.QUADS)
		// gl.End()
		gl.Disable(gl.BLEND)
		gl.Enable(gl.TEXTURE_2D)
	*/

}

type drawer struct {
	vertices []float32
	elements []uint32
	vao      uint32
	vbo      uint32
	ebo      uint32
	prog     *drawProgram
	position uint32
	texcoord uint32
}

func NewDrawer() *drawer {
	d := &drawer{
		vertices: []float32{
			// position, textureCoord
			-0.5, 0.5, 0.0, 0.0, // Top-left
			0.5, 0.5, 1.0, 0.0, // Top-right
			0.5, -0.5, 1.0, 1.0, // Bottom-right
			-0.5, -0.5, 0.0, 1.0, // Bottom-left
		},
		elements: []uint32{
			0, 1, 2,
			2, 3, 0,
		},
	}
	gl.GenVertexArrays(1, &d.vao)
	gl.GenBuffers(1, &d.vbo)
	gl.GenBuffers(1, &d.ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, d.ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(d.elements), gl.Ptr(d.elements), gl.STATIC_DRAW)
	d.prog = newDrawProgram()
	d.position = d.prog.getAttribLocation("position\x00")
	d.texcoord = d.prog.getAttribLocation("texcoord\x00")

	return d
}

func (d *drawer) Draw(x, y, w, h float32, t *Texture) {
	applyCanvas()

	d.vertices[0] = x
	d.vertices[1] = y + h
	d.vertices[4] = x + w
	d.vertices[5] = y + h
	d.vertices[8] = x + w
	d.vertices[9] = y
	d.vertices[12] = x
	d.vertices[13] = y

	//TODO(therjak): include the glOrtho and glViewport stuff from GL_SetCanvas

	gl.UseProgram(d.prog.prog)
	gl.BindVertexArray(d.vao)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, d.ebo)
	gl.BindBuffer(gl.ARRAY_BUFFER, d.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(d.vertices), gl.Ptr(d.vertices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(d.texcoord)
	gl.VertexAttribPointer(d.texcoord, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))

	textureManager.Bind(t)

	gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
}

func (d *drawer) Delete() {
	d.prog.Delete()
	gl.DeleteBuffers(1, &d.ebo)
	gl.DeleteBuffers(1, &d.vbo)
	gl.DeleteVertexArrays(1, &d.vao)
}

type drawProgram struct {
	frag uint32
	vert uint32
	prog uint32
}

func newDrawProgram() *drawProgram {
	d := &drawProgram{
		vert: getShader(vertexSourceDrawer, gl.VERTEX_SHADER),
		frag: getShader(fragmentSourceDrawer, gl.FRAGMENT_SHADER),
		prog: gl.CreateProgram(),
	}
	gl.AttachShader(d.prog, d.vert)
	gl.AttachShader(d.prog, d.frag)
	gl.LinkProgram(d.prog)
	return d
}

func (d *drawProgram) getAttribLocation(attrib string) uint32 {
	return uint32(gl.GetAttribLocation(d.prog, gl.Str(attrib)))
}

func (d *drawProgram) Delete() {
	gl.DeleteProgram(d.prog)
	gl.DeleteShader(d.vert)
	gl.DeleteShader(d.frag)
}

var (
	qDrawer *drawer
)

//export Draw_Init
func Draw_Init() {
	C.Draw_LoadPics()
	qDrawer = NewDrawer()
}

//export Draw_Destroy
func Draw_Destroy() {
	qDrawer.Delete()
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
	C.GL_SetCanvas(C.canvastype(c))
}

func applyCanvas() {
	switch qCanvas {
	case CANVAS_DEFAULT:
		gl.Viewport(0, 0, viewport.width, viewport.height)
	case CANVAS_CONSOLE:
		gl.Viewport(0, 0, viewport.width, viewport.height)
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
	case CANVAS_STATUSBAR:
		s := cvars.ScreenStatusbarScale.Value()
		if s < 1 {
			s = 1
		}
		dw := float32(viewport.width) / 320
		if s > dw {
			s = dw
		}
		if cl.DeathMatch() {
			gl.Viewport(0, 0, viewport.width, int32(48*s))
		} else {
			gl.Viewport(
				int32((float32(viewport.width)-320*s)/2),
				0,
				int32(320*s),
				int32(48*s))
		}
	case CANVAS_WARPIMAGE:
		//gl.Viewport(0,0,viewport.width,viewport.height)
	case CANVAS_CROSSHAIR:
		//gl.Viewport(0,0,viewport.width,viewport.height)
	case CANVAS_BOTTOMRIGHT:
		s := float32(viewport.width) / float32(console.width)
		gl.Viewport(
			int32(float32(viewport.width-320)*s),
			int32(float32(viewport.height-200)*s),
			int32(320*s),
			int32(200*s))
	default:
		// case CANVAS_NONE:
		Error("SetCanvas: bad canvas type")
	}
}

type Color struct {
	R, G, B, A uint8
}

type QPic struct {
	Width   int
	Height  int
	Texture *Texture
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
	C.Draw_CharacterQuad(C.int(x), C.int(y), C.char(num))
}

func DrawCharacterCopper(x, y int, num int) {
	if y <= -8 {
		// Off screen
		return
	}
	num += 128
	num &= 255
	C.Draw_CharacterQuad(C.int(x), C.int(y), C.char(num))
}

func DrawPicture(x, y int, p *QPic) {
	//qDrawer.Draw(float32(x), float32(y), float32(p.Width), float32(p.Height), p.Texture)
	pic := C.QPIC{
		width:   C.int(p.Width),
		height:  C.int(p.Height),
		texture: C.uint32_t(p.Texture.glID),
		sl:      0,
		tl:      0,
		sh:      1,
		th:      1,
	}
	C.Draw_Pic2(C.int(x), C.int(y), pic)
}

func DrawPictureAlpha(x, y int, p *QPic, alpha float32) {
	gl.BlendColor(0, 0, 0, alpha)
	gl.BlendFunc(gl.CONSTANT_ALPHA, gl.ONE_MINUS_CONSTANT_ALPHA)
	gl.Enable(gl.BLEND)

	pic := C.QPIC{
		width:   C.int(p.Width),
		height:  C.int(p.Height),
		texture: C.uint32_t(p.Texture.glID),
		sl:      0,
		tl:      0,
		sh:      1,
		th:      1,
	}
	C.Draw_Pic2(C.int(x), C.int(y), pic)

	gl.Disable(gl.BLEND)
}

func DrawTransparentPictureTranslate(x, y int, p *QPic, top, bottom int) {
	pic := C.QPIC{
		width:   C.int(p.Width),
		height:  C.int(p.Height),
		texture: C.uint32_t(p.Texture.glID),
		sl:      0,
		tl:      0,
		sh:      1,
		th:      1,
	}
	C.Draw_TransPicTranslate2(C.int(x), C.int(y), pic, C.int(top), C.int(bottom))
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

	DrawPicture(0, 0, pic)

	if alpha < 1 {
		gl.Disable(gl.BLEND)
	}
}

func DrawFadeScreen() {
	C.Draw_FadeScreen()
}

func DrawFill(x, y, w, h int, c int, alpha float32) {
	col := Color{
		R: palette.table[c*4],
		G: palette.table[c*4+1],
		B: palette.table[c*4+2],
		A: uint8(alpha * 255),
	}
	drawRect(float32(x), float32(y), float32(w), float32(h), col)
	C.Draw_Fill(C.int(x), C.int(y), C.int(w), C.int(h), C.int(c), C.float(alpha))
}

func DrawTileClear(x, y, w, h int) {
	C.Draw_TileClear(C.int(x), C.int(y), C.int(w), C.int(h))
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
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterCopper(nx, y, int(t[i]))
		nx += 8
	}
}

func DrawStringWhite(x, y int, t string) {
	// TODO: unify into one draw call
	nx := x
	for i := 0; i < len(t); i++ {
		DrawCharacterWhite(nx, y, int(t[i]))
		nx += 8
	}
}
