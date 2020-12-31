package quakelib

//#include <stdlib.h>
//#include <stdint.h>
//extern float skyflatcolor[3];
//extern uint32_t skybox_textures[6];
//extern uint32_t solidskytexture2;
//extern uint32_t alphaskytexture2;
//extern float skyfog;
//extern float skymins[2][6];
//extern float skymaxs[2][6];
//extern int rs_brushpasses;
//void Sky_Init(void);
//void Sky_NewMap(void);
//void Fog_EnableGFog(void);
//void Fog_DisableGFog(void);
//void Sky_DrawSkyBox(void);
//void Sky_ProcessTextureChains(void);
//void Sky_ProcessEntities(void);
//void Sky_DrawFace(int axis);
import "C"

import (
	"fmt"
	"strconv"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/bsp"
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/texture"
)

func init() {
	cmd.AddCommand("sky", skyCommand)
	cvars.RSkyFog.SetCallback(func(cv *cvar.Cvar) {
		C.skyfog = C.float(cv.Value())
	})

}

func skyCommand(args []cmd.QArg, _ int) {
	switch len(args) {
	case 0:
		conlog.Printf("\"sky\" is \"%s\"\n", sky.boxName)
	case 1:
		sky.LoadBox(args[0].String())
	default:
		conlog.Printf("usage: sky <skyname>\n")
	}
}

type qSky struct {
	boxName      string
	boxTextures  [6]*texture.Texture
	solidTexture *texture.Texture
	alphaTexture *texture.Texture
	flat         Color
	mins         [2][6]float32
	maxs         [2][6]float32
}

var sky qSky

//export HasSkyBox
func HasSkyBox() bool {
	return len(sky.boxName) != 0
}

func ClearSkyBox() {
	sky.boxName = ""
	sky.boxTextures = [6]*texture.Texture{}
	C.skybox_textures[0] = 0
	C.skybox_textures[1] = 0
	C.skybox_textures[2] = 0
	C.skybox_textures[3] = 0
	C.skybox_textures[4] = 0
	C.skybox_textures[5] = 0
}

//export SkyInit
func SkyInit() {
	C.Sky_Init()
}

//export SkyDrawSky
func SkyDrawSky() {
	sky.Draw()
}

//export SkyNewMap
func SkyNewMap() {
	sky.NewMap()
}

func (s *qSky) NewMap() {
	ClearSkyBox()
	C.skyfog = C.float(cvars.RSkyFog.Value())

	s.boxName = ""
	s.boxTextures = [6]*texture.Texture{}
	for _, e := range cl.worldModel.Entities {
		if n, _ := e.Name(); n != "worldspawn" {
			continue
		}
		if p, ok := e.Property("sky"); ok {
			s.LoadBox(p)
		}
		if p, ok := e.Property("skyfog"); ok {
			v, err := strconv.ParseFloat(p, 32)
			if err == nil {
				C.skyfog = C.float(v)
			}
		} else if p, ok := e.Property("skyname"); ok { // half-life
			s.LoadBox(p)
		} else if p, ok := e.Property("glsky"); ok { // quake lives
			s.LoadBox(p)
		}
		return
	}
}

var (
	skySuf = [6]string{"rt", "bk", "lf", "ft", "up", "dn"}
)

func (s *qSky) LoadBox(name string) {
	if name == s.boxName {
		return
	}
	s.boxName = name
	for i, t := range s.boxTextures {
		textureManager.FreeTexture(t) // clean up textureManager cache
		C.skybox_textures[i] = 0
	}
	s.boxTextures = [6]*texture.Texture{}
	if s.boxName == "" {
		// Turn off skybox
		return
	}
	noneFound := true
	for i, suf := range skySuf {
		n := fmt.Sprintf("gfx/env/%s%s", s.boxName, suf)
		s.boxTextures[i] = textureManager.LoadSkyBox(n)
		if s.boxTextures[i] != nil {
			noneFound = false
		}
	}
	if noneFound {
		// boxName == "" => No DrawSkyBox but only DrawSkyLayers
		s.boxName = ""
		return
	}

	for i := 0; i < 6; i++ {
		if s.boxTextures[i] != nil {
			texmap[s.boxTextures[i].ID()] = s.boxTextures[i]
			C.skybox_textures[i] = C.uint32_t(s.boxTextures[i].ID())
		} else {
			C.skybox_textures[i] = C.uint32_t(unusedTexture)
		}
	}
}

func (s *qSky) LoadTexture(d []byte, skyName, modelName string) {
	// d is a 256*128 texture with the left side being a masked overlay
	// What a mess. It would be better to have the overlay at the bottom.
	front := [128 * 128]byte{}
	back := [128 * 128]byte{}
	var r, g, b, count int
	for i := 0; i < 128; i++ {
		for j := 0; j < 128; j++ {
			sidx := i*256 + j
			didx := i*128 + j
			p := d[sidx]
			if p == 0 {
				front[didx] = 255
			} else {
				front[didx] = p
				pixel := palette.table[p*4 : p*4+4]
				r += int(pixel[0])
				g += int(pixel[1])
				b += int(pixel[2])
				count++ // only count opaque colors
			}
			back[didx] = d[sidx+128]
		}
	}
	fn := fmt.Sprintf("%s:%s_front", modelName, skyName)
	bn := fmt.Sprintf("%s:%s_back", modelName, skyName)
	s.solidTexture = textureManager.LoadSkyTexture(bn, back[:], texture.TexPrefNone)
	s.alphaTexture = textureManager.LoadSkyTexture(fn, front[:], texture.TexPrefAlpha)
	s.flat = Color{
		R: float32(r) / (float32(count) * 255),
		G: float32(g) / (float32(count) * 255),
		B: float32(b) / (float32(count) * 255),
	}

	texmap[s.solidTexture.ID()] = s.solidTexture
	texmap[s.alphaTexture.ID()] = s.alphaTexture
	C.solidskytexture2 = C.uint32_t(s.solidTexture.ID())
	C.alphaskytexture2 = C.uint32_t(s.alphaTexture.ID())
	C.skyflatcolor[0] = C.float(s.flat.R)
	C.skyflatcolor[1] = C.float(s.flat.G)
	C.skyflatcolor[2] = C.float(s.flat.B)
}

type skyVec [3]int

var (
	st2vec      = [6]skyVec{{3, -1, 2}, {-3, 1, 2}, {1, 3, 2}, {-1, -3, 2}, {-2, -1, 3}, {2, -1, -3}}
	vec2st      = [6]skyVec{{-2, 3, 1}, {2, 3, -1}, {1, 3, 2}, {-1, 3, -2}, {-2, -1, 3}, {-2, 1, -3}}
	skyClip     = [6]vec.Vec3{{1, 1, 0}, {1, -1, 0}, {0, -1, 1}, {0, 1, 1}, {1, 0, 1}, {-1, 0, 1}}
	skyTexOrder = [6]int{0, 2, 1, 3, 4, 5}
)

func (sky *qSky) UpdateBounds(vecs []vec.Vec3) {
	// nump == len(vecs)
	// Sky_ProjectPoly
	// TODO: why does this computation feel stupid?
	var sum vec.Vec3
	for _, v := range vecs {
		sum.Add(v)
	}
	av := vec.Vec3{
		math32.Abs(sum[0]),
		math32.Abs(sum[1]),
		math32.Abs(sum[2]),
	}
	axis := func() int {
		switch {
		case av[0] > av[1] && av[0] > av[2]:
			if sum[0] < 0 {
				return 1
			}
			return 0
		case av[1] > av[2] && av[1] > av[0]:
			if sum[1] < 0 {
				return 3
			}
			return 2
		default:
			if sum[2] < 0 {
				return 5
			}
			return 4
		}
	}()
	j := vec2st[axis]
	for _, v := range vecs {
		dv := func() float32 {
			j2 := j[2]
			if j2 > 0 {
				return v[j2-1]
			}
			return -v[-j2-1]
		}()
		s := func() float32 {
			j0 := j[0]
			if j0 < 0 {
				return -v[-j0-1] / dv
			}
			return v[j0-1] / dv
		}()
		t := func() float32 {
			j1 := j[1]
			if j1 < 0 {
				return -v[-j1-1] / dv
			}
			return v[j1-1] / dv
		}()
		if s < sky.mins[0][axis] {
			sky.mins[0][axis] = s
		}
		if t < sky.mins[1][axis] {
			sky.mins[1][axis] = t
		}
		if s > sky.maxs[0][axis] {
			sky.maxs[0][axis] = s
		}
		if t > sky.maxs[0][axis] {
			sky.maxs[0][axis] = t
		}
	}
}

// MAX_CLIP_VERTS = 64
func (s *qSky) ClipPoly(vecs []vec.Vec3, stage int) {
	if stage >= 6 || stage < 0 {
		s.UpdateBounds(vecs)
		return
	}
	front := false
	back := false
	norm := skyClip[stage]
	var sides []int
	var dists []float32
	for _, v := range vecs {
		d := vec.Dot(v, norm)
		dists = append(dists, d)
		switch {
		case d > 0.1:
			front = true
			sides = append(sides, 0) // SIDE_FRONT
		case d < 0.1:
			back = true
			sides = append(sides, 1) // SIDE_BACK
		default:
			sides = append(sides, 2) // SIDE_ON
		}
	}
	if !front || !back {
		// not clipped
		s.ClipPoly(vecs, stage+1)
		return
	}
	// clip it
	var newvf, newvb []vec.Vec3
	for i, v := range vecs {
		j := (i + 1) % len(vecs)
		switch sides[i] {
		case 0:
			newvf = append(newvf, v)
		case 1:
			newvb = append(newvb, v)
		default:
			newvf = append(newvf, v)
			newvb = append(newvb, v)
		}
		if sides[i] == 2 || sides[j] == 2 || sides[i] == sides[j] {
			continue
		}
		d := dists[i] / (dists[i] - dists[j])
		e := vec.Lerp(vecs[i], vecs[j], d)
		newvf = append(newvf, e)
		newvb = append(newvb, e)
	}
	s.ClipPoly(newvf, stage+1)
	s.ClipPoly(newvb, stage+1)
}

var (
	sqrt3 = math32.Sqrt(3)
)

// DrawSkyLayers draws the old-style scrolling cloud layers
func (s *qSky) DrawSkyLayers() {
	if cvars.RSkyAlpha.Value() < 1 {
		// TODO: this needs to go into the shader
		//glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE)
		//defer glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE)
	}
	fc := cvars.GlFarClip.Value() / sqrt3
	// TODO: why do the model/view stuff outside the shader?
	for i := 0; i < 6; i++ {
		if s.mins[0][i] < s.maxs[0][i] && s.mins[1][i] < s.maxs[1][i] {
			C.Sky_DrawFace(C.int(i))
		}
	}
	if s.mins[0][0] < s.maxs[0][0] && s.mins[1][0] < s.maxs[1][0] {
		mins := [2]float32{s.mins[0][0], s.mins[1][0]}
		maxs := [2]float32{s.maxs[0][0], s.maxs[1][0]}
		// 3 -1 2 -> 2, -0, 1
		// v0 --+
		v1 := vec.Vec3{fc, fc, fc}   // -++
		v2 := vec.Vec3{fc, -fc, fc}  // +++
		v3 := vec.Vec3{-fc, fc, -fc} // +-+
		vup := vec.Sub(v2, v3)       // 2fc, -2fc, 2fc
		vright := vec.Sub(v2, v1)    // 0, -2fc, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{fc, fc, -fc}))
	}
	if s.mins[0][1] < s.maxs[0][1] && s.mins[1][1] < s.maxs[1][1] {
		mins := [2]float32{s.mins[0][1], s.mins[1][1]}
		maxs := [2]float32{s.maxs[0][1], s.maxs[1][1]}
		//-3 1, 2 -> -2, 0, 1
		v1 := vec.Vec3{-fc, -fc, fc} // -++
		v2 := vec.Vec3{-fc, fc, fc}  // +++
		v3 := vec.Vec3{-fc, fc, -fc} // +-+
		vup := vec.Sub(v2, v3)       // 0, 0, 2fc
		vright := vec.Sub(v2, v1)    // 0, 2fc, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{-fc, -fc, -fc}))
	}
	if s.mins[0][2] < s.maxs[0][2] && s.mins[1][2] < s.maxs[1][2] {
		mins := [2]float32{s.mins[0][2], s.mins[1][2]}
		maxs := [2]float32{s.maxs[0][2], s.maxs[1][2]}
		//1,3,2 -> 0, 2, 1
		v1 := vec.Vec3{-fc, fc, fc} // -++
		v2 := vec.Vec3{fc, fc, fc}  // +++
		v3 := vec.Vec3{fc, fc, -fc} // +-+
		vup := vec.Sub(v2, v3)      // 0, 0, 2fc
		vright := vec.Sub(v2, v1)   // 2fc, 0, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{-fc, fc, -fc}))
	}
	if s.mins[0][3] < s.maxs[0][3] && s.mins[1][3] < s.maxs[1][3] {
		mins := [2]float32{s.mins[0][3], s.mins[1][3]}
		maxs := [2]float32{s.maxs[0][3], s.maxs[1][3]}
		//-1, -3, 2 -> -0, -2, 1
		v1 := vec.Vec3{fc, -fc, fc}   // -++
		v2 := vec.Vec3{-fc, -fc, fc}  // +++
		v3 := vec.Vec3{-fc, -fc, -fc} // +-+
		vup := vec.Sub(v2, v3)        // 0, 0, 2fc
		vright := vec.Sub(v2, v1)     // -2fc, 0, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{fc, -fc, -fc}))
	}
	if s.mins[0][4] < s.maxs[0][4] && s.mins[1][4] < s.maxs[1][4] {
		mins := [2]float32{s.mins[0][4], s.mins[1][4]}
		maxs := [2]float32{s.maxs[0][4], s.maxs[1][4]}
		//-2, -1, 3 -> -1, -0, 2
		v1 := vec.Vec3{-fc, fc, fc}  // -++
		v2 := vec.Vec3{-fc, -fc, fc} // +++
		v3 := vec.Vec3{fc, -fc, fc}  // +-+
		vup := vec.Sub(v2, v3)       // -2fc, 0, 0
		vright := vec.Sub(v2, v1)    // 0, -2fc, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{fc, fc, fc}))
	}
	if s.mins[0][5] < s.maxs[0][5] && s.mins[1][5] < s.maxs[1][5] {
		mins := [2]float32{s.mins[0][5], s.mins[1][5]}
		maxs := [2]float32{s.maxs[0][5], s.maxs[1][5]}
		//2, -1, -3 -> 1, -0, -2
		v1 := vec.Vec3{fc, fc, -fc}   // -++
		v2 := vec.Vec3{fc, -fc, -fc}  // +++
		v3 := vec.Vec3{-fc, -fc, -fc} // +-+
		vup := vec.Sub(v2, v3)        // 2fc, 0, 0
		vright := vec.Sub(v2, v1)     // 0, -2fc, 0
		s.DrawFace(mins, maxs, vup, vright,
			vec.Add(qRefreshRect.viewOrg, vec.Vec3{-fc, fc, -fc}))
	}
}

func (s *qSky) DrawFace(mins, maxs [2]float32, vup, vright, v vec.Vec3) {
	// di = qmax...
	// verts[0] == v
}

func (s *qSky) Draw() {
	// This is Draw called before everything else is drawn
	const mf = math32.MaxFloat32
	s.mins = [2][6]float32{
		{-mf, -mf, -mf, -mf, -mf, -mf},
		{-mf, -mf, -mf, -mf, -mf, -mf}}
	s.maxs = [2][6]float32{
		{mf, mf, mf, mf, mf, mf},
		{mf, mf, mf, mf, mf, mf}}
	C.skymins = [2][6]C.float{
		{-mf, -mf, -mf, -mf, -mf, -mf},
		{-mf, -mf, -mf, -mf, -mf, -mf}}
	C.skymaxs = [2][6]C.float{
		{mf, mf, mf, mf, mf, mf},
		{mf, mf, mf, mf, mf, mf}}

	C.Fog_DisableGFog()
	defer C.Fog_EnableGFog()

	gl.Disable(gl.TEXTURE_2D)
	// if Fog_GetDensity() > 0
	// glColor3fv(Fog_GetColor())
	// else
	// glColor3fv(skyflatcolor)
	s.processTextureChains()
	C.Sky_ProcessEntities()
	// glColor3fv(1,1,1)
	gl.Enable(gl.TEXTURE_2D)

	if !cvars.RFastSky.Bool() && !(Fog_GetDensity() > 0 && C.skyfog >= 1) {
		gl.DepthFunc(gl.GEQUAL)
		defer gl.DepthFunc(gl.LEQUAL)
		gl.DepthMask(false)
		defer gl.DepthMask(true)

		if len(sky.boxName) != 0 {
			C.Sky_DrawSkyBox()
		} else {
			s.DrawSkyLayers()
		}
	}
}

func (s *qSky) processTextureChains() {
	for _, t := range cl.worldModel.Textures {
		if t == nil {
			continue
		}
		cw := t.TextureChains[0] // 0 == chain_world
		if cw == nil || cw.Flags&bsp.SurfaceDrawSky == 0 {
			continue
		}
		for tc := cw; tc != nil; tc = tc.TextureChain {
			if !tc.Culled {
				s.processPoly(tc.Polys)
			}
		}

	}
	C.Sky_ProcessTextureChains()
}

func (s *qSky) processPoly(p *bsp.Poly) {
	// draw it
	// DrawGLPoly(p)
	C.rs_brushpasses++

	// update sky bounds
	if !cvars.RFastSky.Bool() {

	}
}

// uses
// R_CullModelForEntity
// cl.worldmodel->numtextures
// cl.worldmodel->textures
// cl.worldmodel->entities
// DrawGLPoly
// Fog_GetDensity
// Fog_GetColor

// Fog_DisableGFog()
// Fog_EnableGFog()
// r_origin -> == qRefreshRect.viewOrg
// rs_skypolys
// rs_skypasses
