package quakelib

// void PR_ExecuteProgram(int p);
// int GetPRArgC();
// int GetPRXStatement();
// int GetPRXFuncName();
// void SetPRTrace(int t);
import "C"

import (
	"quake/conlog"
	"quake/progs"
	"strings"
)

var (
	operationNames = []string{
		"DONE",
		"MUL_F", "MUL_V", "MUL_FV", "MUL_VF",
		"DIV",
		"ADD_F", "ADD_V",
		"SUB_F", "SUB_V",
		"EQ_F", "EQ_V", "EQ_S", "EQ_E", "EQ_FNC",
		"NE_F", "NE_V", "NE_S", "NE_E", "NE_FNC",
		"LE", "GE", "LT", "GT",
		"INDIRECT", "INDIRECT", "INDIRECT", "INDIRECT", "INDIRECT",
		"INDIRECT",
		"ADDRESS",
		"STORE_F", "STORE_V", "STORE_S", "STORE_ENT", "STORE_FLD",
		"STORE_FNC",
		"STOREP_F", "STOREP_V", "STOREP_S", "STOREP_ENT", "STOREP_FLD",
		"STOREP_FNC",
		"RETURN",
		"NOT_F", "NOT_V", "NOT_S", "NOT_ENT", "NOT_FNC",
		"IF", "IFNOT",
		"CALL0", "CALL1", "CALL2", "CALL3", "CALL4",
		"CALL5", "CALL6", "CALL7", "CALL8",
		"STATE",
		"GOTO",
		"AND", "OR",
		"BITAND", "BITOR"}
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

func vmFuncName() string {
	id := C.GetPRXFuncName()
	s, err := progsdat.String(int32(id))
	if err != nil {
		return ""
	}
	return s
}

func vmVarString(first int) string {
	var b strings.Builder

	for i := first; i < prArgC(); i++ {
		idx := progsdat.RawGlobalsI[progs.OffsetParm0+i*3]
		s, err := progsdat.String(idx)
		if err != nil {
			conlog.DWarning("PF_VarString: nil string.\n")
			break
		}
		b.WriteString(s)
	}
	if b.Len() > 255 {
		conlog.DWarning("PF_VarString: %d characters exceeds standard limit of 255.\n", b.Len())
	}
	return b.String()
}

func vmTraceOn() {
	C.SetPRTrace(1)
}

func vmTraceOff() {
	C.SetPRTrace(0)
}

type stackElem struct {
	statement int32
	function  *progs.Function
}

type virtualMachine struct {
	xfunction  *progs.Function
	stack      []stackElem // len(stack) == pr_depth
	localStack []int32     // len(localStack) == localstack_used
	statement  int32
	trace      bool
}

const (
	maxStackDepth = 32
	maxLocalStack = 2024
)

var (
	vm = virtualMachine{
		stack:      make([]stackElem, 0, 32),
		localStack: make([]int32, 0, 2024),
	}
)

//Returns the new program statement counter
func (v *virtualMachine) enterFunction(f *progs.Function) int32 {
	if len(v.stack) == cap(v.stack) {
		runError("stack overflow")
	}
	v.stack = append(v.stack, stackElem{
		statement: v.statement,
		function:  f,
	})

	// save off any locals that the new function steps on
	c := f.Locals
	if len(v.localStack)+int(c) > cap(v.localStack) {
		runError("PR_ExecuteProgram: locals stack overflow\n")
	}
	for i := int32(0); i < c; i++ {
		v.localStack = append(v.localStack, progsdat.RawGlobalsI[f.ParmStart+i])
	}

	// copy parameters
	o := f.ParmStart
	for i := int32(0); i < f.NumParms; i++ {
		for j := byte(0); j < f.ParmSize[i]; j++ {
			progsdat.RawGlobalsI[o] = progsdat.RawGlobalsI[progs.OffsetParm0+i*3+int32(j)]
			o++
		}
	}

	v.xfunction = f
	return f.FirstStatement - 1 // offset the s++ -- THERJAK: What s++
}

func (v *virtualMachine) leaveFunction() int32 {
	if len(v.stack) == 0 {
		HostError("prog stack underflow")
	}

	// Restore locals from the stack
	c := int(v.xfunction.Locals)
	if len(v.localStack) < c {
		runError("PR_ExecuteProgram: locals stack underflow")
	}

	nl := len(v.localStack) - c
	for i := 0; i < c; i++ {
		progsdat.RawGlobalsI[int(v.xfunction.ParmStart)+i] = v.localStack[nl+i]
	}
	v.localStack = v.localStack[:nl]

	// up stack
	v.stack = v.stack[:len(v.stack)-1]
	v.xfunction = v.stack[len(v.stack)-1].function
	return v.stack[len(v.stack)-1].statement
}
