// SPDX-License-Identifier: GPL-2.0-or-later

package conlog

var (
	p func(string, ...any)
)

func SetPrintf(f func(string, ...any)) {
	p = f
}

func Printf(format string, v ...any) {
	p(format, v...)
}
