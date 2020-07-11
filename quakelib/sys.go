package quakelib

import "C"

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/therjak/goquake/qtime"
)

var (
/*
	//int, what is this for?
	memSize = 0 //?
	//void* what is this for?
	membase = null
*/
)

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

//export Sys_DoubleTime
func Sys_DoubleTime() C.double {
	return C.double(qtime.QTime().Seconds())
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
