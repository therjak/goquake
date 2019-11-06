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
