package commandline

import (
	"flag"
	"testing"
)

func TestBoolInt(t *testing.T) {
	var flags flag.FlagSet
	flags.Init("test", flag.ContinueOnError)
	a := boolInt{false, 4}
	b := boolInt{false, 5}
	c := boolInt{true, 6}
	d := boolInt{false, 7}
	e := boolInt{false, 8}
	f := boolInt{true, 9}
	flags.Var(&a, "a", "usage")
	flags.Var(&b, "b", "usage")
	flags.Var(&c, "c", "usage")
	flags.Var(&d, "d", "usage")
	flags.Var(&e, "e", "usage")
	flags.Var(&f, "f", "usage")
	if err := flags.Parse([]string{"-a", "-b=3", "-e=true", "-f=false"}); err != nil {
		t.Error(err)
	}
	if a.set != true {
		t.Errorf("a.set = %v", a.set)
	}
	if b.set != true {
		t.Errorf("b.set = %v", b.set)
	}
	if c.set != true {
		t.Errorf("c.set = %v", c.set)
	}
	if d.set != false {
		t.Errorf("d.set = %v", d.set)
	}
	if e.set != true {
		t.Errorf("e.set = %v", e.set)
	}
	if f.set != false {
		t.Errorf("f.set = %v", f.set)
	}
	if a.num != 4 {
		t.Errorf("a.num = %v", a.num)
	}
	if b.num != 3 {
		t.Errorf("b.num = %v", b.num)
	}
	if c.num != 6 {
		t.Errorf("c.num = %v", c.num)
	}
	if d.num != 7 {
		t.Errorf("d.num = %v", d.num)
	}
}
