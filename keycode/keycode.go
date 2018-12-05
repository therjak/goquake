package keycode

const (
	TAB           = 9
	ENTER         = 13
	ESCAPE        = 27
	SPACE         = 32
	BACKSPACE     = 127
	UPARROW       = 128
	DOWNARROW     = 129
	LEFTARROW     = 130
	RIGHTARROW    = 131
	ALT           = 132
	CTRL          = 133
	SHIFT         = 134
	F1            = 135
	F2            = 136
	F3            = 137
	F4            = 138
	F5            = 139
	F6            = 140
	F7            = 141
	F8            = 142
	F9            = 143
	F10           = 144
	F11           = 145
	F12           = 146
	INS           = 147
	DEL           = 148
	PGDN          = 149
	PGUP          = 150
	HOME          = 151
	END           = 152
	KP_NUMLOCK    = 153
	KP_SLASH      = 154
	KP_STAR       = 155
	KP_MINUS      = 156
	KP_HOME       = 157
	KP_UPARROW    = 158
	KP_PGUP       = 159
	KP_PLUS       = 160
	KP_LEFTARROW  = 161
	KP_5          = 162
	KP_RIGHTARROW = 163
	KP_END        = 164
	KP_DOWNARROW  = 165
	KP_PGDN       = 166
	KP_ENTER      = 167
	KP_INS        = 168
	KP_DEL        = 169
	COMMAND       = 170
	MOUSE1        = 200
	MOUSE2        = 201
	MOUSE3        = 202
	JOY1          = 203
	JOY2          = 204
	JOY3          = 205
	JOY4          = 206
	AUX1          = 207
	AUX2          = 208
	AUX3          = 209
	AUX4          = 210
	AUX5          = 211
	AUX6          = 212
	AUX7          = 213
	AUX8          = 214
	AUX9          = 215
	AUX10         = 216
	AUX11         = 217
	AUX12         = 218
	AUX13         = 219
	AUX14         = 220
	AUX15         = 221
	AUX16         = 222
	AUX17         = 223
	AUX18         = 224
	AUX19         = 225
	AUX20         = 226
	AUX21         = 227
	AUX22         = 228
	AUX23         = 229
	AUX24         = 230
	AUX25         = 231
	AUX26         = 232
	AUX27         = 233
	AUX28         = 234
	AUX29         = 235
	AUX30         = 236
	AUX31         = 237
	AUX32         = 238
	MWHEELUP      = 239
	MWHEELDOWN    = 240
	MOUSE4        = 241
	MOUSE5        = 242
	LTHUMB        = 243
	RTHUMB        = 244
	LSHOULDER     = 245
	RSHOULDER     = 246
	ABUTTON       = 247
	BBUTTON       = 248
	XBUTTON       = 249
	YBUTTON       = 250
	LTRIGGER      = 251
	RTRIGGER      = 252
	PAUSE         = 255
)

var (
	s2k = map[string]int{
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

		"SEMICOLON": ';', // because a raw semicolon seperates commands

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

func reverseMap(m map[string]int) map[int]string {
	r := make(map[int]string)
	for k, v := range m {
		r[v] = k
	}
	return r
}

func KeyToString(k int) string {
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

func StringToKey(s string) int {
	if len(s) == 0 {
		return -1
	}
	if len(s) == 1 {
		return int(s[0])
	}
	v, ok := s2k[s]
	if ok {
		return v
	}
	return -1
}
