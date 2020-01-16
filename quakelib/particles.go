package quakelib

import (
	"math/rand"
	"quake/commandline"
	"quake/cvars"
	"quake/math/vec"

	"github.com/chewxy/math32"
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

	//TODO(THERJAK): particleTextures
	//TODO(THERJAK): cvar r_particles callback
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
		p.origin = vec.Vec3{
			origin[0] + float32((rand.Int31()%32)-16),
			origin[1] + float32((rand.Int31()%32)-16),
			origin[2] + float32((rand.Int31()%32)-16),
		}
		p.velocity = vec.Vec3{
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
		}
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

		p.origin = vec.Vec3{
			origin[0] + float32((rand.Int31()%32)-16),
			origin[1] + float32((rand.Int31()%32)-16),
			origin[2] + float32((rand.Int31()%32)-16),
		}
		p.velocity = vec.Vec3{
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
		}
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
		p.origin = vec.Vec3{
			origin[0] + float32((rand.Int31()%32)-16),
			origin[1] + float32((rand.Int31()%32)-16),
			origin[2] + float32((rand.Int31()%32)-16),
		}
		p.velocity = vec.Vec3{
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
			float32((rand.Int31() % 512) - 256),
		}
		if i&1 == 1 {
			p.typ = ParticleTypeBlob
			p.color = 66 + rand.Int()%6
		} else {
			p.typ = ParticleTypeBlob2
			p.color = 150 + rand.Int()%6
		}
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
