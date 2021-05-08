// SPDX-License-Identifier: GPL-2.0-or-later

package qtime

import (
	"time"
)

var (
	startTime = time.Now()
)

func QTime() time.Duration {
	return time.Now().Sub(startTime)
}
