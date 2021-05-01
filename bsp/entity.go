// SPDX-License-Identifier: GPL-2.0-or-later
package bsp

import (
	"bytes"
)

type Entity struct {
	properties map[string]string
	src        []byte
}

func NewEntity(p []byte) *Entity {
	e := &Entity{properties: make(map[string]string), src: p}
	// parse the entity line by line
	lines := bytes.Split(p, []byte("\n"))
	for _, l := range lines {
		// look for something of the form
		// "key" "value"
		q := bytes.IndexByte(l, '"')
		if q == -1 {
			continue
		}
		r := l[q+1:]
		q = bytes.IndexByte(r, '"')
		if q == -1 {
			continue
		}
		key := string(r[:q])
		r = r[q+1:]
		q = bytes.IndexByte(r, '"')
		if q == -1 {
			continue
		}
		r = r[q+1:]
		q = bytes.IndexByte(r, '"')
		if q == -1 {
			continue
		}
		value := string(r[:q])
		e.properties[key] = value
	}
	return e
}

func (e *Entity) Property(name string) (string, bool) {
	v, ok := e.properties[name]
	return v, ok
}

func (e *Entity) Name() (string, bool) {
	v, ok := e.properties["classname"]
	return v, ok
}

func (e *Entity) PropertyNames() []string {
	n := []string{}
	for k := range e.properties {
		n = append(n, k)
	}
	return n
}

func ParseEntities(data []byte) []*Entity {
	/*
		The data looks like:
		{
		  "name" "value"
		  "name2" "value2"
		}
		{
		  "name3" "value"
		  {
		    ()()()...
		  }
		}
		But I have not seen the nested stuff
	*/
	// First split the entities
	es := []*Entity{}
	var ess [][]byte
	var ob, q int
	start := -1
	for i, b := range data {
		switch b {
		case '{':
			if q != 0 {
				break
			}
			if start == -1 {
				start = i
			} else {
				ob++
			}
		case '}':
			if q != 0 {
				break
			}
			if start == -1 {
				// Bad input
				return nil
			}
			if ob == 0 {
				ess = append(ess, data[start:i+1])
				start = -1
			} else {
				ob--
			}
		case '"':
			if q == 0 {
				q++
			} else {
				q--
			}
		}
	}
	for _, e := range ess {
		es = append(es, NewEntity(e))
	}
	return es
}
