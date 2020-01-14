package quakelib

//struct cvar_s;
//typedef void (*cvarcallback_t)(struct cvar_s*);
//void CallCvarCallback(int id, cvarcallback_t func);
import "C"
import (
	"fmt"
	"io"
	"log"
	"quake/cmd"
	"quake/conlog"
	"quake/cvar"
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

//export CvarGetName
func CvarGetName(id C.int) *C.char {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return nil
	}
	return C.CString(cv.Name())
}

//export Cvar_VariableValue
func Cvar_VariableValue(name *C.char) C.float {
	n := C.GoString(name)
	return C.float(CvarVariableValue(n))
}

func CvarVariableValue(n string) float32 {
	if cv, ok := cvar.Get(n); ok {
		return cv.Value()
	}
	return 0
}

//export CvarVariableString
func CvarVariableString(name *C.char) *C.char {
	n := C.GoString(name)
	if cv, ok := cvar.Get(n); ok {
		return C.CString(cv.String())
	}
	return nil
}

func CvarReset(n string) {
	if cv, ok := cvar.Get(n); ok {
		cv.Reset()
	} else {
		log.Printf("Cvar not found %v", n)
		conlog.Printf("Cvar_Reset: variable %v not found\n", n)
	}
}

//export CvarSetQuick
func CvarSetQuick(id C.int, value *C.char) {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return
	}
	cv.SetByString(C.GoString(value))
}

//export CvarSetValueQuick
func CvarSetValueQuick(id C.int, value C.float) {
	cv, err := cvar.GetByID(int(id))
	if err != nil {
		log.Println(err)
		return
	}
	cv.SetValue(float32(value))
}

//export Cvar_Set
func Cvar_Set(name *C.char, value *C.char) {
	n := C.GoString(name)
	v := C.GoString(value)
	cvarSet(n, v)
}

func cvarSet(name, value string) {
	if cv, ok := cvar.Get(name); ok {
		cv.SetByString(value)
	} else {
		log.Printf("Cvar not found %v", name)
		conlog.Printf("Cvar_Set: variable %v not found\n", name)
	}
}

//export Cvar_SetValue
func Cvar_SetValue(name *C.char, value C.float) {
	n := C.GoString(name)
	CvarSetValue(n, float32(value))
}

func CvarSetValue(name string, value float32) {
	if cv, ok := cvar.Get(name); ok {
		cv.SetValue(value)
	} else {
		log.Printf("Cvar not found %v", name)
		conlog.Printf("Cvar_SetValue: variable %v not found\n", name)
	}
}

//export CvarRegister
func CvarRegister(name *C.char, value *C.char, flags C.int) C.int {
	n := C.GoString(name)
	/* this is a design error, why report?
	  if cmd.Exists(n) {
			conlog.Printf("Cvar-RegisterVariable: %s is a command\n", n)
			return -1
		}
	*/
	v := C.GoString(value)
	cv, err := cvar.Register(n, v, int(flags))
	if err != nil {
		conlog.Printf("%v\n", err)
		return -1
	}
	return C.int(cv.ID())
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
	conlog.Printf("cycle got %s: %s\n", args[0].String(), args[i].String())
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
