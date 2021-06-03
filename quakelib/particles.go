// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/commandline"
	"github.com/therjak/goquake/cvar"
	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/glh"
	"github.com/therjak/goquake/math/vec"

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

var (
	particleDrawer *qParticleDrawer
)

type qParticleDrawer struct {
	vao        *glh.VertexArray
	vbo        *glh.Buffer
	prog       *glh.Program
	projection int32
	modelview  int32

	texture            glh.Texture
	textures           [2]glh.Texture
	textureScaleFactor float32

	// to reduce the number of allocations
	vertices []float32
}

func newParticleDrawProgram() (*glh.Program, error) {
	return glh.NewProgram(vertexSourceParticleDrawer, fragmentSourceParticleDrawer)
}

func newParticleDrawer() *qParticleDrawer {
	d := &qParticleDrawer{}
	d.vao = glh.NewVertexArray()
	d.vbo = glh.NewBuffer(glh.ArrayBuffer)
	var err error
	d.prog, err = newParticleDrawProgram()
	if err != nil {
		Error(err.Error())
	}
	d.projection = d.prog.GetUniformLocation("projection")
	d.modelview = d.prog.GetUniformLocation("modelview")

	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

	d.textures[0] = glh.NewTexture2D()
	d.textures[1] = glh.NewTexture2D()
	d.textures[0].Bind()
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R16F, 64, 64, 0, gl.RED, gl.FLOAT, gl.Ptr(p1TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	d.textures[1].Bind()
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.R16F, 2, 2, 0, gl.RED, gl.FLOAT, gl.Ptr(p2TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	d.texture = d.textures[0]
	d.textureScaleFactor = float32(1.27)
	return d
}

func (d *qParticleDrawer) Draw(ps []particle) {
	up := vec.Scale(1.5, qRefreshRect.viewUp)
	right := vec.Scale(1.5, qRefreshRect.viewRight)

	fwd := qRefreshRect.viewForward
	origin := qRefreshRect.viewOrg

	d.vertices = d.vertices[0:0]
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

		// x, y, z, tx, ty, r, g, b
		d.vertices = append(d.vertices,
			o[0], o[1], o[2], 0, 0, c[0], c[1], c[2],
			(o[0] + u[0]), (o[1] + u[1]), (o[2] + u[2]), 1, 0, c[0], c[1], c[2],
			(o[0] + r[0]), (o[1] + r[1]), (o[2] + r[2]), 0, 1, c[0], c[1], c[2])
	}
	if len(d.vertices) == 0 {
		return
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)
	defer gl.Disable(gl.BLEND)

	gl.DepthMask(false)
	defer gl.DepthMask(true)

	d.prog.Use()
	d.vao.Bind()

	d.vbo.Bind()
	d.vbo.SetData(4*len(d.vertices), gl.Ptr(d.vertices))

	gl.EnableVertexAttribArray(0)
	defer gl.DisableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	defer gl.DisableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2)
	defer gl.DisableVertexAttribArray(2)

	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(0))
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, 4*8, gl.PtrOffset(3*4))
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, 4*8, gl.PtrOffset(5*4))

	view.projection.SetAsUniform(d.projection)
	view.modelView.SetAsUniform(d.modelview)

	d.texture.Bind()

	gl.DrawArrays(gl.TRIANGLES, 0, int32(numVert))

	// We bound a texture without the texture manager.
	// Tell the texture manager that its cache is invalid.
	textureManager.ClearBindings()
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
			cRand.Float32(),
			cRand.Float32(),
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

}

func CreateParticleDrawer() {
	particleDrawer = newParticleDrawer()
}

func init() {
	cvars.RParticles.SetCallback(func(cv *cvar.Cvar) {
		d := particleDrawer
		switch int(cv.Value()) {
		case 1:
			d.texture = d.textures[0]
			d.textureScaleFactor = float32(1.27)
		case 2:
			d.texture = d.textures[1]
			d.textureScaleFactor = float32(1.0)
		}
	})
}

func particlesDeinit() {
	// to be clean this could be run on ui shutdown
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
func particlesReadPointFile(_ []cmd.QArg, _ int) {
	// This is a file to debug maps. They should not be part of a pak.
	// It's to show ingame where the map has holes.
	log.Printf("pointfile")
	// TODO(THERJAK):
	// p.dieTime = 99999 // that is > 27h
	// p.typ = ParticleTypeStatic
	// p.velocity = vec.Vec3{}
	//
	//
	/*
	  FILE *f;
	  vec3_t org;
	  int r;
	  int c;
	  particle_t *p;
	  char name[MAX_QPATH];

	  if (CLS_GetState() != ca_connected) return;  // need an active map.

	  q_snprintf(name, sizeof(name), "maps/%s.pts", cl.mapname);

	  f = fopen(name, "r");
	  if (!f) {
	    Con_Printf("couldn't open %s\n", name);
	    return;
	  }

	  Con_Printf("Reading %s...\n", name);
	  c = 0;
	  for (;;) {
	    r = fscanf(f, "%f %f %f\n", &org[0], &org[1], &org[2]);
	    if (r != 3) break;
	    c++;

	    if (!free_particles) {
	      Con_Printf("Not enough free particles\n");
	      break;
	    }
	    p = free_particles;
	    free_particles = p->next;
	    p->next = active_particles;
	    active_particles = p;

	    p->die = 99999;
	    p->color = (-c) & 15;
	    p->type = pt_static;
	    VectorCopy(vec3_origin, p->vel);
	    VectorCopy(org, p->org);
	  }

	  fclose(f);
	  Con_Printf("%i points read\n", c);
	*/
}

func init() {
	cmd.AddCommand("pointfile", particlesReadPointFile)
}

// randVec returns a randomized vector with radius at most r
func randVec(r int) vec.Vec3 {
	d := 2 * r
	return vec.Vec3{
		float32(cRand.Intn(d) - r),
		float32(cRand.Intn(d) - r),
		float32(cRand.Intn(d) - r),
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
		p.ramp = float32(cRand.Uint32n(4))
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

		p.dieTime = now + 1 + float32(cRand.Uint32n(8))*0.5

		p.ramp = float32(cRand.Uint32n(4))
		p.origin = vec.Add(origin, randVec(16))
		p.velocity = randVec(256)
		if i&1 == 1 {
			p.typ = ParticleTypeBlob
			p.color = 66 + cRand.Intn(6)
		} else {
			p.typ = ParticleTypeBlob2
			p.color = 150 + cRand.Intn(6)
		}
	}
}

func particlesRunEffect(origin, dir vec.Vec3, color, count int, now float32) {
	for i := 0; i < count; i++ {
		l := len(freeParticles)
		if l == 0 {
			return
		}
		p := freeParticles[l-1]
		p.used = true
		freeParticles = freeParticles[:l-1]

		if count == 1024 { // rocket explosion, dead?
			p.dieTime = now + 5
			p.color = ramp1[0]
			p.ramp = float32(cRand.Uint32n(4))
			p.origin = vec.Add(origin, randVec(8))
			p.velocity = randVec(256)
			if i&1 != 0 {
				p.typ = ParticleTypeExplode
			} else {
				p.typ = ParticleTypeExplode2
			}
		} else {
			p.dieTime = now + 0.1*float32((cRand.Uint32n(5)))
			p.color = (color &^ 7) + cRand.Intn(8)
			p.typ = ParticleTypeSlowGrav
			p.origin = vec.Add(origin, randVec(8))
			p.velocity = vec.Scale(15, dir)
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

			p.dieTime = now + 2 + float32(cRand.Uint32n(32))*0.02
			p.color = 224 + cRand.Intn(8)
			p.typ = ParticleTypeSlowGrav

			dir := vec.Vec3{
				float32(j*8 + cRand.Intn(8)),
				float32(i*8 + cRand.Intn(8)),
				256,
			}

			p.origin = vec.Vec3{
				origin[0] + dir[0],
				origin[1] + dir[1],
				origin[2] + float32(cRand.Uint32n(64)),
			}
			vel := float32(50 + cRand.Uint32n(64))
			normalDir := dir.Normalize()
			p.velocity = *normalDir.Scale(vel)
		}
	}
}

var (
	teleportSplashIs = []int{-16, -12, -8, -4, 0, 4, 8, 12, 16}
	teleportSplashJs = []int{-16, -12, -8, -4, 0, 4, 8, 12, 16}
	teleportSplashKs = []int{-24, -20, -16, -12, -8, -4, 0, 4, 8, 12, 16, 20, 24, 28, 32}
)

func particlesAddTeleportSplash(origin vec.Vec3, now float32) {
	for _, i := range teleportSplashIs {
		for _, j := range teleportSplashJs {
			for _, k := range teleportSplashKs {
				l := len(freeParticles)
				if l == 0 {
					return
				}
				p := freeParticles[l-1]
				*p = particle{}
				p.used = true
				freeParticles = freeParticles[:l-1]

				p.dieTime = now + 0.2 + float32(cRand.Uint32n(8))*0.02
				p.color = 7 + cRand.Intn(8)
				p.typ = ParticleTypeSlowGrav

				dir := vec.Vec3{
					float32(j * 8),
					float32(i * 8),
					float32(k * 8),
				}

				p.origin = vec.Vec3{
					origin[0] + float32(i+cRand.Intn(4)),
					origin[1] + float32(j+cRand.Intn(4)),
					origin[2] + float32(j+cRand.Intn(4)),
				}
				vel := float32(50 + cRand.Uint32n(64))
				normalDir := dir.Normalize()
				p.velocity = vec.Scale(vel, normalDir)
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
	if vl != 0 {
		v.Scale(1 / vl)
	}
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
		p.used = true
		freeParticles = freeParticles[:l-1]

		vl -= dec
		p.velocity = vec.Vec3{}
		p.dieTime = now + 2

		switch typ {
		case 0: // rocket trail
			p.ramp = float32(cRand.Uint32n(4))
			p.color = ramp3[int(p.ramp)]
			p.typ = ParticleTypeFire
			p.origin = vec.Add(start, randVec(3))
		case 1: //smoke smoke
			p.ramp = float32(cRand.Uint32n(4) + 2)
			p.color = ramp3[int(p.ramp)]
			p.typ = ParticleTypeFire
			p.origin = vec.Add(start, randVec(3))
		case 2: // blood
			p.color = 67 + cRand.Intn(4)
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
			p.color = 67 + (cRand.Intn(4))
			p.typ = ParticleTypeGrav
			p.origin = vec.Add(start, randVec(3))
			vl -= 3 // make it 'slight'
		case 6: // voor trail
			p.color = 9*16 + 8 + (cRand.Intn(4))
			p.typ = ParticleTypeStatic
			p.dieTime = now + 0.3
			p.origin = vec.Vec3{
				start[0] + float32(cRand.Intn(16)-8),
				start[1] + float32(cRand.Intn(16)-8),
				start[2] + float32(cRand.Intn(16)-8),
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
