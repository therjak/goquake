// SPDX-License-Identifier: GPL-2.0-or-later
package palette

import (
	"goquake/texture"
)

type Palette [256 * 4]uint8

var (
	Table                Palette
	TableFullBright      Palette
	TableFullBrightFence Palette
	TableNoBright        Palette
	TableNoBrightFence   Palette
	TableConsoleChars    Palette
)

func init() {
	pi := 0
	for i := 0; i < 256; i++ {
		rgba := texture.Palette[i]
		Table[pi] = rgba.R
		Table[pi+1] = rgba.G
		Table[pi+2] = rgba.B
		Table[pi+3] = rgba.A
		TableFullBright[pi+3] = rgba.A
		TableNoBright[pi+3] = rgba.A
		pi += 4
	}

	blend := 4 * 224
	// keep 0-223 black
	copy(TableFullBright[blend:], Table[blend:])
	// keep 224-255 black
	copy(TableNoBright[:blend], Table[:blend])

	copy(TableFullBrightFence[:], TableFullBright[:])
	copy(TableNoBrightFence[:], TableNoBright[:])

	// Make the last color transparent
	Table[256*4-1] = 0
	copy(TableConsoleChars[:], Table[:])
	// Make the first color transparent (black)
	TableConsoleChars[3] = 0

	copy(TableFullBrightFence[255*4:], []uint8{0, 0, 0, 0})
	copy(TableNoBrightFence[255*4:], []uint8{0, 0, 0, 0})
}

func (p *Palette) Convert(data []byte) []byte {
	nd := make([]byte, 0, len(data)*4)
	for _, d := range data {
		idx := int(d) * 4
		pixel := p[idx : idx+4]
		nd = append(nd, pixel...)
	}
	return nd
}
