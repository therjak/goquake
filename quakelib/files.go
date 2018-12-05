package quakelib

// void setInt(int* l, int v);
import "C"

import (
	"log"
	"path/filepath"
	"quake/filesystem"
	image "quake/image"
	"unsafe"
)

//export COM_AddGameDirectoryGo
func COM_AddGameDirectoryGo(base, dir *C.char) {
	d := filepath.Join(C.GoString(base), C.GoString(dir))
	filesystem.AddGameDir(d)
}

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

//export COM_FileExists
func COM_FileExists(name *C.char) int {
	_, err := filesystem.GetFile(C.GoString(name))
	if err != nil {
		return 0
	}
	return 1
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

//export Image_Write
func Image_Write(name *C.char, data *C.uchar, width C.int, height C.int) C.int {
	n := C.GoString(name)
	w := int(width)
	h := int(height)
	d := C.GoBytes(unsafe.Pointer(data), width*height*4)
	if err := image.Write(n, d, w, h); err != nil {
		return 0
	}
	return 1
}
