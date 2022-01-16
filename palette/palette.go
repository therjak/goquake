// SPDX-License-Identifier: GPL-2.0-or-later
package palette

import (
	"fmt"
	"goquake/filesystem"
)

var (
	Table                [256 * 4]uint8
	TableFullBright      [256 * 4]uint8
	TableFullBrightFence [256 * 4]uint8
	TableNoBright        [256 * 4]uint8
	TableNoBrightFence   [256 * 4]uint8
	TableConsoleChars    [256 * 4]uint8
)

func Init() error {
	b, err := filesystem.GetFileContents("gfx/palette.lmp")
	if err != nil {
		return fmt.Errorf("Couln't load gfx/palette.lmp")
	}
	// b is rgb 8bit, we want rgba float32
	if 4*len(b) != 3*len(Table) {
		return fmt.Errorf("Palette has wrong size: %v", len(b))
	}
	bi := 0
	pi := 0
	for i := 0; i < 256; i++ {
		Table[pi] = b[bi]
		Table[pi+1] = b[bi+1]
		Table[pi+2] = b[bi+2]
		Table[pi+3] = 255
		TableFullBright[pi+3] = 255
		TableNoBright[pi+3] = 255
		pi += 4
		bi += 3
	}
	// orig changed the last value to alpha 0?

	blend := 4 * 224
	// keep 0-223 black
	copy(TableFullBright[blend:], Table[blend:])
	// keep 224-255 black
	copy(TableNoBright[:blend], Table[:blend])

	copy(TableFullBrightFence[:], TableFullBright[:])
	copy(TableNoBrightFence[:], TableNoBright[:])

	Table[256*4-1] = 0
	copy(TableConsoleChars[:], Table[:])
	TableConsoleChars[3] = 0

	copy(TableFullBrightFence[255*4:], []uint8{0, 0, 0, 0})
	copy(TableNoBrightFence[255*4:], []uint8{0, 0, 0, 0})
	return nil
}
