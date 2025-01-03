// SPDX-License-Identifier: GPL-2.0-or-later

package math

type Number interface {
	int64 | float64 | float32 | int
}

func Clamp[K Number](min, val, max K) K {
	if min > val {
		return min
	} else if max < val {
		return max
	}
	return val
}
