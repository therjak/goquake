package bsp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestQLit(t *testing.T) {
	litData := make([]byte, 4)
	litData[0] = 'Q'
	litData[1] = 'L'
	litData[2] = 'I'
	litData[3] = 'T'
	buf := bytes.NewReader(litData)
	var magic uint32
	binary.Read(buf, binary.LittleEndian, &magic)
	if qlit != magic {
		t.Error("qlit != litdata")
	}
}
