package gokonfi

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dnswlt/gokonfi/token"
	"github.com/google/go-cmp/cmp"
)

func newTestScanner(input string) *Scanner {
	return NewScanner(input, nil)
}

func compareTokenTypes(t *testing.T, actual, expected []token.TokenType) {
	if len(actual) != len(expected) {
		t.Fatalf("Unexpected number of tokens: got %d, expected %d", len(actual), len(expected))
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("Expected token %s at index %d, got %s", expected[i], i, actual[i])
		}
	}
}

func TestScanSymbols(t *testing.T) {
	symbols := "+-*/(){}.:"
	s := newTestScanner(symbols)
	tokenTypes := []token.TokenType{}
	for !s.AtEnd() {
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning symbols: %s", err)
		}
		tokenTypes = append(tokenTypes, tok.Typ)
	}
	expected := []token.TokenType{token.Plus, token.Minus, token.Times, token.Div,
		token.LeftParen, token.RightParen, token.LeftBrace, token.RightBrace, token.Dot,
		token.Colon}
	compareTokenTypes(t, tokenTypes, expected)
}

func TestScanOperators(t *testing.T) {
	tests := []struct {
		op   string
		want token.TokenType
	}{
		{op: "+", want: token.Plus},
		{op: "-", want: token.Minus},
		{op: "*", want: token.Times},
		{op: "/", want: token.Div},
		{op: "@", want: token.Merge},
		{op: ".", want: token.Dot},
		{op: "!", want: token.Not},
		{op: ":", want: token.Colon},
		{op: "::", want: token.OfType},
		{op: "(", want: token.LeftParen},
		{op: ")", want: token.RightParen},
		{op: "{", want: token.LeftBrace},
		{op: "}", want: token.RightBrace},
		{op: "[", want: token.LeftSquare},
		{op: "]", want: token.RightSquare},
		{op: "==", want: token.Equal},
		{op: "!=", want: token.NotEqual},
		{op: "<", want: token.LessThan},
		{op: "<=", want: token.LessEq},
		{op: ">", want: token.GreaterThan},
		{op: ">=", want: token.GreaterEq},
		{op: "&&", want: token.LogicalAnd},
		{op: "||", want: token.LogicalOr},
	}
	for _, test := range tests {
		s := newTestScanner(test.op)
		got, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning symbol: %s", err)
		}
		if got.Typ != test.want {
			t.Errorf("Want token %s, got %s", test.want, got.Typ)
		}
	}
}

func TestScanExpr(t *testing.T) {
	symbols := "2 * (3 + 4)"
	s := newTestScanner(symbols)
	tokenTypes := []token.TokenType{}
	for !s.AtEnd() {
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning symbols: %s", err)
		}
		tokenTypes = append(tokenTypes, tok.Typ)
	}
	expected := []token.TokenType{token.IntLiteral, token.Times, token.LeftParen, token.IntLiteral,
		token.Plus, token.IntLiteral, token.RightParen}
	compareTokenTypes(t, tokenTypes, expected)
}

func TestScanSkipsWhitespace(t *testing.T) {
	s := newTestScanner("     \t    \n   +\nx   \t\t\n   +")
	tokenTypes := []token.TokenType{}
	for !s.AtEnd() {
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning symbols: %s", err)
		}
		tokenTypes = append(tokenTypes, tok.Typ)
	}
	expected := []token.TokenType{token.Plus, token.Ident, token.Plus}
	compareTokenTypes(t, tokenTypes, expected)
}

func TestScanUnknown(t *testing.T) {
	s := newTestScanner("3 $")
	s.NextToken()
	_, err := s.NextToken()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if serr, ok := err.(*ScanError); !ok {
		t.Fatal("Expected ScanError, got something else")
	} else if serr.Pos() != 2 {
		t.Fatalf("Expected ScanError at 2, got it at %d", serr.Pos())
	}
}

func TestScanDouble(t *testing.T) {
	for _, dstr := range []string{"1.23", ".01", "1.", "123.4", "1e9", "17.4e-19", "0.0"} {
		s := newTestScanner(dstr)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning double literal: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != token.DoubleLiteral {
			t.Fatalf("Expected DoubleLiteral token, got %s", tok.Typ)
		}
		if tok.Val != dstr {
			t.Fatalf("Expected %s as Val, got %s", dstr, tok.Val)
		}
	}
}

func TestScanInt(t *testing.T) {
	for _, istr := range []string{"0", "9", "90", "1234"} {
		s := newTestScanner(istr)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning int literal: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != token.IntLiteral {
			t.Fatalf("Expected IntLiteral token, got %s", tok.Typ)
		}
		if tok.Val != istr {
			t.Fatalf("Expected %s as Val, got %s", istr, tok.Val)
		}

	}
}

func TestScanIntRemainder(t *testing.T) {
	s := newTestScanner("1a")
	_, err := s.NextToken()
	if err != nil {
		t.Fatalf("Error scanning int literal: %s", err)
	}
	if s.rem() != "a" {
		t.Fatalf("Expected remainder \"a\", got %s", s.rem())
	}
}

func TestScanIdentifiers(t *testing.T) {
	for _, istr := range []string{"x", "y1", "_a", "_", "_1", "longWithUpper_100"} {
		s := newTestScanner(istr)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning identifier: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != token.Ident {
			t.Fatalf("Expected Ident token, got %s", tok.Typ)
		}
		if tok.Val != istr {
			t.Fatalf("Expected %s as Val, got %s", istr, tok.Val)
		}

	}
}

func TestScanIdentifiersInvalidChars(t *testing.T) {
	for _, str := range []string{"x.a", "x$", "x?"} {
		s := newTestScanner(str)
		s.NextToken()
		if s.rem() != str[1:] {
			t.Fatalf("Expected remainder %s, got %s", str[1:], s.rem())
		}
	}
}

func TestScanKeywords(t *testing.T) {
	type TestData struct {
		input        string
		expectedType token.TokenType
	}
	for _, td := range []TestData{
		{"let", token.Let},
		{"func", token.Func},
		{"template", token.Template},
		{"if", token.If},
		{"then", token.Then},
		{"else", token.Else},
		{"nil", token.Nil},
	} {
		s := newTestScanner(td.input)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning keyword: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != td.expectedType {
			t.Fatalf("Expected Keyword token, got %s", tok.Typ)
		}
		if tok.Val != td.input {
			t.Fatalf("Expected %s as Val, got %s", td.input, tok.Val)
		}

	}
}

func TestScanOnelineString(t *testing.T) {
	type TestData struct {
		input, expected string
	}
	inputs := []TestData{
		{`"foo's bar"`, "foo's bar"},
		{`''`, ""},
		{`'Dollar $ is OK'`, "Dollar $ is OK"},
		{`'Must escape \${}'`, "Must escape ${}"},
		{`'{} is OK'`, "{} is OK"},
		{`'Say "hi"'`, "Say \"hi\""},
		{`"a\nb\tc\\\n\r\"\'"`, "a\nb\tc\\\n\r\"'"},
	}
	for _, td := range inputs {
		s := newTestScanner(td.input)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning identifier: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != token.StrLiteral {
			t.Fatalf("Expected StrLiteral token, got %s", tok.Typ)
		}
		if tok.Val != td.expected {
			t.Fatalf("Expected %s as Val, got %s", td.expected, tok.Val)
		}
	}
}

func TestScanRawString(t *testing.T) {
	type TestData struct {
		input, want string
	}
	inputs := []TestData{
		{"`very raw`", `very raw`},
		{"`very\r\nraw`", "very\nraw"},
		{"`  very\r\n  raw\n`", "  very\n  raw\n"},
		{"`" + `no \n\r\t"' escape` + "`", `no \n\r\t"' escape`},
	}
	for _, td := range inputs {
		s := newTestScanner(td.input)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning identifier: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.rem())
		}
		if tok.Typ != token.StrLiteral {
			t.Fatalf("Expected StrLiteral token, got %s", tok.Typ)
		}
		if tok.Val != td.want {
			t.Fatalf("Expected %s as Val, got %s", td.want, tok.Val)
		}
	}
}

func TestScanFormatString(t *testing.T) {
	tests := []struct {
		input          string
		expectedTokens int
	}{
		{`"${a}"`, 1},
		{`"${a} ${b}"`, 3},
		{`" ${a} ${b} ${c} "`, 7},
		// Empty ${} is discarded, but yields a FormatStrLiteral nonetheless.
		{`"foo${}bar"`, 2},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			s := newTestScanner(test.input)
			tok, err := s.NextToken()
			if err != nil {
				t.Fatalf("Error scanning identifier: %s", err)
			}
			if !s.AtEnd() {
				t.Errorf("Expected to be at end. Remaining substring: %s", s.rem())
			}
			if tok.Typ != token.FormatStrLiteral {
				t.Fatalf("Expected FormatStrLiteral token, got %s", tok.Typ)
			}
			if tok.Val != "" {
				t.Errorf("Expected empty Val, got %s", tok.Val)
			}
			got := len(tok.Fmt.Values)
			if got != test.expectedTokens {
				t.Errorf("Expected %d tokens, got %d", test.expectedTokens, got)
			}
		})
	}
}

func TestScanFormatStringValue(t *testing.T) {
	const (
		tId   = token.Ident
		tPlus = token.Plus
		tStr  = token.StrLiteral
		tInt  = token.IntLiteral
		tEoi  = token.EndOfInput
		tLb   = token.LeftBrace
		tRb   = token.RightBrace
		tCol  = token.Colon
	)
	tests := []struct {
		input     string
		wantIndex int               // Index at which we expect the FormattedValue
		wantTypes []token.TokenType // wanted token types, excluding the mandatory EndOfInput
	}{
		{input: `"alpha ${a+b}"`, wantIndex: 1, wantTypes: []token.TokenType{tId, tPlus, tId}},
		{input: `"${'a'}"`, wantIndex: 0, wantTypes: []token.TokenType{tStr}},
		{input: `"${1}"`, wantIndex: 0, wantTypes: []token.TokenType{tInt}},
		{input: `"/path/to/${'glory'}"`, wantIndex: 1, wantTypes: []token.TokenType{tStr}},
		{input: `'${"a"}'`, wantIndex: 0, wantTypes: []token.TokenType{tStr}},
		{input: `"${{a: 1}}"`, wantIndex: 0, wantTypes: []token.TokenType{tLb, tId, tCol, tInt, tRb}},
		{input: `"${'{'}"`, wantIndex: 0, wantTypes: []token.TokenType{tStr}},
		{input: `"${ '}' }"`, wantIndex: 0, wantTypes: []token.TokenType{tStr}},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			s := newTestScanner(test.input)
			tok, err := s.NextToken()
			if err != nil {
				t.Fatalf("Error scanning input: %s", err)
			}
			if tok.Typ != token.FormatStrLiteral {
				t.Fatalf("Want FormatStrLiteral, got %s", tok.Typ)
			}
			if tok.Fmt == nil {
				t.Fatalf(".Fmt is nil")
			}
			if len(tok.Fmt.Values) <= test.wantIndex {
				t.Fatalf("Want FormattedValue at index %d, but only have %d values.", test.wantIndex, len(tok.Fmt.Values))
			}
			got, ok := tok.Fmt.Values[test.wantIndex].(token.FormattedValue)
			if !ok {
				t.Fatalf("Want FormattedValue, got %T", tok.Fmt.Values[test.wantIndex])
			}
			gotTokenTypes := make([]token.TokenType, len(got.Tokens))
			for i, tok := range got.Tokens {
				gotTokenTypes[i] = tok.Typ
			}
			// Every token sequence in an interpolated expression should end with EndOfInput,
			// since that is what the parser uses to detect its own end of tokens.
			test.wantTypes = append(test.wantTypes, tEoi)
			if diff := cmp.Diff(test.wantTypes, gotTokenTypes); diff != "" {
				t.Fatalf("FormattedValue.Token mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestScanErrors(t *testing.T) {
	tests := []struct {
		input    string
		want     string
		wantRune rune
	}{
		{input: "$", want: "invalid lexeme", wantRune: '$'},
		{input: "a$", want: "invalid lexeme", wantRune: '$'},
		{input: "123\\", want: "invalid lexeme", wantRune: '\\'},
		{input: "1\n2\r\n#", want: "invalid lexeme", wantRune: '#'},
		{input: `"foo`, want: "end of input", wantRune: 0},
		{input: `123'000`, want: "end of input", wantRune: 0},
		// You cannot have \ nor the format string delimiter (here: ") anywhere inside the format string.
		{input: `"${ '\"' }"`, want: "backslash", wantRune: '\\'},
		{input: `"${ '"' }"`, want: "end of string", wantRune: '"'},
		// Format strings cannot contain newlines.
		{input: "\"${ \n }\"", want: "newline", wantRune: '\n'},
		{input: "\"${ \r }\"", want: "newline", wantRune: '\r'},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			toks, err := newTestScanner(test.input).ScanAll()
			if err == nil {
				t.Fatalf("expected error, got %v", toks)
			}
			gotErr, ok := err.(*ScanError)
			if !ok {
				t.Fatalf("expected ScanError, got %v", err)
			}
			if !strings.Contains(gotErr.msg, test.want) {
				t.Errorf("want err \"%s\", got \"%s\"", test.want, gotErr.msg)
			}
			gotPos := int(gotErr.Pos())
			var gotRune rune = 0
			if gotPos < len(test.input) {
				gotRune, _ = utf8.DecodeRuneInString(test.input[gotPos:])
			}
			if gotRune != test.wantRune {
				t.Errorf("want error at character `%c`, got it at `%c` (@%d)", test.wantRune, gotRune, gotErr.Pos())
			}
		})
	}
}
