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
	"unsafe"

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
	CANVAS_BOTTOMLEFT  canvas = C.CANVAS_BOTTOMLEFT
	CANVAS_BOTTOMRIGHT canvas = C.CANVAS_BOTTOMRIGHT
	CANVAS_TOPRIGHT    canvas = C.CANVAS_TOPRIGHT
	CANVAS_INVALID     canvas = C.CANVAS_INVALID
)

//export Draw_Init
func Draw_Init() {
	C.Draw_LoadPics()
}

func SetCanvas(c canvas) {
	C.GL_SetCanvas(C.canvastype(c))
}

type Color struct {
	R, G, B, A uint8
}

type Picture *C.qpic_t

type QPic struct {
	pic     Picture
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
	// TODO(therjak): this cast must die. they do not even have the same size...
	if p.pic != nil {
		d := (*C.glpic_t)(unsafe.Pointer(&p.pic.data[0]))
		pic := C.QPIC{
			width:   p.pic.width,
			height:  p.pic.height,
			texture: d.gltexture,
			sl:      0,
			tl:      0,
			sh:      1,
			th:      1,
		}
		C.Draw_Pic2(C.int(x), C.int(y), pic)
	} else {
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
}

func DrawPictureAlpha(x, y int, p *QPic, alpha float32) {
	// TODO(therjak): this cast must die. they do not even have the same size...
	gl.BlendColor(0, 0, 0, alpha)
	gl.BlendFunc(gl.CONSTANT_ALPHA, gl.ONE_MINUS_CONSTANT_ALPHA)
	gl.Enable(gl.BLEND)

	if p.pic != nil {
		d := (*C.glpic_t)(unsafe.Pointer(&p.pic.data[0]))
		pic := C.QPIC{
			width:   p.pic.width,
			height:  p.pic.height,
			texture: d.gltexture,
			sl:      d.sl,
			tl:      d.tl,
			sh:      d.sh,
			th:      d.th,
		}
		C.Draw_Pic2(C.int(x), C.int(y), pic)
	} else {
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

	gl.Disable(gl.BLEND)
}

func DrawTransparentPictureTranslate(x, y int, p *QPic, top, bottom int) {
	// TODO(therjak): this cast must die. they do not even have the same size...
	if p.pic != nil {
		d := (*C.glpic_t)(unsafe.Pointer(&p.pic.data[0]))
		pic := C.QPIC{
			width:   p.pic.width,
			height:  p.pic.height,
			texture: d.gltexture,
			sl:      d.sl,
			tl:      d.tl,
			sh:      d.sh,
			th:      d.th,
		}
		C.Draw_TransPicTranslate2(C.int(x), C.int(y), pic, C.int(top), C.int(bottom))
	} else {
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

var (
	vertexShaderSource = `#version 330 core
  layout (location = 0) in vec3 aPos;

  void main() {
    gl_Position = vec4(aPos.x, aPos.y, aPos.z, 1.0);
	}
	`
	fragmentShaderSource = `#version 330 core
	out vec4 FragColor;

	void main() {
		FragColor = vec4(1.0f, 0.5f, 0.2f, 1.0f);
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

//getShader(vertexShaderSource, gl.VERTEX_SHADER)
//getShader(fragmentShaderSoure, gl.FRAGMENT_SHADER)

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
