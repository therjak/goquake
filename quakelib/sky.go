package quakelib

//void Sky_Init(void);
//void Sky_DrawSky(void);
//void Sky_NewMap(void);
//void Sky_LoadSkyBox(const char *name);
//void Sky_LoadTextureInt(const unsigned char* src, const char* skyName, const char* modelName);
import "C"

import (
	"fmt"
	"github.com/chewxy/math32"
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/texture"
	"unsafe"
)

func init() {
	cmd.AddCommand("sky", skyCommand)
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

//export SkyInit
func SkyInit() {
	C.Sky_Init()
}

//export SkyDrawSky
func SkyDrawSky() {
	C.Sky_DrawSky()
}

//export SkyNewMap
func SkyNewMap() {
	C.Sky_NewMap()
	sky.NewMap()
}

func (s *qSky) NewMap() {
	s.boxName = ""
	s.boxTextures = [6]*texture.Texture{}
	// TODO:
	// skyfog
	// parse cl.worldmodel.entities
}

//export SkyLoadSkyBox
func SkyLoadSkyBox(c *C.char) {
	C.Sky_LoadSkyBox(c)
	return

	name := C.GoString(c)
	sky.LoadBox(name)
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
		textureManager.FreeTexture(t)
		s.boxTextures[i] = nil
	}
	if s.boxName == "" {
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
}

//export SkyLoadTexture
func SkyLoadTexture(src *C.uchar, skyName *C.char, modelName *C.char) {
	C.Sky_LoadTextureInt(src, skyName, modelName)
	return
	s := C.GoString(skyName)
	m := C.GoString(modelName)
	b := C.GoBytes(unsafe.Pointer(src), 256*128)
	sky.LoadTexture(b, s, m)
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

// uses
// cl_numvisedicts
// cl_visedicts
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
// gl_mtexable
// rs_skypolys
// rs_skypasses
