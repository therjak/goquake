package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
import "C"

type Texture C.gltexture_t
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

var (
	texmap map[TexID]TextureP
)

func init() {
	texmap = make(map[TexID]TextureP)
}

//export GetNoTexture
func GetNoTexture() TexID {
	return TexID(C.notexture.texnum)
}

//export GetTextureWidth
func GetTextureWidth(id TexID) C.uint {
	return texmap[id].width
}

//export GetTextureHeight
func GetTextureHeight(id TexID) C.uint {
	return texmap[id].height
}

//export GLBind
func GLBind(id TexID) {
	C.GL_Bind(texmap[id])
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
	texmap[TexID(t.texnum)] = t

	return TexID(t.texnum)
}

//export TexMgrReloadImage
func TexMgrReloadImage(id TexID, shirt C.int, pants C.int) {
	C.TexMgr_ReloadImage(texmap[id], shirt, pants)
}

//export TexMgrFreeTexture
func TexMgrFreeTexture(id TexID) {
	C.TexMgr_FreeTexture(texmap[id])
}

//export TexMgrFrameUsage
func TexMgrFrameUsage() float32 {
	return float32(C.TexMgr_FrameUsage())
}

//export TexMgrFreeTexturesForOwner
func TexMgrFreeTexturesForOwner(owner *C.qmodel_t) {
	C.TexMgr_FreeTexturesForOwner(owner)
}

//export TexMgrInit
func TexMgrInit() {
	C.TexMgr_Init()
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

//export GLSelectTexture
func GLSelectTexture(target uint32) {
	C.GL_SelectTexture(C.GLenum(target))
}

//export GLDisableMultitexture
func GLDisableMultitexture() {
	// selects texture unit 0
	C.GL_DisableMultitexture()
}

//export GLEnableMultitexture
func GLEnableMultitexture() {
	// selects texture unit 1
	C.GL_EnableMultitexture()
}

//export GLClearBindings
func GLClearBindings() {
	C.GL_ClearBindings()
}
