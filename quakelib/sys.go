// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"log"
	"runtime/debug"
)

/*
func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
*/

func Error(format string, v ...interface{}) {
	debug.PrintStack()
	host.Shutdown()
	log.Fatalf(format, v...)
}

func Sys_Quit() {
	host.Shutdown()
	quitChan <- true
}
