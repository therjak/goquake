// SPDX-License-Identifier: GPL-2.0-or-later

package conlog

import (
	"context"
	"log/slog"
)

var (
	p  func(string, ...interface{})
	sp func(string, ...interface{})
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

func DPrint(msg string, args ...any) {
	slog.Log(context.Background(), -4, msg, args...)
}

func SafePrintf(format string, v ...interface{}) {
	sp(format, v...)
}

func Warning(format string, v ...interface{}) {
	SafePrintf("\x02Warning: ")
	Printf(format, v...)
}

func DWarning(msg string, args ...any) {
	slog.Log(context.Background(), -2, msg, args...)
}
