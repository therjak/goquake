package quakelib

// void PR_ExecuteProgram(int p);
// int GetPRArgC();
// int GetPRXStatement();
import "C"

import (
	"quake/conlog"
	"quake/progs"
	"strings"
)

func PRExecuteProgram(p int32) {
	C.PR_ExecuteProgram(C.int(p))
}

func prArgC() int {
	return int(C.GetPRArgC())
}

func prXStatement() int {
	return int(C.GetPRXStatement())
}

func vmVarString(first int) string {
	var b strings.Builder

	for i := first; i < prArgC(); i++ {
		idx := progsdat.RawGlobalsI[progs.OffsetParm0+i*3]
		s := PRGetString(int(idx))
		if s == nil {
			conlog.DWarning("PF_VarString: nil string.\n")
			break
		}
		b.WriteString(*s)
	}
	if b.Len() > 255 {
		conlog.DWarning("PF_VarString: %d characters exceeds standard limit of 255.\n", b.Len())
	}
	return b.String()
}
