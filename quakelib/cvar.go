// SPDX-License-Identifier: GPL-2.0-or-later
package quakelib

//struct cvar_s;
//typedef void (*cvarcallback_t)(struct cvar_s*);
//void CallCvarCallback(int id, cvarcallback_t func);
import "C"
import (
	"fmt"
	"io"
	"log"

	"goquake/cmd"
	"goquake/conlog"
	"goquake/cvar"
)

//export CvarGetValue
func CvarGetValue(id C.int) C.float {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return 0
	}
	return C.float(cv.Value())
}

//export CvarGetString
func CvarGetString(id C.int) *C.char {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return nil
	}
	return C.CString(cv.String())
}

func CvarVariableValue(n string) float32 {
	if cv, ok := cvar.Get(n); ok {
		return cv.Value()
	}
	return 0
}

func CvarReset(n string) {
	if cv, ok := cvar.Get(n); ok {
		cv.Reset()
	} else {
		log.Printf("Cvar not found %v", n)
		conlog.Printf("Cvar_Reset: variable %v not found\n", n)
	}
}

func cvarSet(name, value string) {
	if cv, ok := cvar.Get(name); ok {
		cv.SetByString(value)
	} else {
		log.Printf("Cvar not found %v", name)
		conlog.Printf("Cvar_Set: variable %v not found\n", name)
	}
}

func CvarSetValue(name string, value float32) {
	if cv, ok := cvar.Get(name); ok {
		cv.SetValue(value)
	} else {
		log.Printf("Cvar not found %v", name)
		conlog.Printf("Cvar_SetValue: variable %v not found\n", name)
	}
}

//export CvarGetID
func CvarGetID(name *C.char) C.int {
	cv, ok := cvar.Get(C.GoString(name))
	if !ok {
		return -1
	}
	return C.int(cv.ID())
}

//export CvarSetCallback
func CvarSetCallback(id C.int, f C.cvarcallback_t) {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return
	}
	cv.SetCallback(func(_ *cvar.Cvar) {
		C.CallCvarCallback(id, f)
	})
}

func init() {
	cmd.AddCommand("cvarlist", printCvarList)
	cmd.AddCommand("toggle", CvarToggle)
	cmd.AddCommand("cycle", CvarCycle)
	cmd.AddCommand("inc", CvarInc)
	cmd.AddCommand("reset", CvarReset_f)
	cmd.AddCommand("resetall", CvarResetAll)
	cmd.AddCommand("resetcfg", CvarResetCfg)
}

func printCvarList(args []cmd.QArg, _ int) {
	// TODO(therjak):
	// this should probably print the syntax of cvarlist if len(args) > 1
	switch len(args) {
	default:
		printPartialCvarList(args[1])
		return
	case 0:
		printFullCvarList()
	}
}

func CvarResetAll(_ []cmd.QArg, _ int) {
	// bail if args not empty?
	for _, cv := range cvar.All() {
		cv.Reset()
	}
}

func CvarResetCfg(_ []cmd.QArg, _ int) {
	// bail if args not empty?
	for _, cv := range cvar.All() {
		if cv.Archive() {
			cv.Reset()
		}
	}
}

func printFullCvarList() {
	cvars := cvar.All()
	for _, v := range cvars {
		conlog.SafePrintf("%s%s %s \"%s\"\n",
			func() string {
				if v.Archive() {
					return "*"
				}
				return " "
			}(),
			func() string {
				if v.Notify() {
					return "s"
				}
				return " "
			}(),
			v.Name(),
			v.String())
	}
	conlog.SafePrintf("%v cvars\n", len(cvars))
}

func printPartialCvarList(p cmd.QArg) {
	log.Printf("TODO")
	// if beginning of name == p
	// same as ListFull
	// in length print add ("beginning with \"%s\"", p)
}

func CvarToggle(args []cmd.QArg, _ int) {
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		if cv, ok := cvar.Get(arg); ok {
			cv.Toggle()
		} else {
			log.Printf("toggle: Cvar not found %v", arg)
			conlog.Printf("toggle: variable %v not found\n", arg)
		}
		break
	default:
		conlog.Printf("toggle <cvar> : toggle cvar\n")
		break
	}
}

func CvarCycle(args []cmd.QArg, _ int) {
	if len(args) < 2 {
		conlog.Printf("cycle <cvar> <value list>: cycle cvar through a list of values\n")
		return
	}
	cv, ok := cvar.Get(args[0].String())
	if !ok {
		conlog.Printf("Cvar_Set: variable %v not found\n", args[0].String())
		return
	}
	// TODO: make entries in args[1:] unique
	oldValue := cv.String()
	i := 0
	for i < len(args)-1 {
		i++
		if oldValue == args[i].String() {
			break
		}
	}
	i %= len(args) - 1
	i++
	cv.SetByString(args[i].String())
}

func CvarInc(args []cmd.QArg, _ int) {
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		CvarSetValue(arg, CvarVariableValue(arg)+1)
	case 2:
		arg := args[0].String()
		CvarSetValue(arg, CvarVariableValue(arg)+float32(args[1].Float64()))
	default:
		conlog.Printf("inc <cvar> [amount] : increment cvar\n")
	}
}

func CvarReset_f(args []cmd.QArg, _ int) {
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		CvarReset(arg)
	default:
		conlog.Printf("reset <cvar> : reset cvar to default\n")
	}
}

func writeCvarVariables(w io.Writer) {
	for _, c := range cvar.All() {
		if c.Archive() {
			w.Write([]byte(fmt.Sprintf("%s \"%s\"\n", c.Name(), c.String())))
		}
	}
}
