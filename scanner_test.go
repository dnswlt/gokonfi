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
	expected := []token.TokenType{token.PlusOp, token.MinusOp, token.TimesOp, token.DivOp,
		token.LeftParen, token.RightParen, token.LeftBrace, token.RightBrace, token.DotOp,
		token.Colon}
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
	expected := []token.TokenType{token.PlusOp, token.Ident, token.PlusOp}
	compareTokenTypes(t, tokenTypes, expected)
}

func TestScanUnknown(t *testing.T) {
	s := NewScanner("3 @")
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
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.Rem())
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
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.Rem())
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
	if s.Rem() != "a" {
		t.Fatalf("Expected remainder \"a\", got %s", s.Rem())
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
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.Rem())
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
		if s.Rem() != str[1:] {
			t.Fatalf("Expected remainder %s, got %s", str[1:], s.Rem())
		}
	}
}

func TestScanKeywords(t *testing.T) {
	for _, istr := range []string{"let", "func"} {
		s := NewScanner(istr)
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning keyword: %s", err)
		}
		if !s.AtEnd() {
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.Rem())
		}
		if tok.Typ != token.Keyword {
			t.Fatalf("Expected Keyword token, got %s", tok.Typ)
		}
		if tok.Val != istr {
			t.Fatalf("Expected %s as Val, got %s", istr, tok.Val)
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
			t.Fatalf("Expected to be at end. Remaining substring: %s", s.Rem())
		}
		if tok.Typ != token.StrLiteral {
			t.Fatalf("Expected StrLiteral token, got %s", tok.Typ)
		}
		if tok.Val != td.expected {
			t.Fatalf("Expected %s as Val, got %s", td.expected, tok.Val)
		}
	}
}
