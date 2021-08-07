// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//struct cvar_s;
//typedef void (*cvarcallback_t)(struct cvar_s*);
//void CallCvarCallback(int id, cvarcallback_t func);
import "C"
import (
	"log"

	"goquake/cvar"
)

//export CvarGetValue
func CvarGetValue(id C.int) C.float {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return 0
	}
	return C.float(cv.Value())
}

//export CvarGetString
func CvarGetString(id C.int) *C.char {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return nil
	}
	return C.CString(cv.String())
}

//export CvarGetID
func CvarGetID(name *C.char) C.int {
	cv, ok := cvar.Get(C.GoString(name))
	if !ok {
		return -1
	}
	return C.int(cv.ID())
}

//export CvarSetCallback
func CvarSetCallback(id C.int, f C.cvarcallback_t) {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return
	}
	cv.SetCallback(func(_ *cvar.Cvar) {
		C.CallCvarCallback(id, f)
	})
}
