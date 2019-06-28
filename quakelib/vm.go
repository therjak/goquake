package quakelib

// void PR_ExecuteProgram(int p);
// int GetPRArgC();
// int GetPRXStatement();
// int GetPRXFuncName();
// void SetPRTrace(int t);
import "C"

import (
	"log"
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
	return f.FirstStatement
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

//  The interpretation main loop
func (v *virtualMachine) ExecuteProgram(fnum int32) {
	/*
	  eval_t *ptr;
	  dstatement_t *st;
	  dfunction_t *f, *newf;
	  int ed;
	*/

	if fnum == 0 || int(fnum) >= len(v.prog.Functions) {
		if v.prog.Globals.Self != 0 {
			edictPrint(int(v.prog.Globals.Self))
		}
		log.Printf("PR_ExecuteProgram %d", fnum)
		HostError("PR_ExecuteProgram: NULL function")
	}

	f := &v.prog.Functions[fnum]

	v.trace = false

	// make a stack frame
	exitdepth := len(v.stack)

	currentStatement := v.enterFunction(f)

	st := func() *progs.Statement {
		return &v.prog.Statements[currentStatement]
	}

	OPAF := func() float32 {
		return v.prog.RawGlobalsF[st().A]
	}
	OPBF := func() float32 {
		return v.prog.RawGlobalsF[st().B]
	}
	setOPCF := func(X float32) {
		v.prog.RawGlobalsF[st().C] = X
	}

	OPAI := func() int32 {
		return v.prog.RawGlobalsI[st().A]
	}
	OPBI := func() int32 {
		return v.prog.RawGlobalsI[st().B]
	}
	setOPBI := func(X int32) {
		v.prog.RawGlobalsI[st().B] = X
	}
	//SOPCI := func(X int32) {
	//	v.prog.RawGlobalsI[st().C] = X
	//}

	OPAV1 := func() float32 {
		return v.prog.RawGlobalsF[st().A]
	}
	OPAV2 := func() float32 {
		return v.prog.RawGlobalsF[st().A+1]
	}
	OPAV3 := func() float32 {
		return v.prog.RawGlobalsF[st().A+2]
	}
	OPBV1 := func() float32 {
		return v.prog.RawGlobalsF[st().B]
	}
	OPBV2 := func() float32 {
		return v.prog.RawGlobalsF[st().B+1]
	}
	OPBV3 := func() float32 {
		return v.prog.RawGlobalsF[st().B+2]
	}
	setOPBV1 := func(X float32) {
		v.prog.RawGlobalsF[st().B] = X
	}
	setOPBV2 := func(X float32) {
		v.prog.RawGlobalsF[st().B+1] = X
	}
	setOPBV3 := func(X float32) {
		v.prog.RawGlobalsF[st().B+2] = X
	}
	setOPCV1 := func(X float32) {
		v.prog.RawGlobalsF[st().C] = X
	}
	setOPCV2 := func(X float32) {
		v.prog.RawGlobalsF[st().C+1] = X
	}
	setOPCV3 := func(X float32) {
		v.prog.RawGlobalsF[st().C+2] = X
	}

	BOOL := func(X bool) float32 {
		if X {
			return 1
		}
		return 0
	}

	startprofile := int32(0)
	profile := int32(0)

	//hack to offset the first increment of currentStatement
	currentStatement--
	for {
		currentStatement++
		profile++
		if profile > 100000 {
			v.statement = currentStatement - int32(len(v.prog.Statements))
			v.RunError("runaway loop error")
		}

		if v.trace {
			v.printStatement(v.prog.Statements[currentStatement])
		}

		switch st().Operator {
		case operatorADD_F:
			setOPCF(OPAF() + OPBF())
		case operatorADD_V:
			setOPCV1(OPAV1() + OPBV1())
			setOPCV2(OPAV2() + OPBV2())
			setOPCV3(OPAV3() + OPBV3())

		case operatorSUB_F:
			setOPCF(OPAF() - OPBF())
		case operatorSUB_V:
			setOPCV1(OPAV1() - OPBV1())
			setOPCV2(OPAV2() - OPBV2())
			setOPCV3(OPAV3() - OPBV3())

		case operatorMUL_F:
			setOPCF(OPAF() * OPBF())
		case operatorMUL_V:
			setOPCF(
				OPAV1()*OPBV1() +
					OPAV2()*OPBV2() +
					OPAV3()*OPBV3())
		case operatorMUL_FV:
			setOPCV1(OPAF() * OPBV1())
			setOPCV2(OPAF() * OPBV2())
			setOPCV3(OPAF() * OPBV3())
		case operatorMUL_VF:
			setOPCV1(OPBF() * OPAV1())
			setOPCV2(OPBF() * OPAV2())
			setOPCV3(OPBF() * OPAV3())

		case operatorDIV_F:
			setOPCF(OPAF() / OPBF())

		case operatorBITAND:
			// This hurts
			r := (int(OPAF()) & int(OPBF()))
			setOPCF(float32(r))
		case operatorBITOR:
			// This hurts
			r := (int(OPAF()) | int(OPBF()))
			setOPCF(float32(r))

		case operatorGE:
			setOPCF(BOOL(OPAF() >= OPBF()))
		case operatorLE:
			setOPCF(BOOL(OPAF() <= OPBF()))
		case operatorGT:
			setOPCF(BOOL(OPAF() > OPBF()))
		case operatorLT:
			setOPCF(BOOL(OPAF() < OPBF()))
		case operatorAND:
			setOPCF(BOOL((OPAF() != 0) && (OPBF() != 0)))
		case operatorOR:
			setOPCF(BOOL((OPAF() != 0) || (OPBF() != 0)))

		case operatorNOT_F:
			setOPCF(BOOL(OPAF() == 0))
		case operatorNOT_V:
			setOPCF(BOOL(
				(OPAV1() == 0) &&
					(OPAV2() == 0) &&
					(OPAV3() == 0)))
			/*
			   case OP_NOT_S:
			     setOPCF(!OPAI || !*PR_GetString(OPAI));
			*/
		case operatorNOT_FNC:
			setOPCF(BOOL(OPAI() == 0))
		case operatorNOT_ENT:
			setOPCF(BOOL(OPAI() == 0))

		case operatorEQ_F:
			setOPCF(BOOL(OPAF() == OPBF()))
		case operatorEQ_V:
			setOPCF(BOOL(
				(OPAV1() == OPBV1()) &&
					(OPAV2() == OPBV2()) &&
					(OPAV3() == OPBV3())))
			/*
			   case OP_EQ_S:
			     setOPCF(!strcmp(PR_GetString(OPAI), PR_GetString(OPBI)));
			*/
		case operatorEQ_E:
			setOPCF(BOOL(OPAI() == OPBI()))
		case operatorEQ_FNC:
			setOPCF(BOOL(OPAI() == OPBI()))
		case operatorNE_F:
			setOPCF(BOOL(OPAF() != OPBF()))
		case operatorNE_V:
			setOPCF(BOOL(
				(OPAV1() != OPBV1()) ||
					(OPAV2() != OPBV2()) ||
					(OPAV3() != OPBV3())))
			/*
			   case operatorNE_S:
			     SOPCF(strcmp(PR_GetString(OPAI), PR_GetString(OPBI)));
			*/
		case operatorNE_E:
			setOPCF(BOOL(OPAI() != OPBI()))
		case operatorNE_FNC:
			setOPCF(BOOL(OPAI() != OPBI()))

		case operatorSTORE_F,
			operatorSTORE_ENT,
			operatorSTORE_FLD, // integers
			operatorSTORE_S,
			operatorSTORE_FNC: // pointers
			setOPBI(OPAI())
		case operatorSTORE_V:
			setOPBV1(OPAV1())
			setOPBV2(OPAV2())
			setOPBV3(OPAV3())

			/*
			   case OP_STOREP_F:
			   case OP_STOREP_ENT:
			   case OP_STOREP_FLD:  // integers
			   case OP_STOREP_S:
			   case OP_STOREP_FNC:  // pointers
			     ptr = (eval_t *)((byte *)EVars(0) + OPBI);
			     ptr->_int = OPAI;
			     break;
			   case OP_STOREP_V:
			     ptr = (eval_t *)((byte *)EVars(0) + OPBI);
			     ptr->vector[0] = OPAV1;
			     ptr->vector[1] = OPAV2;
			     ptr->vector[2] = OPAV3;
			     break;

			   case OP_ADDRESS:
			     ed = OPAI;
			     if (ed == 0 && SV_State() == ss_active) {
			       pr_xstatement = st - pr_statements;
			       PR_RunError("assignment to world entity");
			     }
			     SOPCI((byte *)((int *)EVars(OPAI) + OPBI) - (byte *)EVars(0));
			     break;

			   case OP_LOAD_F:
			   case OP_LOAD_FLD:
			   case OP_LOAD_ENT:
			   case OP_LOAD_S:
			   case OP_LOAD_FNC:
			     SOPCI(((eval_t *)((int *)EVars(OPAI) + OPBI))->_int);
			     break;

			   case OP_LOAD_V:
			     ptr = (eval_t *)((int *)EVars(OPAI) + OPBI);
			     SOPCV1(ptr->vector[0]);
			     SOPCV2(ptr->vector[1]);
			     SOPCV3(ptr->vector[2]);
			     break;
			*/
		case operatorIFNOT:
			if OPAI() == 0 {
				currentStatement += int32(st().B) - 1 // -1 to offset the st++
			}

		case operatorIF:
			if OPAI() != 0 {
				currentStatement += int32(st().B) - 1 // -1 to offset the st++
			}

		case operatorGOTO:
			currentStatement += int32(st().A) - 1 // -1 to offset the st++
			/*
			   case OP_CALL0:
			   case OP_CALL1:
			   case OP_CALL2:
			   case OP_CALL3:
			   case OP_CALL4:
			   case OP_CALL5:
			   case OP_CALL6:
			   case OP_CALL7:
			   case OP_CALL8:
			     pr_xfunction->profile += profile - startprofile;
			     startprofile = profile;
			     pr_xstatement = st - pr_statements;
			     pr_argc = st->op - OP_CALL0;
			     if (!OPAI) PR_RunError("NULL function");
			     newf = &pr_functions[OPAI];
			     if (newf->first_statement < 0) {  // Built-in function
			       int i = -newf->first_statement;
			       if (i >= pr_numbuiltins) PR_RunError("Bad builtin call number %d", i);
			       pr_builtins[i]();
			       break;
			     }
			     // Normal function
			     st = &pr_statements[PR_EnterFunction(newf)];
			     break;
			*/
		case operatorDONE, operatorRETURN:
			v.xfunction.Profile += profile - startprofile
			startprofile = profile
			v.statement = currentStatement - int32(len(v.prog.Statements))
			*(v.prog.Globals.Returnf()) = [3]float32{OPAV1(), OPAV2(), OPAV3()}
			currentStatement = v.leaveFunction()
			if len(v.stack) == exitdepth { // Done
				return
			}
		case operatorSTATE:
			/*
			   EVars(Pr_global_struct_self())->nextthink =
			       Pr_global_struct_time() + 0.1;
			   EVars(Pr_global_struct_self())->frame = OPAF;
			   EVars(Pr_global_struct_self())->think = OPBI;
			*/
		default:
			v.statement = currentStatement - int32(len(v.prog.Statements))
			v.RunError("Bad opcode %d", v.prog.Statements[currentStatement].Operator)
		}
	}
}
