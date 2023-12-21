// SPDX-License-Identifier: GPL-2.0-or-later

package cbuf

import (
	"testing"
)

func TestWait(t *testing.T) {
	c := CommandBuffer{}
	runCount := 0
	c.SetCommandExecutors([]Efunc{
		func(cb *CommandBuffer, a Arguments) (bool, error) {
			runCount++
			return true, nil
		}})
	c.AddText("wait\n")
	c.AddText("test\n")
	c.AddText("test\n")
	c.AddText("wait\n")
	c.AddText("test\n")
	c.Execute()
	if runCount != 0 {
		t.Errorf("runCount=%v, want %v", runCount, 0)
	}
	c.Execute()
	if runCount != 2 {
		t.Errorf("runCount=%v, want %v", runCount, 2)
	}
	c.Execute()
	if runCount != 3 {
		t.Errorf("runCount=%v, want %v", runCount, 3)
	}
}
