package quakelib

import (
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
	particleTexture  uint32
	particleTextures [2]uint32

	particleTextureScaleFactor = float32(1.27)

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
	max := commandline.Particles()
	if max < minParticles {
		max = minParticles
	}
	particles = make([]particle, max)
	freeParticles = make([]*particle, 0, max)
	for i := 0; i < len(particles); i++ {
		freeParticles = append(freeParticles, &particles[i])
	}

	gl.GenTextures(2, &particleTextures[0])
	gl.BindTexture(gl.TEXTURE_2D, particleTextures[0])
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, 64, 64, 0, gl.RED, gl.FLOAT, gl.Ptr(p1TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	gl.BindTexture(gl.TEXTURE_2D, particleTextures[1])
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RED, 2, 2, 0, gl.RED, gl.FLOAT, gl.Ptr(p2TextureData))
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)

	// particleTexture1 = textureManager.loadParticleImage("particle1", 64, 64, p1TextureData)
	// particleTexture2 = textureManager.loadParticleImage("particle2", 2, 2, p2TextureData)

	particleTexture = particleTextures[0]
	particleTextureScaleFactor = float32(1.27)

	//TODO(THERJAK): cvar r_particles callback
	// if r_particles == 1
	// texture1 && factor 1.27
	// if r_particles == 2
	// texture2 && factor 1.0
}

func particlesDeinit() {
	gl.DeleteTextures(2, &particleTextures[0])
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
	grav := frameTime * cvars.ServerGravity.Value() * 0.5
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
	// up := vec.Scale(1.5, qRefreshRect.viewUp)
	// right := vec.Scale(1.5, qRefreshRect.viewRight)

	// vup == qRefreshRect.viewUp
	// vright == qRefreshRect.viewRight
	// vpn == qRefreshRect.viewForward
}
