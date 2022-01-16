// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//#include <stdlib.h>
//#include "gl_model.h"
//void CLPrecacheModel(const char* cn, int i);
import "C"

import (
	"fmt"
	"log"
	"strings"
	"unsafe"

	"goquake/bsp"
	"goquake/cvars"
	"goquake/mdl"
	"goquake/model"
	"goquake/spr"
	"goquake/texture"
)

var (
	models map[string]model.Model
)

func init() {
	// TODO: at some point this should get cleaned up
	models = make(map[string]model.Model)
}

//export ModClearAllGo
func ModClearAllGo() {
	// TODO: disable for now as we do not correctly use faiface/mainthread
	// and getting the gc clean up the models would crash
	return
	// models = make(map[string]model.Model)
}

func loadModel(name string) (model.Model, error) {
	m, ok := models[name]
	if ok {
		// No need, already loaded
		return m, nil
	}
	mods, err := model.Load(name)
	if err != nil {
		log.Printf("LoadModel err: %v", err)
		return nil, err
	}
	for _, m := range mods {
		models[m.Name()] = m
		setExtraFlags(m)
		loadTextures(m)
	}
	m, ok = models[name]
	if ok {
		return m, nil
	}
	return nil, fmt.Errorf("LoadModel err: %v", err)
}

func CLPrecacheModel(name string, i int) {
	cn := C.CString(name)
	C.CLPrecacheModel(cn, C.int(i))
	C.free(unsafe.Pointer(cn))
}

func setExtraFlags(m model.Model) {
	switch mt := m.(type) {
	case *mdl.Model:
		if strings.Contains(cvars.RNoLerpList.String(), mt.Name()) {
			mt.AddFlag(mdl.NoLerp)
		}
		if strings.Contains(cvars.RNoShadowList.String(), mt.Name()) {
			mt.AddFlag(mdl.NoShadow)
		}
		if strings.Contains(cvars.RFullBrightList.String(), mt.Name()) {
			mt.AddFlag(mdl.FullBrightHack)
		}
	}
}

func checkFullbrights(data []byte) bool {
	for _, d := range data {
		if d > 223 {
			return true
		}
	}
	return false
}

func loadTextures(m model.Model) {
	switch mt := m.(type) {
	case *spr.Model:
		for _, rf := range mt.Data.Frames {
			for _, f := range rf.Frames {
				textureManager.addActiveTexture(f.Texture)
				textureManager.loadIndexed(f.Texture, f.Texture.Data)
			}
		}
	case *mdl.Model:
		for _, t := range mt.Textures {
			for _, st := range t {
				textureManager.addActiveTexture(st)
				textureManager.loadIndexed(st, st.Data)
			}
		}
		for _, t := range mt.FBTextures {
			for _, st := range t {
				textureManager.addActiveTexture(st)
				textureManager.loadIndexed(st, st.Data)
			}
		}
	case *bsp.Model:
		for _, t := range mt.Textures {
			// THIS SHOULD MOSTLY MOVE INTO bsp/loader
			// Warp is missing

			// we have bsp texture data in t.Data []byte
			// t.Texture, t.Fullbright and t.Warp are still nil

			// Bad hack
			if strings.HasPrefix(t.Name(), "sky") {
				// it is currently handled in CL_ParseServerInfo but shouldn't
				continue
			}
			if len(t.Data) == 0 {
				continue
			}

			var extra texture.TexPref
			if strings.HasPrefix(t.Name(), "{") {
				extra = texture.TexPrefAlpha
			}

			if checkFullbrights(t.Data) {
				tName := fmt.Sprintf("%s:%s", mt.Name(), t.Name())
				t.Texture = texture.NewTexture(
					int32(t.Width),
					int32(t.Height),
					texture.TexPrefMipMap|texture.TexPrefNoBright|extra,
					tName,
					texture.ColorTypeIndexed,
					t.Data)
				textureManager.addActiveTexture(t.Texture)
				textureManager.loadIndexed(t.Texture, t.Texture.Data)
				fbName := fmt.Sprintf("%s:%s_glow", mt.Name(), t.Name())
				t.Fullbright = texture.NewTexture(
					int32(t.Width),
					int32(t.Height),
					texture.TexPrefMipMap|texture.TexPrefFullBright|extra,
					fbName,
					texture.ColorTypeIndexed,
					t.Data)
				textureManager.addActiveTexture(t.Fullbright)
				textureManager.loadIndexed(t.Fullbright, t.Fullbright.Data)
			} else {
				t.Texture = texture.NewTexture(
					int32(t.Width),
					int32(t.Height),
					texture.TexPrefMipMap|extra,
					t.Name(),
					texture.ColorTypeIndexed,
					t.Data)
				textureManager.addActiveTexture(t.Texture)
				textureManager.loadIndexed(t.Texture, t.Texture.Data)
			}
		}
		for _, s := range mt.Surfaces {
			if s.LightmapTexture != nil {
				textureManager.addActiveTexture(s.LightmapTexture)
				textureManager.loadLightMap(s.LightmapTexture)
			}
		}
	}
}
