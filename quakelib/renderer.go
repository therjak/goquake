package quakelib

import "C"
import (
	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/progs"
	"log"
)

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

//export R_framecount
func R_framecount() int {
	return renderer.frameCount
}

//export R_framecount_inc
func R_framecount_inc() {
	renderer.frameCount++
}

//export R_framecount_reset
func R_framecount_reset() {
	renderer.frameCount = 0
}

//export R_visframecount
func R_visframecount() int {
	return renderer.visFrameCount
}

//export R_visframecount_inc
func R_visframecount_inc() {
	renderer.visFrameCount++
}

//export R_visframecount_reset
func R_visframecount_reset() {
	renderer.visFrameCount = 0
}

//export R_dlightframecount
func R_dlightframecount() int {
	return renderer.lightFrameCount
}

//export R_dlightframecount_up
func R_dlightframecount_up() {
	// gets executed before frameCount was increased
	renderer.lightFrameCount = renderer.frameCount + 1
}

//export R_CullBox
func R_CullBox(mins, maxs *C.float) bool {
	return renderer.CullBox(p2v3(mins), p2v3(maxs))
}

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
	scaleForward := math32.Cos(ar)
	scaleSide := math32.Sin(ar)
	p.normal = vec.Add(vec.Scale(scaleForward, forward), vec.Scale(scaleSide, side))
	p.dist = vec.Dot(qRefreshRect.viewOrg, p.normal)
	p.UpdateSignBits()
}

//export R_SetFrustum
func R_SetFrustum(fovx, fovy float32) {
	renderer.SetFrustum(fovx, fovy)
}

func (r *qRenderer) SetFrustum(fovx, fovy float32) {
	// We do not use qRefreshRect.fovX/fovY directly as water has an effect on these values
	r.frustum[0].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, fovx/2-90) // left
	r.frustum[1].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewRight, 90-fovx/2) // right
	r.frustum[2].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, 90-fovy/2)    // bottom
	r.frustum[3].TurnVector(qRefreshRect.viewForward, qRefreshRect.viewUp, fovy/2-90)    // top
}

//export R_DrawViewModel
func R_DrawViewModel() {
	renderer.DrawWeaponModel()
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
	if weapon.Model.Type != model.ModAlias {
		// this fixes a crash
		return
	}

	// hack the depth range to prevent view model from poking into walls
	gl.DepthRange(0, 0.3)
	r.DrawAliasModel(weapon)
	gl.DepthRange(0, 1)
}

var circleDrawer *qCircleDrawer

type qCircleDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32
}

type qCircle struct {
	origin     vec.Vec3
	radius     float32
	innerColor [3]float32
	outerColor [3]float32
}

func newCircleDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexCircleSource, fragmentCircleSource)
}

func newCircleDrawer() *qCircleDrawer {
	d := &qCircleDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer()
	var err error
	d.prog, err = newCircleDrawProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")
	return d
}

func (cd *qCircleDrawer) Draw(cs []qCircle, viewOrigin, viewForward, viewRight, viewUp vec.Vec3) {
	// viewport(0,0,screen.width, screen.height)

	gl.DepthMask(false)
	gl.Disable(gl.TEXTURE_2D)

	//glShadeModel(GL_SMOOTH);
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)

	projection := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	modelview := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	// NOTE about the matrix element order:
	// [00][04][08][12]
	// [01][05][09][13]
	// [02][06][10][14]
	// [03][07][11][15]

	// projection is a matrix based on
	// fov, screen.width/screen.height, nearclip, farclip
	// fov = tan(fovx * pi / 360), checked
	// aspect ratio is correct
	// fov 0 0 0
	// 0 aspect, 0,0
	// ....
	// all values are coming from
	// fovy = atan(0.75/(sh/sw)*tan(fovx/360pi)) * 360/pi
	// xmax = 4 * tan(fovx * pi / 360)
	// ymax = 4 * tan(fovy * pi / 360)
	// glFrustum(-xmax, xmax, -ymax, ymax, 4, cvars.gl_farclip)
	// -> 8/-2xmax, 0, 0, 0
	//    0, 8/-2ymax, 0, 0
	//    0, 0, -(far+4)/(far-4), -(2*far*4)/(far-4)
	//    0, 0, -1, 0
	gl.GetFloatv(0x0BA7, &projection[0])

	// modelview should be
	// viewright.x,viewUp.x,-viewForward.x,0
	// viewright.y,viewUp.y,-viewForward.y,0
	// viewright.z,viewUp.z,-viewForward.z,0
	// ?| ?| ?| 1
	// it get set by
	// glRotatef(-90, 1, 0, 0)
	// glRotatef(90, 0, 0, 1)
	// glRotatef(-viewangles[2], 1, 0, 0)
	// glRotatef(-viewangles[0], 0, 1, 0)
	// glRotatef(-viewangles[1], 0, 0, 1)
	// glTranslate(-vieworg[0], -vieworg[1], -vieworg[2])
	gl.GetFloatv(0x0BA6, &modelview[0])
	/*
		log.Printf("mv: %v", modelview)
		log.Printf("ex: %v, %v, %v, %v, %v",
			qRefreshRect.viewOrg,
			qRefreshRect.viewForward,
			qRefreshRect.viewRight,
			qRefreshRect.viewUp,
			qRefreshRect.viewAngles)
	*/
	if false {
		log.Printf("something")
	}

	cd.prog.Use()
	cd.vao.Bind()
	cd.vbo.Bind(gl.ARRAY_BUFFER)
	// TODO: remove this allocation
	data := make([]float32, 0, len(cs)*(3+1+3+3))
	for _, c := range cs {
		data = append(data,
			c.origin[0], c.origin[1], c.origin[2],
			c.radius,
			c.innerColor[0], c.innerColor[1], c.innerColor[2],
			c.outerColor[0], c.outerColor[1], c.outerColor[2])
	}
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(data), gl.Ptr(data), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(0))
	gl.VertexAttribPointer(1, 1, gl.FLOAT, false, 4*8, gl.PtrOffset(3*4))
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(4*4))
	gl.VertexAttribPointer(3, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(7*4))

	gl.UniformMatrix4fv(cd.projection, 1, false, &projection[0])
	gl.UniformMatrix4fv(cd.modelview, 1, false, &modelview[0])

	// gl.DrawArrays(gl.TRIANGLES, 0, int32(len(cs)))

	gl.DisableVertexAttribArray(0)
	gl.DisableVertexAttribArray(1)
	gl.DisableVertexAttribArray(2)
	gl.DisableVertexAttribArray(3)

	gl.Disable(gl.BLEND)
	gl.Enable(gl.TEXTURE_2D)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(true)
}

func drawVertex(v vec.Vec3) {
	// dummy for gl.Vertex3fv
	// probably something similar to qParticleDrawer.Draw
}

func (r *qRenderer) RenderDynamicLight(l *DynamicLight) {

	rad := l.Radius * 0.35
	d := vec.Sub(l.Origin, qRefreshRect.viewOrg)
	if d.Length() < rad {
		// view is inside the dynamic light
		view.addLightBlend(1, 0.5, 0, l.Radius*0.0003)
		return
	}

	// TODO: this should probably all get into the shader
	// parameters inside should be rad, l.Origin, qRefreshRect
	center := vec.Sub(l.Origin, vec.Scale(rad, qRefreshRect.viewForward))
	drawVertex(center)
	// sprockets
	for i := float32(16); i >= 0; i-- {
		a := i / 16 * math32.Pi * 2
		s, c := math32.Sincos(a)
		v := l.Origin
		v.Add(vec.Scale(c*rad, qRefreshRect.viewRight))
		v.Add(vec.Scale(s*rad, qRefreshRect.viewUp))
		drawVertex(v)
	}

	/*

	   int i, j;
	   float a;
	   vec3_t v;
	   float rad;

	   glBegin(GL_TRIANGLE_FAN);
	   glColor3f(0.2, 0.1, 0.0);
	   for (i = 0; i < 3; i++) v[i] = light->origin[i] - vpn[i] * rad;
	   glVertex3fv(v);
	   glColor3f(0, 0, 0);
	   for (i = 16; i >= 0; i--) {
	     a = i / 16.0 * M_PI * 2;
	     for (j = 0; j < 3; j++)
	       v[j] =
	           light->origin[j] + vright[j] * cos(a) * rad + vup[j] * sin(a) * rad;
	     glVertex3fv(v);
	   }
	   glEnd();

	*/
}

func (r *qRenderer) RenderDynamicLights() {
	if !cvars.GlFlashBlend.Bool() {
		// TODO(therjak): disabling flashblend is broken since transparent console
		return
	}
	if circleDrawer == nil {
		circleDrawer = newCircleDrawer()
	}

	r.lightFrameCount++
	// TODO: remove this allociation
	cs := make([]qCircle, 0, len(cl.dynamicLights))
	for i := range cl.dynamicLights {
		dl := &cl.dynamicLights[i]
		if dl.DieTime < cl.time || dl.Radius == 0 {
			continue
		}
		// TODO: why do we need this scaling of radius. can the radius be
		// 'right' from the start?
		rad := dl.Radius * 0.35
		d := vec.Sub(dl.Origin, qRefreshRect.viewOrg)
		if d.Length() < rad {
			// view is inside the dynamic light
			view.addLightBlend(1, 0.5, 0, dl.Radius*0.0003)
			continue
		}

		cs = append(cs, qCircle{
			origin:     dl.Origin,
			radius:     rad,
			innerColor: [3]float32{0.2, 0.1, 0.0},
			outerColor: [3]float32{0, 0, 0},
		})
		// r.RenderDynamicLight(dl)
	}
	if len(cs) == 0 {
		return
	}
	circleDrawer.Draw(cs,
		qRefreshRect.viewOrg,
		qRefreshRect.viewForward,
		qRefreshRect.viewRight,
		qRefreshRect.viewUp)
	// glColor3f(1, 1, 1);
}
