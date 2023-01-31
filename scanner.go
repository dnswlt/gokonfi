package gokonfi

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Scanner contains the full input and current scanning state.
type Scanner struct {
	input string
	mark  int
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

func (s *Scanner) setMark() {
	s.mark = s.pos
}

func (s *Scanner) advance() rune {
	if s.AtEnd() {
		return 0
	}
	r, size := utf8.DecodeRuneInString(s.input[s.pos:])
	s.pos += size
	return r
}

func (s *Scanner) peek() rune {
	if s.AtEnd() {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(s.input[s.pos:])
	return r
}

func (s *Scanner) match(expected rune) bool {
	if s.peek() == expected {
		s.advance()
		return true
	}
	return false
}

func (s *Scanner) NextToken() (Token, error) {
	// Iterate until a token is found, skipping comments and whitespace.
	for !s.AtEnd() {
		s.setMark()
		r := s.advance()
		if r == utf8.RuneError {
			return Token{}, &ScanError{pos: s.mark, msg: "Invalid UTF-8 code point"}
		}
		// Advance scanner
		tok := func(t TokenType) (Token, error) {
			return Token{Typ: t, Pos: s.mark, End: s.pos, Val: s.input[s.mark:s.pos]}, nil
		}
		// Check for identfier, which has too many possible first characters for a switch:
		if r == '_' || unicode.IsLetter(r) {
			return s.ident()
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
				return s.number()
			}
			return tok(Dot)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return s.number()
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
		return Token{}, &ScanError{pos: s.mark, msg: fmt.Sprintf("Invalid lexeme '%c'", r)}
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

func (s *Scanner) token(typ TokenType) Token {
	return Token{Typ: typ, Pos: s.mark, End: s.pos, Val: s.input[s.mark:s.pos]}
}

func (s *Scanner) ident() (Token, error) {
	cur := s.mark
	for cur < len(s.input) {
		r, size := utf8.DecodeRuneInString(s.input[cur:])
		if !(unicode.IsLetter(r) || r == '_' || cur > s.mark && unicode.IsDigit(r)) {
			break
		}
		cur += size
	}
	if cur > s.mark {
		s.pos = cur
		typ := Ident
		if _, ok := keywords[s.input[s.mark:s.pos]]; ok {
			typ = Keyword
		}
		return s.token(typ), nil
	}
	return Token{}, &ScanError{pos: s.mark, msg: "Invalid identifier"}
}

// Parses IntLiterals and DoubleLiterals.
func (s *Scanner) number() (Token, error) {
	re := regexp.MustCompile(`^(?:\d+[eE][+-]?\d+|\d*\.\d+(?:[eE][+-]?\d+)?|\d+\.\d*(?:[eE][+-]?\d+)?|(\d+))`)
	ix := re.FindStringSubmatchIndex(s.input[s.mark:])
	if ix == nil {
		return Token{}, &ScanError{pos: s.mark, msg: "Invalid double literal"}
	}
	s.pos = ix[1]
	typ := IntLiteral
	if ix[2] < 0 {
		// Did not match the group for integer literals.
		typ = DoubleLiteral
	}
	return s.token(typ), nil
}

func (s *Scanner) stringLit(delim rune) (Token, error) {
	ndelim := 1 // 1st delim was already parsed.
	for !s.AtEnd() && s.match(delim) {
		ndelim++
	}
	switch ndelim {
	case 1:
		// Parse string contents
		return s.stringOneline(delim)
	case 2:
		// Empty string
		return Token{Typ: StrLiteral, Pos: s.mark, End: s.pos, Val: ""}, nil
	case 3:
		return s.stringMultiline(delim)
	}
	return Token{}, &ScanError{pos: s.mark, msg: "Invalid string literal"}
}

func (s *Scanner) stringOneline(delim rune) (Token, error) {
	var b strings.Builder
	for !s.AtEnd() {
		r := s.advance()
		if r == delim {
			return Token{Typ: StrLiteral, Pos: s.mark, End: s.pos, Val: b.String()}, nil
		} else if r == '\n' || r == '\r' {
			return Token{}, &ScanError{pos: s.pos, msg: "Unexpected newline in string literal"}
		} else if r == '\\' {
			// TODO: this will yield a slightly confusing error message when we're at EOI.
			r = s.advance()
			switch r {
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case '"':
				b.WriteRune('"')
			case '\'':
				b.WriteRune('\'')
			case '\\':
				b.WriteRune('\\')
			default:
				return Token{}, &ScanError{pos: s.pos, msg: fmt.Sprintf("Invalid escape character '%c'", r)}
			}
		} else {
			b.WriteRune(r)
		}

	}
	return Token{}, &ScanError{pos: s.pos, msg: "End of input while scanning string literal"}
}

func (s *Scanner) stringMultiline(delim rune) (Token, error) {
	return Token{}, &ScanError{pos: s.mark, msg: "Multiline strings are not implemented yet"}
}
