package gokonfi

import (
	"testing"

	"github.com/dnswlt/gokonfi/token"
)

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
	s := NewScanner(symbols)
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
		{op: ".", want: token.Dot},
		{op: "!", want: token.Not},
		{op: ":", want: token.Colon},
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
		s := NewScanner(test.op)
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
	s := NewScanner(symbols)
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
	s := NewScanner("     \t    \n   +\nx   \t\t\n   +")
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
	s := NewScanner("3 $")
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
		s := NewScanner(dstr)
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
		s := NewScanner(istr)
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
	s := NewScanner("1a")
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
		s := NewScanner(istr)
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
		s := NewScanner(str)
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
	} {
		s := NewScanner(td.input)
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
		{`'Say "hi"'`, "Say \"hi\""},
		{`"a\nb\tc\\\n\r\"\'"`, "a\nb\tc\\\n\r\"'"},
	}
	for _, td := range inputs {
		s := NewScanner(td.input)
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
