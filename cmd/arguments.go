// SPDX-License-Identifier: GPL-2.0-or-later

package cmd

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type QArg struct {
	a string
}

func (a QArg) String() string {
	return a.a
}

func (a QArg) Int() int {
	r, err := strconv.ParseInt(a.a, 10, 0)
	if err != nil {
		return 0
	}
	return int(r)
}

func (a QArg) Float32() float32 {
	r, err := strconv.ParseFloat(a.a, 32)
	if err != nil {
		return 0
	}
	return float32(r)
}

func (a QArg) Float64() float64 {
	r, err := strconv.ParseFloat(a.a, 64)
	if err != nil {
		return 0
	}
	return r
}

func (a QArg) Bool() bool {
	switch a.a {
	case "1", "t", "T", "true", "TRUE", "True", "On", "ON", "on":
		return true
	default:
		return false
	}
}

type Arguments struct {
	// each arg on its own
	args []QArg
	// concat of args[1:]
	full string
}

func (c *Arguments) Argv(i int) QArg {
	if i < 0 || i >= len(c.args) {
		log.Printf("Got Argv out of bounds %v, %v", i, len(c.args))
		if len(c.args) > 0 {
			log.Printf("Arg0 is %v", c.args[0])
		}
		return QArg{""}
	}
	return c.args[i]
}

func (c *Arguments) Full() string {
	return c.full
}

func (c *Arguments) Args() []QArg {
	return c.args
}

func (c *Arguments) ArgumentString() string {
	// args[0] is the cmd
	if len(c.args) < 2 {
		return ""
	}
	r := strings.TrimPrefix(c.full, c.args[0].String())
	r = strings.TrimLeftFunc(r, unicode.IsSpace)
	// we want to remove " around the text.
	// the end is not that important but the result should not start with " or
	// space.
	if len(r) > 1 {
		if r[0] == '"' {
			r = strings.Trim(r, "\"\t\n\v\f\r ")
		}
	}
	return r
}

// Message returns the message send to the target in argument 1.
// Expects the first two arguments to be cmd and target.
func (c *Arguments) Message() string {
	if len(c.args) < 3 {
		return ""
	}
	t := c.args[1].String()
	return c.full[strings.Index(c.full, t)+len(t)+1:]
}

func Parse(s string) (args Arguments) {
	args.full = strings.TrimFunc(s, unicode.IsSpace)
	args.args = []QArg{}

	l := lex(args.full)
	for {
		i := l.nextItem()

		switch i.typ {
		case itemChar, itemWord:
			args.args = append(args.args, QArg{i.val})
		case itemString:
			s := i.val
			s = strings.TrimPrefix(s, `"`)
			s = strings.TrimSuffix(s, `"`)
			args.args = append(args.args, QArg{s})
		case itemSpace, itemComment1:
			continue
		case itemEOF:
			return
		default:
			log.Printf("got item type %v with value %v", i.typ, i.val)
			return
		}
	}
}

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemString   // quoted string includes quotes
	itemChar     // '{','}','(',')','\'',':'
	itemSpace    // <=32
	itemComment1 //
	itemComment2 /* */
	itemWord     //
)
const eof = -1

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	if len(i.val) > 10 {
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

type stateFn func(*lexer) stateFn

type lexer struct {
	input string
	start int
	pos   int
	width int
	items chan item
	state stateFn
}

func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan item, 2),
		state: lexAction,
	}
	return l
}

func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			l.state = l.state(l)
		}
	}
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func belowSpace(c rune) bool {
	return c <= ' '
}

func lexWord(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isQuakeRune(r):
			// absorb
		default:
			l.backup()
			l.emit(itemWord)
			break Loop
		}
	}
	return lexAction
}

func lexAction(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof || isEndOfLine(r):
		l.emit(itemEOF)
		return nil
	case isSpace(r):
		return lexSpace
	case r == '"':
		return lexQuote
	case r == '/':
		// special look-ahead so we don't break l.backup().
		if l.pos < len(l.input) {
			r := l.input[l.pos]
			if r == '/' {
				// just drop the rest of this line
				l.emit(itemEOF)
				return nil
			}
		}
		fallthrough
	case isQuakeRune(r):
		l.backup()
		return lexWord
	default:
		return l.errorf("unhandled char: %#U", r)
	}
}

func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(itemSpace)
	return lexAction
}

func lexQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '"':
			break Loop
		case eof, '\n':
			return l.errorf("unterminated string")
		}
	}
	l.emit(itemString)
	return lexAction
}

func isQuakeRune(r rune) bool {
	// this is an ugly ascii workaround
	return r > ' '
}

func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
