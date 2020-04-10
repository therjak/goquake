package net

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	prfl "quake/protocol/flags"
	"strings"
)

type QReader struct {
	r *bytes.Reader
}

func NewQReader(data []byte) *QReader {
	return &QReader{bytes.NewReader(data)}
}

func (q *QReader) ReadInt8() (int8, error) {
	var r int8
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) UnreadByte() {
	q.r.UnreadByte()
}

func (q *QReader) ReadByte() (byte, error) {
	var r byte
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadUint8() (uint8, error) {
	var r uint8
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadInt16() (int16, error) {
	var r int16
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadUint16() (uint16, error) {
	var r uint16
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadInt32() (int32, error) {
	var r int32
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadUint32() (uint32, error) {
	var r uint32
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) ReadFloat32() (float32, error) {
	var r float32
	err := binary.Read(q.r, binary.LittleEndian, &r)
	return r, err
}

func (q *QReader) Read(data interface{}) error {
	return binary.Read(q.r, binary.LittleEndian, data)
}

// 13.3 fixed point coords, max range +-4096
func (q *QReader) ReadCoord16() (float32, error) {
	i, err := q.ReadInt16()
	return float32(i) * (1.0 / 8.0), err
}

// 16.8 fixed point coords, max range +-32768
func (q *QReader) ReadCoord24() (float32, error) {
	// We need to read both before handling the errors if we do not want to change
	// the logic.
	// TODO: Do we need to keep the logic?
	i16, err1 := q.ReadInt16()
	i8, err2 := q.ReadUint8()
	if err1 != nil {
		return 0, err1
	}
	if err2 != nil {
		return 0, err2
	}
	return float32(i16) + (float32(i8) * (1.0 / 255.0)), nil
}

func (q *QReader) ReadCoord32f() (float32, error) {
	return q.ReadFloat32()
}

// TODO(therjak):
// it is not needed to always check these flags, just change the called function
// whenever cl.protocolflags would be changed
func (q *QReader) ReadCoord(flags uint16) (float32, error) {
	if flags&prfl.COORDFLOAT != 0 {
		return q.ReadFloat32()
	} else if flags&prfl.COORDINT32 != 0 {
		i, err := q.ReadInt32()
		return float32(i) * (1.0 / 16.0), err
	} else if flags&prfl.COORD24BIT != 0 {
		return q.ReadCoord24()
	}
	return q.ReadCoord16()
}

func (q *QReader) ReadAngle(flags uint32) (float32, error) {
	if flags&prfl.ANGLEFLOAT != 0 {
		return q.ReadFloat32()
	} else if flags&prfl.ANGLESHORT != 0 {
		i, err := q.ReadInt16()
		return float32(i) * (360.0 / 65536.0), err
	}
	i, err := q.ReadInt8()
	return float32(i) * (360.0 / 256.0), err
}

func (q *QReader) ReadAngle16(flags uint32) (float32, error) {
	if flags&prfl.ANGLEFLOAT != 0 {
		return q.ReadFloat32()
	}
	i, err := q.ReadInt16()
	return float32(i) * (360.0 / 65536.0), err
}

func (q *QReader) ReadString() (string, error) {
	sb := strings.Builder{}
	for {
		b, err := q.ReadByte()
		if err != nil {
			return "", err
		}
		if b == 0 {
			break
		}
		sb.WriteByte(b)
	}
	return sb.String(), nil
}

// Len returns the number of bytes of the unread portion of the slice.
func (q *QReader) Len() int {
	return q.r.Len()
}

func (q *QReader) BeginReading() {
	i, _ := q.r.Seek(0, io.SeekCurrent)
	log.Printf("BeginReading while at %d", i)
	q.r.Seek(0, io.SeekStart)
}
