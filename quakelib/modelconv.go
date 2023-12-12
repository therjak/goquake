// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"fmt"
	"log"
	"strings"

	"goquake/bsp"
	"goquake/cvars"
	"goquake/mdl"
	"goquake/model"
	"goquake/spr"
)

var (
	models map[string]model.Model
)

func init() {
	// TODO: at some point this should get cleaned up
	models = make(map[string]model.Model)
}

func ModClearAllGo() {
	// TODO: disable for now as we do not correctly use faiface/mainthread
	// and getting the gc clean up the models would crash
	// return
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

func loadTextures(m model.Model) {
	switch mt := m.(type) {
	case *spr.Model:
		for _, rf := range mt.Data.Frames {
			for _, f := range rf.Frames {
				textureManager.addActiveTexture(f.Texture)
				textureManager.loadRGBA(f.Texture, f.Texture.Data)
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
		mt.UploadBuffer()
	case *bsp.Model:
		for _, t := range mt.Textures {
			if t.SolidSky != nil {
				textureManager.addActiveTexture(t.SolidSky)
				textureManager.loadIndexed(t.SolidSky, t.SolidSky.Data)
			}
			if t.AlphaSky != nil {
				textureManager.addActiveTexture(t.AlphaSky)
				textureManager.loadIndexed(t.AlphaSky, t.AlphaSky.Data)
			}
			if t.Texture != nil {
				textureManager.addActiveTexture(t.Texture)
				textureManager.loadIndexed(t.Texture, t.Texture.Data)
			}
			if t.Fullbright != nil {
				textureManager.addActiveTexture(t.Fullbright)
				textureManager.loadIndexed(t.Fullbright, t.Fullbright.Data)
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
