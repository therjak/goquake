// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (

	// "github.com/chewxy/math32"
	"goquake/bsp"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/glh"
	"goquake/math/vec"
	"goquake/texture"
	"time"

	"github.com/go-gl/gl/v4.6-core/gl"
)

const (
	BLOCK_WIDTH     = 128
	BLOCK_HEIGHT    = 128
	MAX_LIGHTMAPS   = 512
	LIGHTMAP_BYTES  = 4
	LIGHTMAP_FORMAT = gl.RGBA
)

type glRect struct {
	l, t, w, h uint16
}

type lightmap struct {
	texture    *texture.Texture
	poly       *bsp.Poly
	modified   bool
	rectChange glRect
	allocated  [BLOCK_WIDTH]int
	data       [BLOCK_WIDTH * BLOCK_HEIGHT * LIGHTMAP_BYTES]byte
}

var (
	lightmaps             [MAX_LIGHTMAPS]lightmap
	lastLightmapAllocated int
	blockLights           [BLOCK_WIDTH * BLOCK_HEIGHT * 3]uint
)

type qBrushDrawer struct {
	vao           *glh.VertexArray
	vbo           *glh.Buffer
	ebo           *glh.Buffer
	prog          *glh.Program
	projection    int32
	modelview     int32
	tex           int32
	lmTex         int32
	fullBrightTex int32
	useFullBright int32
	useOverBright int32
	useAlphaTest  int32
	alpha         int32
	fogDensity    int32
	fogColor      int32
	turb          int32
	time          int32
	vbo_indices   []uint32
	startTime     time.Time
}

func newBrushDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceBrushDrawer, fragmentSourceBrushDrawer)
}

func newBrushDrawer() (*qBrushDrawer, error) {
	d := &qBrushDrawer{startTime: time.Now()}
	d.vao = glh.NewVertexArray()
	d.ebo = glh.NewBuffer(glh.ElementArrayBuffer)
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	var err error
	d.prog, err = newBrushDrawProgram()
	if err != nil {
		return nil, err
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	d.tex = d.prog.GetUniformLocation("Tex")
	d.lmTex = d.prog.GetUniformLocation("LMTex")
	d.fullBrightTex = d.prog.GetUniformLocation("FullbrightTex")
	d.useFullBright = d.prog.GetUniformLocation("UseFullbrightTex")
	d.useOverBright = d.prog.GetUniformLocation("UseOverbright")
	d.useAlphaTest = d.prog.GetUniformLocation("UseAlphaTest")
	d.alpha = d.prog.GetUniformLocation("Alpha")
	d.fogDensity = d.prog.GetUniformLocation("FogDensity")
	d.fogColor = d.prog.GetUniformLocation("FogColor")
	d.turb = d.prog.GetUniformLocation("Turb")
	d.time = d.prog.GetUniformLocation("Time")
	d.vbo_indices = make([]uint32, 0, 4096)
	return d, nil
}

var (
	// brushDrawer *qBrushDrawer
	brushDrawer *qBrushDrawer
)

func CreateBrushDrawer() error {
	var err error
	brushDrawer, err = newBrushDrawer()
	return err
}

func (d *qBrushDrawer) buildVertexBuffer() {
	// Gets called once per map
	idx := 0
	var buf []float32
	for _, m := range cl.modelPrecache {
		switch w := m.(type) {
		case *bsp.Model:
			for _, s := range w.Surfaces {
				// Why? We are changing the model again...
				s.VboFirstVert = idx
				idx += len(s.Polys.Verts)
				for _, v := range s.Polys.Verts {
					buf = append(buf,
						v.Pos[0], v.Pos[1], v.Pos[2],
						v.S, v.T, // includes texture repeats
						v.LightMapS, v.LightMapT)
				}
			}
		}
	}
	d.vbo.Bind()
	d.vbo.SetData(4*len(buf), gl.Ptr(buf))
}

// This are only used for 'secondary' bsp eg doors
func (r *qRenderer) DrawBrushModel(e *Entity, model *bsp.Model) {
	const epsilon = 0.03125 // (1/32) to keep floating point happy
	if r.cullBrush(e, model) {
		return
	}
	modelOrg := vec.Sub(qRefreshRect.viewOrg, e.Origin)
	if e.Angles[0] != 0 || e.Angles[1] != 0 || e.Angles[2] != 0 {
		tmp := modelOrg
		f, r, u := vec.AngleVectors(e.Angles)
		modelOrg[0] = vec.Dot(tmp, f)
		modelOrg[1] = -vec.Dot(tmp, r)
		modelOrg[2] = vec.Dot(tmp, u)
	}

	// calculate dynamic lighting for model if it's not an instanced model
	if !cvars.GlFlashBlend.Bool() /*&& model.firstmodelsurface != 0*/ {
		markLights(model.Nodes[model.Hulls[0].FirstClipNode])
	}

	if cvars.GlZFix.Bool() {
		e.Origin.Sub(vec.Vec3{epsilon, epsilon, epsilon})
	}

	modelview := view.modelView.Copy()
	modelview.Translate(e.Origin[0], e.Origin[1], e.Origin[2])
	modelview.RotateZ(e.Angles[1])
	// stupid quake bug, it should be -angles[0]
	modelview.RotateY(e.Angles[0])
	modelview.RotateX(e.Angles[2])

	if cvars.GlZFix.Bool() {
		e.Origin.Add(vec.Vec3{epsilon, epsilon, epsilon})
	}

	for _, t := range model.Textures {
		if t != nil {
			t.TextureChains[chainModel] = nil
		}
	}
	// for i := range(lightmap) {
	//  lightmap[i].polys = nil
	// }
	// ...

	for _, s := range model.Surfaces {
		p := s.Plane
		dot := vec.Dot(modelOrg, p.Normal) - p.Dist
		if (s.Flags&bsp.SurfacePlaneBack != 0 && dot < -bsp.BackFaceEpsilon) ||
			(s.Flags&bsp.SurfacePlaneBack == 0 && dot > bsp.BackFaceEpsilon) {
			s.TextureChain = s.TexInfo.Texture.TextureChains[chainModel]
			s.TexInfo.Texture.TextureChains[chainModel] = s
		}
	}
	r.drawTextureChains(modelview, model, e, chainModel)
}

func (r *qRenderer) DrawWorld(model *bsp.Model, mv *glh.Matrix) {
	r.drawTextureChains(mv, model, nil, chainWorld)
}

func init() {
	cvars.GlOverBright.SetCallback(func(*cvar.Cvar) {
		rebuildAllLightMaps()
	})
}

func rebuildAllLightMaps() {
	if cl.worldModel == nil {
		// this is probably not the exact test necessary but good enough?
		return
	}

	// 0 is worldModel
	for i := 1; i < len(cl.modelPrecache); i++ {
		mod, ok := cl.modelPrecache[i].(*bsp.Model)
		if !ok {
			continue
		}
		for _, s := range mod.Surfaces {
			if s.Flags&bsp.SurfaceDrawTiled != 0 {
				continue
			}
			var lights []bsp.DynamicLight
			for i := range cl.dynamicLights {
				lights = append(lights, &cl.dynamicLights[i])
			}
			s.BuildLightMap(lightStyleValues, renderer.frameCount, lights, cvars.GlOverBright.Bool())
			textureManager.loadLightMap(s.LightmapTexture)
		}
	}
	/*
		// Should no longer be needed
			for i := range lightmaps {
				lm := &lightmaps[i]
				if lm.allocated[0] == 0 {
					break
				}
				textureManager.Bind(lm.texture)
				gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0,
					BLOCK_WIDTH, BLOCK_HEIGHT,
					LIGHTMAP_FORMAT, gl.UNSIGNED_BYTE, gl.Ptr(lm.data))
			}
	*/
}

func (r *qRenderer) drawTextureChainsNoTexture(mv *glh.Matrix, model *bsp.Model, e *Entity, chain int) {
	// THERJAK: Why is this even needed? shouldn't the texture one be enough?
	entalpha := float32(1)
	if e != nil {
		entalpha = entAlphaDecode(e.Alpha)
	}
	if entalpha < 1.0 {
		gl.DepthMask(false)
		defer gl.DepthMask(true)
		gl.Enable(gl.BLEND)
		defer gl.Disable(gl.BLEND)
		// TODO: add in the shader:
		// gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		// glColor4f(1,1,1,entalpha)
	}
	for _, t := range model.Textures {
		if t == nil ||
			t.TextureChains[chain] == nil ||
			t.TextureChains[chain].Flags&bsp.SurfaceNoTexture == 0 {
			continue
		}
		bound := false
		for s := t.TextureChains[chain]; s != nil; s = s.TextureChain {
			if s.Culled {
				continue
			}
			if !bound {
				t.Texture.Bind()
				bound = true
			}
			// DrawGLPoly(s.polys)
		}
	}
}

func (d *qBrushDrawer) drawTextureChains(mv *glh.Matrix, model *bsp.Model, e *Entity, chain int) {
	// Compare R_DrawTextureChains_GLSL, recent quakespasm
	entalpha := float32(1)
	if e != nil {
		entalpha = entAlphaDecode(e.Alpha)
	}
	if entalpha < 1.0 {
		gl.DepthMask(false)
		defer gl.DepthMask(true)
		gl.Enable(gl.BLEND)
		defer gl.Disable(gl.BLEND)
	}
	d.prog.Use()

	d.vao.Bind()
	d.vbo.Bind()
	d.ebo.Bind()

	gl.EnableVertexAttribArray(0) // Vert
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1) // TexCoords
	defer gl.DisableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2) // LMCoords
	defer gl.DisableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 7*4, 0)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 7*4, 3*4)
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, 7*4, 5*4)

	gl.Uniform1i(d.tex, 0) // Match gl.TEXTRUE0, see below
	gl.Uniform1i(d.lmTex, 1)
	gl.Uniform1i(d.fullBrightTex, 2)
	gl.Uniform1i(d.useFullBright, 0) // remove? gets overwritten below
	var useOverBright int32
	if cvars.GlOverBrightModels.Bool() {
		useOverBright = 1
	}
	gl.Uniform1i(d.useOverBright, useOverBright)
	gl.Uniform1i(d.useAlphaTest, 0)
	gl.Uniform1f(d.alpha, entalpha)
	gl.Uniform1f(d.fogDensity, fog.Density)
	gl.Uniform4f(d.fogColor, fog.Color.R, fog.Color.G, fog.Color.B, 0)
	gl.Uniform1f(d.time, float32(time.Since(d.startTime).Seconds()))
	view.projection.SetAsUniform(d.projection)
	mv.SetAsUniform(d.modelview)

	for _, t := range model.Textures {
		if t == nil ||
			t.TextureChains[chain] == nil ||
			t.TextureChains[chain].Flags&(bsp.SurfaceNoTexture|bsp.SurfaceDrawSky) != 0 {
			continue
		}
		turb := func(flags int) float32 {
			if flags&bsp.SurfaceDrawTurb != 0 {
				return 1
			}
			return 0
		}(t.TextureChains[chain].Flags)
		gl.Uniform1f(d.turb, turb)
		// TODO: check water alpha

		frame := 0
		if e != nil {
			frame = e.Frame
		}

		bound := false
		var lastLightmap *texture.Texture
		for s := t.TextureChains[chain]; s != nil; s = s.TextureChain {
			if s.Culled {
				continue
			}
			if !bound {
				ani := textureAnimation(t, frame)
				textureManager.BindUnit(ani.Texture, gl.TEXTURE0)
				if cvars.GlFullBrights.Bool() && ani.Fullbright != nil {
					textureManager.BindUnit(ani.Fullbright, gl.TEXTURE2)
					gl.Uniform1i(d.useFullBright, 1)
				} else {
					gl.Uniform1i(d.useFullBright, 0)
				}

				bound = true
				lastLightmap = s.LightmapTexture
			}

			if lastLightmap != s.LightmapTexture {
				if len(d.vbo_indices) > 0 {
					// TODO: this handling of ebo needs improvement
					d.ebo.SetData(4*len(d.vbo_indices), gl.Ptr(d.vbo_indices))
					gl.DrawElements(gl.TRIANGLES, int32(len(d.vbo_indices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
					d.vbo_indices = d.vbo_indices[:0]
				}
			}
			textureManager.BindUnit(s.LightmapTexture, gl.TEXTURE1)
			lastLightmap = s.LightmapTexture

			for i := 2; i < s.NumEdges; i++ {
				d.vbo_indices = append(d.vbo_indices,
					uint32(s.VboFirstVert),
					uint32(s.VboFirstVert+i-1),
					uint32(s.VboFirstVert+i))
			}
		}
		if len(d.vbo_indices) > 0 {
			// TODO: this handling of ebo needs improvement
			d.ebo.SetData(4*len(d.vbo_indices), gl.Ptr(d.vbo_indices))
			gl.DrawElements(gl.TRIANGLES, int32(len(d.vbo_indices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
			d.vbo_indices = d.vbo_indices[:0]
		}
	}
	textureManager.SelectTextureUnit(gl.TEXTURE0)
}

func textureAnimation(t *bsp.Texture, frame int) *bsp.Texture {
	// R_TextureAnimation
	// TODO: alternate_anims
	// TODO: base anims
	// relative = cl.time*10 % anim_total
	// return anims[relative]
	return t
}

func (r *qRenderer) drawTextureChains(mv *glh.Matrix, model *bsp.Model, e *Entity, chain int) {
	// TODO: shouldn't rebuildAllLightmaps already uploaded the lightmap?
	// R_BuildLighmapChains(model,chain)
	// R_UploadLightmaps()

	r.drawTextureChainsNoTexture(mv, model, e, chain)
	brushDrawer.drawTextureChains(mv, model, e, chain)
}

func waterAlphaForSurface(s *bsp.Surface) float32 {
	orWater := func(v float32) float32 {
		if v > 0 {
			return v
		}
		return mapAlphas.water
	}
	switch {
	case s.Flags&bsp.SurfaceDrawLava != 0:
		return orWater(mapAlphas.lava)
	case s.Flags&bsp.SurfaceDrawTele != 0:
		return orWater(mapAlphas.tele)
	case s.Flags&bsp.SurfaceDrawSlime != 0:
		return orWater(mapAlphas.slime)
	case s.Flags&bsp.SurfaceDrawTurb != 0:
		return mapAlphas.water
	default:
		// TODO
		return mapAlphas.water
	}
}
