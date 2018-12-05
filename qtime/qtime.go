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
