package quakelib

import (
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/image"
	"github.com/therjak/goquake/wad"
	"log"
	"strconv"
	"strings"

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

type texMgr struct {
	multiTextureEnabled bool
	currentTarget       uint32
	currentTexture      [3]*glh.Texture
	glModeIndex         int // TODO(therjak): glmode_idx is still split between c and go

	activeTextures map[*Texture]bool
	maxAnisotropy  float32
	maxTextureSize int32
}

type colorType int

const (
	colorTypeIndexed colorType = iota
	colorTypeRGBA
	colorTypeLightmap
)

type Texture struct {
	glID   *glh.Texture
	width  int32 // mipmap can make it differ from source width
	height int32
	flags  TexPref
	name   string
	typ    colorType
	data   []byte
}

func NewTexture(w, h int32, flags TexPref, name string, typ colorType, data []byte) *Texture {
	t := &Texture{
		glID:   glh.NewTexture(),
		width:  w,
		height: h,
		flags:  flags,
		name:   name,
		typ:    typ,
		data:   data,
	}
	return t
}

const (
	unusedTexture = glh.TexID(^uint32(0))
)

var (
	texmap         map[glh.TexID]*Texture
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
	texmap = make(map[glh.TexID]*Texture)
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

func (tm *texMgr) LoadConsoleChars() *Texture {
	data := wad.GetConsoleChars()
	if len(data) != 128*128 {
		Error("ConsoleChars not found")
		return nil
	}
	t := NewTexture(128, 128,
		TexPrefAlpha|TexPrefNearest|TexPrefNoPicMip|TexPrefConChars,
		"gfx.wad:conchars", colorTypeIndexed, data)
	textureManager.addActiveTexture(t)
	textureManager.loadIndexed(t, data)
	texmap[t.glID.ID()] = t
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
	t := NewTexture(int32(w), int32(h), flags, name, colorTypeRGBA, data)
	tm.addActiveTexture(t)
	tm.loadRGBA(t, data)
	texmap[t.glID.ID()] = t
	return t
}

func (tm *texMgr) loadIndexdTex(name string, w, h int, flags TexPref, data []byte) *Texture {
	t := NewTexture(
		int32(w),
		int32(h),
		flags,
		name,
		colorTypeIndexed,
		data)
	tm.addActiveTexture(t)
	tm.loadIndexed(t, data)
	texmap[t.glID.ID()] = t
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

func (tm *texMgr) loadParticleImage(name string, width, height int32, data []byte) *Texture {
	t := NewTexture(width, height,
		TexPrefPersist|TexPrefAlpha|TexPrefLinear,
		name, colorTypeRGBA, data)
	textureManager.addActiveTexture(t)
	textureManager.loadRGBA(t, data)
	return t
}

func (tm *texMgr) LoadSkyTexture(n string, d []byte, flags TexPref) *Texture {
	t := NewTexture(128, 128, flags, n, colorTypeIndexed, d)
	tm.addActiveTexture(t)
	tm.loadIndexed(t, d)
	texmap[t.glID.ID()] = t
	return t
}

func (tm *texMgr) LoadSkyBox(n string) *Texture {
	img, err := image.Load(n)
	if err != nil {
		return nil
	}
	s := img.Bounds().Size()

	t := NewTexture(int32(s.X), int32(s.Y), TexPrefNone, n, colorTypeRGBA, img.Pix)
	tm.addActiveTexture(t)
	tm.loadRGBA(t, img.Pix)
	texmap[t.glID.ID()] = t
	return t
}

func (tm *texMgr) FreeTexture(t *Texture) {
	if inReloadImages {
		// Stupid workaround. Needs real fix.
		return
	}
	if t == noTexture {
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

var (
	glWarpImageSize int32
)

func (tm *texMgr) RecalcWarpImageSize(w, h int) {
	s := tm.safeTextureSize(512)
	for int(s) > w || int(s) > h {
		s >>= 1
	}
	glWarpImageSize = s

	// TODO(therjak): there should be a better way.
	dummy := make([]float32, s*s*3)
	for t, b := range tm.activeTextures {
		if b && (t.flags&TexPrefWarpImage != 0) {
			tm.Bind(t)
			gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, s, s, 0, gl.RGB, gl.FLOAT, gl.Ptr(dummy))
			t.width = s
			t.height = s
		}
	}
}

func (tm *texMgr) SelectTextureUnit(target uint32) {
	// THERJAK: we have at least 80 texture units, why use only 3?
	if target == tm.currentTarget {
		return
	}
	gl.ActiveTexture(target)
	tm.currentTarget = target
}

func (tm *texMgr) Bind(t *Texture) {
	if t == nil {
		t = nullTexture
	}
	if t.glID != tm.currentTexture[tm.currentTarget-gl.TEXTURE0] {
		tm.currentTexture[tm.currentTarget-gl.TEXTURE0] = t.glID
		t.glID.Bind()
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
	if t == nil {
		return
	}
	if t.glID == tm.currentTexture[0] {
		tm.currentTexture[0] = nil
	}
	if t.glID == tm.currentTexture[1] {
		tm.currentTexture[1] = nil
	}
	if t.glID == tm.currentTexture[2] {
		tm.currentTexture[2] = nil
	}

	delete(texmap, t.glID.ID())
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

func (tm *texMgr) ClearBindings() {
	tm.currentTexture = [3]*glh.Texture{nil, nil, nil}
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
				texels += k.width * k.height * 4 / 3
			} else {
				texels += k.width * k.height
			}
		}
	}
	mb := float32(texels) * (24 / 8) / (1000 * 1000)
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
	safeW := tm.safeTextureSize(t.width >> picmip)
	safeH := tm.safeTextureSize(t.height >> picmip)
	for t.width > safeW {
		log.Printf("safeW")
		// half width
		data = downScaleWidth(t.width, t.height, data)
		t.width >>= 1
		// if t.flags&TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.width, t.height, data)
		// }
	}
	for t.height > safeH {
		log.Printf("safeH")
		// half height
		data = downScaleHeight(t.width, t.height, data)
		t.height >>= 1
		// if t.flags&TexPrefAlpha != 0 {
		// TODO(therjak): is this needed?
		//   alphaEdgeFix(t.width, t.height, data)
		//}
	}
	// Orig uses the 'old' values 3 or 4
	internalformat := int32(gl.RGB)
	if t.flags&TexPrefAlpha != 0 {
		internalformat = gl.RGBA
	}
	tm.Bind(t)
	gl.TexImage2D(gl.TEXTURE_2D, 0, internalformat, t.width, t.height,
		0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))

	gl.GenerateMipmap(gl.TEXTURE_2D)
	tm.SetFilterModes(t)
}

func (tm *texMgr) loadLightMap(t *Texture, data []byte) {
	tm.Bind(t)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, t.width, t.height,
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
		alphaEdgeFix(t.width, t.height, nd)
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
