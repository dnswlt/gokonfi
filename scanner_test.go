package gokonfi

import (
	"testing"
)

func TestScanSymbols(t *testing.T) {
	symbols := "+-*/(){}.:"
	s := NewScanner(symbols)
	tokenTypes := []TokenType{}
	for !s.AtEnd() {
		tok, err := s.NextToken()
		if err != nil {
			t.Fatalf("Error scanning symbols: %s", err)
		}
		tokenTypes = append(tokenTypes, tok.Typ)
	}
	expected := []TokenType{PlusOp, MinusOp, TimesOp, DivOp, LeftParen, RightParen, LeftBrace, RightBrace, Dot, Colon}
	if len(tokenTypes) != len(expected) {
		t.Fatalf("Unexpected number of tokens: got %d, expected %d", len(tokenTypes), len(expected))
	}
	for i := range tokenTypes {
		if tokenTypes[i] != expected[i] {
			t.Fatalf("Expected token %s at index %d, got %s", expected[i], i, tokenTypes[i])
		}
	}
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
		if tok.Typ != DoubleLiteral {
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
		if tok.Typ != IntLiteral {
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
