// SPDX-License-Identifier: GPL-2.0-or-later

package net

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestReadAck(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	out := make(chan msg, 1)
	acks := make(chan uint32, 1)
	go readUDP(c1, out, acks)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(8|NETFLAG_ACK))
	binary.Write(&buf, binary.BigEndian, uint32(42))
	c2.Write(buf.Bytes())

	got := <-acks
	if got != 42 {
		t.Errorf("Got wrong ack sequence")
	}
}

func TestUDPReadUnreliable(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	out := make(chan msg, 1)
	acks := make(chan uint32, 1)
	go readUDP(c1, out, acks)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(12|NETFLAG_UNRELIABLE))
	// With unreliable we can start with an arbitrary sequence number
	// as we might have missed the previous ones
	binary.Write(&buf, binary.BigEndian, uint32(42))
	buf.Write([]byte{1, 2, 45, 5})
	c2.Write(buf.Bytes())
	buf.Reset()

	got := <-out
	want := []byte{2, 1, 2, 45, 5}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}

	// A packet we already know of should not cause a signal
	binary.Write(&buf, binary.BigEndian, uint32(12|NETFLAG_UNRELIABLE))
	binary.Write(&buf, binary.BigEndian, uint32(42))
	buf.Write([]byte{1, 2, 45, 5})
	c2.Write(buf.Bytes())
	buf.Reset()

	binary.Write(&buf, binary.BigEndian, uint32(11|NETFLAG_UNRELIABLE))
	binary.Write(&buf, binary.BigEndian, uint32(44))
	buf.Write([]byte{83, 212, 43})
	c2.Write(buf.Bytes())
	buf.Reset()

	got = <-out
	want = []byte{2, 83, 212, 43}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}

	// An old packet should not cause a signal
	binary.Write(&buf, binary.BigEndian, uint32(14|NETFLAG_UNRELIABLE))
	binary.Write(&buf, binary.BigEndian, uint32(30))
	buf.Write([]byte{11, 21, 3, 23, 23, 23})
	c2.Write(buf.Bytes())
	buf.Reset()

	binary.Write(&buf, binary.BigEndian, uint32(16|NETFLAG_UNRELIABLE))
	binary.Write(&buf, binary.BigEndian, uint32(45))
	buf.Write([]byte{25, 11, 53, 62, 62, 66, 67, 68})
	c2.Write(buf.Bytes())
	buf.Reset()

	got = <-out
	want = []byte{2, 25, 11, 53, 62, 62, 66, 67, 68}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}
}

func TestUDPReadReliableSinglePacket(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	out := make(chan msg, 1)
	acks := make(chan uint32, 1)
	ret := make([]byte, 8) // For the ACK
	go readUDP(c1, out, acks)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(12|NETFLAG_DATA|NETFLAG_EOM))
	// With reliable we have to start with sequence number 0
	binary.Write(&buf, binary.BigEndian, uint32(0))
	buf.Write([]byte{1, 2, 45, 5})
	c2.Write(buf.Bytes())
	buf.Reset()
	if i, err := c2.Read(ret); err != nil || i != 8 {
		t.Fatalf("Could not read ACK")
	}
	wantRet := []byte{0, 2, 0, 8, 0, 0, 0, 0} // ACK(2), len(8), seq(0)
	if !bytes.Equal(ret, wantRet) {
		t.Errorf("wrong Ack. want %v got %v", wantRet, ret)
	}

	got := <-out
	want := []byte{1, 1, 2, 45, 5}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}

	// A packet we already know of should not cause a signal but still cause an ACK
	binary.Write(&buf, binary.BigEndian, uint32(12|NETFLAG_DATA|NETFLAG_EOM))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	buf.Write([]byte{1, 2, 45, 5})
	c2.Write(buf.Bytes())
	buf.Reset()
	if _, err := c2.Read(ret); err != nil {
		t.Fatalf("Could not read ACK")
	}
	wantRet = []byte{0, 2, 0, 8, 0, 0, 0, 0}
	if !bytes.Equal(ret, wantRet) {
		t.Errorf("wrong Ack. want %v got %v", wantRet, ret)
	}

	// A new message should still count
	binary.Write(&buf, binary.BigEndian, uint32(11|NETFLAG_DATA|NETFLAG_EOM))
	binary.Write(&buf, binary.BigEndian, uint32(1)) // increment sequence
	buf.Write([]byte{83, 212, 43})
	c2.Write(buf.Bytes())
	buf.Reset()
	if _, err := c2.Read(ret); err != nil {
		t.Fatalf("Could not read ACK")
	}
	wantRet = []byte{0, 2, 0, 8, 0, 0, 0, 1}
	if !bytes.Equal(ret, wantRet) {
		t.Errorf("wrong Ack. want %v got %v", wantRet, ret)
	}

	got = <-out
	want = []byte{1, 83, 212, 43}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}
}

func TestUDPReadDualMessage(t *testing.T) {
	// Verify that we do not reuse the memory of the outgoing message in an invalid way
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	out := make(chan msg, 4)
	acks := make(chan uint32, 1)
	go readUDP(c1, out, acks)

	go func() {
		// For this bug to happen we need to send from a different go routine.
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, uint32(57|NETFLAG_UNRELIABLE))
		binary.Write(&buf, binary.BigEndian, uint32(0))
		// I am not sure about the slice internals but this is a real message and
		// it produced this error.
		buf.Write([]byte{2, 7, 180, 232, 18, 67, 15, 34, 70, 12, 20, 1, 17, 0, 0,
			65, 100, 0, 25, 25, 0, 0, 0, 1, 207, 1, 2, 7, 167, 4, 253, 0, 54, 64, 3,
			128, 46, 128, 47, 128, 59, 128, 65, 128, 81, 128, 82, 128, 83})
		c2.Write(buf.Bytes())
		buf.Reset()

		binary.Write(&buf, binary.BigEndian, uint32(58|NETFLAG_UNRELIABLE))
		binary.Write(&buf, binary.BigEndian, uint32(1))
		buf.Write([]byte{2, 7, 80, 237, 18, 67, 15, 162, 66, 12, 20, 0, 1, 17, 0,
			0, 65, 100, 0, 25, 25, 0, 0, 0, 1, 207, 1, 2, 7, 213, 4, 253, 0, 54, 62,
			3, 128, 46, 128, 47, 128, 59, 128, 65, 128, 81, 128, 82, 128, 83})
		c2.Write(buf.Bytes())
		buf.Reset()
	}()

	got := <-out
	want := []byte{2, 2, 7, 180, 232, 18, 67, 15, 34, 70, 12, 20, 1, 17, 0, 0,
		65, 100, 0, 25, 25, 0, 0, 0, 1, 207, 1, 2, 7, 167, 4, 253, 0, 54, 64, 3,
		128, 46, 128, 47, 128, 59, 128, 65, 128, 81, 128, 82, 128, 83}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable message 1. want %v, got %v", want, got.data)
	}
	got = <-out
	want = []byte{2, 2, 7, 80, 237, 18, 67, 15, 162, 66, 12, 20, 0, 1, 17, 0,
		0, 65, 100, 0, 25, 25, 0, 0, 0, 1, 207, 1, 2, 7, 213, 4, 253, 0, 54, 62,
		3, 128, 46, 128, 47, 128, 59, 128, 65, 128, 81, 128, 82, 128, 83}
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable message 2. want %v, got %v", want, got.data)
	}
}

func TestUDPReadReliableMultiPacket(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	out := make(chan msg, 1)
	acks := make(chan uint32, 1)
	ret := make([]byte, 8) // For the ACK
	go readUDP(c1, out, acks)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, uint32(12|NETFLAG_DATA))
	// With reliable we have to start with sequence number 0
	binary.Write(&buf, binary.BigEndian, uint32(0))
	buf.Write([]byte{1, 2, 45, 5})
	c2.Write(buf.Bytes())
	buf.Reset()
	if i, err := c2.Read(ret); err != nil || i != 8 {
		t.Fatalf("Could not read ACK")
	}
	wantRet := []byte{0, 2, 0, 8, 0, 0, 0, 0} // ACK(2), len(8), seq(0)
	if !bytes.Equal(ret, wantRet) {
		t.Errorf("wrong Ack. want %v got %v", wantRet, ret)
	}

	binary.Write(&buf, binary.BigEndian, uint32(11|NETFLAG_DATA|NETFLAG_EOM))
	binary.Write(&buf, binary.BigEndian, uint32(1)) // increment sequence
	buf.Write([]byte{83, 212, 43})
	c2.Write(buf.Bytes())
	buf.Reset()
	if _, err := c2.Read(ret); err != nil {
		t.Fatalf("Could not read ACK")
	}
	wantRet = []byte{0, 2, 0, 8, 0, 0, 0, 1}
	if !bytes.Equal(ret, wantRet) {
		t.Errorf("wrong Ack. want %v got %v", wantRet, ret)
	}

	got := <-out
	want := []byte{1, 1, 2, 45, 5, 83, 212, 43} // 1 + msg1 + msg2
	if !bytes.Equal(got.data, want) {
		t.Fatalf("Got wrong unreliable sequence. want %v, got %v", want, got.data)
	}
}

func TestUDPWriteUnreliable(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	in := make(chan msg, 1)
	acks := make(chan uint32, 1)
	canWrite := make(chan bool, 1)
	go writeUDP(c1, in, acks, canWrite)

	in <- msg{data: []byte{2, 1, 2, 45, 5}}
	got := make([]byte, 50)
	i, err := c2.Read(got)
	if err != nil {
		t.Fatalf("Could not read from connection: %v", err)
	}
	if i != 4+8 {
		t.Errorf("Got wrong number of bytes: want %v got %v", 4+8, i)
	}
	want := []byte{0, 0x10, 0, 12, 0, 0, 0, 0, 1, 2, 45, 5}
	if !bytes.Equal(want, got[:i]) {
		t.Errorf("Got wrong message: want %v, got %v", want, got[:i])
	}

	in <- msg{data: []byte{2, 84, 212, 43}}
	i, err = c2.Read(got)
	if err != nil {
		t.Fatalf("Could not read from connection: %v", err)
	}
	if i != 3+8 {
		t.Errorf("Got wrong number of bytes: want %v got %v", 3+8, i)
	}
	want = []byte{0, 0x10, 0, 11, 0, 0, 0, 1, 84, 212, 43}
	if !bytes.Equal(want, got[:i]) {
		t.Errorf("Got wrong message: want %v, got %v", want, got[:i])
	}
}

func TestUDPWriteReliable(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c2.Close()
	c1.SetDeadline(time.Time{})
	c2.SetDeadline(time.Time{})
	in := make(chan msg, 1)
	acks := make(chan uint32, 1)
	canWrite := make(chan bool, 1)
	go writeUDP(c1, in, acks, canWrite)

	in <- msg{data: []byte{1, 1, 2, 45, 5}}
	got := make([]byte, 50)
	i, err := c2.Read(got)
	if err != nil {
		t.Fatalf("Could not read from connection: %v", err)
	}
	if i != 4+8 {
		t.Errorf("Got wrong number of bytes: want %v got %v", 4+8, i)
	}
	// 0x00 0x09 == NETFLAG_DATA + NETFLAG_EOM
	want := []byte{0, 0x09, 0, 12, 0, 0, 0, 0, 1, 2, 45, 5}
	if !bytes.Equal(want, got[:i]) {
		t.Errorf("Got wrong message: want %v, got %v", want, got[:i])
	}
	acks <- 0 // ack the sequence 0,0,0,0
	next := <-canWrite
	if !next {
		t.Fatal("canWrite did return false")
	}

	in <- msg{data: []byte{1, 84, 212, 43}}
	i, err = c2.Read(got)
	if err != nil {
		t.Fatalf("Could not read from connection: %v", err)
	}
	if i != 3+8 {
		t.Errorf("Got wrong number of bytes: want %v got %v", 3+8, i)
	}
	want = []byte{0, 0x09, 0, 11, 0, 0, 0, 1, 84, 212, 43}
	if !bytes.Equal(want, got[:i]) {
		t.Errorf("Got wrong message: want %v, got %v", want, got[:i])
	}
	acks <- 1 // ack the 0,0,0,1
	next = <-canWrite
	if !next {
		t.Fatal("canWrite did return false")
	}
}

// Missing tests:
// Long UDPWriteReliable with split message (needs message at least 32001 long)
// Resend case if no ack for UDPWriteReliable (find good way to mock the timer)
