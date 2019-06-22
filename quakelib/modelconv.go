package quakelib

//#include "gl_model.h"
import "C"

import (
	"fmt"
	"log"
	"quake/model"
)

//export SVSetModelByName
func SVSetModelByName(n *C.char, idx int, localModel int) {
	name := C.GoString(n)
	nm := func() *model.QModel {
		cm, ok := models[name]
		if ok {
			return cm
		}
		log.Printf("TODO!!! SetModel: %d, %s new", idx, name)
		return nil
	}()
	if int(idx) == len(sv.models) {
		sv.models = append(sv.models, nm)
	} else {
		sv.models[int(idx)] = nm
	}
}

var (
	models map[string]*model.QModel
)

func init() {
	// TODO: at some point this should get cleaned up
	models = make(map[string]*model.QModel)
}

//export ModClearAllGo
func ModClearAllGo() {
	models = make(map[string]*model.QModel)
}

//export LoadModelGo
func LoadModelGo(name *C.char) {
	loadModel(C.GoString(name))
}

func loadModel(name string) {
	_, ok := models[name]
	if ok {
		// No need, already loaded
		return
	}
	mods, err := model.Load(name)
	if err != nil {
		log.Printf("LoadModel err: %v", err)
	}
	for _, m := range mods {
		models[m.Name] = m
	}
}

//export EDLoadEntitiesGo
func EDLoadEntitiesGo() {
	loadEntities(sv.worldModel.Entities)
}

//export CLSetWorldModel
func CLSetWorldModel(m *C.qmodel_t) {
	name := C.GoString(&m.name[0])
	cm, ok := models[name]
	if ok {
		cl.worldModel = cm
		return
	}
	mods, err := model.Load(name)
	if err != nil {
		log.Printf("CL - LoadModel err: %v", err)
	}
	for _, m := range mods {
		if m.Name == name {
			cl.worldModel = m
		}
	}
}

//export SVSetWorldModelByName
func SVSetWorldModelByName(n *C.char) {
	// This has already a lot of SV_SpawnServer
	name := C.GoString(n)
	log.Printf("New world: %s", name)
	sv.worldModel = nil
	sv.modelPrecache = sv.modelPrecache[:0]
	sv.soundPrecache = sv.soundPrecache[:0]
	sv.models = append(sv.models, nil)
	sv.models = sv.models[:1]
	log.Printf("New world starts with %d models", len(sv.models))
	cm, ok := models[name]
	if ok {
		sv.worldModel = cm
	} else {
		log.Fatalf("Missing the world model")
		return
	}
	sv.modelPrecache = append(sv.modelPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	sv.modelPrecache = append(sv.modelPrecache, name)
	sv.soundPrecache = append(sv.soundPrecache, string([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	sv.models = append(sv.models, sv.worldModel)
	for i := 1; i < len(sv.worldModel.Submodels); i++ {
		nn := fmt.Sprintf("*%d", i)
		nm, ok := models[nn]
		if !ok {
			log.Printf("Missing model %d", i)
			continue
		}
		sv.modelPrecache = append(sv.modelPrecache, nn)
		sv.models = append(sv.models, nm)
	}

	clearWorld()
}
