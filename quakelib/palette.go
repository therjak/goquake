package quakelib

import (
	"github.com/therjak/goquake/filesystem"
)

type qPalette struct {
	table                [256 * 4]uint8
	tableFullBright      [256 * 4]uint8
	tableFullBrightFence [256 * 4]uint8
	tableNoBright        [256 * 4]uint8
	tableNoBrightFence   [256 * 4]uint8
	tableConsoleChars    [256 * 4]uint8
}

var (
	palette qPalette
)

func (p *qPalette) Init() {
	b, err := filesystem.GetFileContents("gfx/palette.lmp")
	if err != nil {
		Error("Couln't load gfx/palette.lmp")
	}
	// b is rgb 8bit, we want rgba float32
	if 4*len(b) != 3*len(p.table) {
		Error("Palette has wrong size: %v", len(b))
	}
	bi := 0
	pi := 0
	for i := 0; i < 256; i++ {
		p.table[pi] = b[bi]
		p.table[pi+1] = b[bi+1]
		p.table[pi+2] = b[bi+2]
		p.table[pi+3] = 255
		p.tableFullBright[pi+3] = 255
		p.tableNoBright[pi+3] = 255
		pi += 4
		bi += 3
	}
	// orig changed the last value to alpha 0?

	blend := 4 * 224
	// keep 0-223 black
	copy(p.tableFullBright[blend:], p.table[blend:])
	// keep 224-255 black
	copy(p.tableNoBright[:blend], p.table[:blend])

	copy(p.tableFullBrightFence[:], p.tableFullBright[:])
	copy(p.tableNoBrightFence[:], p.tableNoBright[:])

	p.table[256*4-1] = 0
	copy(p.tableConsoleChars[:], p.table[:])
	p.tableConsoleChars[3] = 0

	copy(p.tableFullBrightFence[255*4:], []uint8{0, 0, 0, 0})
	copy(p.tableNoBrightFence[255*4:], []uint8{0, 0, 0, 0})
}
