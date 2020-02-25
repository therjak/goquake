package quakelib

import (
	"log"
	"math"
	"math/rand"
	"quake/commandline"
	"quake/cvars"
	"quake/math/vec"

	"github.com/chewxy/math32"
	"github.com/go-gl/gl/v4.6-core/gl"
)

type particleType int

const (
	ParticleTypeStatic particleType = iota
	ParticleTypeGrav
	ParticleTypeSlowGrav
	ParticleTypeFire
	ParticleTypeExplode
	ParticleTypeExplode2
	ParticleTypeBlob
	ParticleTypeBlob2
)

const (
	vertexSourceParticleDrawer = `
#version 410
in vec3 vcolor;
in vec3 vposition;
in vec2 vtexcoord;
out vec2 Texcoord;
out vec3 InColor;
uniform mat4 projection;
uniform mat4 modelview;

void main() {
	Texcoord = vtexcoord;
	InColor = vcolor;
	gl_Position = projection * modelview * vec4(vposition, 1.0);
}
` + "\x00"

	fragmentSourceParticleDrawer = `
#version 410
in vec2 Texcoord;
in vec3 InColor;
out vec4 frag_color;
uniform sampler2D tex;

void main() {
	float color = texture(tex, Texcoord).r;
	frag_color.rgb = InColor;
	frag_color.a = color; // texture has only one chan
	frag_color = clamp(frag_color, vec4(0,0,0,0), vec4(1,1,1,1));
}
` + "\x00"
)

var (
	particleDrawer *qParticleDrawer
)

type qParticleDrawer struct {
	vao        uint32
	vbo        uint32
	prog       uint32
	position   uint32
	color      uint32
	texcoord   uint32
	projection int32
	modelview  int32

	texture            uint32
	textures           [2]uint32
	textureScaleFactor float32
}

func newParticleDrawProgram() uint32 {
	vert := getShader(vertexSourceParticleDrawer, gl.VERTEX_SHADER)
	log.Printf("vertex: %s", vertexSourceParticleDrawer)
	frag := getShader(fragmentSourceParticleDrawer, gl.FRAGMENT_SHADER)
	d := gl.CreateProgram()
	gl.AttachShader(d, vert)
	gl.AttachShader(d, frag)
	gl.LinkProgram(d)
	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	return d
}

func newParticleDrawer() *qParticleDrawer {
	d := &qParticleDrawer{}
	gl.GenVertexArrays(1, &d.vao)
	gl.GenBuffers(1, &d.vbo)
	d.prog = newParticleDrawProgram()
	d.color = uint32(gl.GetAttribLocation(d.prog, gl.Str("vcolor\x00")))
	d.texcoord = uint32(gl.GetAttribLocation(d.prog, gl.Str("vtexcoord\x00")))
	d.position = uint32(gl.GetAttribLocation(d.prog, gl.Str("vposition\x00")))
	d.projection = gl.GetUniformLocation(d.prog, gl.Str("projection\x00"))
	d.modelview = gl.GetUniformLocation(d.prog, gl.Str("modelview\x00"))

	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	gl.GenTextures(2, &d.textures[0])
	gl.BindTexture(gl.TEXTURE_2D, d.textures[0])
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R16F, 64, 64, 0, gl.RED, gl.FLOAT, gl.Ptr(p1TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	gl.BindTexture(gl.TEXTURE_2D, d.textures[1])
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R16F, 2, 2, 0, gl.RED, gl.FLOAT, gl.Ptr(p2TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	d.texture = d.textures[0]
	d.textureScaleFactor = float32(1.27)
	return d
}

func (d *qParticleDrawer) cleanup() {
	gl.DeleteProgram(d.prog)
	gl.DeleteBuffers(1, &d.vbo)
	gl.DeleteVertexArrays(1, &d.vao)
	gl.DeleteTextures(2, &d.textures[0])
}

func (d *qParticleDrawer) Draw(ps []particle) {
	up := vec.Scale(1.5, qRefreshRect.viewUp)
	right := vec.Scale(1.5, qRefreshRect.viewRight)

	fwd := qRefreshRect.viewForward
	origin := qRefreshRect.viewOrg

	vertices := []float32{}
	numVert := uint32(0)
	for _, p := range ps {
		if !p.used {
			continue
		}
		o := p.origin
		scale := (o[0]-origin[0])*fwd[0] +
			(o[1]-origin[1])*fwd[1] +
			(o[2]-origin[2])*fwd[2]
		if scale < 20 {
			scale = 1 + 0.08
		} else {
			scale = 1 + scale*0.004
		}
		scale *= d.textureScaleFactor
		u := vec.Scale(scale, up)
		r := vec.Scale(scale, right)

		ci := p.color * 4
		c := vec.Vec3{
			float32(palette.table[ci]) / 255,
			float32(palette.table[ci+1]) / 255,
			float32(palette.table[ci+2]) / 255,
		}
		numVert += 3

		// add
		// glRotatef(-90,1,0,0)
		// glRotatef(90,0,0,1)
		// aka z is up
		//		v1 := vec.Vec3{-o[2], -o[0], o[1]}
		//		v2 := vec.Add(v1, vec.Vec3{-u[2], -u[0], u[1]})
		//		v3 := vec.Add(v1, vec.Vec3{-r[2], -r[0], r[1]})

		// add
		// glRotatef(-qRefreshRect.viewAngles[2], 1,0,0)
		// glRotatef(-qRefreshRect.viewAngles[0], 0,1,0)
		// glRotatef(-qRefreshRect.viewAngles[1], 0,0,1)

		// add
		// glTranslatef(-qRefreshRect.viewOrg[0], -qRefreshRect.viewOrg[1], -qRefreshRect.viewOrg[2])
		//		v1.Sub(qRefreshRect.viewOrg)
		//		v2.Sub(qRefreshRect.viewOrg)
		//		v3.Sub(qRefreshRect.viewOrg)

		// x, y, z, tx, ty, r, g, b
		vertices = append(vertices,
			//			v1[0], v1[1], v1[2], 0, 0, c[0], c[1], c[2],
			//			v2[0], v2[1], v2[2], 1, 0, c[0], c[1], c[2],
			//			v3[0], v3[1], v3[2], 0, 1, c[0], c[1], c[2])

			// orig:
			o[0], o[1], o[2], 0, 0, c[0], c[1], c[2],
			(o[0] + u[0]), (o[1] + u[1]), (o[2] + u[2]), 1, 0, c[0], c[1], c[2],
			(o[0] + r[0]), (o[1] + r[1]), (o[2] + r[2]), 0, 1, c[0], c[1], c[2])
	}
	if len(vertices) == 0 {
		return
	}

	projection := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	gl.GetFloatv(0x0BA7, &projection[0])
	modelview := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	gl.GetFloatv(0x0BA6, &modelview[0])
	// gl_viewport: 0x0x0ba2
	// gl_texture_matrix: 0x0ba8

	defer func() {
		// To clean up again
		gl.MatrixLoadIdentityEXT(gl.PATH_PROJECTION_NV)
		gl.Viewport(
			int32(qRefreshRect.viewRect.x),
			viewport.height-int32(qRefreshRect.viewRect.y+qRefreshRect.viewRect.height),
			int32(qRefreshRect.viewRect.width),
			int32(qRefreshRect.viewRect.height))
		xmax := 4 * math.Tan(float64(qRefreshRect.fovX)*math.Pi/360)
		ymax := 4 * math.Tan(float64(qRefreshRect.fovY)*math.Pi/360)
		gl.MatrixFrustumEXT(gl.PATH_PROJECTION_NV, -xmax, xmax, -ymax, ymax, 4, float64(cvars.GlFarClip.Value()))

		gl.MatrixLoadIdentityEXT(gl.PATH_MODELVIEW_NV)
		gl.MatrixRotatefEXT(gl.PATH_MODELVIEW_NV, -90, 1, 0, 0)
		gl.MatrixRotatefEXT(gl.PATH_MODELVIEW_NV, 90, 0, 0, 1)

		gl.MatrixRotatefEXT(gl.PATH_MODELVIEW_NV, -qRefreshRect.viewAngles[2], 1, 0, 0)
		gl.MatrixRotatefEXT(gl.PATH_MODELVIEW_NV, -qRefreshRect.viewAngles[0], 0, 1, 0)
		gl.MatrixRotatefEXT(gl.PATH_MODELVIEW_NV, -qRefreshRect.viewAngles[1], 0, 0, 1)

		gl.MatrixTranslatefEXT(gl.PATH_MODELVIEW_NV, -qRefreshRect.viewOrg[0], -qRefreshRect.viewOrg[1], -qRefreshRect.viewOrg[2])
	}()

	gl.Disable(gl.DEPTH_TEST)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)

	gl.DepthMask(false)

	gl.UseProgram(d.prog)
	gl.BindVertexArray(d.vao)

	gl.BindBuffer(gl.ARRAY_BUFFER, d.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(d.position)
	gl.VertexAttribPointer(d.position, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(0))

	gl.EnableVertexAttribArray(d.texcoord)
	gl.VertexAttribPointer(d.texcoord, 2, gl.FLOAT, false, 4*8, gl.PtrOffset(3*4))

	gl.EnableVertexAttribArray(d.color)
	gl.VertexAttribPointer(d.color, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(5*4))

	gl.UniformMatrix4fv(d.projection, 1, false, &projection[0])
	gl.UniformMatrix4fv(d.modelview, 1, false, &modelview[0])

	gl.BindTexture(gl.TEXTURE_2D, d.texture)

	gl.DrawArrays(gl.TRIANGLES, 0, int32(numVert))

	gl.DisableVertexAttribArray(d.color)
	gl.DisableVertexAttribArray(d.texcoord)
	gl.DisableVertexAttribArray(d.position)

	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
	gl.Enable(gl.DEPTH_TEST)
}

type particle struct {
	origin   vec.Vec3
	color    int
	velocity vec.Vec3
	ramp     float32
	dieTime  float32 // if dieTime < now => dead, needs to be server controlled
	typ      particleType
	used     bool
}

var (
	particles       []particle
	freeParticles   []*particle // a stack of free ones
	ramp1           = [8]int{0x6f, 0x6d, 0x6b, 0x69, 0x67, 0x65, 0x63, 0x61}
	ramp2           = [8]int{0x6f, 0x6e, 0x6d, 0x6c, 0x6b, 0x6a, 0x68, 0x66}
	ramp3           = [8]int{0x6d, 0x6b, 6, 5, 4, 3, 0, 0}
	angleVelocities = [162]vec.Vec3{}
)

func init() {
	for i := 0; i < len(angleVelocities); i++ {
		angleVelocities[i] = vec.Vec3{
			// orig has 0 - 2.55 but this gets multiplied by time and into sin/cos
			// so there should be no point to do anything fancy
			rand.Float32(),
			rand.Float32(),
			0, // not used
		}
	}
}

const (
	minParticles = 512
)

var (
	p1TextureData = genCycle()
	// very small cycle in a 2x2 image. aka a pixel...
	p2TextureData = []float32{1, 0, 0, 0}
)

func genCycle() []float32 {
	// simple visualisation of the build cycle image:
	// *0
	// 00
	const r = 16
	const r2 = r * 2
	const rr = r * r
	const s = 64
	cycle := func(x, y float32) float32 {
		if x >= r2 || y >= r2 {
			return 0
		}
		x -= r
		y -= r
		cr := x*x + y*y
		cr /= rr
		if cr >= 1 {
			return 0
		}
		a := 8 * (1 - cr) // increase the sharpness of the transition
		if a >= 1 {
			return 1
		}
		return a
	}
	d := make([]float32, s*s)
	i := 0
	for x := float32(0); x < s; x++ {
		for y := float32(0); y < s; y++ {
			d[i] = cycle(x, y)
			i++
		}
	}
	return d
}

func particlesInit() {
	// this is part of ui startup
	max := commandline.Particles()
	if max < minParticles {
		max = minParticles
	}
	particles = make([]particle, max)
	freeParticles = make([]*particle, 0, max)
	for i := 0; i < len(particles); i++ {
		freeParticles = append(freeParticles, &particles[i])
	}

	//TODO(THERJAK): cvar r_particles callback
	// if r_particles == 1
	// texture1 && factor 1.27
	// if r_particles == 2
	// texture2 && factor 1.0

	particleDrawer = newParticleDrawer()
}

func particlesDeinit() {
	// to be clean this could be run on ui shutdown
	particleDrawer.cleanup()
	particleDrawer = nil
}

func particlesAddEntity(origin vec.Vec3, now float32) {
	for i := 0; i < 162; i++ {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		p.used = true
		freeParticles = freeParticles[:l-1]

		angle := now * angleVelocities[i][0]
		sy := math32.Sin(angle)
		cy := math32.Cos(angle)
		angle = now * angleVelocities[i][1]
		sp := math32.Sin(angle)
		cp := math32.Cos(angle)
		forward := vec.Vec3{
			cp * cy,
			cp * sy,
			-sp,
		}

		p.dieTime = now + 0.01
		p.color = 0x6f
		p.typ = ParticleTypeExplode
		p.origin = vec.Vec3{
			origin[0] + avertexNormals[i][0]*64 + forward[0]*16,
			origin[1] + avertexNormals[i][1]*64 + forward[1]*16,
			origin[2] + avertexNormals[i][2]*64 + forward[2]*16,
		}
	}
}

func particlesClear() {
	freeParticles = freeParticles[:0]
	for i := 0; i < len(particles); i++ {
		particles[i].dieTime = 0
		particles[i].used = false
		freeParticles = append(freeParticles, &particles[i])
	}
}

// on cmd "pointfile"
func particlesReadPointFile() {
	// TODO(THERJAK):
	// p.dieTime = 99999 // that is > 27h
	// p.typ = ParticleTypeStatic
	// p.velocity = vec.Vec3{}
}

// randVec returns a randomized vector with radius at most r
func randVec(r int) vec.Vec3 {
	d := 2 * r
	return vec.Vec3{
		float32((rand.Int() % d) - r),
		float32((rand.Int() % d) - r),
		float32((rand.Int() % d) - r),
	}
}

func particlesAddExplosion(origin vec.Vec3, now float32) {
	for i := 0; i < 1024; i++ {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		p.used = true
		freeParticles = freeParticles[:l-1]

		p.dieTime = now + 5
		p.color = 0x6f
		p.ramp = float32(rand.Int31() & 3)
		p.origin = vec.Add(origin, randVec(16))
		p.velocity = randVec(256)
		if i&1 == 1 {
			p.typ = ParticleTypeExplode
		} else {
			p.typ = ParticleTypeExplode2
		}
	}
}

func particlesAddExplosion2(origin vec.Vec3, colorStart, colorLength int, now float32) {
	for i := 0; i < 512; i++ {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		p.used = true
		freeParticles = freeParticles[:l-1]

		p.dieTime = now + 0.3
		p.color = colorStart + (i % colorLength)
		p.typ = ParticleTypeBlob

		p.origin = vec.Add(origin, randVec(16))
		p.velocity = randVec(256)
	}
}

func particlesAddBlobExplosion(origin vec.Vec3, now float32) {
	for i := 0; i < 1024; i++ {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		p.used = true
		freeParticles = freeParticles[:l-1]

		p.dieTime = now + 1 + float32(rand.Int31()&8)*0.5

		p.ramp = float32(rand.Int31() & 3)
		p.origin = vec.Add(origin, randVec(16))
		p.velocity = randVec(256)
		if i&1 == 1 {
			p.typ = ParticleTypeBlob
			p.color = 66 + rand.Int()%6
		} else {
			p.typ = ParticleTypeBlob2
			p.color = 150 + rand.Int()%6
		}
	}
}

func particlesAddLavaSplash(origin vec.Vec3, now float32) {
	for i := -16; i < 16; i++ {
		for j := -16; j < 16; j++ {
			l := len(freeParticles)
			if l == 0 {
				return
			}
			p := freeParticles[l-1]
			p.used = true
			freeParticles = freeParticles[:l-1]

			p.dieTime = now + 2 + float32(rand.Int()&31)*0.02
			p.color = 224 + (rand.Int() & 7)
			p.typ = ParticleTypeSlowGrav

			dir := vec.Vec3{
				float32(j*8 + rand.Int()&7),
				float32(i*8 + rand.Int()&7),
				256,
			}

			p.origin = vec.Vec3{
				origin[0] + dir[0],
				origin[1] + dir[1],
				origin[2] + float32(rand.Int()&63),
			}
			vel := float32(50 + rand.Int()&63)
			normalDir := dir.Normalize()
			p.velocity = *normalDir.Scale(vel)
		}
	}
}

func particlesAddTeleportSplash(origin vec.Vec3, now float32) {
	for i := -16; i < 16; i += 4 {
		for j := -16; j < 16; j += 4 {
			for k := -24; j < 32; k += 4 {
				l := len(freeParticles)
				if l == 0 {
					return
				}
				p := freeParticles[l-1]
				p.used = true
				freeParticles = freeParticles[:l-1]

				p.dieTime = now + 0.2 + float32(rand.Int()&7)*0.02
				p.color = 7 + (rand.Int() & 7)
				p.typ = ParticleTypeSlowGrav

				dir := vec.Vec3{
					float32(j * 8),
					float32(i * 8),
					float32(k * 8),
				}

				p.origin = vec.Vec3{
					origin[0] + float32(i+rand.Int()&3),
					origin[1] + float32(j+rand.Int()&3),
					origin[2] + float32(j+rand.Int()&3),
				}
				vel := float32(50 + rand.Int()&63)
				normalDir := dir.Normalize()
				p.velocity = *normalDir.Scale(vel)
			}
		}
	}
}

var (
	rocketTrailTraceCount = 0
)

func particlesAddRocketTrail(start, end vec.Vec3, typ int, now float32) {
	v := vec.Sub(end, start)
	vl := v.Length()
	dec := float32(3)
	if typ >= 128 {
		dec = 1
		typ -= 128
	}

	for vl > 0 {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		*p = particle{}
		p.used = true
		freeParticles = freeParticles[:l-1]

		vl -= dec
		p.velocity = vec.Vec3{}
		p.dieTime = now + 2

		switch typ {
		case 0: // rocket trail
			p.color = (rand.Int() & 3)
			p.typ = ParticleTypeFire
			p.origin = vec.Add(start, randVec(3))
		case 1: //smoke smoke
			p.color = 2 + (rand.Int() & 3)
			p.typ = ParticleTypeFire
			p.origin = vec.Add(start, randVec(3))
		case 2: // blood
			p.color = 67 + (rand.Int() & 3)
			p.typ = ParticleTypeGrav
			p.origin = vec.Add(start, randVec(3))
		case 3, 5: // tracer
			p.color = ((rocketTrailTraceCount & 4) << 1)
			if typ == 3 {
				p.color += 52
			} else {
				p.color += 230
			}
			rocketTrailTraceCount++
			p.dieTime = now + 0.5
			p.typ = ParticleTypeStatic
			p.origin = start
			if rocketTrailTraceCount&1 != 0 {
				p.velocity[0] = 30 * v[0]
				p.velocity[1] = 30 * -v[1]
			} else {
				p.velocity[0] = 30 * -v[1]
				p.velocity[1] = 30 * v[0]
			}
		case 4: // slight blood
			p.color = 67 + (rand.Int() & 3)
			p.typ = ParticleTypeGrav
			p.origin = vec.Add(start, randVec(3))
			vl -= 3 // make it 'slight'
		case 6: // voor trail
			p.color = 9*16 + 8 + (rand.Int() & 3)
			p.typ = ParticleTypeStatic
			p.dieTime = now + 0.3
			p.origin = vec.Vec3{
				start[0] + float32((rand.Int()&15)-8),
				start[1] + float32((rand.Int()&15)-8),
				start[2] + float32((rand.Int()&15)-8),
			}
		}
		start.Add(v)
	}
}

func particlesRun(now float32, lastFrame float32) {
	frameTime := now - lastFrame
	t3 := frameTime * 15
	t2 := frameTime * 10
	t1 := frameTime * 5
	grav := frameTime * cvars.ServerGravity.Value() * 0.05
	dvel := frameTime * 4

	for i := 0; i < len(particles); i++ {
		p := &particles[i]
		if !p.used {
			continue
		}
		if p.dieTime < now {
			p.used = false
			freeParticles = append(freeParticles, p)
			continue
		}
		p.origin.Add(vec.Scale(frameTime, p.velocity))

		switch p.typ {
		case ParticleTypeFire:
			p.ramp += t1
			if p.ramp >= 6 {
				p.dieTime = 0
			} else {
				p.color = ramp3[int(p.ramp)]
			}
			p.velocity[2] += grav

		case ParticleTypeExplode:
			p.ramp += t2
			if p.ramp >= 8 {
				p.dieTime = 0
			} else {
				p.color = ramp1[int(p.ramp)]
			}
			p.velocity.Add(vec.Scale(dvel, p.velocity))
			p.velocity[2] -= grav

		case ParticleTypeExplode2:
			p.ramp += t3
			if p.ramp >= 6 {
				p.dieTime = 0
			} else {
				p.color = ramp2[int(p.ramp)]
			}
			p.velocity.Sub(vec.Scale(frameTime, p.velocity))
			p.velocity[2] -= grav

		case ParticleTypeBlob:
			p.velocity.Add(vec.Scale(dvel, p.velocity))
			p.velocity[2] -= grav

		case ParticleTypeBlob2:
			p.velocity[0] -= p.velocity[0] * dvel
			p.velocity[1] -= p.velocity[1] * dvel
			p.velocity[2] -= grav

		case ParticleTypeGrav, ParticleTypeSlowGrav:
			p.velocity[2] -= grav
		}
	}
}

func particlesDraw() {
	if !cvars.RParticles.Bool() {
		return
	}
	particleDrawer.Draw(particles)
}
