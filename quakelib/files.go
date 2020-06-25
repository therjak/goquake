package quakelib

// void setInt(int* l, int v);
import "C"

import (
	"log"

	"github.com/therjak/goquake/filesystem"
	image "github.com/therjak/goquake/image"
)

//export COM_LoadFileGo
func COM_LoadFileGo(name *C.char, length *C.int) *C.uchar {
	n := C.GoString(name)
	b, err := filesystem.GetFileContents(n)
	if err != nil {
		log.Printf("Could not load file %v, %v", n, err)
		return nil
	}
	C.setInt(length, C.int(len(b)))
	return (*C.uchar)(C.CBytes(b))
}

//export Image_LoadImage
func Image_LoadImage(name *C.char, width *C.int, height *C.int) *C.uchar {
	n := C.GoString(name)
	img, err := image.Load(n)
	if err != nil {
		return nil
	}
	s := img.Bounds().Size()
	C.setInt(width, C.int(s.X))
	C.setInt(height, C.int(s.Y))
	return (*C.uchar)(C.CBytes(img.Pix))
}
