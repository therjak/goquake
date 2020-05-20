package quakelib

//#include "gl_model.h"
//#include "gl_texmgr.h"
// int GetRFrameCount();
// extern int gl_warpimagesize;
// extern texture_t *r_notexture_mip, *r_notexture_mip2;
import "C"

import (
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/image"
	"github.com/therjak/goquake/wad"
	"log"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

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

type colorType int

const (
	colorTypeIndexed colorType = iota
	colorTypeRGBA
	colorTypeLightmap
)

type Texture struct {
	glID         uint32
	glWidth      int32 // mipmap can make it differ from source width
	glHeight     int32
	sourceWidth  int32
	sourceHeight int32
	flags        TexPref
	name         string
	crc          uint16 // for some caching
	boundFrame   int
	typ          colorType
	data         []byte
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

	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
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
		typ:          colorTypeLightmap,
		data:         d,
	}
	textureManager.addActiveTexture(t)
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

func (tm *texMgr) LoadConsoleChars() *Texture {
	data := wad.GetConsoleChars()
	if len(data) != 128*128 {
		Error("ConsoleChars not found")
		return nil
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
		typ:          colorTypeIndexed,
		data:         data,
	}
	textureManager.addActiveTexture(t)
	textureManager.loadIndexed(t, data)
	texmap[TexID(t.glID)] = t
	return t
}

func (tm *texMgr) LoadNoTex(name string, w, h int, data []byte) *Texture {
	flags := TexPrefNearest | TexPrefPersist | TexPrefNoPicMip
	return tm.loadRGBATex(name, w, h, flags, data)
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

func (tm *texMgr) loadRGBATex(name string, w, h int, flags TexPref, data []byte) *Texture {
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
		typ:          colorTypeRGBA,
		data:         data,
	}
	tm.addActiveTexture(t)
	tm.loadRGBA(t, data)
	texmap[TexID(t.glID)] = t
	return t
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
		typ:          colorTypeIndexed,
		data:         data,
	}
	tm.addActiveTexture(t)
	tm.loadIndexed(t, data)
	texmap[TexID(t.glID)] = t
	return t
}

func (tm *texMgr) LoadBacktile() *Texture {
	name := "backtile"
	p := wad.GetPic(name)
	if p == nil {
		Error("Draw_LoadPics: couldn't load backtile")
		return nil
	}
	return textureManager.LoadWadTex(name, p.Width, p.Height, p.Data)
}

//export TexMgrLoadParticleImage
func TexMgrLoadParticleImage(name *C.char, width C.int,
	height C.int, data *C.byte) TexID {
	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	t := textureManager.loadParticleImage(C.GoString(name), int32(width), int32(height), d)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

func (tm *texMgr) loadParticleImage(name string, width, height int32, data []byte) *Texture {
	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      width,
		glHeight:     height,
		sourceWidth:  width,
		sourceHeight: height,
		flags:        TexPrefPersist | TexPrefAlpha | TexPrefLinear,
		name:         name,
		typ:          colorTypeRGBA,
		data:         data,
	}
	textureManager.addActiveTexture(t)
	textureManager.loadRGBA(t, data)
	return t
}

//export TexMgrLoadSkyTexture
func TexMgrLoadSkyTexture(name *C.char, data *C.byte, flags C.unsigned) TexID {

	n := C.GoString(name)
	d := C.GoBytes(unsafe.Pointer(data), 128*128)
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
		typ:          colorTypeIndexed,
		data:         d,
	}
	textureManager.addActiveTexture(t)
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
		typ:          colorTypeRGBA,
		data:         img.Pix,
	}
	textureManager.addActiveTexture(t)
	textureManager.loadRGBA(t, img.Pix)
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

//export TexMgrLoadImage2
func TexMgrLoadImage2(owner *C.qmodel_t, name *C.char, width C.int,
	height C.int, format C.enum_srcformat, data *C.byte, source_file *C.char,
	source_offset C.src_offset_t, flags C.unsigned) TexID {

	d, ct := func() ([]byte, colorType) {
		switch format {
		case C.SRC_RGBA:
			return C.GoBytes(unsafe.Pointer(data), width*height*4), colorTypeRGBA
		default: // C.SRC_INDEXED
			return C.GoBytes(unsafe.Pointer(data), width*height), colorTypeIndexed
		}
	}()

	var tn uint32
	gl.GenTextures(1, &tn)
	t := &Texture{
		glID:         tn,
		glWidth:      int32(width),
		glHeight:     int32(height),
		sourceWidth:  int32(width),
		sourceHeight: int32(height),
		name:         C.GoString(name),
		flags:        TexPref(flags),
		typ:          ct,
		data:         d,
	}
	textureManager.addActiveTexture(t)
	switch format {
	case C.SRC_RGBA:
		textureManager.loadRGBA(t, d)
	default: // C.SRC_INDEXED
		textureManager.loadIndexed(t, d)
	}
	texmap[TexID(t.glID)] = t
	return TexID(t.glID)
}

//export TexMgrReloadImage
func TexMgrReloadImage(id TexID, shirt C.int, pants C.int) {
	t := texmap[id]
	textureManager.ReloadImage(t)
	// this is actually a 'map texture & reload'
	// it should work quite different to the others.
	// It also implies indexed colors
}

//export TexMgrFreeTexture
func TexMgrFreeTexture(id TexID) {
	textureManager.FreeTexture(texmap[id])
}

func (tm *texMgr) FreeTexture(t *Texture) {
	if inReloadImages {
		// Stupid workaround. Needs real fix.
		return
	}

	delete(tm.activeTextures, t)
	tm.deleteTexture(t)
}

//export TexMgrFrameUsage
func TexMgrFrameUsage() float32 {
	return textureManager.FrameUsage()
}

//export TexMgrFreeTexturesForOwner
func TexMgrFreeTexturesForOwner(owner *C.qmodel_t) {
	// TODO(therjak): free all activeTextures with this owner
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
	noTexture = textureManager.LoadNoTex("notexture", 2, 2, []byte{
		159, 91, 83, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 159, 91, 83, 255,
	})
	nullTexture = textureManager.LoadNoTex("nulltexture", 2, 2, []byte{
		127, 191, 255, 255, 0, 0, 0, 255,
		0, 0, 0, 255, 127, 191, 255, 255,
	})

	// Mod_Init is called before, so we need to do this here
	C.r_notexture_mip.gltexture = C.uint(noTexture.glID)
	C.r_notexture_mip2.gltexture = C.uint(noTexture.glID)
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

var (
	glWarpImageSize int32
)

func (tm *texMgr) RecalcWarpImageSize() {
	s := tm.safeTextureSize(512)
	for s > int32(screen.Width) || s > int32(screen.Height) {
		s >>= 1
	}
	glWarpImageSize = s
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
			tm.ReloadImage(k)
		}
	}
	inReloadImages = false
}

func (tm *texMgr) ReloadImage(t *Texture) {
	// TODO(therjak): color mapping for shirts and pants
	/*
	  if (glt->shirt > -1 && glt->pants > -1) {
	    // create new translation table
	    for (i = 0; i < 256; i++) translation[i] = i;

	    shirt = glt->shirt * 16;
	    if (shirt < 128) {
	      for (i = 0; i < 16; i++) translation[TOP_RANGE + i] = shirt + i;
	    } else {
	      for (i = 0; i < 16; i++) translation[TOP_RANGE + i] = shirt + 15 - i;
	    }

	    pants = glt->pants * 16;
	    if (pants < 128) {
	      for (i = 0; i < 16; i++) translation[BOTTOM_RANGE + i] = pants + i;
	    } else {
	      for (i = 0; i < 16; i++) translation[BOTTOM_RANGE + i] = pants + 15 - i;
	    }

	    // translate texture
	    size = glt->width * glt->height;
	    dst = translated = (byte *)Hunk_Alloc(size);
	    src = data;

	    for (i = 0; i < size; i++) *dst++ = translation[*src++];

	    data = translated;
	  }
	*/
	switch t.typ {
	case colorTypeIndexed:
		tm.loadIndexed(t, t.data)
	case colorTypeRGBA:
		tm.loadRGBA(t, t.data)
	case colorTypeLightmap:
		tm.loadLightMap(t, t.data)
	}
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

func (tm *texMgr) addActiveTexture(t *Texture) {
	tm.activeTextures[t] = true
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

func (tm *texMgr) loadLightMap(t *Texture, data []byte) {
	tm.Bind(t)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, t.glWidth, t.glHeight,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	tm.SetFilterModes(t)
}

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
