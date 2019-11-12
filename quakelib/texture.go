package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// int GetRFrameCount();
import "C"

import (
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

type TextureP C.gltexture_tp

// gl.TEXTURE0 == GL_TEXTURE0_ARB
// gl.TEXTURE1 == GL_TEXTURE1_ARB
// gl.TEXTURE_2D == GL_TEXTURE_2D
// GL_UNUSED_TEXTURE is quake specific "^uint32(0)"
//
// gl.ActiveTexture == GL_SelectTextureFunc
// gl.BindTexture == glBindTexture
// gl.Enable == glEnable
// gl.Disable == glDisable

type TexPref uint32

const (
	TexPrefMipMap TexPref = 1 << iota
	TexPrefLinear
	TexPrefNearest
	TexPrefAlpha
	TexPrefPad
	TexPrefPersist
	TexPrefOverwrite
	TexPrefNoPicMip
	TexPrefFullBright
	TexPrefNoBright
	TexPrefConChars
	TexPrefWarpImage
	TexPrefNone TexPref = 0
)

type glMode struct {
	magfilter float32
	minfilter float32
	name      string
}

var (
	glModes = [6]glMode{
		{gl.NEAREST, gl.NEAREST, "GL_NEAREST"},
		{gl.NEAREST, gl.NEAREST_MIPMAP_NEAREST, "GL_NEAREST_MIPMAP_NEAREST"},
		{gl.NEAREST, gl.NEAREST_MIPMAP_LINEAR, "GL_NEAREST_MIPMAP_LINEAR"},
		{gl.LINEAR, gl.LINEAR, "GL_LINEAR"},
		{gl.LINEAR, gl.LINEAR_MIPMAP_NEAREST, "GL_LINEAR_MIPMAP_NEAREST"},
		{gl.LINEAR, gl.LINEAR_MIPMAP_LINEAR, "GL_LINEAR_MIPMAP_LINEAR"},
	}
)

func describeTextureModes(_ []cmd.QArg, _ int) {
	for i, m := range glModes {
		conlog.SafePrintf("   %2d: %s", i+1, m.name)
	}
	conlog.Printf("%d modes\n", len(glModes))
}

func init() {
	cmd.AddCommand("gl_describetexturemodes", describeTextureModes)
	cmd.AddCommand("imagelist", func(_ []cmd.QArg, _ int) {
		textureManager.logTextures()
	})
}

type TexID uint32

type texMgr struct {
	multiTextureEnabled bool
	currentTarget       uint32
	currentTexture      [3]uint32
	glModeIndex         int // TODO(therjak): glmode_idx is still split between c and go

	activeTextures map[*Texture]bool
	maxAnisotropy  float32
}

//export GetMTexEnabled
func GetMTexEnabled() bool {
	return textureManager.multiTextureEnabled
}

type Texture struct {
	glID         uint32
	cp           TextureP
	glWidth      int32 // mipmap can make it differ from source width
	glHeight     int32
	sourceWidth  int32
	sourceHeight int32
	flags        TexPref
	name         string
}

const (
	unusedTexture = ^uint32(0)
)

var (
	texmap         map[TexID]*Texture
	textureManager = texMgr{
		currentTarget:  gl.TEXTURE0,
		glModeIndex:    len(glModes) - 1,
		activeTextures: make(map[*Texture]bool),
		maxAnisotropy:  1,
	}
	noTexture   *Texture
	nullTexture *Texture
)

func init() {
	texmap = make(map[TexID]*Texture)
	cvars.GlTextureMode.SetByString(glModes[textureManager.glModeIndex].name)
	cvars.GlTextureAnisotropy.SetCallback(func(cv *cvar.Cvar) {
		textureManager.anisotropyCallback(cv)
	})
	cvars.GlTextureMode.SetCallback(func(cv *cvar.Cvar) {
		textureManager.textureModeCallback(cv)
	})
}

//export GetNoTexture
func GetNoTexture() TexID {
	return TexID(noTexture.glID)
}

//export GetTextureWidth
func GetTextureWidth(id TexID) int32 {
	return int32(texmap[id].cp.width)
}

//export GetTextureHeight
func GetTextureHeight(id TexID) int32 {
	return int32(texmap[id].cp.height)
}

//export TexMgrLoadImage
func TexMgrLoadImage(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, format C.enum_srcformat, data *C.byte, source_file *C.char,
	source_offset C.src_offset_t, flags C.unsigned) TexID {

	t := C.TexMgr_LoadImage(owner, name, width,
		height, format, data,
		source_file,
		source_offset, flags)

	// Note texnum 0 is reserved in opengl so it can not natually occur.
	gt := ConvertCTex(t)
	for k, _ := range textureManager.activeTextures {
		if k.glID == gt.glID {
			(*k) = *gt
			texmap[TexID(t.texnum)] = k
		}
	}

	return TexID(t.texnum)
}

//export TexMgrLoadImage2
func TexMgrLoadImage2(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, format C.enum_srcformat, data *C.byte, source_file *C.char,
	source_offset C.src_offset_t, flags C.unsigned) TextureP {

	t := C.TexMgr_LoadImage(owner, name, width,
		height, format, data,
		source_file,
		source_offset, flags)

	// Note texnum 0 is reserved in opengl so it can not natually occur.
	gt := ConvertCTex(t)
	for k, _ := range textureManager.activeTextures {
		if k.glID == gt.glID {
			(*k) = *gt
			texmap[TexID(t.texnum)] = k
		}
	}

	return t
}

//export TexMgrReloadImage
func TexMgrReloadImage(id TexID, shirt C.int, pants C.int) {
	textureManager.ReloadImage(texmap[id], int(shirt), int(pants))
}

//export TexMgrFreeTexture
func TexMgrFreeTexture(id TexID) {
	if inReloadImages {
		// Stupid workaround. Needs real fix.
		return
	}
	C.TexMgr_FreeTexture(texmap[id].cp)
}

//export TexMgrFrameUsage
func TexMgrFrameUsage() float32 {
	return textureManager.FrameUsage()
}

//export TexMgrFreeTexturesForOwner
func TexMgrFreeTexturesForOwner(owner *C.qmodel_t) {
	C.TexMgr_FreeTexturesForOwner(owner)
}

func ConvertCTex(ct TextureP) *Texture {
	return &Texture{
		glID:         uint32(ct.texnum),
		cp:           ct,
		glWidth:      int32(ct.width),
		glHeight:     int32(ct.height),
		sourceWidth:  int32(ct.source_width),
		sourceHeight: int32(ct.source_height),
		flags:        TexPref(ct.flags),
		name:         C.GoString(&ct.name[0]),
	}
}

//export TexMgrInit
func TexMgrInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	C.TexMgr_Init()
	nullTexture = ConvertCTex(C.nulltexture)
	noTexture = ConvertCTex(C.notexture)
}

//export TexMgrDeleteTextureObjects
func TexMgrDeleteTextureObjects() {
	// This only discards all opengl objects. They get recreated
	// in TexMgrReloadImages
	textureManager.DeleteTextureObjects()
}

//export TexMgrReloadImages
func TexMgrReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects
	textureManager.ReloadImages()
}

//export TexMgrReloadNobrightImages
func TexMgrReloadNobrightImages() {
	textureManager.ReloadNoBrightImages()
}

//export TexMgrPadConditional
func TexMgrPadConditional(s int) int {
	return int(C.TexMgr_PadConditional(C.int(s)))
}

//export TexMgrRecalcWarpImageSize
func TexMgrRecalcWarpImageSize() {
	C.TexMgr_RecalcWarpImageSize()
}

//export GLDisableMultitexture
func GLDisableMultitexture() {
	textureManager.DisableMultiTexture()
}

//export GLEnableMultitexture
func GLEnableMultitexture() {
	textureManager.EnableMultiTexture()
}

//export GLSelectTexture
func GLSelectTexture(target uint32) {
	textureManager.SelectTextureUnit(target)
}

func (tm *texMgr) SelectTextureUnit(target uint32) {
	// THERJAK: we have at least 80 texture units, why use only 3?
	if target == tm.currentTarget {
		return
	}
	gl.ActiveTexture(target)
	tm.currentTarget = target
}

//export GL_Bind
func GL_Bind(t TextureP) {
	if t == nil {
		textureManager.Bind(nullTexture)
		log.Printf("Bind texture nil")
		return
	}
	textureManager.Bind(ConvertCTex(t))
	return
}

//export GLBind
func GLBind(id TexID) {
	textureManager.Bind(texmap[id])
}

func (tm *texMgr) Bind(t *Texture) {
	if t == nil {
		t = nullTexture
	}
	if t.glID != tm.currentTexture[tm.currentTarget-gl.TEXTURE0] {
		tm.currentTexture[tm.currentTarget-gl.TEXTURE0] = t.glID
		gl.BindTexture(gl.TEXTURE_2D, t.glID)
		t.cp.visframe = C.GetRFrameCount()
	}
}

//export GL_DeleteTexture2
func GL_DeleteTexture2(t TextureP) {
	textureManager.removeActiveTexture(uint32(t.texnum))
	textureManager.deleteTexture(texmap[TexID(t.texnum)])
}

//export GL_DeleteTexture
func GL_DeleteTexture(t TextureP) {
	textureManager.deleteTexture(texmap[TexID(t.texnum)])
}

func (tm *texMgr) DeleteTextureObjects() {
	// This only discards all opengl objects. They get recreated
	// in TexMgrReloadImages
	for k, v := range tm.activeTextures {
		if v {
			tm.deleteTexture(k)
		}
	}
}

var (
	inReloadImages = false
)

func (tm *texMgr) ReloadNoBrightImages() {
	for k, v := range tm.activeTextures {
		if v {
			if k.flags&TexPrefNoBright != 0 {
				tm.ReloadImage(k, -1, -1)
			}
		}
	}
}

func (tm *texMgr) ReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects

	// Workaround for some recursion
	inReloadImages = true
	for k, v := range tm.activeTextures {
		if v {
			gl.GenTextures(1, &k.glID)
			k.cp.texnum = C.uint(k.glID)
			tm.ReloadImage(k, -1, -1)
		}
	}
	inReloadImages = false
}

func (tm *texMgr) ReloadImage(t *Texture, top, bottom int) {
	C.TexMgr_ReloadImage(t.cp, C.int(top), C.int(bottom))
}

func (tm *texMgr) deleteTexture(t *Texture) {
	gl.DeleteTextures(1, &t.glID)
	if t.glID == tm.currentTexture[0] {
		tm.currentTexture[0] = unusedTexture
	}
	if t.glID == tm.currentTexture[1] {
		tm.currentTexture[1] = unusedTexture
	}
	if t.glID == tm.currentTexture[2] {
		tm.currentTexture[2] = unusedTexture
	}

	delete(texmap, TexID(t.glID))
	t.glID = 0
	t.cp.texnum = 0
}

func (tm *texMgr) DisableMultiTexture() {
	// selects texture unit 0
	if tm.multiTextureEnabled {
		gl.Disable(gl.TEXTURE_2D)
		GLSelectTexture(gl.TEXTURE0)
		tm.multiTextureEnabled = false
	}
}

func (tm *texMgr) EnableMultiTexture() {
	// selects texture unit 1
	GLSelectTexture(gl.TEXTURE1)
	gl.Enable(gl.TEXTURE_2D)
	tm.multiTextureEnabled = true
}

//export TexMgr_SetFilterModes
func TexMgr_SetFilterModes(ct TextureP) {
	t := ConvertCTex(ct)
	textureManager.SetFilterModes(t)
}

func (tm *texMgr) SetFilterModes(t *Texture) {
	tm.Bind(t)
	m := glModes[tm.glModeIndex]
	switch {
	case t.flags&TexPrefNearest != 0:
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	case t.flags&TexPrefLinear != 0:
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	case t.flags&TexPrefMipMap != 0:
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, m.magfilter)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, m.minfilter)
		v := cvars.GlTextureAnisotropy.Value()
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY, v)
	default:
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, m.magfilter)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, m.magfilter)
	}
}

func (tm *texMgr) anisotropyCallback(cv *cvar.Cvar) {
	// TexMgr_Anisotropy_f
	val := cv.Value()
	switch {
	case val < 1:
		cv.SetByString("1")
	case val > tm.maxAnisotropy:
		cv.SetValue(tm.maxAnisotropy)
	default:
		for k, v := range tm.activeTextures {
			if v {
				if k.flags&TexPrefMipMap != 0 {
					tm.Bind(k)
					m := glModes[tm.glModeIndex]
					gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, m.magfilter)
					gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, m.magfilter)
					gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY, val)
				}
			}
		}
	}
}
func (tm *texMgr) textureModeCallback(cv *cvar.Cvar) {
	name := cv.String()
	for i, m := range glModes {
		if m.name == name {
			if tm.glModeIndex != i {
				tm.glModeIndex = i
			}
			for k, v := range tm.activeTextures {
				if v {
					tm.SetFilterModes(k)
				}
			}
			statusbar.MarkChanged()
			// TODO: WarpImages need a redraw too?
			return
		}
	}
	// Try to fix the cvar value and recursivly call again
	ln := strings.ToLower(name)
	for _, m := range glModes {
		if strings.ToLower(m.name) == ln {
			cv.SetByString(m.name)
			return
		}
	}
	i, _ := strconv.Atoi(name)
	if i >= 1 && i <= len(glModes) {
		cv.SetByString(glModes[i-1].name)
		return
	}
	conlog.Printf("\"%s\" is not a valid texturemade\n", name)
	cv.SetByString(glModes[tm.glModeIndex].name)
}

//export GLClearBindings
func GLClearBindings() {
	textureManager.ClearBindings()
}

func (tm *texMgr) ClearBindings() {
	tm.currentTexture = [3]uint32{unusedTexture, unusedTexture, unusedTexture}
}

//export GL_TexImage2D
func GL_TexImage2D(target uint32, level int32, internalformat int32, width int32, height int32, border int32, format uint32, xtype uint32, pixels unsafe.Pointer) {
	gl.TexImage2D(target, level, internalformat, width, height, border, format, xtype, pixels)
}

//export GL_GenTextures
func GL_GenTextures(n int32, t *uint32) {
	gl.GenTextures(n, t)
}

func (tm *texMgr) addActiveTexture(t *Texture) {
	tm.activeTextures[t] = true
}

func (tm *texMgr) removeActiveTexture(tid uint32) {
	for k, v := range tm.activeTextures {
		if v && k.glID == tid {
			delete(tm.activeTextures, k)
			return
		}
	}
}

//export GL_GenTextures2
func GL_GenTextures2(t TextureP) {
	var tn uint32
	gl.GenTextures(1, &tn)
	t.texnum = C.uint(tn)
	textureManager.addActiveTexture(ConvertCTex(t))
}

//export GL_TexParameterf
func GL_TexParameterf(target uint32, pname uint32, param float32) {
	gl.TexParameterf(target, pname, param)
}

//export GL_GetIntegerv
func GL_GetIntegerv(pname uint32, data *int32) {
	gl.GetIntegerv(pname, data)
}

func (tm *texMgr) logTextures() {
	texels, mb := tm.getTextureUsage()
	conlog.Printf("%d textures %d pixels %.1f megabytes\n",
		len(tm.activeTextures), texels, mb)
}

func (tm *texMgr) FrameUsage() float32 {
	_, mb := tm.getTextureUsage()
	return mb
}

func (tm *texMgr) getTextureUsage() (int32, float32) {
	texels := int32(0)
	for k, v := range tm.activeTextures {
		log.Printf("Texture %s, %s, %d, %d, %v",
			k.name, C.GoString(&(k.cp.name[0])), k.cp.height, k.cp.width, v)
		if v {
			if k.flags&TexPrefMipMap != 0 {
				texels += k.glWidth * k.glHeight * 4 / 3
			} else {
				texels += k.glWidth * k.glHeight
			}
		}
	}
	mb := float32(texels) * (cvars.VideoBitsPerPixel.Value() / 8) / (1000 * 1000)
	return texels, mb
}
