package gokonfi

import (
	"fmt"
	"regexp"
	"unicode"
	"unicode/utf8"
)

// Scanner contains the full input and current scanning state.
type Scanner struct {
	input string
	pos   int
}

// Creates a new scanner from the given input.
func NewScanner(input string) Scanner {
	return Scanner{input: input, pos: 0}
}

//go:generate stringer -type=TokenType
type TokenType int32

const (
	Unspecified TokenType = iota
	// Literals
	IntLiteral
	DoubleLiteral
	StrLiteral
	// Operators
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
	Dot
	// Separators
	Comma
	LeftParen
	RightParen
	LeftBrace
	RightBrace
	Colon
	// Identifiers
	Ident
	Keyword
	// Don't treat end of input as an error, but use a special token.
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

var (
	keywords = map[string]bool{
		"func": true,
		"let":  true,
	}
)

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
		// Check for identfier, which has too many possible first characters for a switch:
		if r == '_' || unicode.IsLetter(r) {
			return s.ident(start)
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
		case '"', '\'':
			return s.stringLit(r)
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

func (s *Scanner) ident(start int) (Token, error) {
	cur := start
	for cur < len(s.input) {
		r, size := utf8.DecodeRuneInString(s.input[cur:])
		if !(unicode.IsLetter(r) || r == '_' || cur > start && unicode.IsDigit(r)) {
			break
		}
		cur += size
	}
	if cur > start {
		s.pos = cur
		typ := Ident
		if _, ok := keywords[s.input[start:s.pos]]; ok {
			typ = Keyword
		}
		return Token{Typ: typ, Pos: start, End: s.pos, Val: s.input[start:s.pos]}, nil
	}
	return Token{}, &ScanError{pos: start, msg: "Invalid identifier"}
}

// Parses IntLiterals and DoubleLiterals.
func (s *Scanner) number(start int) (Token, error) {
	re := regexp.MustCompile(`^(?:\d+[eE][+-]?\d+|\d*\.\d+(?:[eE][+-]?\d+)?|\d+\.\d*(?:[eE][+-]?\d+)?|(\d+))`)
	ix := re.FindStringSubmatchIndex(s.input[start:])
	if ix == nil {
		return Token{}, &ScanError{pos: start, msg: "Invalid double literal"}
	}
	s.pos = ix[1]
	typ := IntLiteral
	if ix[2] < 0 {
		// Did not match the group for integer literals.
		typ = DoubleLiteral
	}
	return Token{Typ: typ, Pos: start, End: s.pos, Val: s.input[start:s.pos]}, nil
}

func (s *Scanner) stringLit(start int, delim rune) (Token, error) {
	ndelim := 1 // 1st delim was already parsed.
	for !s.AtEnd() && s.match(delim) {
		ndelim++
	}
	switch ndelim {
	case 1:
		// Parse string contents
		return s.stringOneline(start, delim)
	case 2:
		// Empty string
		return Token{Typ: StrLiteral, Pos: start, End: s.pos, Val: ""}, nil
	case 3:
		return s.stringMultiline(start, delim)
	}
	return Token{}, &ScanError{pos: start, msg: "Invalid string literal"}
}

func (s *Scanner) stringOneline(start int, delim rune) (Token, error) {

}

func (s *Scanner) stringMultiline(start int, delim rune) (Token, error) {

}
