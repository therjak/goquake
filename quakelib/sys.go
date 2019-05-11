package quakelib

//#include <stdlib.h>
//#include "host_shutdown.h"
import "C"

import (
	"fmt"
	"log"
	"os"
	"quake/conlog"
	"quake/qtime"
	"runtime/debug"
	"strings"
	"time"
	"unsafe"
)

var (
	/*
		//int, what is this for?
		memSize = 0 //?
		//void* what is this for?
		membase = null
	*/
	screenDisabled = false
)

//export ScreenDisabled
func ScreenDisabled() C.int {
	if screenDisabled {
		return 1
	}
	return 0
}

//export SetScreenDisabled
func SetScreenDisabled(b C.int) {
	screenDisabled = (b != 0)
}

//export Sys_DoubleTime
func Sys_DoubleTime() C.double {
	return C.double(qtime.QTime().Seconds())
}

func Error(format string, v ...interface{}) {
	debug.PrintStack()
	C.Host_Shutdown()
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
	C.Host_Shutdown()
	os.Exit(0)
}

//export Sys_Sleep
func Sys_Sleep(ms C.ulong) {
	time.Sleep(time.Millisecond * time.Duration(ms))
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

//export Sys_PrintServerProtocol
func Sys_PrintServerProtocol(i C.int, s *C.char) {
	log.Printf("Server using protocol %v (%s)\n", int(i), C.GoString(s))
}

//export REPORT_BadCall
func REPORT_BadCall() {
	fmt.Printf("Go BadCall\n")
}

//export REPORT_INT
func REPORT_INT(in C.int) {
	fmt.Printf("Go ReportInt %v\n", in)
}

//export REPORT_STR
func REPORT_STR(in *C.char) {
	fmt.Printf(C.GoString(in))
}

func SVClientPrintf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	log.Print(s)
	HostClient().Printf(s)
}

//do not use. use conlog.Printf
func conPrintf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	cstr := C.CString(s)
	defer C.free(unsafe.Pointer(cstr))
	log.Print(s)
	C.Con_PrintStr(cstr)
}

//do not use. use conlog.Printf
func conPrintStr(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	cstr := C.CString(s)
	defer C.free(unsafe.Pointer(cstr))
	C.Con_PrintStr(cstr)
}

//do not use. use conlog.SafePrintf
func conSafePrintf(format string, v ...interface{}) {
	tmp := ScreenDisabled()
	screenDisabled = true
	defer SetScreenDisabled(tmp)
	conPrintStr(format, v...)
}

func init() {
	conlog.SetPrintf(conPrintf)
	conlog.SetSafePrintf(conSafePrintf)
}

const (
	// 40 chars, starts with 1d, ends with 1f, 1e between
	quakeBar = "\x1d\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1e\x1f\n"
)

//export ConPrintBar
func ConPrintBar() {
	if consoleLineWidth >= len(quakeBar) {
		conlog.Printf(quakeBar)
	} else {
		var b strings.Builder
		b.WriteByte('\x1d')
		for i := 2; i < consoleLineWidth; i++ {
			b.WriteByte('\x1e')
		}
		b.WriteByte('\x1f')
		conlog.Printf(b.String())
	}
}
