package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// int GetRFrameCount();
import "C"

import (
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvars"
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
}

type TexID uint32

type texMgr struct {
	multiTextureEnabled bool
	currentTarget       uint32
	currentTexture      [3]uint32
	glModeIndex         int // TODO(therjak): glmode_idx is still split between c and go

	activeTextures map[*Texture]bool
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
	}
	noTexture   *Texture
	nullTexture *Texture
)

func init() {
	texmap = make(map[TexID]*Texture)
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
	texmap[TexID(t.texnum)] = ConvertCTex(t)

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
	texmap[TexID(t.texnum)] = ConvertCTex(t)

	return t
}

//export TexMgrReloadImage
func TexMgrReloadImage(id TexID, shirt C.int, pants C.int) {
	C.TexMgr_ReloadImage(texmap[id].cp, shirt, pants)
}

//export TexMgrFreeTexture
func TexMgrFreeTexture(id TexID) {
	C.TexMgr_FreeTexture(texmap[id].cp)
}

//export TexMgrFrameUsage
func TexMgrFrameUsage() float32 {
	return float32(C.TexMgr_FrameUsage())
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
	}
}

//export TexMgrInit
func TexMgrInit() {
	C.TexMgr_Init()
	nullTexture = ConvertCTex(C.nulltexture)
	noTexture = ConvertCTex(C.notexture)
}

//export TexMgrDeleteTextureObjects
func TexMgrDeleteTextureObjects() {
	// This only discards all opengl objects. They get recreated
	// in TexMgrReloadImages
	C.TexMgr_DeleteTextureObjects()
}

//export TexMgrReloadImages
func TexMgrReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects
	C.TexMgr_ReloadImages()
}

//export TexMgrReloadNobrightImages
func TexMgrReloadNobrightImages() {
	C.TexMgr_ReloadNobrightImages()
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
	textureManager.addActiveTexture(ConvertCTex(t))
	var tn uint32
	gl.GenTextures(1, &tn)
	t.texnum = C.uint(tn)
}

//export GL_TexParameterf
func GL_TexParameterf(target uint32, pname uint32, param float32) {
	gl.TexParameterf(target, pname, param)
}

//export GL_GetIntegerv
func GL_GetIntegerv(pname uint32, data *int32) {
	gl.GetIntegerv(pname, data)
}
