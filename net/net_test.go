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

func TestReadUnreliable(t *testing.T) {
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
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
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
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
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
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
	}
}

func TestReadReliableSinglePacket(t *testing.T) {
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
	if bytes.Compare(ret, wantRet) != 0 {
		t.Errorf("wrong Ack. got %v want %v", ret, wantRet)
	}

	got := <-out
	want := []byte{1, 1, 2, 45, 5}
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
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
	if bytes.Compare(ret, wantRet) != 0 {
		t.Errorf("wrong Ack. got %v want %v", ret, wantRet)
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
	if bytes.Compare(ret, wantRet) != 0 {
		t.Errorf("wrong Ack. got %v want %v", ret, wantRet)
	}

	got = <-out
	want = []byte{1, 83, 212, 43}
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
	}
}

func TestReadReliableMultiPacket(t *testing.T) {
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
	if bytes.Compare(ret, wantRet) != 0 {
		t.Errorf("wrong Ack. got %v want %v", ret, wantRet)
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
	if bytes.Compare(ret, wantRet) != 0 {
		t.Errorf("wrong Ack. got %v want %v", ret, wantRet)
	}

	got := <-out
	want := []byte{1, 1, 2, 45, 5, 83, 212, 43} // 1 + msg1 + msg2
	if bytes.Compare(got.data, want) != 0 {
		t.Fatalf("Got wrong unreliable sequence. got %v, want %v", got.data, want)
	}
}
