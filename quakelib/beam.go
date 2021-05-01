// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"github.com/chewxy/math32"
	"github.com/therjak/goquake/math"
	"github.com/therjak/goquake/math/vec"
	"github.com/therjak/goquake/model"
	"github.com/therjak/goquake/rand"
)

type beam struct {
	entity  int16
	model   model.Model
	endTime float64
	start   vec.Vec3
	end     vec.Vec3
}

var (
	beams [32]beam
)

func clearBeams() {
	beams = [32]beam{}
}

func parseBeam(name string, ent int16, s, e vec.Vec3) {
	m, err := loadModel(name)
	if err != nil {
		return
	}

	set := func(b *beam) {
		b.entity = ent
		b.model = m
		b.endTime = cl.time + 0.2
		b.start = s
		b.end = e
	}

	for i := range beams {
		b := &beams[i]
		if b.entity == ent {
			set(b)
			return
		}
	}
	for i := range beams {
		b := &beams[i]
		if b.model == nil || b.endTime < cl.time {
			set(b)
			return
		}
	}
}

func (c *Client) updateTempEntities() {
	ClearTempEntities()
	rg := rand.New(uint32(c.time * 1000)) // to freeze while paused

	for i := range beams {
		b := &beams[i]
		if b.model == nil || b.endTime < c.time {
			continue
		}
		// if coming from the player
		if int(b.entity) == c.viewentity {
			b.start = c.Entity().Origin
		}

		yaw := float32(0)
		var pitch float32
		dist := vec.Sub(b.end, b.start)
		if dist[0] == 0 && dist[1] == 0 {
			if dist[2] > 0 {
				pitch = 90
			} else {
				pitch = 270
			}
		} else {
			yaw = math.Round(math32.Atan2(dist[1], dist[0]) * 180 / math32.Pi)
			if yaw < 0 {
				yaw += 360
			}
			forward := math.Sqrt(dist[0]*dist[0] + dist[1]*dist[1])
			pitch = math.Round(math32.Atan2(dist[2], forward) * 180 / math32.Pi)
			if pitch < 0 {
				pitch += 360
			}
		}

		origin := b.start
		d := dist.Length()
		for d > 0 {
			e := c.NewTempEntity()
			if e == nil {
				return
			}
			e.Origin = origin
			e.Model = b.model
			e.Angles = vec.Vec3{pitch, yaw, math32.Mod(rg.Float32(), 360)}
			origin.Add(vec.Scale(30, dist))
			d -= 30
		}
	}
}
