package gokonfi

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dnswlt/gokonfi/token"
)

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
		"nil":      token.Nil,
	}
	// Used to extract integer and double literals.
	numberRegexp = regexp.MustCompile(`^(?:\d+[eE][+-]?\d+|\d*\.\d+(?:[eE][+-]?\d+)?|\d+\.\d*(?:[eE][+-]?\d+)?|(\d+))`)
)

// Scanner contains the full input and the current scanning state.
type Scanner struct {
	input string
	mark  int // Used to keep track of the start of multi-character tokens.
	pos   int // Next position in input to be scanned.
	off   int // Offset of input[0] in a broader context. Nonzero only for child scanners.
}

// Creates a new scanner from the given input.
func NewScanner(input string) *Scanner {
	return &Scanner{input: input}
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
	return token.Token{Typ: typ, Pos: s.tmark(), End: s.tpos(), Val: val}, nil
}

// childScanner returns a new child Scanner that will process a substring of s.input
// ranging from start to end (exclusive). The child scanner will have its offset set
// to start, so the tokens it returns will have Pos values from start upwards, so that
// their positions are meaningful in the context of the parent scanner's input.
func (s *Scanner) childScanner(start, end int) *Scanner {
	if end >= len(s.input) {
		end = len(s.input)
	}
	return &Scanner{input: s.input[start:end], off: start}
}

func (s *Scanner) tpos() token.Pos {
	return token.Pos(s.pos + s.off)
}

func (s *Scanner) tmark() token.Pos {
	return token.Pos(s.mark + s.off)
}

func (s *Scanner) fail(format string, args ...any) error {
	return s.failat(s.mark, format, args...)
}

func (s *Scanner) failat(pos int, format string, args ...any) error {
	return &ScanError{pos: token.Pos(pos), msg: fmt.Sprintf(format, args...)}
}

// ScanError is the error type typically returned by calls to Scanner methods.
type ScanError struct {
	pos token.Pos
	msg string
}

// Returns the position at which the ScanError occurred.
func (s *ScanError) Pos() token.Pos {
	return s.pos
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("scanError: %s at position %d", e.msg, e.pos)
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
			return token.Token{}, s.fail("invalid UTF-8 code point")
		}
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
		return token.Token{}, s.fail("invalid lexeme '%c'", r)
	}
	return s.token(token.EndOfInput)
}

// Scans all tokens in the scanner's remaining input.
// If the scan is successful, the last token
// will always be [token.EndOfInput]. If any errors occur duing the scan,
// all tokens scanned so far are returned, together with an error.
func (s *Scanner) ScanAll() ([]token.Token, error) {
	r := []token.Token{}
	for {
		t, err := s.NextToken()
		if err != nil {
			return r, err
		}
		r = append(r, t)
		if t.Typ == token.EndOfInput {
			break
		}
	}
	return r, nil
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
	return token.Token{}, s.fail("invalid identifier")
}

// Parses IntLiterals and DoubleLiterals.
func (s *Scanner) number() (token.Token, error) {
	ix := numberRegexp.FindStringSubmatchIndex(s.input[s.mark:])
	if ix == nil {
		return token.Token{}, s.fail("invalid double literal")
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
	case 2, 6:
		// Empty string
		return s.tokenVal(token.StrLiteral, "")
	case 3:
		return s.stringMultiline(delim)
	}
	return token.Token{}, s.fail("invalid string literal")
}

func (s *Scanner) stringOneline(delim rune) (token.Token, error) {
	var parts []token.FormatStrValue // Parts collected for a format string.
	partPos := s.tpos()              // Start position of the current format string part.
	var b strings.Builder
	for !s.AtEnd() {
		r := s.advance()
		if r == delim {
			// Reached the end of the string.
			if len(parts) > 0 {
				// We're in a format string.
				if b.Len() > 0 {
					parts = append(parts, token.FormatStrPart{Val: b.String(), Pos: partPos, End: s.tpos()})
				}
				return token.Token{
					Typ: token.FormatStrLiteral,
					Pos: s.tmark(),
					End: s.tpos(),
					Val: "", // format strings have all data in .Fmt.
					Fmt: &token.FormatStr{Values: parts}}, nil
			}
			// Regular string.
			return s.tokenVal(token.StrLiteral, b.String())
		} else if r == '\n' || r == '\r' {
			return token.Token{}, s.failat(s.pos, "unexpected newline in string literal")
		} else if r == '$' && s.match('{') {
			if b.Len() > 0 {
				part := token.FormatStrPart{Val: b.String(), Pos: partPos, End: s.tpos()}
				parts = append(parts, part)
				b.Reset()
			}
			exprStart := s.pos // s.pos points at the first character inside the ${} (after '{').
			err := s.skipFormatStringExpr(delim)
			if err != nil {
				return token.Token{}, err
			}
			exprEnd := s.pos - 1 // s.pos points at the first character outside the ${} (after '}').
			if exprStart == exprEnd {
				// Ignore empty interpolation ${}.
				continue
			}
			exprTokens, err := s.childScanner(exprStart, exprEnd).ScanAll()
			if err != nil {
				return token.Token{}, err
			}
			partPos = s.tpos()
			part := token.FormattedValue{Tokens: exprTokens, Pos: token.Pos(exprStart), End: token.Pos(exprEnd)}
			parts = append(parts, part)
		} else if r == '\\' {
			r = s.advance()
			switch r {
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			case '"', '\'', '\\', '$':
				b.WriteRune(r)
			default:
				return token.Token{}, s.failat(s.pos, "invalid escape character '%c'", r)
			}
		} else {
			b.WriteRune(r)
		}

	}
	return token.Token{}, s.failat(s.pos, "end of input while scanning string literal")
}

// Advances the scanner so it points at the character following the '}' that closes
// the format string interpolated expression.
//
// When calling this method, s must point at the first character of the interpolated expression.
func (s *Scanner) skipFormatStringExpr(delim rune) error {
	depth := 0
	inString := false
	for !s.AtEnd() {
		r := s.advance()
		switch r {
		case delim:
			return s.failat(s.pos, "end of string in interpolated expression")
		case '\n', '\r':
			return s.failat(s.pos, "newline in interpolated expression")
		case '\\':
			return s.failat(s.pos, "interpolated expression cannot contain a backslash")
		case '\'', '"':
			// One of these is delim, so we only end up here for the other string delimiter,
			// which can be used to delimit string literals inside the interpolated expression.
			inString = !inString
		case '}':
			if depth == 0 && !inString {
				// Reached end of interpolated expression
				return nil
			} else if !inString {
				depth--
			}
		case '{':
			if !inString {
				depth++
			}
		}
	}
	return s.fail("end of input")
}

func (s *Scanner) stringMultiline(delim rune) (token.Token, error) {
	return token.Token{}, s.fail("multiline strings are not implemented yet")
}
