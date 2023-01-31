package gokonfi

import (
	"testing"

	"github.com/dnswlt/gokonfi/token"
)

func scanTokens(input string) ([]token.Token, error) {
	s := NewScanner(input)
	r := []token.Token{}
	for {
		t, err := s.NextToken()
		if err != nil {
			return nil, err
		}
		r = append(r, t)
		if t.Typ == token.EndOfInput {
			break
		}
	}
	return r, nil
}

func TestParseExpr(t *testing.T) {
	ts, err := scanTokens("1 + 3")
	if err != nil {
		t.Fatalf("Unexpected error while scanning the input: %s", err)
	}
	p := NewParser(ts)
	e, err := p.Expression()
	if err != nil {
		t.Fatalf("Could not parse expression: %s", err)
	}
	be, ok := e.(*BinaryExpr)
	if !ok {
		t.Fatalf("Expected a binary expression, got sth else")
	}
	if be.Op != token.Plus {
		t.Fatalf("Expected Plus operator, got %s", be.Op)
	}
}
