// SPDX-License-Identifier: GPL-2.0-or-later

package cbuf

import "testing"

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		in     string
		wantF  string
		wantAS string
		wantMS string
		wantA  []QArg
	}{
		{
			in:     `say hello world`,
			wantF:  `say hello world`,
			wantAS: `hello world`,
			wantMS: `world`,
			wantA:  []QArg{{"say"}, {"hello"}, {"world"}},
		},
		{
			in:     `say "hello world"`,
			wantF:  `say "hello world"`,
			wantAS: `hello world`,
			wantMS: ``,
			wantA:  []QArg{{"say"}, {"hello world"}},
		},
		{
			in:     ` say_team  foo bar baz `,
			wantF:  `say_team  foo bar baz`,
			wantAS: `foo bar baz`,
			wantMS: `bar baz`,
			wantA:  []QArg{{"say_team"}, {"foo"}, {"bar"}, {"baz"}},
		},
	} {
		arg := Parse(tc.in)
		if tc.wantF != arg.Full() {
			t.Errorf("Parse(%q).Full()=%q, want %q", tc.in, arg.Full(), tc.wantF)
		}
		if tc.wantAS != arg.ArgumentString() {
			t.Errorf("Parse(%q).ArgumentString()=%q, want %q", tc.in, arg.ArgumentString(), tc.wantAS)
		}
		if tc.wantMS != arg.Message() {
			t.Errorf("Parse(%q).Message()=%q, want %q", tc.in, arg.Message(), tc.wantMS)
		}
		as := arg.Args()
		if len(tc.wantA) != len(as) {
			t.Fatalf("Parse(%q).Args() has len(%d), want %d", tc.in, len(as), len(tc.wantA))
		}
		for i := range tc.wantA {
			if tc.wantA[i] != as[i] {
				t.Errorf("Arg[%d]=%q, want %q", i, as[i], tc.wantA[i])
			}
		}
	}
}
