package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// int GetRFrameCount();
import "C"

import (
	"github.com/go-gl/gl/v4.6-core/gl"
	"log"
	"unsafe"
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

type TexID uint32

type texMgr struct {
	multiTextureEnabled bool
	currentTarget       uint32
	currentTexture      [3]uint32
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
}

const (
	unusedTexture = ^uint32(0)
)

var (
	texmap         map[TexID]*Texture
	textureManager = texMgr{
		currentTarget: gl.TEXTURE0,
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
	C.TexMgr_DeleteTextureObjects()
}

//export TexMgrReloadImages
func TexMgrReloadImages() {
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

//export GL_TexParameterf
func GL_TexParameterf(target uint32, pname uint32, param float32) {
	gl.TexParameterf(target, pname, param)
}

//export GL_GetIntegerv
func GL_GetIntegerv(pname uint32, data *int32) {
	gl.GetIntegerv(pname, data)
}
