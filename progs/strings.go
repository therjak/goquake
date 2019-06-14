package progs

import "fmt"

func init() {
	//	engineStringMap = make(map[string]int)
}

type LoadedProg struct {
	*prog
	engineStrings []string
}

//-- call this LoadProgs and let it return something called progs.LoadedProg
func LoadProgs() (*LoadedProg, error) {
	lp, err := loadProgs()
	if err != nil {
		return nil, err
	}
	r := &LoadedProg{lp, make([]string, 0)}
	r.AddString("")
	return r, nil
}

func (p *LoadedProg) NewString(s string) int {
	// TODO:
	// replace \n with '\n' and all other \x with just '\'
	p.engineStrings = append(p.engineStrings, s)
	i := len(p.engineStrings)
	return -i
}

func (p *LoadedProg) AddString(s string) int {
	for i, es := range p.engineStrings {
		if es == s {
			return -(i + 1) // see String func
		}
	}
	return p.NewString(s)
}

func (p *LoadedProg) String(n int32) (string, error) {
	if n >= 0 {
		s, ok := p.Strings[n]
		if !ok {
			return "", fmt.Errorf("String: request of %v, is unknown", n)
		}
		return s, nil
	}
	// n is negative, so -(n + 1) is our index
	index := -(n + 1)
	if int32(len(p.engineStrings)) <= index {
		return "", fmt.Errorf("String: request of %v, is unknown", n)
	}
	return p.engineStrings[index], nil
}

func (p *LoadedProg) findFieldDef(name string) (Def, error) {
	for _, d := range p.FieldDefs {
		n, err := p.String(d.SName)
		if err != nil {
			continue
		}
		if name == n {
			return d, nil
		}
	}
	return Def{}, fmt.Errorf("FieldDef '%s' not found", name)
}

func (p *LoadedProg) findGlobalDef(name string) (Def, error) {
	for _, d := range p.GlobalDefs {
		n, err := p.String(d.SName)
		if err != nil {
			continue
		}
		if name == n {
			return d, nil
		}
	}
	return Def{}, fmt.Errorf("GlobalDef '%s' not found", name)
}

func (p *LoadedProg) findFunction(name string) (Function, error) {
	for _, f := range p.Functions {
		n, err := p.String(f.SName)
		if err != nil {
			continue
		}
		if name == n {
			return f, nil
		}
	}
	return Function{}, fmt.Errorf("Function '%s' not found", name)
}
