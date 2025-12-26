// SPDX-License-Identifier: GPL-2.0-or-later

package quakelib

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"goquake/math/vec"
	"goquake/progs"
)

var errProgram = errors.New("Program error")

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

func (v *virtualMachine) funcName() string {
	id := v.xfunction.SName
	s, err := v.prog.String(int32(id))
	if err != nil {
		return ""
	}
	return s
}

func (v *virtualMachine) varString(first int) string {
	var b strings.Builder

	for i := first; i < v.argc; i++ {
		idx := v.prog.RawGlobalsI[progs.OffsetParm0+i*3]
		s, err := v.prog.String(idx)
		if err != nil {
			slog.Debug("PF_VarString: nil string")
			break
		}
		b.WriteString(s)
	}
	if b.Len() > 255 {
		slog.Debug("PF_VarString: characters exceeds standard limit of 255.", slog.Int("Len", b.Len()))
	}
	return b.String()
}

type stackElem struct {
	statement int32
	function  *progs.Function
}

type virtualMachine struct {
	xfunction  *progs.Function
	stack      []stackElem
	localStack []int32
	statement  int32
	trace      bool
	prog       *progs.LoadedProg
	argc       int
	builtins   []func(s *Server) error

	// only to prevent recursion
	changeLevelIssued bool
}

const (
	maxStackDepth = 1024
	maxLocalStack = 16384
)

var (
	vm *virtualMachine
)

func NewVirtualMachine() *virtualMachine {
	v := &virtualMachine{
		stack:      make([]stackElem, 0, maxStackDepth),
		localStack: make([]int32, 0, maxLocalStack),
	}
	v.builtins = []func(s *Server) error{
		v.fixme,
		v.makeVectors,   // void(entity e) makevectors		= #1
		v.setOrigin,     // void(entity e, vector o) setorigin	= #2
		v.setModel,      // void(entity e, string m) setmodel	= #3
		v.setSize,       // void(entity e, vector min, vector max) setsize	= #4
		v.fixme,         // void(entity e, vector min, vector max) setabssize	= #5
		v.doBreak,       // void() break				= #6
		v.random,        // float() random			= #7
		v.sound,         // void(entity e, float chan, string samp) sound	= #8
		v.normalize,     // vector(vector v) normalize		= #9
		v.terminalError, // void(string e) error			= #10
		v.objError,      // void(string e) objerror		= #11
		v.vlen,          // float(vector v) vlen			= #12
		v.vecToYaw,      // float(vector v) vectoyaw		= #13
		v.spawn,         // entity() spawn			= #14
		v.remove,        // void(entity e) remove		= #15
		v.traceline,     // float(vector v1, vector v2, float tryents) traceline = #16
		v.checkClient,   // entity() clientlist			= #17
		v.find,          // entity(entity start, .string fld, string match) find	= #18
		v.precacheSound, // void(string s) precache_sound	= #19
		v.precacheModel, // void(string s) precache_model	= #20
		v.stuffCmd,      // void(entity client, string s)stuffcmd	= #21
		v.findRadius,    // entity(vector org, float rad) findradius	= #22
		v.bprint,        // void(string s) bprint		= #23
		v.sprint,        // void(entity client, string s) sprint	= #24
		v.dprint,        // void(string s) dprint		= #25
		v.ftos,          // void(string s) ftos			= #26
		v.vtos,          // void(string s) vtos			= #27
		v.coredump, v.traceOn, v.traceOff,
		v.eprint,   // void(entity e) debug print an entire entity
		v.walkMove, // float(float yaw, float dist) walkmove
		v.fixme,    // float(float yaw, float dist) walkmove
		v.dropToFloor, v.lightStyle, v.rint, v.floor, v.ceil, v.fixme,
		v.checkBottom, v.pointContents, v.fixme, v.fabs, v.aim, v.cvar,
		v.localCmd, v.nextEnt, v.particle, v.changeYaw, v.fixme,
		v.vecToAngles,

		v.writeByte, v.writeChar, v.writeShort, v.writeLong, v.writeCoord,
		v.writeAngle, v.writeString, v.writeEntity,

		v.fixme, v.fixme, v.fixme, v.fixme, v.fixme, v.fixme, v.fixme,

		v.moveToGoal, v.precacheFile, v.makeStatic,

		v.changeLevel, v.fixme,

		v.cvarSet, v.centerPrint,

		v.ambientSound,

		v.precacheModel,
		v.precacheSound, // precache_sound2 is different only for qcc
		v.precacheFile,

		v.setSpawnParms,
		// 2021 release
		v.finaleFinished, // float() finaleFinished = # 79
		v.fixme,          // void localsound (entity client, string sample) = #80
		v.fixme,          // void draw_point (vector point, float colormap, float lifetime, float depthtest) = #81
		v.fixme,          // void draw_line (vector start, vector end, float colormap, float lifetime, float depthtest) = #82
		v.fixme,          // void draw_arrow (vector start, vector end, float colormap, float size, float lifetime, float depthtest) = #83
		v.fixme,          // void draw_ray (vector start, vector direction, float length, float colormap, float size, float lifetime, float depthtest) = #84
		v.fixme,          // void draw_circle (vector origin, float radius, float colormap, float lifetime, float depthtest) = #85
		v.fixme,          // void draw_bounds (vector min, vector max, float colormap, float lifetime, float depthtest) = #86
		v.fixme,          // void draw_worldtext (string s, vector origin, float size, float lifetime, float depthtest) = #87
		v.fixme,          // void draw_sphere (vector origin, float radius, float colormap, float lifetime, float depthtest) = #88
		v.fixme,          // void draw_cylinder (vector origin, float halfHeight, float radius, float colormap, float lifetime, float depthtest) = #89
		v.centerPrint,    // #90
		v.bprint,
		v.sprint,
	}
	return v
}

func (v *virtualMachine) printStatement(s progs.Statement) {
	name := "unknown"
	if int(s.Operator) < len(operationNames) {
		name = operationNames[s.Operator]
	}

	if s.Operator == operatorIF || s.Operator == operatorIFNOT {
		slog.Info(name,
			slog.String("branch", v.prog.GlobalString(s.A)),
			slog.Int("B", int(s.B)))
	} else if s.Operator == operatorGOTO {
		slog.Info(name, slog.Int("branch", int(s.A)))
	} else if d := s.Operator - operatorSTORE_F; d < 6 {
		slog.Info(name,
			slog.String("A", v.prog.GlobalString(s.A)),
			slog.String("B", v.prog.GlobalStringNoContents(s.B)))
	} else {
		a, b, c := "", "", ""
		if s.A != 0 {
			a = v.prog.GlobalString(s.A)
		}
		if s.B != 0 {
			b = v.prog.GlobalString(s.B)
		}
		if s.C != 0 {
			c = v.prog.GlobalStringNoContents(s.C)
		}
		slog.Info(name, slog.String("A", a), slog.String("B", b), slog.String("C", c))
	}
}

func (v *virtualMachine) printFunction(f *progs.Function) {
	if f == nil {
		slog.Warn("<NO FUNCTION>")
	} else {
		file, _ := v.prog.String(f.SFile)
		name, _ := v.prog.String(f.SName)
		slog.Info("FUNCTION", slog.String("file", file), slog.String("name", name))
	}
}

func (v *virtualMachine) stackTrace() {
	v.printFunction(v.xfunction)
	if len(v.stack) == 0 {
		slog.Warn("<NO STACK>")
		return
	}
	for i := len(v.stack) - 1; i >= 0; i-- {
		v.printFunction(v.stack[i].function)
	}
}

func (v *virtualMachine) abort() {
	v.printStatement(v.prog.Statements[v.statement])
	v.stackTrace()

	// dump the stack so host_error can shutdown functions
	v.stack = v.stack[:0]
}

// Returns the new program statement counter
func (v *virtualMachine) enterFunction(f *progs.Function) (int32, error) {
	if len(v.stack) == cap(v.stack) {
		slog.Error("stack overflow")
		v.abort()
		return 0, errProgram
	}
	v.stack = append(v.stack, stackElem{
		statement: v.statement,
		function:  v.xfunction,
	})

	// save off any locals that the new function steps on
	c := f.Locals
	if len(v.localStack)+int(c) > cap(v.localStack) {
		slog.Error("PR_ExecuteProgram: locals stack overflow")
		v.abort()
		return 0, errProgram
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
	return f.FirstStatement, nil
}

func (v *virtualMachine) leaveFunction() (int32, error) {
	if len(v.stack) == 0 {
		return 0, fmt.Errorf("prog stack underflow")
	}

	// Restore locals from the stack
	c := int(v.xfunction.Locals)
	if len(v.localStack) < c {
		slog.Error("PR_ExecuteProgram: locals stack underflow")
		v.abort()
		return 0, errProgram
	}

	nl := len(v.localStack) - c
	for i := 0; i < c; i++ {
		v.prog.RawGlobalsI[int(v.xfunction.ParmStart)+i] = v.localStack[nl+i]
	}
	v.localStack = v.localStack[:nl]

	// up stack
	top := v.stack[len(v.stack)-1]
	v.stack = v.stack[:len(v.stack)-1]
	v.xfunction = top.function
	return top.statement, nil
}

// The interpretation main loop
func (v *virtualMachine) ExecuteProgram(fnum int32, s *Server) error {
	if fnum == 0 || int(fnum) >= len(v.prog.Functions) {
		if v.prog.Globals.Self != 0 {
			s.edictPrint(int(v.prog.Globals.Self))
		}
		return fmt.Errorf("PR_ExecuteProgram: NULL function, %d", fnum)
	}

	f := &v.prog.Functions[fnum]

	v.trace = false

	// make a stack frame
	exitdepth := len(v.stack)

	currentStatement, err := v.enterFunction(f)
	if err != nil {
		return err
	}

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
	setOPCI := func(X int32) {
		v.prog.RawGlobalsI[st().C] = X
	}

	OPAV := func() vec.Vec3 {
		a := st().A
		return vec.Vec3{
			v.prog.RawGlobalsF[a],
			v.prog.RawGlobalsF[a+1],
			v.prog.RawGlobalsF[a+2]}
	}
	OPBV := func() vec.Vec3 {
		b := st().B
		return vec.Vec3{
			v.prog.RawGlobalsF[b],
			v.prog.RawGlobalsF[b+1],
			v.prog.RawGlobalsF[b+2]}
	}
	setOPBV := func(X vec.Vec3) {
		b := st().B
		v.prog.RawGlobalsF[b] = X[0]
		v.prog.RawGlobalsF[b+1] = X[1]
		v.prog.RawGlobalsF[b+2] = X[2]
	}
	setOPCV := func(X vec.Vec3) {
		c := st().C
		v.prog.RawGlobalsF[c] = X[0]
		v.prog.RawGlobalsF[c+1] = X[1]
		v.prog.RawGlobalsF[c+2] = X[2]
	}

	BOOL := func(X bool) float32 {
		if X {
			return 1
		}
		return 0
	}

	//hack to offset the first increment of currentStatement
	currentStatement--
	for {
		currentStatement++

		if v.trace {
			v.printStatement(v.prog.Statements[currentStatement])
		}

		switch st().Operator {
		case operatorADD_F:
			setOPCF(OPAF() + OPBF())
		case operatorADD_V:
			setOPCV(vec.Add(OPAV(), OPBV()))

		case operatorSUB_F:
			setOPCF(OPAF() - OPBF())
		case operatorSUB_V:
			setOPCV(vec.Sub(OPAV(), OPBV()))

		case operatorMUL_F:
			setOPCF(OPAF() * OPBF())
		case operatorMUL_V:
			setOPCF(vec.Dot(OPAV(), OPBV()))
		case operatorMUL_FV:
			setOPCV(vec.Scale(OPAF(), OPBV()))
		case operatorMUL_VF:
			setOPCV(vec.Scale(OPBF(), OPAV()))

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
			setOPCF(BOOL(OPAV() == vec.Vec3{0, 0, 0}))
		case operatorNOT_S:
			i := OPAI()
			_, err := v.prog.String(i)
			setOPCF(BOOL(i == 0 || err != nil))
		case operatorNOT_FNC:
			setOPCF(BOOL(OPAI() == 0))
		case operatorNOT_ENT:
			setOPCF(BOOL(OPAI() == 0))

		case operatorEQ_F:
			setOPCF(BOOL(OPAF() == OPBF()))
		case operatorEQ_V:
			setOPCF(BOOL(OPAV() == OPBV()))
		case operatorEQ_S:
			a := OPAI()
			sa, erra := v.prog.String(a)
			b := OPBI()
			sb, errb := v.prog.String(b)
			setOPCF(BOOL(
				(erra != nil && errb != nil) ||
					(erra == nil && errb == nil && sa == sb)))
		case operatorEQ_E:
			setOPCF(BOOL(OPAI() == OPBI()))
		case operatorEQ_FNC:
			setOPCF(BOOL(OPAI() == OPBI()))
		case operatorNE_F:
			setOPCF(BOOL(OPAF() != OPBF()))
		case operatorNE_V:
			setOPCF(BOOL(OPAV() != OPBV()))
		case operatorNE_S:
			a := OPAI()
			sa, erra := v.prog.String(a)
			b := OPBI()
			sb, errb := v.prog.String(b)
			setOPCF(BOOL(
				(erra != errb) ||
					(erra == nil && errb == nil && sa != sb)))
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
			setOPBV(OPAV())

		case operatorSTOREP_F,
			operatorSTOREP_ENT,
			operatorSTOREP_FLD, // integers
			operatorSTOREP_S,
			operatorSTOREP_FNC: // pointers
			o := OPBI()
			entvars.Set0RawI(o, OPAI())

		case operatorSTOREP_V:
			o := OPBI()
			value := OPAV()
			entvars.Set0RawF(o, value[0])
			entvars.Set0RawF(o+4, value[1])
			entvars.Set0RawF(o+8, value[2])

		case operatorADDRESS:
			ed := OPAI()
			if ed == 0 && s.state == ServerStateActive {
				v.statement = currentStatement
				slog.Error("assignment to world entity")
				v.abort()
				return errProgram
			}
			setOPCI(entvars.Address(OPAI(), OPBI()))

		case operatorLOAD_F,
			operatorLOAD_FLD,
			operatorLOAD_ENT,
			operatorLOAD_S,
			operatorLOAD_FNC:
			i := entvars.RawI(OPAI(), +OPBI())
			setOPCI(i)

		case operatorLOAD_V:
			ve := [3]float32{
				entvars.RawF(OPAI(), OPBI()),
				entvars.RawF(OPAI(), OPBI()+1),
				entvars.RawF(OPAI(), OPBI()+2),
			}
			setOPCV(ve)

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

		case operatorCALL0,
			operatorCALL1,
			operatorCALL2,
			operatorCALL3,
			operatorCALL4,
			operatorCALL5,
			operatorCALL6,
			operatorCALL7,
			operatorCALL8:
			v.statement = currentStatement
			v.argc = int(st().Operator) - operatorCALL0
			if OPAI() == 0 {
				slog.Error("NULL function")
				v.abort()
				return errProgram
			}
			newf := &v.prog.Functions[OPAI()]
			if newf.FirstStatement < 0 {
				// Built-in function
				i := int(-newf.FirstStatement)
				if i >= len(v.builtins) {
					slog.Error("Bad builtin call number", slog.Int("num", i))
					v.abort()
					return errProgram
				}
				if err := v.builtins[i](s); err != nil {
					return err
				}
			} else {
				// Normal function
				if s, err := v.enterFunction(newf); err != nil {
					return err
				} else {
					currentStatement = s - 1
				}
			}

		case operatorDONE, operatorRETURN:
			v.statement = currentStatement
			*(v.prog.Globals.Returnf()) = OPAV()
			if s, err := v.leaveFunction(); err != nil {
				return err
			} else {
				currentStatement = s
			}
			if len(v.stack) == exitdepth { // Done
				return nil
			}

		case operatorSTATE:
			ev := entvars.Get(int(v.prog.Globals.Self))
			ev.NextThink = v.prog.Globals.Time + 0.1
			ev.Frame = OPAF()
			ev.Think = OPBI()

		default:
			v.statement = currentStatement
			slog.Error("Bad opcode", slog.Int("opcode", int(v.prog.Statements[currentStatement].Operator)))
			v.abort()
			return errProgram
		}
	}
	return nil
}
