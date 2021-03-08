// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

/*
typedef void (*xcommand_t)(void);
void callQuakeFunc(xcommand_t f);
*/
import "C"

import (
	"github.com/therjak/goquake/cmd"
)

//export Cmd_AddCommand
func Cmd_AddCommand(cmd_name *C.char, f C.xcommand_t) {
	name := C.GoString(cmd_name)
	cmd.AddCommand(name, func(_ []cmd.QArg, _ int) { C.callQuakeFunc(f) })
}
