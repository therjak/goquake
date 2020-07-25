package quakelib

import "C"

import (
	"fmt"
)

//export VIDGLSwapControl
func VIDGLSwapControl() bool {
	return glSwapControl
}

//export SetVIDGLSwapControl
func SetVIDGLSwapControl(v C.int) {
	glSwapControl = (v != 0)
}

//export VIDGetSwapInterval
func VIDGetSwapInterval() int {
	return getSwapInterval()
}

//export VID_Init_Go
func VID_Init_Go() {
	err := videoInit()
	if err != nil {
		fmt.Printf("%v", err)
		Error(fmt.Sprintf("%v", err))
	}
}
