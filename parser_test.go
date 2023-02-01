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

func TestParseFieldAcc(t *testing.T) {
	ts, err := scanTokens("{}.a.b")
	if err != nil {
		t.Fatalf("Unexpected error while scanning the input: %s", err)
	}
	p := NewParser(ts)
	e, err := p.Expression()
	if err != nil {
		t.Fatalf("Could not parse expression: %s", err)
	}
	fa, ok := e.(*FieldAcc)
	if !ok {
		t.Fatalf("Expected a FieldAcc expression, got sth else")
	}
	if fa.Name != "b" {
		t.Fatalf("Expected .b field access, got %v", fa.Name)
	}
}

func TestParseRecordExpr(t *testing.T) {
	ts, err := scanTokens(`{
		let a: 9
		x: 1 
		y: "a"
	}`)
	if err != nil {
		t.Fatalf("Unexpected error while scanning the input: %s", err)
	}
	p := NewParser(ts)
	e, err := p.Expression()
	if err != nil {
		t.Fatalf("Could not parse expression: %s", err)
	}
	r, ok := e.(*RecExpr)
	if !ok {
		t.Fatalf("Expected a record expression, got sth else")
	}
	if r == nil {
		t.Fatalf("Unexpected nil record expression")
	}
	// Let binding
	if len(r.LetVars) != 1 {
		t.Fatalf("Expected one let binding, got %d", len(r.LetVars))
	}
	if a, ok := r.LetVars["a"]; !ok {
		t.Fatal("Missing 'a' let binding")
	} else if a.Val.(*IntLiteral).Val != 9 {
		t.Fatalf("Expected a to be 9, got %v", a.Val)
	}
	// Fields
	if len(r.Fields) != 2 {
		t.Fatalf("Expected two fields, got %d", len(r.Fields))
	}
	if x, ok := r.Fields["x"]; !ok {
		t.Fatal("Missing 'x' field")
	} else if x.Val.(*IntLiteral).Val != 1 {
		t.Fatalf("Expected .x to be 1, got %v", x.Val)
	}
	if y, ok := r.Fields["y"]; !ok {
		t.Fatal("Missing 'y' field")
	} else if y.Val.(*StrLiteral).Val != "a" {
		t.Fatalf("Expected .y to be \"a\", got \"%v\"", y.Val)
	}
}
