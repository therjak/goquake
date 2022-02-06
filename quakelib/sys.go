// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import "C"

import (
	"log"
	"runtime/debug"
)

var (
/*
	//int, what is this for?
	memSize = 0 //?
	//void* what is this for?
	membase = null
*/
)

func Must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

//export ScreenDisabled
func ScreenDisabled() C.int {
	if screen.disabled {
		return 1
	}
	return 0
}

//export SetScreenDisabled
func SetScreenDisabled(b C.int) {
	screen.disabled = (b != 0)
}

func Error(format string, v ...interface{}) {
	debug.PrintStack()
	host.Shutdown()
	log.Fatalf(format, v...)
}

//export Go_Error
func Go_Error(c *C.char) {
	Error(C.GoString(c))
}

//export Go_Error_S
func Go_Error_S(c *C.char, s *C.char) {
	Error(C.GoString(c), C.GoString(s))
}

//export Go_Error_I
func Go_Error_I(c *C.char, i C.int) {
	Error(C.GoString(c), int(i))
}

//export Sys_Quit
func Sys_Quit() {
	host.Shutdown()
	quitChan <- true
}

//export Sys_Print
func Sys_Print(c *C.char) {
	log.Print(C.GoString(c))
}

//export Sys_Print_S
func Sys_Print_S(c *C.char, s *C.char) {
	log.Printf(C.GoString(c), C.GoString(s))
}

//export Sys_Print_I
func Sys_Print_I(c *C.char, i C.int) {
	log.Printf(C.GoString(c), int(i))
}

//export Sys_Print_F
func Sys_Print_F(c *C.char, f C.float) {
	log.Printf(C.GoString(c), float32(f))
}
