package progs

import "fmt"

func init() {
	//	engineStringMap = make(map[string]int)
}

type LoadedProg struct {
	*prog
	engineStrings []string
	// engineStringMap map[string]int
}

//-- call this LoadProgs and let it return something called progs.LoadedProg
func LoadProgs() (*LoadedProg, error) {
	lp, err := loadProgs()
	if err != nil {
		return nil, err
	}
	r := &LoadedProg{lp, make([]string, 0)} // , make(map[string]int)}
	// r.fillEngineStrings(r.Strings)
	return r, nil
}

/*
func (p *LoadedProg) fillEngineStrings(ks map[int]string) {
	// just ignore duplicates
	for k, v := range ks {
		_, ok := p.engineStringMap[v]
		if !ok {
			p.engineStringMap[v] = k
		}
	}
}
*/

func (p *LoadedProg) NewString(s string) int {
	// TODO:
	// replace \n with '\n' and all other \x with just '\'
	p.engineStrings = append(p.engineStrings, s)
	i := len(p.engineStrings)
	// p.engineStringMap[s] = -i
	return -i
}

func (p *LoadedProg) AddString(s string) int {
	// TODO: prevent duplicates
	/*
		v, ok := engineStringMap[s]
		if ok {
			log.Printf("PR_SetEngineString1 %v, %d", s, v)
			return C.int(v)
		}
	*/
	return p.NewString(s)
}

func (p *LoadedProg) String(n int) (string, error) {
	if n >= 0 {
		s, ok := p.Strings[n]
		if !ok {
			return "", fmt.Errorf("String: request of %v, is unknown", n)
		}
		return s, nil
	}
	// n is negative, so -(n + 1) is our index
	index := -(n + 1)
	if len(p.engineStrings) <= index {
		return "", fmt.Errorf("String: request of %v, is unknown", n)
	}
	return p.engineStrings[index], nil
}
