// SPDX-License-Identifier: GPL-2.0-or-later

package keycode

type KeyCode int

type KeyCodeSlice []KeyCode

func (p KeyCodeSlice) Len() int           { return len(p) }
func (p KeyCodeSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p KeyCodeSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

const (
	TAB           = KeyCode(9)
	ENTER         = KeyCode(13)
	ESCAPE        = KeyCode(27)
	SPACE         = KeyCode(32)
	BACKSPACE     = KeyCode(127)
	UPARROW       = KeyCode(128)
	DOWNARROW     = KeyCode(129)
	LEFTARROW     = KeyCode(130)
	RIGHTARROW    = KeyCode(131)
	ALT           = KeyCode(132)
	CTRL          = KeyCode(133)
	SHIFT         = KeyCode(134)
	F1            = KeyCode(135)
	F2            = KeyCode(136)
	F3            = KeyCode(137)
	F4            = KeyCode(138)
	F5            = KeyCode(139)
	F6            = KeyCode(140)
	F7            = KeyCode(141)
	F8            = KeyCode(142)
	F9            = KeyCode(143)
	F10           = KeyCode(144)
	F11           = KeyCode(145)
	F12           = KeyCode(146)
	INS           = KeyCode(147)
	DEL           = KeyCode(148)
	PGDN          = KeyCode(149)
	PGUP          = KeyCode(150)
	HOME          = KeyCode(151)
	END           = KeyCode(152)
	KP_NUMLOCK    = KeyCode(153)
	KP_SLASH      = KeyCode(154)
	KP_STAR       = KeyCode(155)
	KP_MINUS      = KeyCode(156)
	KP_HOME       = KeyCode(157)
	KP_UPARROW    = KeyCode(158)
	KP_PGUP       = KeyCode(159)
	KP_PLUS       = KeyCode(160)
	KP_LEFTARROW  = KeyCode(161)
	KP_5          = KeyCode(162)
	KP_RIGHTARROW = KeyCode(163)
	KP_END        = KeyCode(164)
	KP_DOWNARROW  = KeyCode(165)
	KP_PGDN       = KeyCode(166)
	KP_ENTER      = KeyCode(167)
	KP_INS        = KeyCode(168)
	KP_DEL        = KeyCode(169)
	COMMAND       = KeyCode(170)
	MOUSE1        = KeyCode(200)
	MOUSE2        = KeyCode(201)
	MOUSE3        = KeyCode(202)
	JOY1          = KeyCode(203)
	JOY2          = KeyCode(204)
	JOY3          = KeyCode(205)
	JOY4          = KeyCode(206)
	AUX1          = KeyCode(207)
	AUX2          = KeyCode(208)
	AUX3          = KeyCode(209)
	AUX4          = KeyCode(210)
	AUX5          = KeyCode(211)
	AUX6          = KeyCode(212)
	AUX7          = KeyCode(213)
	AUX8          = KeyCode(214)
	AUX9          = KeyCode(215)
	AUX10         = KeyCode(216)
	AUX11         = KeyCode(217)
	AUX12         = KeyCode(218)
	AUX13         = KeyCode(219)
	AUX14         = KeyCode(220)
	AUX15         = KeyCode(221)
	AUX16         = KeyCode(222)
	AUX17         = KeyCode(223)
	AUX18         = KeyCode(224)
	AUX19         = KeyCode(225)
	AUX20         = KeyCode(226)
	AUX21         = KeyCode(227)
	AUX22         = KeyCode(228)
	AUX23         = KeyCode(229)
	AUX24         = KeyCode(230)
	AUX25         = KeyCode(231)
	AUX26         = KeyCode(232)
	AUX27         = KeyCode(233)
	AUX28         = KeyCode(234)
	AUX29         = KeyCode(235)
	AUX30         = KeyCode(236)
	AUX31         = KeyCode(237)
	AUX32         = KeyCode(238)
	MWHEELUP      = KeyCode(239)
	MWHEELDOWN    = KeyCode(240)
	MOUSE4        = KeyCode(241)
	MOUSE5        = KeyCode(242)
	LTHUMB        = KeyCode(243)
	RTHUMB        = KeyCode(244)
	LSHOULDER     = KeyCode(245)
	RSHOULDER     = KeyCode(246)
	ABUTTON       = KeyCode(247)
	BBUTTON       = KeyCode(248)
	XBUTTON       = KeyCode(249)
	YBUTTON       = KeyCode(250)
	LTRIGGER      = KeyCode(251)
	RTRIGGER      = KeyCode(252)
	PAUSE         = KeyCode(255)
)

var (
	s2k = map[string]KeyCode{
		"TAB":        TAB,
		"ENTER":      ENTER,
		"ESCAPE":     ESCAPE,
		"SPACE":      SPACE,
		"BACKSPACE":  BACKSPACE,
		"UPARROW":    UPARROW,
		"DOWNARROW":  DOWNARROW,
		"LEFTARROW":  LEFTARROW,
		"RIGHTARROW": RIGHTARROW,

		"ALT":   ALT,
		"CTRL":  CTRL,
		"SHIFT": SHIFT,

		//	{"KP_NUMLOCK", K_KP_NUMLOCK},
		"KP_SLASH":      KP_SLASH,
		"KP_STAR":       KP_STAR,
		"KP_MINUS":      KP_MINUS,
		"KP_HOME":       KP_HOME,
		"KP_UPARROW":    KP_UPARROW,
		"KP_PGUP":       KP_PGUP,
		"KP_PLUS":       KP_PLUS,
		"KP_LEFTARROW":  KP_LEFTARROW,
		"KP_5":          KP_5,
		"KP_RIGHTARROW": KP_RIGHTARROW,
		"KP_END":        KP_END,
		"KP_DOWNARROW":  KP_DOWNARROW,
		"KP_PGDN":       KP_PGDN,
		"KP_ENTER":      KP_ENTER,
		"KP_INS":        KP_INS,
		"KP_DEL":        KP_DEL,

		"F1":  F1,
		"F2":  F2,
		"F3":  F3,
		"F4":  F4,
		"F5":  F5,
		"F6":  F6,
		"F7":  F7,
		"F8":  F8,
		"F9":  F9,
		"F10": F10,
		"F11": F11,
		"F12": F12,

		"INS":  INS,
		"DEL":  DEL,
		"PGDN": PGDN,
		"PGUP": PGUP,
		"HOME": HOME,
		"END":  END,

		"COMMAND": COMMAND,

		"MOUSE1": MOUSE1,
		"MOUSE2": MOUSE2,
		"MOUSE3": MOUSE3,
		"MOUSE4": MOUSE4,
		"MOUSE5": MOUSE5,

		"JOY1": JOY1,
		"JOY2": JOY2,
		"JOY3": JOY3,
		"JOY4": JOY4,

		"AUX1":  AUX1,
		"AUX2":  AUX2,
		"AUX3":  AUX3,
		"AUX4":  AUX4,
		"AUX5":  AUX5,
		"AUX6":  AUX6,
		"AUX7":  AUX7,
		"AUX8":  AUX8,
		"AUX9":  AUX9,
		"AUX10": AUX10,
		"AUX11": AUX11,
		"AUX12": AUX12,
		"AUX13": AUX13,
		"AUX14": AUX14,
		"AUX15": AUX15,
		"AUX16": AUX16,
		"AUX17": AUX17,
		"AUX18": AUX18,
		"AUX19": AUX19,
		"AUX20": AUX20,
		"AUX21": AUX21,
		"AUX22": AUX22,
		"AUX23": AUX23,
		"AUX24": AUX24,
		"AUX25": AUX25,
		"AUX26": AUX26,
		"AUX27": AUX27,
		"AUX28": AUX28,
		"AUX29": AUX29,
		"AUX30": AUX30,
		"AUX31": AUX31,
		"AUX32": AUX32,

		"PAUSE": PAUSE,

		"MWHEELUP":   MWHEELUP,
		"MWHEELDOWN": MWHEELDOWN,

		"SEMICOLON": ';', // because a raw semicolon separates commands

		"BACKQUOTE": '`', // because a raw backquote may toggle the console
		"TILDE":     '~', // because a raw tilde may toggle the console

		"LTHUMB":    LTHUMB,
		"RTHUMB":    RTHUMB,
		"LSHOULDER": LSHOULDER,
		"RSHOULDER": RSHOULDER,
		"ABUTTON":   ABUTTON,
		"BBUTTON":   BBUTTON,
		"XBUTTON":   XBUTTON,
		"YBUTTON":   YBUTTON,
		"LTRIGGER":  LTRIGGER,
		"RTRIGGER":  RTRIGGER,
	}
	k2s = reverseMap(s2k)
)

func reverseMap(m map[string]KeyCode) map[KeyCode]string {
	r := make(map[KeyCode]string)
	for k, v := range m {
		r[v] = k
	}
	return r
}

func KeyToString(k KeyCode) string {
	if k == -1 {
		return "<KEY NOT FOUND>"
	}
	if k > 32 && k < 127 {
		return string(k)
	}
	s, ok := k2s[k]
	if ok {
		return s
	}
	return "<UNKNOWN KEYNUM>"
}

func StringToKey(s string) KeyCode {
	if len(s) == 0 {
		return -1
	}
	if len(s) == 1 {
		return KeyCode(s[0])
	}
	v, ok := s2k[s]
	if ok {
		return v
	}
	return -1
}
