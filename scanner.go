package gokonfi

import (
	"fmt"
	"regexp"
	"unicode/utf8"
)

type Scanner struct {
	input string
	pos   int
}

func NewScanner(input string) Scanner {
	return Scanner{input: input, pos: 0}
}

//go:generate stringer -type=TokenType
type TokenType int32

const (
	Unspecified TokenType = iota
	IntLiteral
	DoubleLiteral
	StrLiteral
	PlusOp
	MinusOp
	TimesOp
	DivOp
	Equal
	NotEqual
	LessThan
	LessEq
	GreaterThan
	GreaterEq
	Comma
	Dot
	LeftParen
	RightParen
	LeftBrace
	RightBrace
	Colon
	EndOfInput
)

type Token struct {
	Typ TokenType
	Pos int
	End int
	Val string
}

type ScanError struct {
	pos int
	msg string
}

func (s *ScanError) Pos() int {
	return s.pos
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("scanError: %s at position %d", e.msg, e.pos)
}

func (s *Scanner) AtEnd() bool {
	return s.pos >= len(s.input)
}

func (s *Scanner) Rem() string {
	return s.input[s.pos:]
}

func (s *Scanner) peek() rune {
	if s.AtEnd() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[s.pos:])
	return r
}

func (s *Scanner) advance() {
	if s.AtEnd() {
		return
	}
	_, size := utf8.DecodeRuneInString(s.input[s.pos:])
	s.pos += size
}

func (s *Scanner) match(expected rune) bool {
	if s.AtEnd() {
		return false
	}
	if r, size := utf8.DecodeRuneInString(s.input[s.pos:]); r == expected {
		s.pos += size
		return true
	}
	return false
}

func (s *Scanner) NextToken() (Token, error) {
	// Iterate until a token is found, skipping comments and whitespace.
	for !s.AtEnd() {
		r, size := utf8.DecodeRuneInString(s.input[s.pos:])
		if r == utf8.RuneError {
			return Token{}, &ScanError{pos: s.pos, msg: "Invalid UTF-8 code point"}
		}
		// Advance scanner
		start := s.pos
		s.pos += size
		tok := func(t TokenType) (Token, error) {
			return Token{Typ: t, Pos: start, End: s.pos, Val: s.input[start:s.pos]}, nil
		}
		// Dispatch based on first character.
		switch r {
		case '(':
			return tok(LeftParen)
		case ')':
			return tok(RightParen)
		case '{':
			return tok(LeftBrace)
		case '}':
			return tok(RightBrace)
		case ',':
			return tok(Comma)
		case ':':
			return tok(Colon)
		case '+':
			return tok(PlusOp)
		case '-':
			return tok(MinusOp)
		case '*':
			return tok(TimesOp)
		case '/':
			if s.match('/') {
				s.eatline()
				continue
			}
			return tok(DivOp)
		case '.':
			u := s.peek()
			if u >= '0' && u <= '9' {
				return s.number(start)
			}
			return tok(Dot)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return s.number(start)
		case '<':
			if s.match('=') {
				return tok(LessEq)
			}
			return tok(LessThan)
		case '>':
			if s.match('=') {
				return tok(GreaterEq)
			}
			return tok(GreaterThan)
		case ' ', '\t', '\n', '\r':
			// Skip whitespace
			continue
		}
		return Token{}, &ScanError{pos: start, msg: fmt.Sprintf("Invalid lexeme '%c'", r)}
	}
	return Token{Typ: EndOfInput, Pos: s.pos, End: s.pos, Val: "<eoi>"}, nil
}

func (s *Scanner) eatline() {
	for !s.AtEnd() {
		c, sz := utf8.DecodeRuneInString(s.input[s.pos:])
		s.pos += sz
		if c == '\n' {
			return
		}
	}
}

func (s *Scanner) number(start int) (Token, error) {
	/*
		E			[Ee][+-]?{D}+
		FS			(f|F|l|L)
		{D}+{E}{FS}?		{ count(); return(CONSTANT); }
		{D}*"."{D}+({E})?{FS}?	{ count(); return(CONSTANT); }
		{D}+"."{D}*({E})?{FS}?	{ count(); return(CONSTANT); }
	*/
	re := regexp.MustCompile(`^(?:\d+[eE][+-]?\d+|\d*\.\d+(?:[eE][+-]?\d+)?|\d+\.\d*(?:[eE][+-]?\d+)?|(\d+))`)
	ix := re.FindStringSubmatchIndex(s.input[start:])
	if ix == nil {
		return Token{}, &ScanError{pos: start, msg: "Invalid double literal"}
	}
	s.pos = ix[1]
	typ := IntLiteral
	if ix[2] < 0 {
		typ = DoubleLiteral
	}
	return Token{Typ: typ, Pos: start, End: s.pos, Val: s.input[start:s.pos]}, nil
}
