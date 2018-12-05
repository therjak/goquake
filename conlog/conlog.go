package conlog

import ()

var (
	p  func(string, ...interface{})
	sp func(string, ...interface{})
)

func SetPrintf(f func(string, ...interface{})) {
	p = f
}
func SetSavePrintf(f func(string, ...interface{})) {
	sp = f
}

func Printf(format string, v ...interface{}) {
	p(format, v...)
}

func SafePrintf(format string, v ...interface{}) {
	sp(format, v...)
}
