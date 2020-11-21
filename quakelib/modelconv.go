package quakelib

//#include <stdlib.h>
//#include "gl_model.h"
//void CLPrecacheModel(const char* cn, int i);
import "C"

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/therjak/goquake/model"
)

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

func loadModel(name string) (*model.QModel, error) {
	m, ok := models[name]
	if ok {
		// No need, already loaded
		return m, nil
	}
	mods, err := model.Load(name)
	if err != nil {
		log.Printf("LoadModel err: %v", err)
	}
	for _, m := range mods {
		models[m.Name()] = m
	}
	m, ok = models[name]
	if ok {
		return m, nil
	}
	return nil, fmt.Errorf("LoadModel err: %v", err)
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
		if m.Name() == name {
			cl.worldModel = m
		}
	}
}

func CLPrecacheModel(name string, i int) {
	cn := C.CString(name)
	C.CLPrecacheModel(cn, C.int(i))
	C.free(unsafe.Pointer(cn))
}
