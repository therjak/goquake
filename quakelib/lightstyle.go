package quakelib

//extern int d_lightstylevalue[256];
import "C"

import (
	"fmt"

	"github.com/therjak/goquake/cvars"
	"github.com/therjak/goquake/net"
)

type lightStyle struct {
	average     int
	peak        int
	unprocessed string // only used for demo recording
	lightMap    []int
}

const (
	maxLightStyles = 64
)

//export ReadLightStyle
func ReadLightStyle() {
	err := readLightStyle(cls.inMessage)
	if err != nil {
		Error("svc_lightstyle: %v", err)
	}
}

var (
	lightStyles [maxLightStyles]lightStyle
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

func readLightStyle(msg *net.QReader) error {
	idx, err := msg.ReadByte()
	if err != nil {
		return err
	}
	if idx >= maxLightStyles {
		return fmt.Errorf("> MAX_LIGHTSTYLES")
	}
	style := &lightStyles[idx]
	str, err := msg.ReadString()
	if err != nil {
		return err
	}
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
	for i := 0; i < maxLightStyles; i++ {
		s := &lightStyles[i]
		if len(s.lightMap) == 0 {
			C.d_lightstylevalue[i] = 256
			continue
		}
		switch cvars.RFlatLightStyles.Value() {
		case 1:
			C.d_lightstylevalue[i] = C.int(s.average)
		case 2:
			C.d_lightstylevalue[i] = C.int(s.peak)
		default:
			C.d_lightstylevalue[i] = C.int(s.lightMap[idx%len(lightStyles[i].lightMap)])
		}
	}
}
