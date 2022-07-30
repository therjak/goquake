// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"log"
	"strconv"

	"goquake/bsp"
	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvar"
	"goquake/cvars"
	"goquake/glh"
	"goquake/math"
	"goquake/math/vec"
	"goquake/texture"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
)

func init() {
	addCommand("sky", skyCommand)
	cvars.RSkyFog.SetCallback(func(cv *cvar.Cvar) {
		sky.fog = cv.Value()
	})

}

func skyCommand(args []cmd.QArg, _ int) error {
	switch len(args) {
	case 0:
		conlog.Printf("\"sky\" is \"%s\"\n", sky.boxName)
	case 1:
		sky.LoadBox(args[0].String())
	default:
		conlog.Printf("usage: sky <skyname>\n")
	}
	return nil
}

type qSky struct {
	boxName      string
	solidTexture *texture.Texture
	alphaTexture *texture.Texture
	flat         Color
	mins         [2][6]float32
	maxs         [2][6]float32
	fog          float32
	simpleDrawer *qSimpleSkyDrawer
	boxDrawer    *qSkyBoxDrawer
}

var (
	sky       qSky
	skyDrawer *qSkyDrawer
)

func CreateSkyDrawer() {
	skyDrawer = newSkyDrawer()
	sky.simpleDrawer = newSimpleSkyDrawer()
	sky.boxDrawer = newSkyBoxDrawer()
}

func (s *qSky) newMap(worldspawn *bsp.Entity) {
	s.fog = cvars.RSkyFog.Value()
	s.boxName = ""
	s.boxDrawer.texture = nil
	if p, ok := worldspawn.Property("sky"); ok {
		s.LoadBox(p)
	}
	if p, ok := worldspawn.Property("skyfog"); ok {
		v, err := strconv.ParseFloat(p, 32)
		if err == nil {
			s.fog = float32(v)
		}
	} else if p, ok := worldspawn.Property("skyname"); ok { // half-life
		s.LoadBox(p)
	} else if p, ok := worldspawn.Property("glsky"); ok { // quake lives
		s.LoadBox(p)
	}
}

func (s *qSky) LoadBox(name string) {
	if name == s.boxName {
		return
	}
	s.boxName = name
	textureManager.FreeTexture(s.boxDrawer.texture) // clean up textureManager cache
	s.boxDrawer.texture = nil
	if s.boxName == "" {
		// Turn off skybox
		return
	}
	s.boxDrawer.texture = textureManager.LoadSkyBox(s.boxName)
	if s.boxDrawer.texture == nil {
		// boxName == "" => No DrawSkyBox but only DrawSkyLayers
		s.boxName = ""
	}
}

func (s *qSky) LoadTexture(t *bsp.Texture) {

	s.solidTexture = t.SolidSky
	textureManager.BindUnit(s.solidTexture, gl.TEXTURE0)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	s.alphaTexture = t.AlphaSky
	textureManager.BindUnit(s.alphaTexture, gl.TEXTURE0)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	s.flat = Color{
		R: t.FlatSky.R,
		G: t.FlatSky.G,
		B: t.FlatSky.B,
	}
}

type skyVec [3]int

var (
	st2vec  = [6]skyVec{{3, -1, 2}, {-3, 1, 2}, {1, 3, 2}, {-1, -3, 2}, {-2, -1, 3}, {2, -1, -3}}
	vec2st  = [6]skyVec{{-2, 3, 1}, {2, 3, -1}, {1, 3, 2}, {-1, 3, -2}, {-2, -1, 3}, {-2, 1, -3}}
	skyClip = [6]vec.Vec3{{1, 1, 0}, {1, -1, 0}, {0, -1, 1}, {0, 1, 1}, {1, 0, 1}, {-1, 0, 1}}
)

func (sky *qSky) updateBounds(vecs []vec.Vec3) {
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
		if t > sky.maxs[1][axis] {
			sky.maxs[1][axis] = t
		}
	}
}

// MAX_CLIP_VERTS = 64
func (s *qSky) clipPoly(vecs []vec.Vec3, stage int) {
	if stage >= 6 || stage < 0 {
		s.updateBounds(vecs)
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
		s.clipPoly(vecs, stage+1)
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
	s.clipPoly(newvf, stage+1)
	s.clipPoly(newvb, stage+1)
}

func (s *qSky) processEntities(c Color) {
	if !cvars.RDrawEntities.Bool() {
		return
	}

	viewOrg := qRefreshRect.viewOrg
	for _, e := range visibleEntities {
		var model *bsp.Model
		switch m := e.Model.(type) {
		default:
			continue
		case *bsp.Model:
			if renderer.cullBrush(e, m) {
				continue
			}
			model = m
		}
		if e.Alpha == 1 {
			// invisible
			continue
		}
		modelOrg := vec.Sub(viewOrg, e.Origin)
		var rotated bool
		var fwd, r, u vec.Vec3
		if e.Angles[0] != 0 || e.Angles[1] != 0 || e.Angles[2] != 0 {
			rotated = true
			fwd, r, u = vec.AngleVectors(e.Angles)
			tmp := modelOrg
			modelOrg[0] = vec.Dot(tmp, fwd)
			modelOrg[1] = -vec.Dot(tmp, r)
			modelOrg[2] = vec.Dot(tmp, u)
		}
		for _, su := range model.Surfaces {
			if su.Flags&bsp.SurfaceDrawSky == 0 {
				continue
			}
			dot := vec.Dot(modelOrg, su.Plane.Normal) - su.Plane.Dist
			if (su.Flags&bsp.SurfacePlaneBack != 0 && dot < -0.01) ||
				(su.Flags&bsp.SurfacePlaneBack == 0 && dot > 0.01) {
				// TODO(therjak): remove/cache this alloc
				var poly bsp.Poly
				poly.Verts = make([]bsp.TexCoord, 0, len(su.Polys.Verts))
				for _, v := range su.Polys.Verts {
					if rotated {
						pos := v.Pos
						np := vec.Vec3{
							e.Origin[0] + pos[0]*fwd[0] - pos[1]*r[0] + pos[2]*u[0],
							e.Origin[1] + pos[0]*fwd[1] - pos[1]*r[1] + pos[2]*u[1],
							e.Origin[2] + pos[0]*fwd[2] - pos[1]*r[2] + pos[2]*u[2],
						}
						poly.Verts = append(poly.Verts, bsp.TexCoord{
							Pos: np,
						})
					} else {
						poly.Verts = append(poly.Verts, v)
					}
				}
				s.processPoly(&poly, c)
			}
		}
	}
}

var (
	sqrt3 = math32.Sqrt(3)
)

// DrawSkyLayers draws the old-style scrolling cloud layers
func (s *qSky) DrawSkyLayers() {
	if mapAlphas.sky < 1 {
		// TODO: this needs to go into the shader
		//glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_MODULATE)
		//defer glTexEnvf(GL_TEXTURE_ENV, GL_TEXTURE_ENV_MODE, GL_REPLACE)
	}
	fc := cvars.GlFarClip.Value() / sqrt3
	fc2 := 2 * fc
	// TODO: why do the model/view stuff outside the shader?
	// check qRefreshRect.viewOrg below

	skyDrawer.prog.Use()
	skyDrawer.vao.Bind()
	skyDrawer.ebo.Bind()
	skyDrawer.vbo.Bind()

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2)
	defer gl.DisableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*7, 0)   // pos
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 4*7, 3*4) // solidTexPos
	gl.VertexAttribPointerWithOffset(2, 2, gl.FLOAT, false, 4*7, 5*4) // alphaTexPos

	view.projection.SetAsUniform(skyDrawer.projection)
	view.modelView.SetAsUniform(skyDrawer.modelview)
	gl.Uniform1i(skyDrawer.solidTex, 0)
	gl.Uniform1i(skyDrawer.alphaTex, 1)

	textureManager.BindUnit(s.solidTexture, gl.TEXTURE0)
	defer textureManager.SelectTextureUnit(gl.TEXTURE0)
	textureManager.BindUnit(s.alphaTexture, gl.TEXTURE1)

	sc1 := math32.Mod(float32(cl.time)*8, 128)
	sc2 := math32.Mod(float32(cl.time)*16, 128)
	vertices := make([]float32, 0, 7*4)

	drawFace := func(mins, maxs [2]float32, vup, vright, v vec.Vec3) {
		// Textures are 128x128
		v = vec.Add(qRefreshRect.viewOrg, v)
		p1 := v
		p2 := vec.Add(v, vup)
		p3 := vec.Add(p2, vright)
		p4 := vec.Add(v, vright)
		// TODO: s&t are still wrong for both tex
		vertices = vertices[:0]
		ap := func(p vec.Vec3) {
			v := vec.Sub(p, qRefreshRect.viewOrg)
			v[2] *= 3 // flatten the sphere
			l := 6 * 63 * v.RLength()
			s1 := (sc1 + v[0]*l) / 128
			t1 := (sc1 + v[1]*l) / 128
			s2 := (sc2 + v[0]*l) / 128
			t2 := (sc2 + v[1]*l) / 128
			vertices = append(vertices, p[0], p[1], p[2], s1, t1, s2, t2)
		}
		ap(p1)
		ap(p2)
		ap(p3)
		ap(p4)
		skyDrawer.vbo.SetData(4*len(vertices), gl.Ptr(vertices))
		gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
	}

	// TODO(therjak): this can/should be done in one draw call.
	if s.mins[0][0] < s.maxs[0][0] && s.mins[1][0] < s.maxs[1][0] {
		mins := [2]float32{s.mins[0][0], s.mins[1][0]}
		maxs := [2]float32{s.maxs[0][0], s.maxs[1][0]}
		vup := vec.Vec3{0, 0, fc2}
		vright := vec.Vec3{0, -fc2, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{fc, fc, -fc})
	}
	if s.mins[0][1] < s.maxs[0][1] && s.mins[1][1] < s.maxs[1][1] {
		mins := [2]float32{s.mins[0][1], s.mins[1][1]}
		maxs := [2]float32{s.maxs[0][1], s.maxs[1][1]}
		vup := vec.Vec3{0, 0, fc2}
		vright := vec.Vec3{0, fc2, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{-fc, -fc, -fc})
	}
	if s.mins[0][2] < s.maxs[0][2] && s.mins[1][2] < s.maxs[1][2] {
		mins := [2]float32{s.mins[0][2], s.mins[1][2]}
		maxs := [2]float32{s.maxs[0][2], s.maxs[1][2]}
		vup := vec.Vec3{0, 0, fc2}
		vright := vec.Vec3{fc2, 0, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{-fc, fc, -fc})
	}
	if s.mins[0][3] < s.maxs[0][3] && s.mins[1][3] < s.maxs[1][3] {
		mins := [2]float32{s.mins[0][3], s.mins[1][3]}
		maxs := [2]float32{s.maxs[0][3], s.maxs[1][3]}
		vup := vec.Vec3{0, 0, fc2}
		vright := vec.Vec3{-fc2, 0, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{fc, -fc, -fc})
	}
	if s.mins[0][4] < s.maxs[0][4] && s.mins[1][4] < s.maxs[1][4] {
		mins := [2]float32{s.mins[0][4], s.mins[1][4]}
		maxs := [2]float32{s.maxs[0][4], s.maxs[1][4]}
		vup := vec.Vec3{-fc2, 0, 0}
		vright := vec.Vec3{0, -fc2, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{fc, fc, fc})
	}
	if s.mins[0][5] < s.maxs[0][5] && s.mins[1][5] < s.maxs[1][5] {
		mins := [2]float32{s.mins[0][5], s.mins[1][5]}
		maxs := [2]float32{s.maxs[0][5], s.maxs[1][5]}
		vup := vec.Vec3{fc2, 0, 0}
		vright := vec.Vec3{0, -fc2, 0}
		drawFace(mins, maxs, vup, vright, vec.Vec3{-fc, fc, -fc})
	}
}

// for drawing the single colored sky
type qSimpleSkyDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
	color      int32
	vertices   []float32
}

func newSimpleSkyDrawer() *qSimpleSkyDrawer {
	d := &qSimpleSkyDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	var err error
	d.prog, err = newSimpleSkyProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection") // mat
	d.modelview = d.prog.GetUniformLocation("modelview")   // mat
	d.color = d.prog.GetUniformLocation("in_color")        // vec4
	return d
}

func newSimpleSkyProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexWorldPositionSource, fragmentSourceColorRecDrawer)
}

func (d *qSimpleSkyDrawer) draw(p *bsp.Poly, c Color) {
	d.prog.Use()
	d.vao.Bind()
	d.vbo.Bind()

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*3, 0) // pos

	view.projection.SetAsUniform(d.projection)
	view.modelView.SetAsUniform(d.modelview)
	gl.Uniform4f(d.color, c.R, c.G, c.B, c.A)

	d.vertices = d.vertices[:0]
	for _, v := range p.Verts {
		d.vertices = append(d.vertices, v.Pos[0], v.Pos[1], v.Pos[2])
	}
	d.vbo.SetData(4*len(d.vertices), gl.Ptr(d.vertices))
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, int32(len(p.Verts)))
}

type qSkyDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	ebo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
	solidTex   int32
	alphaTex   int32
}

func newSkyDrawer() *qSkyDrawer {
	d := &qSkyDrawer{}
	elements := []uint32{
		0, 1, 2,
		2, 3, 0,
	}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	d.ebo = glh.NewBuffer(glh.ElementArrayBuffer)
	d.ebo.Bind()
	d.ebo.SetData(4*len(elements), gl.Ptr(elements))
	var err error
	d.prog, err = newSkyProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	d.solidTex = d.prog.GetUniformLocation("solid")
	d.alphaTex = d.prog.GetUniformLocation("alpha")
	return d
}

func newSkyProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexDualTextureSource, fragmentSourceDualTextureDrawer)
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

	color := s.flat
	if fog.Density > 0 {
		color = fog.Color
	}
	// Draw a simple uni color sky
	s.processTextureChains(color)
	s.processEntities(color)

	if !cvars.RFastSky.Bool() && !(fog.GetDensity() > 0 && s.fog >= 1) {
		// Draw better quality sky
		gl.DepthFunc(gl.GEQUAL)
		defer gl.DepthFunc(gl.LEQUAL)
		gl.DepthMask(false)
		defer gl.DepthMask(true)

		if len(sky.boxName) != 0 {
			s.drawBox()
		} else {
			s.DrawSkyLayers()
		}
	}
}

func (s *qSky) processTextureChains(c Color) {
	for _, t := range cl.worldModel.Textures {
		if t == nil {
			continue
		}
		cw := t.TextureChains[chainWorld]
		if cw == nil || cw.Flags&bsp.SurfaceDrawSky == 0 {
			continue
		}
		for tc := cw; tc != nil; tc = tc.TextureChain {
			if !tc.Culled {
				s.processPoly(tc.Polys, c)
			}
		}
	}
}

func (s *qSky) processPoly(p *bsp.Poly, c Color) {
	s.simpleDrawer.draw(p, c)

	// update sky bounds
	if !cvars.RFastSky.Bool() {
		v := make([]vec.Vec3, 0, len(p.Verts))
		for _, p := range p.Verts {
			v = append(v, vec.Sub(p.Pos, qRefreshRect.viewOrg))
		}
		s.clipPoly(v, 0)
	}
}

func (s *qSky) drawBox() {
	log.Printf("Drawing sky box: %v", s.boxName)
	r, g, b, a := fog.GetColor()
	a = math.Clamp32(0, s.fog, 1)
	fogColor := Color{r, g, b, a}
	s.boxDrawer.draw(fogColor)
}

// for drawing the single colored sky
type qSkyBoxDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
	fogColor   int32
	vertices   []float32
	texture    *texture.Texture
}

func newSkyBoxProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceSkybox, fragmentSourceSkybox)
}

func newSkyBoxDrawer() *qSkyBoxDrawer {
	d := &qSkyBoxDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	var err error
	d.prog, err = newSkyBoxProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection") // mat
	d.modelview = d.prog.GetUniformLocation("modelview")   // mat
	// d.fogColor = d.prog.GetUniformLocation("fogColor")     // vec4

	vertices := []float32{
		-1, 1, -1,
		-1, -1, -1,
		1, -1, -1,
		1, -1, -1,
		1, 1, -1,
		-1, 1, -1,

		-1, -1, 1,
		-1, -1, -1,
		-1, 1, -1,
		-1, 1, -1,
		-1, 1, 1,
		-1, -1, 1,

		1, -1, -1,
		1, -1, 1,
		1, 1, 1,
		1, 1, 1,
		1, 1, -1,
		1, -1, -1,

		-1, -1, 1,
		-1, 1, 1,
		1, 1, 1,
		1, 1, 1,
		1, -1, 1,
		-1, -1, 1,

		-1, 1, -1,
		1, 1, -1,
		1, 1, 1,
		1, 1, 1,
		-1, 1, 1,
		-1, 1, -1,

		-1, -1, -1,
		-1, -1, 1,
		1, -1, -1,
		1, -1, -1,
		-1, -1, 1,
		1, -1, 1,
	}
	d.vao.Bind()
	d.vbo.Bind()
	d.vbo.SetData(4*len(vertices), gl.Ptr(vertices))

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 4*3, 0)

	return d
}

func (d *qSkyBoxDrawer) draw(fog Color) {
	mv := view.modelView.Copy()
	// put the viewer back into the center
	mv.Translate(qRefreshRect.viewOrg[0], qRefreshRect.viewOrg[1], qRefreshRect.viewOrg[2])

	d.prog.Use()
	d.vao.Bind()
	d.vbo.Bind()

	view.projection.SetAsUniform(d.projection)
	mv.SetAsUniform(d.modelview)
	// gl.Uniform4f(d.fogColor, fog.R, fog.G, fog.B, fog.A)
	textureManager.BindUnit(d.texture, gl.TEXTURE0)

	// gl.DrawArrays(gl.TRIANGLE_FAN, 0, int32(len(p.Verts)))
}
