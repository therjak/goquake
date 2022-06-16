// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

//extern float fogColor[4];
import "C"

import (
	"fmt"
	"log"

	"goquake/bsp"
	"goquake/cmd"
	"goquake/conlog"
	"goquake/math"
)

const (
	FogDefaultDensity = 0.0
	FogDefaultGray    = 0.3
)

type QFog struct {
	Density float32
	Color   Color

	OldDensity float32
	OldColor   Color

	Time float64 // duration of fade
	Done float64 // time when fade will be done
}

var (
	fog QFog
)

func (f *QFog) Update(density, red, green, blue float32, time float64) {
	density = math.Clamp32(0, density, 1)
	red = math.Clamp32(0, red, 1)
	green = math.Clamp32(0, green, 1)
	blue = math.Clamp32(0, blue, 1)
	// save previous settings for fade
	if time > 0 {
		// check for a fade in progress
		if f.Done > cl.time {
			frac := float32((f.Done - cl.time) / f.Time)
			f.OldDensity = math.Lerp(f.Density, f.OldDensity, frac)
			f.OldColor.R = math.Lerp(f.Color.R, f.OldColor.R, frac)
			f.OldColor.G = math.Lerp(f.Color.G, f.OldColor.G, frac)
			f.OldColor.B = math.Lerp(f.Color.B, f.OldColor.B, frac)
		} else {
			f.OldDensity = f.Density
			f.OldColor.R = f.Color.R
			f.OldColor.G = f.Color.G
			f.OldColor.B = f.Color.B
		}
	}

	f.Density = density
	f.Color.R = red
	f.Color.G = green
	f.Color.B = blue
	f.Time = time
	f.Done = cl.time + time
}

func init() {
	addCommand("fog", fogCommand)
}

func fogCommand(args []cmd.QArg, _ int) error {
	fog.command(args)
	return nil
}

func (f *QFog) command(args []cmd.QArg) {
	switch len(args) {
	default:
		conlog.Printf("usage:\n")
		conlog.Printf("   fog <density>\n")
		conlog.Printf("   fog <red> <green> <blue>\n")
		conlog.Printf("   fog <density> <red> <green> <blue>\n")
		conlog.Printf("current values:\n")
		conlog.Printf("   \"density\" is \"%f\"\n", f.Density)
		conlog.Printf("   \"red\" is \"%f\"\n", f.Color.R)
		conlog.Printf("   \"green\" is \"%f\"\n", f.Color.G)
		conlog.Printf("   \"blue\" is \"%f\"\n", f.Color.B)
	case 1:
		f.Update(args[0].Float32(), f.Color.R, f.Color.G, f.Color.B, 0)
	case 2: // TEST
		f.Update(args[0].Float32(), f.Color.R, f.Color.G, f.Color.B, args[1].Float64())
	case 3:
		f.Update(f.Density, args[0].Float32(), args[1].Float32(), args[2].Float32(), 0)
	case 4:
		f.Update(args[0].Float32(), args[1].Float32(), args[2].Float32(), args[3].Float32(), 0)
	case 5: // TEST
		f.Update(args[0].Float32(), args[1].Float32(), args[2].Float32(), args[3].Float32(), args[4].Float64())
	}
}

func (f *QFog) parseWorldspawn(worldspawn *bsp.Entity) {
	f.Density = FogDefaultDensity
	f.Color.R = FogDefaultGray
	f.Color.G = FogDefaultGray
	f.Color.B = FogDefaultGray
	f.OldDensity = FogDefaultDensity
	f.OldColor.R = FogDefaultGray
	f.OldColor.G = FogDefaultGray
	f.OldColor.B = FogDefaultGray
	f.Time = 0
	f.Done = 0

	if txt, ok := worldspawn.Property("fog"); ok {
		var d, r, g, b float32
		i, err := fmt.Sscanf(txt, "%f %f %f %f", &d, &r, &g, &b)
		if err == nil && i == 4 {
			f.Density = d
			f.Color.R = r
			f.Color.G = g
			f.Color.B = b
		} else {
			log.Printf("Error parsing fog: %v", err)
		}
	}
}

//export Fog_GetColor
func Fog_GetColor() *C.float {
	r, g, b, a := fog.GetColor()
	C.fogColor[0] = C.float(r)
	C.fogColor[1] = C.float(g)
	C.fogColor[2] = C.float(b)
	C.fogColor[3] = C.float(a)
	return &C.fogColor[0]
}

func (f *QFog) GetColor() (float32, float32, float32, float32) {
	var r, g, b float32
	if f.Done > cl.time {
		fade := (f.Done - cl.time) / f.Time
		r = math.Lerp(f.Color.R, f.OldColor.R, float32(fade))
		g = math.Lerp(f.Color.G, f.OldColor.G, float32(fade))
		b = math.Lerp(f.Color.B, f.OldColor.B, float32(fade))
	} else {
		r = f.Color.R
		g = f.Color.G
		b = f.Color.B
	}

	// find closest 24-bit RGB value, so solid-colored sky can match the fog
	// perfectly
	r = math.Round(r*255) / 255
	g = math.Round(g*255) / 255
	b = math.Round(b*255) / 255

	return r, g, b, 1
}

//export Fog_GetDensity
func Fog_GetDensity() float32 {
	return fog.GetDensity()
}

func (f *QFog) GetDensity() float32 {
	if f.Done > cl.time {
		fade := (f.Done - cl.time) / f.Time
		return math.Lerp(f.Density, f.OldDensity, float32(fade))
	}
	return f.Density
}
