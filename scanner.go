package gokonfi

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dnswlt/gokonfi/token"
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

// ScanError is the error type returned by calls to [Scanner.NextToken].
type ScanError struct {
	pos token.Pos
	msg string
}

var (
	keywords = map[string]token.TokenType{
		"func":     token.Func,
		"let":      token.Let,
		"template": token.Template,
		"if":       token.If,
		"then":     token.Then,
		"else":     token.Else,
		"true":     token.BoolLiteral,
		"false":    token.BoolLiteral,
		"nil":      token.NilLiteral,
	}

	numberRegexp = regexp.MustCompile(`^(?:\d+[eE][+-]?\d+|\d*\.\d+(?:[eE][+-]?\d+)?|\d+\.\d*(?:[eE][+-]?\d+)?|(\d+))`)
)

// Returns the position at which the ScanError occurred.
func (s *ScanError) Pos() token.Pos {
	return s.pos
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("scanError: %s at position %d", e.msg, e.pos)
}

// AtEnd returns true if the scanner has processed its input entirely.
func (s *Scanner) AtEnd() bool {
	return s.pos >= len(s.input)
}

func (s *Scanner) rem() string {
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

func (s *Scanner) val() string {
	return s.input[s.mark:s.pos]
}

func (s *Scanner) token(typ token.TokenType) (token.Token, error) {
	return s.tokenVal(typ, s.val())
}

func (s *Scanner) tokenVal(typ token.TokenType, val string) (token.Token, error) {
	return token.Token{Typ: typ, Pos: token.Pos(s.mark), End: token.Pos(s.pos), Val: val}, nil
}

// NextToken scans the next token in the input and advances the scanner state.
// This function is where all the lexing magic happens.
//
// If the scanner has reached the end of the input, it returns [token.EndOfInput].
func (s *Scanner) NextToken() (token.Token, error) {
	// Iterate until a token is found, skipping comments and whitespace.
	for !s.AtEnd() {
		s.setMark()
		r := s.advance()
		if r == utf8.RuneError {
			return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: "Invalid UTF-8 code point"}
		}
		// Advance scanner
		// Check for identfier, which has too many possible first characters for a switch:
		if r == '_' || unicode.IsLetter(r) {
			return s.ident()
		}
		// Dispatch based on first character.
		switch r {
		case '(':
			return s.token(token.LeftParen)
		case ')':
			return s.token(token.RightParen)
		case '{':
			return s.token(token.LeftBrace)
		case '}':
			return s.token(token.RightBrace)
		case '[':
			return s.token(token.LeftSquare)
		case ']':
			return s.token(token.RightSquare)
		case ',':
			return s.token(token.Comma)
		case ':':
			return s.token(token.Colon)
		case '+':
			return s.token(token.Plus)
		case '-':
			return s.token(token.Minus)
		case '*':
			return s.token(token.Times)
		case '@':
			return s.token(token.Merge)
		case '/':
			if s.match('/') {
				s.eatline()
				continue
			}
			return s.token(token.Div)
		case '.':
			u := s.peek()
			if u >= '0' && u <= '9' {
				return s.number()
			}
			return s.token(token.Dot)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return s.number()
		case '<':
			if s.match('=') {
				return s.token(token.LessEq)
			}
			return s.token(token.LessThan)
		case '>':
			if s.match('=') {
				return s.token(token.GreaterEq)
			}
			return s.token(token.GreaterThan)
		case '=':
			if s.match('=') {
				return s.token(token.Equal)
			}
		case '!':
			if s.match('=') {
				return s.token(token.NotEqual)
			}
			return s.token(token.Not)
		case '&':
			if s.match('&') {
				return s.token(token.LogicalAnd)
			}
		case '|':
			if s.match('|') {
				return s.token(token.LogicalOr)
			}
		case '"', '\'':
			return s.stringLit(r)
		case ' ', '\t', '\n', '\r':
			// Skip whitespace
			continue
		}
		return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: fmt.Sprintf("Invalid lexeme '%c'", r)}
	}
	return s.token(token.EndOfInput)
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

func (s *Scanner) ident() (token.Token, error) {
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
		typ := token.Ident
		if kwTyp, ok := keywords[s.val()]; ok {
			typ = kwTyp
		}
		return s.token(typ)
	}
	return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: "Invalid identifier"}
}

// Parses IntLiterals and DoubleLiterals.
func (s *Scanner) number() (token.Token, error) {
	ix := numberRegexp.FindStringSubmatchIndex(s.input[s.mark:])
	if ix == nil {
		return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: "Invalid double literal"}
	}
	s.pos = s.mark + ix[1]
	typ := token.IntLiteral
	if ix[2] < 0 {
		// Did not match the group for integer literals.
		typ = token.DoubleLiteral
	}
	return s.token(typ)
}

func (s *Scanner) stringLit(delim rune) (token.Token, error) {
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
		return s.tokenVal(token.StrLiteral, "")
	case 3:
		return s.stringMultiline(delim)
	}
	return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: "Invalid string literal"}
}

func (s *Scanner) stringOneline(delim rune) (token.Token, error) {
	var b strings.Builder
	for !s.AtEnd() {
		r := s.advance()
		if r == delim {
			return s.tokenVal(token.StrLiteral, b.String())
		} else if r == '\n' || r == '\r' {
			return token.Token{}, &ScanError{pos: token.Pos(s.pos), msg: "Unexpected newline in string literal"}
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
				return token.Token{}, &ScanError{pos: token.Pos(s.pos), msg: fmt.Sprintf("Invalid escape character '%c'", r)}
			}
		} else {
			b.WriteRune(r)
		}

	}
	return token.Token{}, &ScanError{pos: token.Pos(s.pos), msg: "End of input while scanning string literal"}
}

func (s *Scanner) stringMultiline(delim rune) (token.Token, error) {
	return token.Token{}, &ScanError{pos: token.Pos(s.mark), msg: "Multiline strings are not implemented yet"}
}
