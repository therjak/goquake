// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

import (
	"C"
	"encoding/binary"
	"math"
)

// This seems like a hack as either the in or out should be bytes anyway

func big16(in uint16) uint16 {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, in)
	return binary.BigEndian.Uint16(bytes)
}

func big32(in uint32) uint32 {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, in)
	return binary.BigEndian.Uint32(bytes)
}

func little16(in uint16) uint16 {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, in)
	return binary.LittleEndian.Uint16(bytes)
}

func little32(in uint32) uint32 {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, in)
	return binary.LittleEndian.Uint32(bytes)
}

//export LittleShort
func LittleShort(in C.short) C.short {
	a := little16(uint16(in))
	return C.short(a)
}

//export LittleLong
func LittleLong(in C.long) C.long {
	a := little32(uint32(in))
	return C.long(a)
}

//export LittleFloat
func LittleFloat(in C.float) C.float {
	bits := math.Float32bits(float32(in))
	a := little32(bits)
	return C.float(math.Float32frombits(a))
}
