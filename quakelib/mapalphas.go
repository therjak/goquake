// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"goquake/bsp"
	"goquake/cvar"
	"goquake/cvars"
	"strconv"
)

type alphas struct {
	water float32
	lava  float32
	tele  float32
	slime float32
	sky   float32
}

var (
	// used for transparent fluid drawing (drawTextureChainsWater)
	mapAlphas alphas
)

func init() {
	cvars.RWaterAlpha.SetCallback(func(cv *cvar.Cvar) {
		mapAlphas.water = cv.Value()
	})
	cvars.RLavaAlpha.SetCallback(func(cv *cvar.Cvar) {
		mapAlphas.lava = cv.Value()
	})
	cvars.RTeleAlpha.SetCallback(func(cv *cvar.Cvar) {
		mapAlphas.tele = cv.Value()
	})
	cvars.RSlimeAlpha.SetCallback(func(cv *cvar.Cvar) {
		mapAlphas.slime = cv.Value()
	})
	cvars.RSkyAlpha.SetCallback(func(cv *cvar.Cvar) {
		mapAlphas.sky = cv.Value()
	})
	// handle RWaterQuality ?
}

func handleMapAlphas(e *bsp.Entity) {
	mapAlphas.water = cvars.RWaterAlpha.Value()
	mapAlphas.lava = cvars.RLavaAlpha.Value()
	mapAlphas.tele = cvars.RTeleAlpha.Value()
	mapAlphas.slime = cvars.RSlimeAlpha.Value()
	mapAlphas.sky = cvars.RSkyAlpha.Value()

	atof := func(s string) float32 {
		v, err := strconv.ParseFloat(s, 32)
		if err != nil {
			return 0
		}
		return float32(v)
	}
	if v, ok := e.Property("wateralpha"); ok {
		mapAlphas.water = atof(v)
	}
	if v, ok := e.Property("_wateralpha"); ok {
		mapAlphas.water = atof(v)
	}
	if v, ok := e.Property("lavaalpha"); ok {
		mapAlphas.lava = atof(v)
	}
	if v, ok := e.Property("_lavaalpha"); ok {
		mapAlphas.lava = atof(v)
	}
	if v, ok := e.Property("telealpha"); ok {
		mapAlphas.tele = atof(v)
	}
	if v, ok := e.Property("_telealpha"); ok {
		mapAlphas.tele = atof(v)
	}
	if v, ok := e.Property("slimealpha"); ok {
		mapAlphas.slime = atof(v)
	}
	if v, ok := e.Property("_slimealpha"); ok {
		mapAlphas.slime = atof(v)
	}
	if v, ok := e.Property("skyalpha"); ok {
		mapAlphas.sky = atof(v)
	}
}
