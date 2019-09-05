package quakelib

//float GetScreenConsoleCurrentHeight(void);
import "C"

type qconsole struct {
}

var (
	console qconsole
)

func (c *qconsole) currentHeight() int {
	return int(C.GetScreenConsoleCurrentHeight())
}
