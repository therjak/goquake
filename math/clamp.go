// SPDX-License-Identifier: GPL-2.0-or-later
package math

func Clamp(min, val, max float64) float64 {
	if min > val {
		return min
	} else if max < val {
		return max
	}
	return val
}

func Clamp32(min, val, max float32) float32 {
	if min > val {
		return min
	} else if max < val {
		return max
	}
	return val
}

func ClampI(min, val, max int) int {
	if min > val {
		return min
	} else if max < val {
		return max
	}
	return val
}
