// SPDX-License-Identifier: GPL-2.0-or-later

package conlog

var (
	p func(string, ...interface{})
)

func SetPrintf(f func(string, ...interface{})) {
	p = f
}

func Printf(format string, v ...interface{}) {
	p(format, v...)
}
