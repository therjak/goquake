// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//extern int d_lightstylevalue[256];
import "C"

import (
	"fmt"

	"goquake/bsp"
	"goquake/cvars"
)

type lightStyle struct {
	average     int
	peak        int
	unprocessed string // only used for demo recording
	lightMap    []int
}

var (
	lightStyleValues bsp.LightStyles
	lightStyles      [bsp.MaxLightStyles]lightStyle
)

func clearLightStyles() {
	for i := 0; i < len(lightStyles); i++ {
		lightStyles[i] = lightStyle{
			average: 13 * 22,
			peak:    13 * 22,
		}
	}
}

func avgPeak(d []int) (int, int) {
	if len(d) == 0 {
		return 13 * 22, 13 * 22
	}
	s := 0
	m := 0
	for _, v := range d {
		s += v
		if v > m {
			m = v
		}
	}
	return s / len(d), m
}

func readLightStyle(idx int32, str string) error {
	if idx >= bsp.MaxLightStyles {
		return fmt.Errorf("> MAX_LIGHTSTYLES")
	}
	style := &lightStyles[idx]
	style.unprocessed = str
	style.lightMap = make([]int, len(str))
	// we read content with bytes x : 'a' <= x <= 'z'
	// and shift it to zero based and scaled by 22
	for i := 0; i < len(str); i++ {
		style.lightMap[i] = (int(str[i]) - int('a')) * 22
	}
	style.average, style.peak = avgPeak(style.lightMap)
	return nil
}

//export R_AnimateLight
func R_AnimateLight() {
	idx := int(cl.time * 10)
	for i := 0; i < bsp.MaxLightStyles; i++ {
		s := &lightStyles[i]
		if len(s.lightMap) == 0 {
			lightStyleValues[i] = 256
			C.d_lightstylevalue[i] = 256
			continue
		}
		switch cvars.RFlatLightStyles.Value() {
		case 1:
			lightStyleValues[i] = s.average
		case 2:
			lightStyleValues[i] = s.peak
		default:
			lightStyleValues[i] = s.lightMap[idx%len(lightStyles[i].lightMap)]
		}
		C.d_lightstylevalue[i] = C.int(lightStyleValues[i])
	}
}
