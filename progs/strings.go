// SPDX-License-Identifier: GPL-2.0-or-later

package progs

import (
	"fmt"
	"log"
	"strings"
)

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

func (p *LoadedProg) NewString(s string) int32 {
	s = strings.ReplaceAll(s, "\\n", "\n")
	p.engineStrings = append(p.engineStrings, s)
	i := int32(len(p.engineStrings))
	return -i
}

func (p *LoadedProg) AddString(s string) int32 {
	for i, es := range p.engineStrings {
		if es == s {
			return int32(-(i + 1)) // see String func
		}
	}
	return p.NewString(s)
}

// Returns a string with a description and the contents of a global,
// padded to 20 field width
func (p *LoadedProg) GlobalString(n int16) string {
	log.Printf("TODO: GlobalString called %d", n)
	// TODO
	/*
			static char line[512];
		  const char *s;
		  int i;
		  ddef_t *def;
		  void *val;

		  def = ED_GlobalAtOfs(ofs); // FindGlobalDef
		  if (!def)
		    sprintf(line, "%i(?)", ofs);
		  else {
		    eval_t v;
		    v.vector[0] = Pr_globalsf(ofs);
		    v.vector[1] = Pr_globalsf(ofs + 1);
		    v.vector[2] = Pr_globalsf(ofs + 2);
		    s = PR_ValueString(def->type, &v);
		    sprintf(line, "%i(%s)%s", ofs, PR_GetString(def->s_name), s);
		  }

		  i = strlen(line);
		  for (; i < 20; i++) strcat(line, " ");
		  strcat(line, " ");

		  return line;
	*/
	return ""
}

func (p *LoadedProg) GlobalStringNoContents(n int16) string {
	log.Printf("TODO: GlobalStringNoContents called %d", n)
	// TODO
	/*
			static char line[512];
		  int i;
		  ddef_t *def;

		  def = ED_GlobalAtOfs(ofs); // FindGlobalDef
		  if (!def)
		    sprintf(line, "%i(?)", ofs);
		  else
		    sprintf(line, "%i(%s)", ofs, PR_GetString(def->s_name));

		  i = strlen(line);
		  for (; i < 20; i++) strcat(line, " ");
		  strcat(line, " ");

		  return line;
	*/
	return ""
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

func (p *LoadedProg) FindFieldDef(name string) (Def, error) {
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

func (p *LoadedProg) FindGlobalDef(name string) (Def, error) {
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

func (p *LoadedProg) FindFunction(name string) (int, error) {
	for i, f := range p.Functions {
		n, err := p.String(f.SName)
		if err != nil {
			continue
		}
		if name == n {
			return i, nil
		}
	}
	return 0, fmt.Errorf("Function '%s' not found", name)
}
