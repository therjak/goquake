package conlog

var (
	p   func(string, ...interface{})
	sp  func(string, ...interface{})
	dev float32
)

func SetPrintf(f func(string, ...interface{})) {
	p = f
}
func SetSafePrintf(f func(string, ...interface{})) {
	sp = f
}

func Printf(format string, v ...interface{}) {
	p(format, v...)
}

func SetDeveloper(v float32) {
	dev = v
}

func DPrintf(format string, v ...interface{}) {
	if dev != 0 {
		p(format, v...)
	}
}

func DPrintf2(format string, v ...interface{}) {
	if dev >= 2 {
		p(format, v...)
	}
}

func SafePrintf(format string, v ...interface{}) {
	sp(format, v...)
}

func Warning(format string, v ...interface{}) {
	SafePrintf("\x02Warning: ")
	Printf(format, v...)
}

func DWarning(format string, v ...interface{}) {
	if dev == 0 {
		return
	}
	SafePrintf("\x02Warning: ")
	Printf(format, v...)
}
