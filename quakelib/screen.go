package quakelib

//#include <stdlib.h>
// int SCR_ModalMessage(const char *text, float timeout);
// void SCR_BeginLoadingPlaque(void);
// void SCR_EndLoadingPlaque(void);
// void SCR_UpdateScreen(void);
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"quake/cmd"
	"quake/conlog"
	"quake/image"
	"unsafe"

	"github.com/go-gl/gl/v4.6-core/gl"
)

var (
	screen qScreen
)

type qScreen struct {
	disabled bool
}

func ModalMessage(message string, timeout float32) bool {
	m := C.CString(message)
	defer C.free(unsafe.Pointer(m))
	return C.SCR_ModalMessage(m, C.float(timeout)) != 0
}

func SCR_BeginLoadingPlaque() {
	C.SCR_BeginLoadingPlaque()
}

func SCR_EndLoadingPlaque() {
	C.SCR_EndLoadingPlaque()
}

func SCR_UpdateScreen() {
	C.SCR_UpdateScreen()
}

func init() {
	cmd.AddCommand("screenshot", screenShot)
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func screenShot(_ []cmd.QArg, _ int) {
	var pngName string
	var fileName string
	for i := 0; i < 10000; i++ {
		pngName = fmt.Sprintf("spasm%04d.png", i)
		fileName = filepath.Join(gameDirectory, pngName)
		if !fileExists(fileName) {
			break
		}
		if i == 9999 {
			conlog.Printf("Coun't find an unused filename\n")
			return
		}
	}
	buffer := make([]byte, viewport.width*viewport.height*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(viewport.x, viewport.y, viewport.width, viewport.height,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(buffer))
	err := image.Write(fileName, buffer, int(viewport.width), int(viewport.height))
	if err != nil {
		conlog.Printf("Coudn't create screenshot file\n")
	} else {
		conlog.Printf("Wrote %s\n", pngName)
	}
}
