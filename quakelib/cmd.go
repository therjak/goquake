package quakelib

/*
typedef void (*xcommand_t)(void);
void callQuakeFunc(xcommand_t f);
*/
import "C"

import (
	"log"
	"quake/cmd"
)

//export Cmd_AddCommand
func Cmd_AddCommand(cmd_name *C.char, f C.xcommand_t) {
	name := C.GoString(cmd_name)
	cmd.AddCommand(name, func(_ []cmd.QArg) { C.callQuakeFunc(f) })
}

//export Cmd_Exists
func Cmd_Exists(cmd_name *C.char) C.int {
	ok := cmd.Exists(C.GoString(cmd_name))
	if ok {
		return 1
	}
	return 0
}

//export Cmd_Argc
func Cmd_Argc() C.int {
	// log.Printf("Argc: %v", args.Argc())
	return C.int(cmd.Argc())
}

//export Cmd_ArgsInt
func Cmd_ArgsInt() *C.char {
	log.Printf("Args called")
	s := cmd.Full()
	return C.CString(s)
}

//export Cmd_ArgvInt
func Cmd_ArgvInt(i C.int) *C.char {
	return C.CString(cmd.Argv(int(i)).String())
}

//export Cmd_ArgvAsInt
func Cmd_ArgvAsInt(i C.int) C.int {
	r := cmd.Argv(int(i)).Int()
	return C.int(r)
}

//export Cmd_ArgvAsDouble
func Cmd_ArgvAsDouble(i C.int) C.double {
	return C.double(cmd.ArgvAsDouble(int(i)))
}

//export Cmd_ParseString
func Cmd_ParseString(in *C.char) {
	s := C.GoString(in)
	cmd.Parse(s)
}
