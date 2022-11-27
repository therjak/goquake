// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"image"
	"log"
	"strconv"
	"strings"

	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/glh"
	qimage "goquake/image"
	"goquake/palette"
	"goquake/texture"
	"goquake/wad"

	"github.com/go-gl/gl/v4.6-core/gl"
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

func describeTextureModes(_ cmd.Arguments, p, s int) error {
	for i, m := range glModes {
		conlog.SafePrintf("   %2d: %s", i+1, m.name)
	}
	conlog.Printf("%d modes\n", len(glModes))
	return nil
}

func init() {
	addCommand("gl_describetexturemodes", describeTextureModes)
	addCommand("imagelist", func(_ cmd.Arguments, p, s int) error {
		textureManager.logTextures()
		return nil
	})
}

type texMgr struct {
	multiTextureEnabled bool
	currentTarget       uint32
	currentTexture      [3]*texture.Texture
	glModeIndex         int // TODO(therjak): glmode_idx is still split between c and go

	activeTextures map[*texture.Texture]bool
	maxAnisotropy  float32
	maxTextureSize int32
}

const (
	unusedTexture = glh.TexID(^uint32(0))
)

var (
	textureManager = texMgr{
		currentTarget:  gl.TEXTURE0,
		glModeIndex:    len(glModes) - 1,
		activeTextures: make(map[*texture.Texture]bool),
		maxAnisotropy:  1,
	}
	nullTexture *texture.Texture
)

func init() {
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

func (tm *texMgr) Init() {
	// get correct maxAnisotropy
	gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &tm.maxAnisotropy)
}

func (tm *texMgr) LoadConsoleChars() *texture.Texture {
	data := wad.GetConsoleChars()
	if len(data) != 128*128*4 {
		Error("ConsoleChars not found")
		return nil
	}
	t := texture.NewTexture(128, 128,
		texture.TexPrefAlpha|texture.TexPrefNearest|texture.TexPrefNoPicMip|texture.TexPrefConChars,
		"gfx.wad:conchars", texture.ColorTypeRGBA, data)
	tm.addActiveTexture(t)
	tm.loadRGBA(t, data)
	return t
}

func (tm *texMgr) LoadNoTex(name string, w, h int, data []byte) *texture.Texture {
	flags := texture.TexPrefNearest | texture.TexPrefPersist | texture.TexPrefNoPicMip
	return tm.loadRGBATex(name, w, h, flags, data)
}

func (tm *texMgr) LoadInternalTex(name string, w, h int, data []byte) *texture.Texture {
	flags := texture.TexPrefNearest | texture.TexPrefAlpha | texture.TexPrefPersist |
		texture.TexPrefPad | texture.TexPrefNoPicMip
	return tm.loadIndexdTex(name, w, h, flags, data)
}

func (tm *texMgr) LoadWadTex(name string, w, h int, data []byte) *texture.Texture {
	flags := texture.TexPrefAlpha | texture.TexPrefPad | texture.TexPrefNoPicMip
	return tm.loadIndexdTex(name, w, h, flags, data)
}

func (tm *texMgr) loadRGBATex(name string, w, h int, flags texture.TexPref, data []byte) *texture.Texture {
	t := texture.NewTexture(int32(w), int32(h), flags, name, texture.ColorTypeRGBA, data)
	tm.addActiveTexture(t)
	tm.loadRGBA(t, data)
	return t
}

func (tm *texMgr) loadIndexdTex(name string, w, h int, flags texture.TexPref, data []byte) *texture.Texture {
	t := texture.NewTexture(
		int32(w),
		int32(h),
		flags,
		name,
		texture.ColorTypeIndexed,
		data)
	tm.addActiveTexture(t)
	tm.loadIndexed(t, data)
	return t
}

func (tm *texMgr) LoadBacktile() *texture.Texture {
	name := "backtile"
	p := wad.GetPic(name)
	if p == nil {
		Error("Draw_LoadPics: couldn't load backtile")
		return nil
	}
	return tm.LoadWadTex(name, p.Width, p.Height, p.Data)
}

func (tm *texMgr) loadParticleImage(name string, width, height int32, data []byte) *texture.Texture {
	t := texture.NewTexture(width, height,
		texture.TexPrefPersist|texture.TexPrefAlpha|texture.TexPrefLinear,
		name, texture.ColorTypeRGBA, data)
	tm.addActiveTexture(t)
	tm.loadRGBA(t, data)
	return t
}

var skySuf = [6]string{"rt", "lf", "up", "dn", "bk", "ft"}

func (tm *texMgr) LoadSkyBox(boxName string) *texture.Texture {
	var imgs [6]*image.NRGBA
	for i, suf := range skySuf {
		n := fmt.Sprintf("gfx/env/%s%s", boxName, suf)
		img, err := qimage.Load(n)
		if err != nil {
			// TODO: is it ok to allow incomplete cubemaps?
			return nil
		}
		imgs[i] = img
	}
	t := texture.NewCubeTexture(0, 0, texture.TexPrefNone, boxName, texture.ColorTypeRGBA, nil)
	tm.addActiveTexture(t)
	tm.BindUnit(t, gl.TEXTURE0)
	for i, img := range imgs {
		s := img.Bounds().Size()
		gl.TexImage2D(gl.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(i),
			0, gl.RGB, int32(s.X), int32(s.Y), 0, gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	}
	gl.TexParameterf(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_CUBE_MAP, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)

	return t
}

func (tm *texMgr) FreeTexture(t *texture.Texture) {
	if inReloadImages {
		// Stupid workaround. Needs real fix.
		return
	}

	delete(tm.activeTextures, t)
	tm.deleteTexture(t)
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
		cv = pad(cv)
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

func (tm *texMgr) SelectTextureUnit(target uint32) {
	// THERJAK: we have at least 80 texture units, why use only 3?
	if target == tm.currentTarget {
		return
	}
	gl.ActiveTexture(target)
	tm.currentTarget = target
}

func (tm *texMgr) Bind(t *texture.Texture) {
	if t == nil {
		t = nullTexture
	}
	if t != tm.currentTexture[tm.currentTarget-gl.TEXTURE0] {
		tm.currentTexture[tm.currentTarget-gl.TEXTURE0] = t
		t.Bind()
	}
}

func (tm *texMgr) BindUnit(t *texture.Texture, target uint32) {
	if t != tm.currentTexture[target-gl.TEXTURE0] {
		tm.SelectTextureUnit(target)
		tm.Bind(t)
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
			if k.Flags(texture.TexPrefNoBright) {
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
			tm.ReloadImage(k)
		}
	}
	inReloadImages = false
}

func (tm *texMgr) ReloadImage(t *texture.Texture) {
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
	switch t.Typ {
	case texture.ColorTypeIndexed:
		tm.loadIndexed(t, t.Data)
	case texture.ColorTypeRGBA:
		tm.loadRGBA(t, t.Data)
	case texture.ColorTypeLightmap:
		tm.loadLightMap(t)
	}
}

func (tm *texMgr) deleteTexture(t *texture.Texture) {
	if t == nil {
		return
	}
	if t == tm.currentTexture[0] {
		tm.currentTexture[0] = nil
	}
	if t == tm.currentTexture[1] {
		tm.currentTexture[1] = nil
	}
	if t == tm.currentTexture[2] {
		tm.currentTexture[2] = nil
	}
}

func (tm *texMgr) DisableMultiTexture() {
	// selects texture unit 0
	if tm.multiTextureEnabled {
		gl.Disable(gl.TEXTURE_2D)
		tm.SelectTextureUnit(gl.TEXTURE0)
		tm.multiTextureEnabled = false
	}
}

func (tm *texMgr) EnableMultiTexture() {
	// selects texture unit 1
	tm.SelectTextureUnit(gl.TEXTURE1)
	gl.Enable(gl.TEXTURE_2D)
	tm.multiTextureEnabled = true
}

func (tm *texMgr) SetFilterModes(t *texture.Texture) {
	tm.BindUnit(t, gl.TEXTURE0)
	m := glModes[tm.glModeIndex]
	switch {
	case t.Flags(texture.TexPrefNearest):
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	case t.Flags(texture.TexPrefLinear):
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	case t.Flags(texture.TexPrefMipMap):
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
	val := cv.Value()
	switch {
	case val < 1:
		cv.SetByString("1")
	case val > tm.maxAnisotropy:
		cv.SetValue(tm.maxAnisotropy)
	default:
		for k, v := range tm.activeTextures {
			if v {
				if k.Flags(texture.TexPrefMipMap) {
					tm.BindUnit(k, gl.TEXTURE0)
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
	// Try to fix the cvar value and recursively call again
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
	conlog.Printf("\"%s\" is not a valid texturemode\n", name)
	cv.SetByString(glModes[tm.glModeIndex].name)
}

func (tm *texMgr) ClearBindings() {
	tm.currentTexture = [3]*texture.Texture{nil, nil, nil}
}

func (tm *texMgr) addActiveTexture(t *texture.Texture) {
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

func (tm *texMgr) getTextureUsage() (int, float32) {
	texels := 0
	for k, v := range tm.activeTextures {
		if v {
			texels += k.Texels()
		}
	}
	mb := float32(texels) * (24 / 8) / (1000 * 1000)
	return texels, mb
}

func (tm *texMgr) loadRGBA(t *texture.Texture, data []byte) {
	picmip := uint32(0)
	if !t.Flags(texture.TexPrefNoPicMip) {
		pv := cvars.GlPicMip.Value()
		if pv > 0 {
			picmip = uint32(pv)
		}
	}
	safeW := tm.safeTextureSize(t.Width >> picmip)
	safeH := tm.safeTextureSize(t.Height >> picmip)
	for t.Width > safeW {
		log.Printf("safeW")
		// half width
		data = downScaleWidth(t.Width, t.Height, data)
		t.Width >>= 1
		// if t.Flags&texture.TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.width, t.height, data)
		// }
	}
	for t.Height > safeH {
		log.Printf("safeH")
		// half height
		data = downScaleHeight(t.Width, t.Height, data)
		t.Height >>= 1
		// if t.Flags&texture.TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.width, t.height, data)
		//}
	}
	// Orig uses the 'old' values 3 or 4
	internalformat := int32(gl.RGB)
	if t.Flags(texture.TexPrefAlpha) {
		internalformat = gl.RGBA
	}
	tm.BindUnit(t, gl.TEXTURE0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internalformat, t.Width, t.Height,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))

	gl.GenerateMipmap(gl.TEXTURE_2D)
	tm.SetFilterModes(t)
}

func (tm *texMgr) loadLightMap(t *texture.Texture) {
	tm.BindUnit(t, gl.TEXTURE0)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, t.Width, t.Height,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(t.Data))
	tm.SetFilterModes(t)
}

func (tm *texMgr) loadIndexed(t *texture.Texture, data []byte) {
	var p *palette.Palette
	switch {
	case t.Flags(texture.TexPrefFullBright) && t.Flags(texture.TexPrefAlpha):
		p = &palette.TableFullBrightFence
	case t.Flags(texture.TexPrefFullBright):
		p = &palette.TableFullBright
	case t.Flags(texture.TexPrefNoBright) && t.Flags(texture.TexPrefAlpha):
		p = &palette.TableNoBrightFence
	case t.Flags(texture.TexPrefNoBright):
		p = &palette.TableNoBright
	case t.Flags(texture.TexPrefConChars):
		p = &palette.TableConsoleChars
	default:
		p = &palette.Table
	}
	nd := p.Convert(data)
	// do we actually need padding?
	if t.Flags(texture.TexPrefAlpha) {
		palette.AlphaEdgeFix(t.Width, t.Height, nd)
	}
	tm.loadRGBA(t, nd)
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
