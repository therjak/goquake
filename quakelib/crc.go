package quakelib

import (
	"C"
)
import (
	"quake/crc"
	"unsafe"
)

//export CRC_Block
func CRC_Block(start *C.uchar, count C.int) uint16 {
	p := C.GoBytes(unsafe.Pointer(start), count)
	return crc.Update(p)
}
