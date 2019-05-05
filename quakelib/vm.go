package quakelib

// void PR_ExecuteProgram(int p);
import "C"

func PRExecuteProgram(p int32) {
	C.PR_ExecuteProgram(C.int(p))
}
