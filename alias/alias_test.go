// SPDX-License-Identifier: GPL-2.0-or-later

package alias

import (
	"fmt"
	"goquake/cbuf"
	"goquake/cmd"
	"goquake/conlog"
	"testing"
)

func TestAliasRegister(t *testing.T) {
	al := New()
	cmds := cmd.New()
	if err := al.Register(cmds); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteAlias(t *testing.T) {
	al := New()
	cmds := cmd.New()
	if err := al.Register(cmds); err != nil {
		t.Fatal(err)
	}
	cb := cbuf.CommandBuffer{}
	worldCount := 0
	p := func(cb *cbuf.CommandBuffer, a cbuf.Arguments) (bool, error) {
		if a.Full() != "world" {
			t.Errorf("Print() = %q, want %q", a.Full(), "world")
		} else {
			worldCount++
		}
		return true, nil
	}
	cb.SetCommandExecutors([]cbuf.Efunc{
		cmds.Execute(), // execute 'alias'
		al.Execute(),   // execute 'hello'
		p,              // execute 'world'
	})

	cb.AddText("alias hello world\n")
	cb.Execute()
	cb.AddText("hello\n")
	cb.AddText("world\n")
	cb.Execute()
	if worldCount != 2 {
		// for 'hello' -> 'world' and 'world'
		t.Errorf("Executed 'world' %d times, want %d", worldCount, 2)
	}
}

func TestPrintAlias(t *testing.T) {
	var pfout, spfout string
	pf := func(s string, a ...any) {
		pfout += fmt.Sprintf(s, a...)
	}
	spf := func(s string, a ...any) {
		spfout += fmt.Sprintf(s, a...)
	}
	conlog.SetPrintf(pf)
	conlog.SetSafePrintf(spf)
	al := New()
	cmds := cmd.New()
	if err := al.Register(cmds); err != nil {
		t.Fatal(err)
	}
	cb := cbuf.CommandBuffer{}
	cb.SetCommandExecutors([]cbuf.Efunc{
		cmds.Execute(), // execute 'alias'
		al.Execute(),   // execute 'hello'
	})

	cb.AddText("alias hello world\n")
	cb.Execute()
	cb.AddText("alias\n")
	cb.Execute()
	if spfout != "  hello: world\n1 alias command(s)\n" {
		t.Errorf("%q", spfout)
	}
	cb.AddText("alias hello\n")
	cb.Execute()
	if pfout != "  hello: world\n" {
		t.Errorf("%q", pfout)
	}
}
