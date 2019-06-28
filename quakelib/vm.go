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

const (
	operatorDONE = iota
	operatorMUL_F
	operatorMUL_V
	operatorMUL_FV
	operatorMUL_VF
	operatorDIV_F
	operatorADD_F
	operatorADD_V
	operatorSUB_F
	operatorSUB_V

	operatorEQ_F
	operatorEQ_V
	operatorEQ_S
	operatorEQ_E
	operatorEQ_FNC

	operatorNE_F
	operatorNE_V
	operatorNE_S
	operatorNE_E
	operatorNE_FNC

	operatorLE
	operatorGE
	operatorLT
	operatorGT

	operatorLOAD_F
	operatorLOAD_V
	operatorLOAD_S
	operatorLOAD_ENT
	operatorLOAD_FLD
	operatorLOAD_FNC

	operatorADDRESS

	operatorSTORE_F
	operatorSTORE_V
	operatorSTORE_S
	operatorSTORE_ENT
	operatorSTORE_FLD
	operatorSTORE_FNC

	operatorSTOREP_F
	operatorSTOREP_V
	operatorSTOREP_S
	operatorSTOREP_ENT
	operatorSTOREP_FLD
	operatorSTOREP_FNC

	operatorRETURN
	operatorNOT_F
	operatorNOT_V
	operatorNOT_S
	operatorNOT_ENT
	operatorNOT_FNC
	operatorIF
	operatorIFNOT
	operatorCALL0
	operatorCALL1
	operatorCALL2
	operatorCALL3
	operatorCALL4
	operatorCALL5
	operatorCALL6
	operatorCALL7
	operatorCALL8
	operatorSTATE
	operatorGOTO
	operatorAND
	operatorOR

	operatorBITAND
	operatorBITOR
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
	prog       *progs.LoadedProg
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

func (v *virtualMachine) printStatement(s progs.Statement) {
	if int(s.Operator) < len(operationNames) {
		conlog.Printf("%10s ", operationNames[s.Operator])
	}

	if s.Operator == operatorIF || s.Operator == operatorIFNOT {
		conlog.Printf("%sbranch %d", v.prog.GlobalString(s.A), s.A)
	} else if s.Operator == operatorGOTO {
		conlog.Printf("branch %d", s.A)
	} else if d := s.Operator - operatorSTORE_F; d < 6 && d >= 0 {
		conlog.Printf("%s", v.prog.GlobalString(s.A))
		conlog.Printf("%s", v.prog.GlobalStringNoContents(s.B))
	} else {
		if s.A != 0 {
			conlog.Printf("%s", v.prog.GlobalString(s.B))
		}
		if s.B != 0 {
			conlog.Printf("%s", v.prog.GlobalString(s.B))
		}
		if s.C != 0 {
			conlog.Printf("%s", v.prog.GlobalStringNoContents(s.C))
		}
	}
	conlog.Printf("\n")
}

func (v *virtualMachine) printFunction(f *progs.Function) {
	if f == nil {
		conlog.Printf("<NO FUNCTION>\n")
	} else {
		file, _ := v.prog.String(f.SFile)
		name, _ := v.prog.String(f.SName)
		conlog.Printf("%12s : %s\n", file, name)
	}
}

func (v *virtualMachine) stackTrace() {
	v.printFunction(v.xfunction)
	if len(v.stack) == 0 {
		conlog.Printf("<NO STACK>\n")
		return
	}
	for i := len(v.stack) - 1; i >= 0; i-- {
		v.printFunction(v.stack[i].function)
	}
}

/*
func init() {
	cmd.AddCommand("profile", PR_Profile_f)
}

void PR_Profile_f(void) {
  int i, num;
  int pmax;
  dfunction_t *f, *best;

  if (!SV_Active()) return;

  num = 0;
  do {
    pmax = 0;
    best = NULL;
    for (i = 0; i < progs->numfunctions; i++) {
      f = &pr_functions[i];
      if (f->profile > pmax) {
        pmax = f->profile;
        best = f;
      }
    }
    if (best) {
      if (num < 10)
        Con_Printf("%7i %s\n", best->profile, PR_GetString(best->s_name));
      num++;
      best->profile = 0;
    }
  } while (best);
}

*/
// Aborts the currently executing function
func (v *virtualMachine) RunError(format string, a ...interface{}) {
	v.printStatement(v.prog.Statements[v.statement])
	v.stackTrace()

	conlog.Printf(format, a...)

	// dump the stack so host_error can shutdown functions
	v.stack = v.stack[:0]

	HostError("Program error")
}

//Returns the new program statement counter
func (v *virtualMachine) enterFunction(f *progs.Function) int32 {
	if len(v.stack) == cap(v.stack) {
		v.RunError("stack overflow")
	}
	v.stack = append(v.stack, stackElem{
		statement: v.statement,
		function:  v.xfunction,
	})

	// save off any locals that the new function steps on
	c := f.Locals
	if len(v.localStack)+int(c) > cap(v.localStack) {
		v.RunError("PR_ExecuteProgram: locals stack overflow\n")
	}
	for i := int32(0); i < c; i++ {
		v.localStack = append(v.localStack, v.prog.RawGlobalsI[f.ParmStart+i])
	}

	// copy parameters
	o := f.ParmStart
	for i := int32(0); i < f.NumParms; i++ {
		for j := byte(0); j < f.ParmSize[i]; j++ {
			v.prog.RawGlobalsI[o] = v.prog.RawGlobalsI[progs.OffsetParm0+i*3+int32(j)]
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
		v.RunError("PR_ExecuteProgram: locals stack underflow")
	}

	nl := len(v.localStack) - c
	for i := 0; i < c; i++ {
		v.prog.RawGlobalsI[int(v.xfunction.ParmStart)+i] = v.localStack[nl+i]
	}
	v.localStack = v.localStack[:nl]

	// up stack
	v.stack = v.stack[:len(v.stack)-1]
	v.xfunction = v.stack[len(v.stack)-1].function
	return v.stack[len(v.stack)-1].statement
}
