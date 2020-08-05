package quakelib

import "C"

import (
	"fmt"
	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
	"github.com/therjak/goquake/math"
	"log"
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

/*

float *Fog_GetColor(void) {
  static float c[4];
  float f;
  int i;

  if (fade_done > CL_Time()) {
    f = (fade_done - CL_Time()) / fade_time;
    c[0] = f * old_red + (1.0 - f) * fog_red;
    c[1] = f * old_green + (1.0 - f) * fog_green;
    c[2] = f * old_blue + (1.0 - f) * fog_blue;
    c[3] = 1.0;
  } else {
    c[0] = fog_red;
    c[1] = fog_green;
    c[2] = fog_blue;
    c[3] = 1.0;
  }

  // find closest 24-bit RGB value, so solid-colored sky can match the fog
  // perfectly
  for (i = 0; i < 3; i++) c[i] = (float)(Q_rint(c[i] * 255)) / 255.0f;

  return c;
}

float Fog_GetDensity(void) {
  float f;

  if (fade_done > CL_Time()) {
    f = (fade_done - CL_Time()) / fade_time;
    return f * old_density + (1.0 - f) * fog_density;
  } else
    return fog_density;
}

void Fog_SetupFrame(void) {
  glFogfv(GL_FOG_COLOR, Fog_GetColor());
  glFogf(GL_FOG_DENSITY, Fog_GetDensity() / 64.0);
}

void Fog_EnableGFog(void) {
  if (Fog_GetDensity() > 0) glEnable(GL_FOG);
}

void Fog_DisableGFog(void) {
  if (Fog_GetDensity() > 0) glDisable(GL_FOG);
}

void Fog_StartAdditive(void) {
  vec3_t color = {0, 0, 0};

  if (Fog_GetDensity() > 0) glFogfv(GL_FOG_COLOR, color);
}

void Fog_StopAdditive(void) {
  if (Fog_GetDensity() > 0) glFogfv(GL_FOG_COLOR, Fog_GetColor());
}
*/

//export Fog_NewMap
func Fog_NewMap() {
	fog.ParseWorldspawn() // for global fog
}

/*
void Fog_SetupState(void) {
	glFogi(GL_FOG_MODE, GL_EXP2);
}

void Fog_Init(void) {
  Cmd_AddCommand("fog", Fog_FogCommand_f);

  // set up global fog
  fog_density = DEFAULT_DENSITY;
  fog_red = DEFAULT_GRAY;
  fog_green = DEFAULT_GRAY;
  fog_blue = DEFAULT_GRAY;

  Fog_SetupState();
}
*/
