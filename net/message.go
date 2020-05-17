package net

import (
	"bytes"
	"encoding/binary"
	"quake/protocol"
)

type Message struct {
	buf bytes.Buffer
}

func (m *Message) Bytes() []byte {
	return m.buf.Bytes()
}

func (m *Message) Len() int {
	return m.buf.Len()
}

func (m *Message) write(data interface{}) {
	binary.Write(&m.buf, binary.LittleEndian, data)
}

func (m *Message) WriteChar(c int) {
	m.write(uint8(c))
}

func (m *Message) WriteByte(c int) {
	m.write(uint8(c))
}

func (m *Message) WriteShort(c int) {
	m.write(int16(c))
}

func (m *Message) WriteLong(c int) {
	m.write(int32(c))
}

func (m *Message) WriteFloat(c float32) {
	m.write(c)
}

func (m *Message) WriteString(c string) {
	if len(c) != 0 {
		m.buf.WriteString(c)
	}
	m.WriteByte(0)
}

func rint(x float32) int {
	if x > 0 {
		return int(x + 0.5)
	}
	return int(x - 0.5)
}

func (m *Message) WriteCoord(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATCOORD != 0 {
		m.WriteFloat(f)
	} else if flags&protocol.PRFL_INT32COORD != 0 {
		m.WriteLong(rint(f * 16))
	} else if flags&protocol.PRFL_24BITCOORD != 0 {
		m.WriteShort(int(f))
		m.WriteByte(rint(f*255) % 255)
	} else {
		m.WriteShort(rint(f * 8))
	}
}

func (m *Message) WriteAngle(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATANGLE != 0 {
		m.WriteFloat(f)
	} else if flags&protocol.PRFL_SHORTANGLE != 0 {
		m.WriteShort(rint(f*65536.0/360) & 65535)
	} else {
		m.WriteByte(rint(f*256.0/360.0) & 255)
	}
}

func (m *Message) WriteAngle16(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATANGLE != 0 {
		m.WriteFloat(f)
	} else {
		m.WriteShort(rint(f*65536.0/360.0) & 65535)
	}
}

func (m *Message) WriteBytes(b []byte) {
	m.buf.Write(b)
}

func (m *Message) HasMessage() bool {
	return m.buf.Len() > 0
}

func (m *Message) ClearMessage() {
	m.buf.Reset()
}
