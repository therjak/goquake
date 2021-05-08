// SPDX-License-Identifier: GPL-2.0-or-later

package crc

const (
	ccittFalse = 0x1021
	cRCInitial = 0xffff
)

type Table struct {
	entries [256]uint16
}

// 16bit CRC used by XMODEM
var ccittFalseTable = makeTable(ccittFalse)

func makeTable(poly uint16) *Table {
	t := &Table{}
	width := uint16(16)
	for i := uint16(0); i < 256; i++ {
		crc := i << (width - 8)
		for j := 0; j < 8; j++ {
			if crc&(1<<(width-1)) != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		t.entries[i] = crc
	}
	return t
}

func update(crc uint16, p []byte) uint16 {
	for _, v := range p {
		crc = ccittFalseTable.entries[byte(crc>>8)^v] ^ (crc << 8)
	}
	return crc
}

func Update(p []byte) uint16 {
	return update(cRCInitial, p)
}
