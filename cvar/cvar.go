// SPDX-License-Identifier: GPL-2.0-or-later

package cvar

import (
	"fmt"
	"log"
	"strconv"

	"goquake/cmd"
	"goquake/conlog"
)

var (
	cvarArray  []*Cvar
	cvarByName = make(map[string]*Cvar)
)

type flag uint64

const (
	// cvar flags bitfield
	NONE        flag = 0
	ARCHIVE     flag = 1
	NOTIFY      flag = 1 << 1
	SERVERINFO  flag = 1 << 2
	USERINFO    flag = 1 << 3
	CHANGED     flag = 1 << 4
	ROM         flag = 1 << 6
	LOCKED      flag = 1 << 8 // locked temporarily
	REGISTERED  flag = 1 << 10
	CALLBACK    flag = 1 << 16
	USERDEFINED flag = 1 << 17 // cvar was created by the user/mod, and needs to be saved a bit differently.
	AUTOCVAR    flag = 1 << 18 // cvar changes need to feed back to qc global changes.
	SETA        flag = 1 << 19 // cvar will be saved with seta.
)

type CallbackFunc func(cv *Cvar)

type Cvar struct {
	archive    bool
	notify     bool
	serverinfo bool
	rom        bool
	user       bool
	seta       bool
	callback   CallbackFunc
	name       string
	// stringValue is the truth, value the derived one
	stringValue  string
	value        float32
	defaultValue string
	id           int
}

func All() []*Cvar {
	return cvarArray
}

func (cv *Cvar) Archive() bool {
	return cv.archive
}

func (cv *Cvar) Notify() bool {
	return cv.notify
}

func (cv *Cvar) UserDefined() bool {
	return cv.user
}

func (cv *Cvar) SetA() bool {
	return cv.seta
}

func (cv *Cvar) SetCallback(cb CallbackFunc) {
	cv.callback = cb
}

func (cv *Cvar) SetByString(s string) {
	if cv.rom {
		return
	}
	cv.stringValue = s
	pf, _ := strconv.ParseFloat(cv.stringValue, 32)
	cv.value = float32(pf)
	if cv.callback != nil {
		cv.callback(cv)
	}
}

func (cv *Cvar) Reset() {
	cv.SetByString(cv.defaultValue)
}

func (cv *Cvar) String() string {
	return cv.stringValue
}

func (cv *Cvar) ID() int {
	return cv.id
}

func (cv *Cvar) Name() string {
	return cv.name
}

func (cv *Cvar) Value() float32 {
	return cv.value
}

func (cv *Cvar) SetValue(value float32) {
	if float32(int(value)) == value {
		v := strconv.FormatInt(int64(value), 10)
		cv.SetByString(v)
	} else {
		v := strconv.FormatFloat(float64(value), 'f', -1, 32)
		cv.SetByString(v)
	}
}

func (cv *Cvar) Toggle() {
	if cv.String() == "1" {
		cv.SetByString("0")
	} else {
		cv.SetByString("1")
	}
}

func (cv *Cvar) Bool() bool {
	return cv.stringValue != "0"
}

func Get(name string) (*Cvar, bool) {
	cv, err := cvarByName[name]
	return cv, err
}

func GetByID(id int) (*Cvar, error) {
	if id < 0 || id >= len(cvarArray) {
		return nil, fmt.Errorf("id out of bounds")
	}
	return cvarArray[id], nil
}

func create(name, value string) *Cvar {
	cv := &Cvar{name: name, defaultValue: value}
	cv.SetByString(value)
	pos := len(cvarArray)
	cvarArray = append(cvarArray, cv)
	cvarByName[name] = cv
	cv.id = pos
	return cv
}

func Register(name, value string, flags flag) (*Cvar, error) {
	if _, ok := cvarByName[name]; ok {
		return nil, fmt.Errorf("Can't register variable %s, already defined\n", name)
	}

	cv := create(name, value)

	if flags&ARCHIVE != 0 {
		cv.archive = true
	}
	if flags&NOTIFY != 0 {
		cv.notify = true
	}
	if flags&SERVERINFO != 0 {
		cv.serverinfo = true
	}
	if flags&ROM != 0 {
		cv.rom = true
	}
	// registered 1<<10, callback 1<<16

	return cv, nil
}

func MustRegister(n, v string, flag flag) *Cvar {
	cv, err := Register(n, v, flag)
	if err != nil {
		log.Panic(n)
	}
	return cv
}

func Execute(a cmd.Arguments) (bool, error) {
	args := a.Args()
	if len(args) == 0 {
		return false, nil
	}
	n := args[0].String()
	cv, ok := Get(n)
	if !ok {
		return false, nil
	}
	if len(args) == 1 {
		conlog.Printf("\"%s\" is \"%s\"\n", cv.Name(), cv.String())
		log.Printf("shown cvar")
		return true, nil
	}
	cv.SetByString(args[1].String())
	return true, nil
}

func init() {
	cmd.Must(cmd.AddCommand("cvarlist", list))
	cmd.Must(cmd.AddCommand("cycle", cycle))
	cmd.Must(cmd.AddCommand("inc", inc))
	cmd.Must(cmd.AddCommand("reset", reset))
	cmd.Must(cmd.AddCommand("resetall", resetAll))
	cmd.Must(cmd.AddCommand("resetcfg", resetCfg))
	cmd.Must(cmd.AddCommand("set", set))
	cmd.Must(cmd.AddCommand("seta", seta))
	cmd.Must(cmd.AddCommand("toggle", toggle))
}

func set(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch {
	case len(args) >= 2:
		if cmd.Exists(args[0].String()) {
			conlog.Printf("conflict with command\n")
			return nil
		}
		if cv, ok := cvarByName[args[0].String()]; ok {
			cv.SetByString(args[1].String())
		} else {
			cv := create(args[0].String(), args[1].String())
			cv.user = true
		}
	default:
		conlog.Printf("set <cvar> <value>\n")
	}
	return nil
}

func seta(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch {
	case len(args) >= 2:
		if cmd.Exists(args[0].String()) {
			conlog.Printf("conflict with command\n")
			return nil
		}
		if cv, ok := cvarByName[args[0].String()]; ok {
			cv.SetByString(args[1].String())
			cv.seta = true
		} else {
			cv := create(args[0].String(), args[1].String())
			cv.seta = true
			cv.archive = true
			cv.user = true
		}
	default:
		conlog.Printf("seta <cvar> <value>\n")
	}
	return nil
}

func toggle(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		if cv, ok := Get(arg); ok {
			cv.Toggle()
		} else {
			log.Printf("toggle: Cvar not found %v", arg)
			conlog.Printf("toggle: variable %v not found\n", arg)
		}
	default:
		conlog.Printf("toggle <cvar> : toggle cvar\n")
	}
	return nil
}

func incr(n string, v float32) {
	if cv, ok := Get(n); ok {
		cv.SetValue(cv.Value() + v)
	} else {
		log.Printf("Cvar not found %v", n)
		conlog.Printf("Cvar_SetValue: variable %v not found\n", n)
	}
}

func inc(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		incr(arg, 1)
	case 2:
		arg := args[0].String()
		incr(arg, args[1].Float32())
	default:
		conlog.Printf("inc <cvar> [amount] : increment cvar\n")
	}
	return nil
}

func reset(a cmd.Arguments) error {
	args := a.Args()[1:]
	switch c := len(args); c {
	case 1:
		arg := args[0].String()
		if cv, ok := Get(arg); ok {
			cv.Reset()
		} else {
			log.Printf("Cvar not found %v", arg)
			conlog.Printf("Cvar_Reset: variable %v not found\n", arg)
		}
	default:
		conlog.Printf("reset <cvar> : reset cvar to default\n")
	}
	return nil
}

func resetAll(_ cmd.Arguments) error {
	// bail if args not empty?
	for _, cv := range All() {
		cv.Reset()
	}
	return nil
}

func resetCfg(_ cmd.Arguments) error {
	// bail if args not empty?
	for _, cv := range All() {
		if cv.Archive() {
			cv.Reset()
		}
	}
	return nil
}

func list(a cmd.Arguments) error {
	// TODO(therjak):
	// this should probably print the syntax of cvarlist if len(args) > 2
	args := a.Args()
	switch len(args) {
	default:
		partialList(args[1])
	case 0, 1:
		fullList()
	}
	return nil
}

func fullList() {
	cvars := All()
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

func partialList(p cmd.QArg) {
	log.Printf("TODO")
	// if beginning of name == p
	// same as ListFull
	// in length print add ("beginning with \"%s\"", p)
}

func cycle(a cmd.Arguments) error {
	args := a.Args()[1:]
	if len(args) < 2 {
		conlog.Printf("cycle <cvar> <value list>: cycle cvar through a list of values\n")
		return nil
	}
	cv, ok := Get(args[0].String())
	if !ok {
		conlog.Printf("Cvar_Set: variable %v not found\n", args[0].String())
		return nil
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
	return nil
}
