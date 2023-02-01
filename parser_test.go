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

// Compare two expressions, ignoring token positions
func compareExpr(t *testing.T, lhs, rhs Expr) {
	switch v := lhs.(type) {
	case *IntLiteral:
		w, ok := rhs.(*IntLiteral)
		if !ok {
			t.Fatalf("Expected rhs of type %T, got %T", v, rhs)
		}
		if v.Val != w.Val {
			t.Fatalf("Expected %d, got %d", v.Val, w.Val)
		}
	case *StrLiteral:
		w, ok := rhs.(*StrLiteral)
		if !ok {
			t.Fatalf("Expected rhs of type %T, got %T", v, rhs)
		}
		if v.Val != w.Val {
			t.Fatalf("Expected \"%s\", got \"%s\"", v.Val, w.Val)
		}
	case *RecExpr:
		w, ok := rhs.(*RecExpr)
		if !ok {
			t.Fatalf("Expected rhs of type %T, got %T", v, rhs)
		}
		if len(v.Fields) != len(w.Fields) {
			t.Fatalf("Expected %d record fields, got %d", len(v.Fields), len(w.Fields))
		}
		for f := range v.Fields {
			if _, ok := w.Fields[f]; !ok {
				t.Fatalf("Expected field .%s in rhs", f)
			} else {
				compareExpr(t, v.Fields[f].Val, w.Fields[f].Val)
			}
		}
	}
}

// Test helpers to generate expressions.
func rec(fields ...*RecField) *RecExpr {
	fieldMap := make(map[string]RecField)
	for _, f := range fields {
		fieldMap[f.Name] = *f
	}
	return &RecExpr{LetVars: make(map[string]LetVar), Fields: fieldMap}
}
func fld(name string, val Expr) *RecField {
	return &RecField{Name: name, Val: val}
}
func intval(i int64) *IntLiteral {
	return &IntLiteral{Val: i}
}
func strval(s string) *StrLiteral {
	return &StrLiteral{Val: s}
}

func TestParseRecordExpr2(t *testing.T) {
	ts, err := scanTokens(`{
		x: 1 
		y: "a"
		z: {
			w: 0
		}
	}`)
	if err != nil {
		t.Fatalf("Unexpected error while scanning the input: %s", err)
	}
	p := NewParser(ts)
	e, err := p.Expression()
	if err != nil {
		t.Fatalf("Could not parse expression: %s", err)
	}
	compareExpr(t, e,
		rec(fld("x", intval(1)),
			fld("y", strval("a")),
			fld("z", rec(fld("w", intval(0))))))
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
