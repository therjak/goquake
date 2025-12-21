// SPDX-License-Identifier: GPL-2.0-or-later

package net

import (
	"bytes"
	"encoding/binary"

	"goquake/protocol"
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

func (m *Message) Write(data interface{}) {
	binary.Write(&m.buf, binary.LittleEndian, data)
}

func (m *Message) WriteChar(c int) {
	m.Write(uint8(c))
}

func (m *Message) WriteByte(c int) {
	m.Write(uint8(c))
}

func (m *Message) WriteInt8(c int8) {
	m.Write(c)
}

func (m *Message) WriteUint8(c uint8) {
	m.Write(c)
}

func (m *Message) WriteShort(c int) {
	m.Write(int16(c))
}

func (m *Message) WriteInt16(c int16) {
	m.Write(c)
}

func (m *Message) WriteUint16(c uint16) {
	m.Write(c)
}

func (m *Message) WriteLong(c int) {
	m.Write(int32(c))
}

func (m *Message) WriteInt32(c int32) {
	m.Write(c)
}

func (m *Message) WriteUint32(c uint32) {
	m.Write(c)
}

func (m *Message) WriteFloat(c float32) {
	m.Write(c)
}

func (m *Message) WriteString(c string) {
	if len(c) != 0 {
		m.buf.WriteString(c)
	}
	m.WriteUint8(0)
}

func rint(x float32) int {
	if x > 0 {
		return int(x + 0.5)
	}
	return int(x - 0.5)
}

func (m *Message) WriteCoord(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATCOORD != 0 {
		m.Write(f)
	} else if flags&protocol.PRFL_INT32COORD != 0 {
		m.Write(int32(rint(f * 16)))
	} else if flags&protocol.PRFL_24BITCOORD != 0 {
		m.Write(int16(f))
		m.Write(uint8(rint(f*255) % 0xff))
	} else {
		m.Write(int16(rint(f * 8)))
	}
}

func (m *Message) WriteAngle(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATANGLE != 0 {
		m.Write(f)
	} else if flags&protocol.PRFL_SHORTANGLE != 0 {
		m.Write(int16(rint(f*65536.0/360) & 0xffff))
	} else {
		m.Write(uint8(rint(f*256.0/360.0) & 0xff))
	}
}

func (m *Message) WriteAngle16(f float32, flags uint32) {
	if flags&protocol.PRFL_FLOATANGLE != 0 {
		m.Write(f)
	} else {
		m.Write(int16(rint(f*65536.0/360.0) & 0xffff))
	}
}

func (m *Message) WriteBytes(b []byte) {
	m.buf.Write(b)
}

func (m *Message) HasMessage() bool {
	return m.buf.Len() > 0
}

// Deprecated: use Reset()
func (m *Message) ClearMessage() {
	m.buf.Reset()
}

func (m *Message) Reset() {
	m.buf.Reset()
}
