package quakelib

//extern float fogColor[4];
import "C"

import (
	"fmt"
	"log"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/math"
)

const (
	FogDefaultDensity = 0.0
	FogDefaultGray    = 0.3
)

type QFog struct {
	Density float32
	Red     float32
	Green   float32
	Blue    float32

	OldDensity float32
	OldRed     float32
	OldGreen   float32
	OldBlue    float32

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
			f.OldRed = math.Lerp(f.Red, f.OldRed, frac)
			f.OldGreen = math.Lerp(f.Green, f.OldGreen, frac)
			f.OldBlue = math.Lerp(f.Blue, f.OldBlue, frac)
		} else {
			f.OldDensity = f.Density
			f.OldRed = f.Red
			f.OldGreen = f.Green
			f.OldBlue = f.Blue
		}
	}

	f.Density = density
	f.Red = red
	f.Green = green
	f.Blue = blue
	f.Time = time
	f.Done = cl.time + time
}

func init() {
	cmd.AddCommand("fog", fogCommand)
}

func fogCommand(args []cmd.QArg, _ int) {
	fog.command(args)
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
		conlog.Printf("   \"red\" is \"%f\"\n", f.Red)
		conlog.Printf("   \"green\" is \"%f\"\n", f.Green)
		conlog.Printf("   \"blue\" is \"%f\"\n", f.Blue)
	case 1:
		f.Update(args[0].Float32(), f.Red, f.Green, f.Blue, 0)
	case 2: // TEST
		f.Update(args[0].Float32(), f.Red, f.Green, f.Blue, args[1].Float64())
	case 3:
		f.Update(f.Density, args[0].Float32(), args[1].Float32(), args[2].Float32(), 0)
	case 4:
		f.Update(args[0].Float32(), args[1].Float32(), args[2].Float32(), args[3].Float32(), 0)
	case 5: // TEST
		f.Update(args[0].Float32(), args[1].Float32(), args[2].Float32(), args[3].Float32(), args[4].Float64())
	}
}

func (f *QFog) ParseWorldspawn() {
	f.Density = FogDefaultDensity
	f.Red = FogDefaultGray
	f.Green = FogDefaultGray
	f.Blue = FogDefaultGray
	f.OldDensity = FogDefaultDensity
	f.OldRed = FogDefaultGray
	f.OldGreen = FogDefaultGray
	f.OldBlue = FogDefaultGray
	f.Time = 0
	f.Done = 0

	for _, e := range cl.worldModel.Entities {
		txt, ok := e["fog"]
		if ok {
			var d, r, g, b float32
			i, err := fmt.Sscanf(txt, "%f %f %f %f", &d, &r, &g, &b)
			if err == nil && i == 4 {
				f.Density = d
				f.Red = r
				f.Green = g
				f.Blue = b
			} else {
				log.Printf("Error parsing fog: %v", err)
			}
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
		r = math.Lerp(f.Red, f.OldRed, float32(fade))
		g = math.Lerp(f.Green, f.OldGreen, float32(fade))
		b = math.Lerp(f.Blue, f.OldBlue, float32(fade))
	} else {
		r = f.Red
		g = f.Green
		b = f.Blue
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

//export Fog_NewMap
func Fog_NewMap() {
	fog.ParseWorldspawn() // for global fog
}
