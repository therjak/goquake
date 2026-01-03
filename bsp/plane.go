// SPDX-License-Identifier: GPL-2.0-or-later

package bsp

import (
	"log/slog"
	"os"
	"runtime/debug"

	"goquake/math/vec"
)

func (p *Plane) BoxOnPlaneSide(mins, maxs vec.Vec3) int {
	if p.Type < 3 {
		if p.Dist <= mins[int(p.Type)] {
			return 1
		}
		if p.Dist >= maxs[int(p.Type)] {
			return 2
		}
		return 3
	}
	d1, d2 := func() (float32, float32) {
		n := p.Normal
		switch p.SignBits {
		case 0:
			d1 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*maxs[2]
			d2 := n[0]*mins[0] + n[1]*mins[1] + n[2]*mins[2]
			return d1, d2
		case 1:
			d1 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*maxs[2]
			d2 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*mins[2]
			return d1, d2
		case 2:
			d1 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*maxs[2]
			d2 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*mins[2]
			return d1, d2
		case 3:
			d1 := n[0]*mins[0] + n[1]*mins[1] + n[2]*maxs[2]
			d2 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*mins[2]
			return d1, d2
		case 4:
			d1 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*mins[2]
			d2 := n[0]*mins[0] + n[1]*mins[1] + n[2]*maxs[2]
			return d1, d2
		case 5:
			d1 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*mins[2]
			d2 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*maxs[2]
			return d1, d2
		case 6:
			d1 := n[0]*maxs[0] + n[1]*mins[1] + n[2]*mins[2]
			d2 := n[0]*mins[0] + n[1]*maxs[1] + n[2]*maxs[2]
			return d1, d2
		case 7:
			d1 := n[0]*mins[0] + n[1]*mins[1] + n[2]*mins[2]
			d2 := n[0]*maxs[0] + n[1]*maxs[1] + n[2]*maxs[2]
			return d1, d2
		default:
			debug.PrintStack()
			slog.Error("BoxOnPlaneSide: Bad signbits")
			os.Exit(1)
			return 0, 0
		}
	}()
	sides := 0
	if d1 >= p.Dist {
		sides = 1
	}
	if d2 < p.Dist {
		sides |= 2
	}
	return sides
}
