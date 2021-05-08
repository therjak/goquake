// SPDX-License-Identifier: GPL-2.0-or-later

package cvar

import (
	"fmt"
	"log"
	"strconv"

	"github.com/therjak/goquake/cmd"
	"github.com/therjak/goquake/conlog"
)

var (
	cvarArray  []*Cvar
	cvarByName = make(map[string]*Cvar)
)

const (
	// cvar flags bitfield
	NONE       = 0
	ARCHIVE    = 1
	NOTIFY     = 1 << 1
	SERVERINFO = 1 << 2
	ROM        = 1 << 6
	REGISTERED = 1 << 10
	CALLBACK   = 1 << 16
)

type CallbackFunc func(cv *Cvar)

type Cvar struct {
	archive    bool
	notify     bool
	serverinfo bool
	rom        bool
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

func Register(name, value string, flags int) (*Cvar, error) {
	if _, ok := cvarByName[name]; ok {
		return nil, fmt.Errorf("Can't register variable %s, already defined\n", name)
	}
	cv := Cvar{name: name, defaultValue: value}
	cv.SetByString(value)

	if flags&1 != 0 {
		cv.archive = true
	}
	if flags&2 != 0 {
		cv.notify = true
	}
	if flags&4 != 0 {
		cv.serverinfo = true
	}
	if flags&64 != 0 {
		cv.rom = true
	}
	// registered 1<<10, callback 1<<16

	pos := len(cvarArray)
	cvarArray = append(cvarArray, &cv)
	cvarByName[name] = &cv
	cv.id = pos

	return &cv, nil
}

func MustRegister(n, v string, flag int) *Cvar {
	cv, err := Register(n, v, flag)
	if err != nil {
		log.Panic(n)
	}
	return cv
}

func Execute(args []cmd.QArg, _ int) bool {
	if len(args) == 0 {
		return false
	}
	n := args[0].String()
	cv, ok := Get(n)
	if !ok {
		return false
	}
	if len(args) == 1 {
		conlog.Printf("\"%s\" is \"%s\"\n", cv.Name(), cv.String())
		log.Printf("shown cvar")
		return true
	}
	cv.SetByString(args[1].String())
	return true
}
