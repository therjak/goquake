// SPDX-License-Identifier: GPL-2.0-or-later

package cvar

import (
	"fmt"
	"log"
	"strconv"

	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
)

type Cvars map[string]*Cvar

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

func New(name, value string, flags flag) *Cvar {
	cv := &Cvar{name: name, defaultValue: value}
	cv.SetByString(value)

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

	return cv
}

func (c *Cvars) All() []*Cvar {
	var r []*Cvar
	for _, cv := range *c {
		r = append(r, cv)
	}
	return r
}

func (c *Cvars) Add(cv *Cvar) error {
	if _, ok := (*c)[cv.name]; ok {
		return fmt.Errorf("Can't register variable %s, already defined\n", cv.name)
	}
	(*c)[cv.name] = cv
	return nil
}

func (c *Cvars) Execute() func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
	return func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
		args := a.Args()
		if len(args) == 0 {
			return false, nil
		}
		n := args[0].String()
		cv, ok := (*c)[n]
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
}

func (c *Cvars) set(cs *cmd.Commands) func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch {
		case len(args) >= 2:
			name, value := args[0].String(), args[1].String()
			if cs.Exists(name) {
				conlog.Printf("conflict with command\n")
				return nil
			}
			if cv, ok := (*c)[name]; ok {
				cv.SetByString(value)
			} else {
				cv := New(name, value, NONE)
				(*c)[name] = cv
				cv.user = true
			}
		default:
			conlog.Printf("set <cvar> <value>\n")
		}
		return nil
	}
}

func (c *Cvars) seta(cs *cmd.Commands) func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch {
		case len(args) >= 2:
			name, value := args[0].String(), args[1].String()
			if cs.Exists(name) {
				conlog.Printf("conflict with command\n")
				return nil
			}
			if cv, ok := (*c)[name]; ok {
				cv.SetByString(value)
				cv.seta = true
			} else {
				cv := New(name, value, NONE)
				(*c)[name] = cv
				cv.seta = true
				cv.archive = true
				cv.user = true
			}
		default:
			conlog.Printf("seta <cvar> <value>\n")
		}
		return nil
	}
}

func (c *Cvars) toggle() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch len(args) {
		case 1:
			arg := args[0].String()
			if cv, ok := (*c)[arg]; ok {
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
}

func (c *Cvars) incr(n string, v float32) {
	if cv, ok := (*c)[n]; ok {
		cv.SetValue(cv.Value() + v)
	} else {
		log.Printf("Cvar not found %v", n)
		conlog.Printf("Cvar_SetValue: variable %v not found\n", n)
	}
}

func (c *Cvars) inc() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch len(args) {
		case 1:
			arg := args[0].String()
			c.incr(arg, 1)
		case 2:
			arg := args[0].String()
			c.incr(arg, args[1].Float32())
		default:
			conlog.Printf("inc <cvar> [amount] : increment cvar\n")
		}
		return nil
	}
}

func (c *Cvars) reset() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		switch len(args) {
		case 1:
			arg := args[0].String()
			if cv, ok := (*c)[arg]; ok {
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
}

func (c *Cvars) resetAll() func(_ cbuf.Arguments) error {
	return func(_ cbuf.Arguments) error {
		// bail if args not empty?
		for _, cv := range *c {
			cv.Reset()
		}
		return nil
	}
}

func (c *Cvars) resetCfg() func(_ cbuf.Arguments) error {
	return func(_ cbuf.Arguments) error {
		// bail if args not empty?
		for _, cv := range *c {
			if cv.Archive() {
				cv.Reset()
			}
		}
		return nil
	}
}

func (c *Cvars) list() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		// TODO(therjak):
		// this should probably print the syntax of cvarlist if len(args) > 2
		args := a.Args()
		switch len(args) {
		default:
			c.partialList(args[1])
		case 0, 1:
			c.fullList()
		}
		return nil
	}
}

func (c *Cvars) fullList() {
	for _, v := range *c {
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
	conlog.SafePrintf("%v cvars\n", len(*c))
}

func (c *Cvars) partialList(p cbuf.QArg) {
	log.Printf("TODO")
	// if beginning of name == p
	// same as ListFull
	// in length print add ("beginning with \"%s\"", p)
}

func (c *Cvars) cycle() func(a cbuf.Arguments) error {
	return func(a cbuf.Arguments) error {
		args := a.Args()[1:]
		if len(args) < 2 {
			conlog.Printf("cycle <cvar> <value list>: cycle cvar through a list of values\n")
			return nil
		}
		cv, ok := (*c)[args[0].String()]
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
}

func (cvs *Cvars) Commands(c *cmd.Commands) error {
	if err := c.Add("cvarlist", cvs.list()); err != nil {
		return err
	}
	if err := c.Add("cycle", cvs.cycle()); err != nil {
		return err
	}
	if err := c.Add("inc", cvs.inc()); err != nil {
		return err
	}
	if err := c.Add("reset", cvs.reset()); err != nil {
		return err
	}
	if err := c.Add("resetall", cvs.resetAll()); err != nil {
		return err
	}
	if err := c.Add("resetcfg", cvs.resetCfg()); err != nil {
		return err
	}
	if err := c.Add("set", cvs.set(c)); err != nil {
		return err
	}
	if err := c.Add("seta", cvs.seta(c)); err != nil {
		return err
	}
	if err := c.Add("toggle", cvs.toggle()); err != nil {
		return err
	}
	return nil
}
