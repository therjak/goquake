package qmsg

import (
	"bytes"
	"encoding/binary"

	"github.com/therjak/goquake/protocol"
)

type ClientWriter interface {
	Bytes() []byte
	WriteByte(c byte) error
	WriteLong(c uint32) error
	WriteShort(c uint16) error
	WriteFloat(f float32) error
	WriteAngle(f float32) error
	WriteAngle16(f float32) error
	WriteString(s string) (int, error)
}

func NewClientWriter(flags uint16) ClientWriter {
	if flags&protocol.ANGLEFLOAT != 0 {
		return &WriterFloatAngle{}
	} else if flags&protocol.ANGLESHORT != 0 {
		return &WriterShortAngle{}
	}
	return &WriterByteAngle{}
}

type BaseWriter struct {
	bytes.Buffer
}

type WriterFloatAngle struct {
	BaseWriter
}
type WriterShortAngle struct {
	BaseWriter
}
type WriterByteAngle struct {
	BaseWriter
}

func (b *BaseWriter) WriteShort(c uint16) error {
	return binary.Write(b, binary.LittleEndian, c)
}

func (b *BaseWriter) WriteLong(c uint32) error {
	return binary.Write(b, binary.LittleEndian, c)
}

func (b *BaseWriter) WriteFloat(f float32) error {
	return binary.Write(b, binary.LittleEndian, f)
}

func rint(x float32) int32 {
	if x > 0 {
		return int32(x + 0.5)
	}
	return int32(x - 0.5)
}

func (b *BaseWriter) WriteFloatAngle(f float32) error {
	return b.WriteFloat(f)
}

func (b *BaseWriter) WriteShortAngle(f float32) error {
	return b.WriteShort(uint16(rint(f*65536.0/360.0) & 0xffff))
}

func (b *BaseWriter) WriteByteAngle(f float32) error {
	return b.WriteByte(byte(rint(f*256.0/360.0) & 0xff))
}

func (b *WriterFloatAngle) WriteAngle(f float32) error {
	return b.WriteFloatAngle(f)
}
func (b *WriterShortAngle) WriteAngle(f float32) error {
	return b.WriteShortAngle(f)
}
func (b *WriterByteAngle) WriteAngle(f float32) error {
	return b.WriteByteAngle(f)
}

func (b *WriterFloatAngle) WriteAngle16(f float32) error {
	return b.WriteFloatAngle(f)
}
func (b *WriterShortAngle) WriteAngle16(f float32) error {
	return b.WriteShortAngle(f)
}
func (b *WriterByteAngle) WriteAngle16(f float32) error {
	return b.WriteShortAngle(f)
}
