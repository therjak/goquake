package bsp

import (
	"goquake/math/vec"
	"goquake/texture"

	"github.com/chewxy/math32"
)

var (
	blockLights [128 * 128 * 3]uint32
)

func (s *Surface) createSurfaceLightmap() {
	smax := (s.extents[S] >> 4) + 1
	tmax := (s.extents[T] >> 4) + 1
	s.LightmapData = make([]byte, smax*tmax*4)
	s.LightmapTexture = texture.NewTexture(int32(smax), int32(tmax),
		texture.TexPrefLinear|texture.TexPrefNoPicMip,
		s.lightmapName, texture.ColorTypeLightmap, s.LightmapData)
	// needs to be done but not in bsp:
	// textureManager.addActiveTexture(s.LightmapTexture)
	// textureManager.loadLightmap(s.LightmapTexture, s.LightmapTextureData)
}

func clampColor(c uint32) byte {
	if c > 255 {
		return 255
	}
	return byte(c)
}

type DynamicLight interface {
	Origin() vec.Vec3
	Radius() float32
	MinLight() float32 // dont' add when contributing less
	Color() vec.Vec3
}

func (s *Surface) BuildLightMap(dynamicStyles LightStyles, frame int, lights []DynamicLight, overbright bool) {
	smax := (s.extents[S] >> 4) + 1
	tmax := (s.extents[T] >> 4) + 1
	size := smax * tmax
	lightmap := s.LightSamples
	for b := range blockLights {
		blockLights[b] = 0
	}
	if len(lightmap) != 0 {
		for m, style := range s.Styles {
			if style == 0xff {
				break
			}
			scale := dynamicStyles[style]
			s.CachedLight[m] = scale // 8.8 fraction
			for i := 0; i < size*3; i++ {
				blockLights[i] += uint32(lightmap[i]) * uint32(scale)
			}
		}
	}
	if s.DLightFrame == frame {
		s.addDynamicLights(lights)
	}

	dst := 0
	src := 0
	var r, g, b uint32
	for i := 0; i < tmax; i++ {
		for j := 0; j < smax; j++ {
			if overbright {
				r = blockLights[src] >> 8
				src++
				g = blockLights[src] >> 8
				src++
				b = blockLights[src] >> 8
				src++
			} else {
				r = blockLights[src] >> 7
				src++
				g = blockLights[src] >> 7
				src++
				b = blockLights[src] >> 7
				src++
			}
			s.LightmapData[dst] = clampColor(r)
			dst++
			s.LightmapData[dst] = clampColor(g)
			dst++
			s.LightmapData[dst] = clampColor(b)
			dst++
			s.LightmapData[dst] = 255
			dst++
		}
	}
}

func (s *Surface) addDynamicLights(lights []DynamicLight) {
	smax := (s.extents[S] >> 4) + 1
	tmax := (s.extents[T] >> 4) + 1
	tex := s.TexInfo
	for i, l := range lights {
		if len(s.DLightBits) >= i {
			break
		}
		if !s.DLightBits[i] {
			continue
		}
		rad := l.Radius()
		dist := vec.Dot(l.Origin(), s.Plane.Normal) + s.Plane.Dist
		rad -= math32.Abs(dist)
		minLight := l.MinLight()
		if rad < minLight {
			continue
		}
		minLight = rad - minLight
		impact := vec.Sub(l.Origin(), vec.Scale(dist, s.Plane.Normal))
		var local [2]float32
		local[S] = vec.Dot(impact, tex.Vecs[S].Pos) + tex.Vecs[S].Offset
		local[T] = vec.Dot(impact, tex.Vecs[T].Pos) + tex.Vecs[T].Offset
		local[S] -= float32(s.textureMins[S])
		local[T] -= float32(s.textureMins[T])
		// 542
		r := l.Color()[0] * 256
		g := l.Color()[1] * 256
		b := l.Color()[2] * 256
		bidx := 0
		for t := 0; t < tmax; t++ {
			td := math32.Abs(local[T] - float32(t*16))
			for s := 0; s < smax; s++ {
				sd := math32.Abs(local[S] - float32(s*16))
				var dist float32
				if sd > td {
					dist = sd + td/2
				} else {
					dist = td + sd/2
				}
				if dist < minLight {
					bnes := rad - dist
					blockLights[bidx] += uint32(bnes * r)
					blockLights[bidx+1] += uint32(bnes * g)
					blockLights[bidx+2] += uint32(bnes * b)
				}
				bidx += 3
			}
		}
	}
}
