// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"goquake/bsp"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/glh"
	"goquake/math/vec"
	"goquake/mdl"
	"goquake/palette"
	"goquake/progs"
	"goquake/spr"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
)

func init() {
	cvars.RClearColor.SetCallback(setClearColor)
}

func setClearColor(cv *cvar.Cvar) {
	s := int(cv.Value()) & 0xff
	r := float32(palette.Table[s*4]) / 255
	g := float32(palette.Table[s*4+1]) / 255
	b := float32(palette.Table[s*4+2]) / 255
	gl.ClearColor(r, g, b, 0)
}

func setupGLState() {
	gl.ClearColor(0.15, 0.15, 0.15, 0)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CW)
	gl.Enable(gl.TEXTURE_2D)
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthRange(0, 1)
	gl.DepthFunc(gl.LEQUAL)
}

type fPlane struct {
	signBits uint8 // caching of plane side tests
	normal   vec.Vec3
	dist     float32
}

type qRenderer struct {
	frustum         [4]fPlane
	frameCount      int
	lightFrameCount int
	visFrameCount   int
}

var renderer qRenderer

// CullBox returns true if the box is completely outside the frustum
func (r *qRenderer) CullBox(mins, maxs vec.Vec3) bool {
	for _, f := range r.frustum {
		switch f.signBits {
		case 0:
			if f.normal[0]*maxs[0]+f.normal[1]*maxs[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 1:
			if f.normal[0]*mins[0]+f.normal[1]*maxs[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 2:
			if f.normal[0]*maxs[0]+f.normal[1]*mins[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 3:
			if f.normal[0]*mins[0]+f.normal[1]*mins[1]+f.normal[2]*maxs[2] < f.dist {
				return true
			}
		case 4:
			if f.normal[0]*maxs[0]+f.normal[1]*maxs[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 5:
			if f.normal[0]*mins[0]+f.normal[1]*maxs[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 6:
			if f.normal[0]*maxs[0]+f.normal[1]*mins[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		case 7:
			if f.normal[0]*mins[0]+f.normal[1]*mins[1]+f.normal[2]*mins[2] < f.dist {
				return true
			}
		}
	}
	return false
}

func (p *fPlane) UpdateSignBits() {
	p.signBits = 0
	if p.normal[0] < 0 {
		p.signBits |= 1 << 0
	}
	if p.normal[1] < 0 {
		p.signBits |= 1 << 1
	}
	if p.normal[2] < 0 {
		p.signBits |= 1 << 2
	}
}

func deg2rad(a float32) float32 {
	a /= 180
	a *= math32.Pi
	return a
}

func (p *fPlane) TurnVector(forward, side vec.Vec3, angle float32) {
	ar := deg2rad(angle)
	scaleSide, scaleForward := math32.Sincos(ar)

	p.normal = vec.Add(vec.Scale(scaleForward, forward), vec.Scale(scaleSide, side))
	p.dist = vec.Dot(qRefreshRect.viewOrg, p.normal)
	p.UpdateSignBits()
}

func (r *qRenderer) SetFrustum(fovx, fovy float32) {
	// We do not use qRefreshRect.fovX/fovY directly as water has an effect on these values
	r.frustum[0].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, fovx/2-90) // left
	r.frustum[1].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, 90-fovx/2) // right
	r.frustum[2].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, 90-fovy/2)    // bottom
	r.frustum[3].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, fovy/2-90)    // top
}

func (r *qRenderer) DrawWeaponModel() {
	if !cvars.RDrawViewModel.Bool() ||
		!cvars.RDrawEntities.Bool() ||
		cvars.ChaseActive.Bool() {
		return
	}
	if cl.items&progs.ItemInvisibility != 0 ||
		cl.stats.health <= 0 {
		return
	}
	weapon := cl.WeaponEntity()
	if weapon.Model == nil {
		return
	}
	switch m := weapon.Model.(type) {
	case *mdl.Model:
		// hack the depth range to prevent view model from poking into walls
		gl.DepthRange(0, 0.3)
		r.DrawAliasModel(weapon, m)
		gl.DepthRange(0, 1)
	}
}

var coneDrawer *qConeDrawer

func CreateConeDrawer() error {
	var err error
	coneDrawer, err = newConeDrawer()
	return err
}

type qConeDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
}

type qCone struct {
	origin     vec.Vec3
	radius     float32
	innerColor [3]float32
	outerColor [3]float32
}

func newConeDrawProgram() (*glh.Program, error) {
	return glh.NewProgramWithGeometry(vertexConeSource, geometryConeSource, fragmentConeSource)
}

func newConeDrawer() (*qConeDrawer, error) {
	d := &qConeDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	var err error
	d.prog, err = newConeDrawProgram()
	if err != nil {
		return nil, err
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	return d, nil
}

func (cd *qConeDrawer) Draw(cs []qCone) {
	gl.DepthMask(false) // to not obstruct the view to particles within the cone
	defer gl.DepthMask(true)

	gl.Enable(gl.BLEND)
	defer gl.Disable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	defer gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	cd.prog.Use()
	cd.vao.Bind()
	cd.vbo.Bind()
	// TODO: remove this allocation
	data := make([]float32, 0, len(cs)*(3+1+3+3))
	for _, c := range cs {
		data = append(data,
			c.origin[0], c.origin[1], c.origin[2],
			c.radius,
			c.innerColor[0], c.innerColor[1], c.innerColor[2],
			c.outerColor[0], c.outerColor[1], c.outerColor[2])
	}
	cd.vbo.SetData(4*len(data), gl.Ptr(data))

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2)
	defer gl.DisableVertexAttribArray(2)
	gl.EnableVertexAttribArray(3)
	defer gl.DisableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*10, 0)
	gl.VertexAttribPointerWithOffset(1, 1, gl.FLOAT, false, 4*10, 3*4)
	gl.VertexAttribPointerWithOffset(2, 3, gl.FLOAT, false, 4*10, 4*4)
	gl.VertexAttribPointerWithOffset(3, 3, gl.FLOAT, false, 4*10, 7*4)

	view.projection.SetAsUniform(cd.projection)
	view.modelView.SetAsUniform(cd.modelview)

	gl.DrawArrays(gl.POINTS, 0, int32(len(cs)))
}

func (r *qRenderer) RenderDynamicLights() {
	if !cvars.GlFlashBlend.Bool() {
		// TODO(therjak): disabling flashblend is broken since transparent console
		return
	}

	r.lightFrameCount++
	// TODO: remove this allociation
	cs := make([]qCone, 0, len(cl.dynamicLights))
	for i := range cl.dynamicLights {
		dl := &cl.dynamicLights[i]
		if dl.dieTime < cl.time || dl.radius == 0 {
			continue
		}
		// TODO: why do we need this scaling of radius. can the radius be
		// 'right' from the start?
		rad := dl.radius * 0.35
		d := vec.Sub(dl.origin, qRefreshRect.viewOrg)
		if d.Length() < rad {
			// view is inside the dynamic light
			view.addLightBlend(1, 0.5, 0, dl.radius*0.0003)
			continue
		}

		cs = append(cs, qCone{
			origin:     dl.origin,
			radius:     rad,
			innerColor: [3]float32{0.2, 0.1, 0.0},
			outerColor: [3]float32{0, 0, 0},
		})
	}
	if len(cs) == 0 {
		return
	}
	coneDrawer.Draw(cs)
}

func entAlphaDecode(a byte) float32 {
	// 0 == ENTALPHA_DEFAULT
	if a == 0 {
		return 1
	}
	return (float32(a) - 1) - 254
}

func (r *qRenderer) DrawEntitiesOnList(alphaPass bool) {
	if !cvars.RDrawEntities.Bool() {
		return
	}
	// r.DrawShadows()
	for _, e := range visibleEntities {
		a := entAlphaDecode(e.Alpha)
		if (a < 1 && !alphaPass) || (a == 1 && alphaPass) {
			continue
		}

		if e == cl.Entity() {
			e.Angles[0] *= 0.3
		}

		switch m := e.Model.(type) {
		case *mdl.Model:
			r.DrawAliasModel(e, m)
		case *bsp.Model:
			r.DrawBrushModel(e, m)
		case *spr.Model:
			r.DrawSpriteModel(e, m)
		}
	}
}

func (r *qRenderer) backFaceCull(s *bsp.Surface) bool {
	viewOrg := qRefreshRect.viewOrg

	dot := func(p *bsp.Plane) float32 {
		switch p.Type {
		case 0, 1, 2:
			return viewOrg[p.Type] - p.Dist
		default:
			return vec.Dot(viewOrg, p.Normal) - p.Dist
		}
	}(s.Plane)

	if (dot < 0) != (s.Flags&bsp.SurfacePlaneBack != 0) {
		return true
	}

	return false
}

func (r *qRenderer) cullSurfaces(textures []*bsp.Texture) {
	for _, tx := range textures {
		if tx == nil || tx.TextureChains[chainWorld] == nil {
			continue
		}
		for s := tx.TextureChains[chainWorld]; s != nil; s = s.TextureChain {
			s.Culled = r.CullBox(s.Mins, s.Maxs) || r.backFaceCull(s)
		}
	}
}

/*
func (r *qRenderer) DrawShadows() {
	if !cvars.RShadows.Bool() {
		return
	}

	// TODO: This depends on the fbo created later
	// Need to revisit after DrawAliasShadow no longer uses the fixed pipeline
	gl.Clear(gl.STENCIL_BUFFER_BIT)
	gl.StencilFunc(gl.EQUAL, 0, ^uint32(0))
	gl.StencilOp(gl.KEEP, gl.KEEP, gl.INCR)
	gl.Enable(gl.STENCIL_TEST)

	for _, e := range visibleEntities {
		switch m := e.Model.(type) {
		case *mdl.Model:
			r.DrawAliasShadow(e, m)
		}
	}

	gl.Disable(gl.STENCIL_TEST)
}*/
