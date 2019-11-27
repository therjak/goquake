package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// int GetRFrameCount();
// extern int gl_warpimagesize;
import "C"

import (
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvar"
	"quake/cvars"
	"quake/image"
	"quake/wad"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
	// "quake/crc"
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
	maxTextureSize int32
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
	crc          uint16 // for some caching
	boundFrame   int
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
	cvars.GlFullBrights.SetCallback(func(_ *cvar.Cvar) {
		textureManager.reloadNoBrightImages()
	})
}

//export GetNoTexture
func GetNoTexture() TexID {
	return TexID(noTexture.glID)
}

//export GetTextureWidth
func GetTextureWidth(id TexID) int32 {
	return int32(texmap[id].glWidth)
}

//export GetTextureHeight
func GetTextureHeight(id TexID) int32 {
	return int32(texmap[id].glHeight)
}

//export TexMgrLoadLightMapImage
func TexMgrLoadLightMapImage(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, data *C.byte, flags C.unsigned) TexID {

	// TODO(therjak): add cache ala
	// if TexPrefOverWrite && owner&name&crc match
	//  return old one

	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      int32(width),
		glHeight:     int32(height),
		sourceWidth:  int32(width),
		sourceHeight: int32(height),
		flags:        TexPref(flags),
		name:         C.GoString(name),
	}
	textureManager.addActiveTexture(t)
	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	textureManager.loadLightMap(t, d)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

//export Go_LoadWad
func Go_LoadWad() {
	err := wad.Load()
	if err != nil {
		Error("Could not load wad: %v", err)
	}
}

//export TexMgrLoadConsoleChars
func TexMgrLoadConsoleChars() TexID {
	data := wad.GetConsoleChars()
	if len(data) != 128*128 {
		conlog.Printf("ConsoleChars not found")
		return 0
	}
	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      128,
		glHeight:     128,
		sourceWidth:  128,
		sourceHeight: 128,
		flags:        TexPrefAlpha | TexPrefNearest | TexPrefNoPicMip | TexPrefConChars,
		name:         "gfx.wad:conchars",
	}
	textureManager.addActiveTexture(t)
	textureManager.loadIndexed(t, data)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

func (tm *texMgr) LoadInternalTex(name string, w, h int, data []byte) *Texture {
	flags := TexPrefNearest | TexPrefAlpha | TexPrefPersist |
		TexPrefPad | TexPrefNoPicMip
	return tm.loadIndexdTex(name, w, h, flags, data)
}

func (tm *texMgr) LoadWadTex(name string, w, h int, data []byte) *Texture {
	flags := TexPrefAlpha | TexPrefPad | TexPrefNoPicMip
	return tm.loadIndexdTex(name, w, h, flags, data)
}

func (tm *texMgr) loadIndexdTex(name string, w, h int, flags TexPref, data []byte) *Texture {
	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      int32(w),
		glHeight:     int32(h),
		sourceWidth:  int32(w),
		sourceHeight: int32(h),
		flags:        flags,
		name:         name,
	}
	tm.addActiveTexture(t)
	tm.loadIndexed(t, data)
	texmap[TexID(t.glID)] = t
	return t
}

//export TexMgrLoadBacktile
func TexMgrLoadBacktile() TexID {
	name := "backtile"
	p := wad.GetPic(name)
	if p == nil {
		return 0
	}
	t := textureManager.LoadWadTex(name, p.Width, p.Height, p.Data)
	return TexID(t.glID)
}

//export TexMgrLoadParticleImage
func TexMgrLoadParticleImage(name *C.char, width C.int,
	height C.int, data *C.byte) TexID {
	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      int32(width),
		glHeight:     int32(height),
		sourceWidth:  int32(width),
		sourceHeight: int32(height),
		flags:        TexPrefPersist | TexPrefAlpha | TexPrefLinear,
		name:         C.GoString(name),
	}
	textureManager.addActiveTexture(t)
	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	textureManager.loadRGBA(t, d)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

//export TexMgrLoadSkyTexture
func TexMgrLoadSkyTexture(name *C.char, data *C.byte, flags C.unsigned) TexID {

	n := C.GoString(name)
	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      128,
		glHeight:     128,
		sourceWidth:  128,
		sourceHeight: 128,
		name:         n,
		flags:        TexPref(flags),
	}
	textureManager.addActiveTexture(t)
	d := C.GoBytes(unsafe.Pointer(data), 128*128)
	textureManager.loadIndexed(t, d)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

//export TexMgrLoadSkyBox
func TexMgrLoadSkyBox(name *C.char) TexID {
	n := C.GoString(name)
	img, err := image.Load(n)
	if err != nil {
		return 0
	}
	s := img.Bounds().Size()

	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      int32(s.X),
		glHeight:     int32(s.Y),
		sourceWidth:  int32(s.X),
		sourceHeight: int32(s.Y),
		name:         n,
	}
	textureManager.addActiveTexture(t)
	textureManager.loadRGBA(t, img.Pix)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
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
	t := texmap[id]
	// this is actually a 'map texture & reload'
	// it should work quite different to the others.
	// It also implies indexed colors
	if t.cp != nil {
		C.TexMgr_ReloadImage(t.cp, shirt, pants)
	} else {
		log.Printf("Reload without c-texture")
	}
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

func pad(s int32) int32 {
	i := int32(1)
	for ; s < i; i <<= 1 {
	}
	return i
}

func (tm *texMgr) safeTextureSize(s int32) int32 {
	cv := int32(cvars.GlMaxSize.Value())
	if cv > 0 {
		cv := pad(cv)
		if cv < s {
			s = cv
		}
	}
	if tm.maxTextureSize < s {
		return tm.maxTextureSize
	}
	return s
}

func (tm *texMgr) padConditional(s int32) int32 {
	// as support for textures with non pot 2 size is required, this is a nop
	return s
}

//export TexMgrInit
func TexMgrInit() {
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &textureManager.maxAnisotropy)
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &textureManager.maxTextureSize)
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
	// It is only called on VID_Restart (resolution change, vid_restart)
	textureManager.ReloadImages()
}

//export TexMgrRecalcWarpImageSize
func TexMgrRecalcWarpImageSize() {
	textureManager.RecalcWarpImageSize()
}

func (tm *texMgr) RecalcWarpImageSize() {
	s := tm.safeTextureSize(512)
	for s > int32(screen.Width) || s > int32(screen.Height) {
		s >>= 1
	}
	C.gl_warpimagesize = C.int(s)

	// TODO(therjak): there should be a better way.
	dummy := make([]float32, s*s*3)
	for t, b := range tm.activeTextures {
		if b && (t.flags&TexPrefWarpImage != 0) {
			tm.Bind(t)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, s, s, 0, gl.RGB, gl.FLOAT, gl.Ptr(dummy))
			t.glWidth = s
			t.glHeight = s
		}
	}
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
	if texmap[id].glID != uint32(id) {
		log.Printf("broken glID: %v, %v", texmap[id].glID, id)
	}
}

func (tm *texMgr) Bind(t *Texture) {
	if t == nil {
		t = nullTexture
	}
	if t.glID != tm.currentTexture[tm.currentTarget-gl.TEXTURE0] {
		tm.currentTexture[tm.currentTarget-gl.TEXTURE0] = t.glID
		gl.BindTexture(gl.TEXTURE_2D, t.glID)
		t.boundFrame = int(C.GetRFrameCount())
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

func (tm *texMgr) reloadNoBrightImages() {
	for k, v := range tm.activeTextures {
		if v {
			if k.flags&TexPrefNoBright != 0 {
				tm.ReloadImage(k)
			}
		}
	}
}

var inReloadImages = false

func (tm *texMgr) ReloadImages() {
	// This is the reverse of TexMgrFreeTexturesObjects

	// Workaround for some recursion
	inReloadImages = true
	for k, v := range tm.activeTextures {
		if v {
			gl.GenTextures(1, &k.glID)
			k.cp.texnum = C.uint(k.glID)
			tm.ReloadImage(k)
		}
	}
	inReloadImages = false
}

func (tm *texMgr) ReloadImage(t *Texture) {
	C.TexMgr_ReloadImage(t.cp, -1, -1)
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

// TexMgr_LoadImage32
func (tm *texMgr) loadRGBA(t *Texture, data []byte) {
	picmip := uint32(0)
	if t.flags&TexPrefNoPicMip == 0 {
		pv := cvars.GlPicMip.Value()
		if pv > 0 {
			picmip = uint32(pv)
		}
	}
	safeW := tm.safeTextureSize(t.glWidth >> picmip)
	safeH := tm.safeTextureSize(t.glHeight >> picmip)
	for t.glWidth > safeW {
		log.Printf("safeW")
		// half width
		data = downScaleWidth(t.glWidth, t.glHeight, data)
		t.glWidth >>= 1
		// if t.flags&TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.glWidth, t.glHeight, data)
		// }
	}
	for t.glHeight > safeH {
		log.Printf("safeH")
		// half height
		data = downScaleHeight(t.glWidth, t.glHeight, data)
		t.glHeight >>= 1
		// if t.flags&TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.glWidth, t.glHeight, data)
		//}
	}
	// Orig uses the 'old' values 3 or 4
	internalformat := int32(gl.RGB)
	if t.flags&TexPrefAlpha != 0 {
		internalformat = gl.RGBA
	}
	tm.Bind(t)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internalformat, t.glWidth, t.glHeight,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))

	gl.GenerateMipmap(gl.TEXTURE_2D)
	tm.SetFilterModes(t)
}

// TexMgr_LoadLightmap
func (tm *texMgr) loadLightMap(t *Texture, data []byte) {
	tm.Bind(t)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, t.glWidth, t.glHeight,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	tm.SetFilterModes(t)
}

// TexMgr_LoadImage8
func (tm *texMgr) loadIndexed(t *Texture, data []byte) {
	var p *[256 * 4]byte
	switch {
	case t.flags&(TexPrefFullBright|TexPrefAlpha) ==
		(TexPrefFullBright | TexPrefAlpha):
		p = &palette.tableFullBrightFence
	case t.flags&TexPrefFullBright != 0:
		p = &palette.tableFullBright
	case t.flags&(TexPrefNoBright|TexPrefAlpha) ==
		(TexPrefNoBright | TexPrefAlpha):
		p = &palette.tableNoBrightFence
	case t.flags&TexPrefNoBright != 0:
		p = &palette.tableNoBright
	case t.flags&TexPrefConChars != 0:
		p = &palette.tableConsoleChars
	default:
		p = &palette.table
	}
	nd := make([]byte, 0, len(data)*4)
	// TODO(therjak): add workaround for 'shot1sid' texture
	// do we actually need padding?
	for _, d := range data {
		idx := int(d) * 4
		pixel := p[idx : idx+4]
		nd = append(nd, pixel...)
	}
	// do we actually need padding?
	if t.flags&TexPrefAlpha != 0 {
		alphaEdgeFix(t.glWidth, t.glHeight, nd)
	}
	tm.loadRGBA(t, nd)
}

func alphaEdgeFix(w, h int32, d []byte) {
	alpha := func(p int32) byte {
		return d[p+3]
	}
	for y := int32(0); y < h; y++ {
		prev := (y - 1 + h) % h
		next := (y + 1) % h
		for x := int32(0); x < w; x++ {
			pp := (x - 1 + w) % w
			np := (x + 1) % w
			prow := prev * w
			crow := y * w
			nrow := next * w
			p := []int32{
				(pp + prow) * 4, (x + prow) * 4, (np + prow) * 4,
				(pp + crow) * 4 /*           */, (np + crow) * 4,
				(pp + nrow) * 4, (x + nrow) * 4, (np + nrow) * 4,
			}
			pixel := (x + crow) * 4
			if alpha(pixel) == 0 {
				r, g, b := int32(0), int32(0), int32(0)
				count := int32(0)
				for _, rp := range p {
					if alpha(rp) != 0 {
						r += int32(d[rp])
						g += int32(d[rp+1])
						b += int32(d[rp+2])
						count++
					}
				}
				if count != 0 {
					d[pixel] = byte(r / count)
					d[pixel+1] = byte(g / count)
					d[pixel+2] = byte(b / count)
				}
			}
		}
	}
}

func merge3Pixel(p1, p2, p3 []byte) [4]byte {
	r1 := float32(p1[0]) / 255
	g1 := float32(p1[1]) / 255
	b1 := float32(p1[2]) / 255
	a1 := float32(p1[3]) / 255
	r2 := float32(p2[0]) / 255
	g2 := float32(p2[1]) / 255
	b2 := float32(p2[2]) / 255
	a2 := float32(p2[3]) / 255
	r3 := float32(p3[0]) / 255
	g3 := float32(p3[1]) / 255
	b3 := float32(p3[2]) / 255
	a3 := float32(p3[3]) / 255
	return [4]byte{
		byte(((r1*a1 + r2*a2 + r3*a3) * 255) / 3),
		byte(((g1*a1 + g2*a2 + g3*a3) * 255) / 3),
		byte(((b1*a1 + b2*a2 + b3*a3) * 255) / 3),
		byte(((a1 + a2 + a3) * 255) / 3),
	}
}

func downScaleWidth(width int32, height int32, data []byte) []byte {
	ndata := make([]byte, len(data)/2)
	for y := int32(0); y < height; y++ {
		for x := int32(0); x < width; x += 2 {
			pp := (x - 1 + width) % width
			np := (x + 1) % width
			row := y * width
			p := (pp + row) * 4
			c := (x + row) * 4
			n := (np + row) * 4
			pixel := merge3Pixel(data[p:p+4], data[c:c+4], data[n:n+4])
			ndata = append(ndata, pixel[:]...)
		}
	}
	return ndata
}

func downScaleHeight(width int32, height int32, data []byte) []byte {
	ndata := make([]byte, len(data)/2)
	for y := int32(0); y < height; y += 2 {
		prev := (y - 1 + height) % height
		next := (y + 1) % height
		for x := int32(0); x < width; x++ {
			p := (x + prev*width) * 4
			c := (x + y*width) * 4
			n := (x + next*width) * 4
			pixel := merge3Pixel(data[p:p+4], data[c:c+4], data[n:n+4])
			ndata = append(ndata, pixel[:]...)
		}
	}
	return ndata
}
