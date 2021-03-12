// SPDX-License-Identifier: GPL-2.0-or-later
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
