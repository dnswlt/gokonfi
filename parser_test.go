package gokonfi

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dnswlt/gokonfi/token"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func scanTokens(input string) ([]token.Token, error) {
	s := NewScanner(input)
	return s.ScanAll()
}

func parse(input string) (Expr, error) {
	ts, err := scanTokens(input)
	if err != nil {
		return nil, err
	}
	p := NewParser(ts)
	res, err := p.Expression()
	if err != nil {
		return nil, err
	}
	if !p.AtEnd() {
		return nil, fmt.Errorf("did not parse entire input")
	}
	return res, nil
}

func TestParseTopLevelExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Expr // Type of Expr that we want
	}{
		{name: "plus", input: "1 + 3", want: (*BinaryExpr)(nil)},
		{name: "minus", input: "1 + 3 - 2", want: (*BinaryExpr)(nil)},
		{name: "eq", input: "1 == 2", want: (*BinaryExpr)(nil)},
		{name: "unary-", input: "-2", want: (*UnaryExpr)(nil)},
		{name: "unary!", input: "!2", want: (*UnaryExpr)(nil)},
		{name: "rec", input: "{x: 1}", want: (*RecExpr)(nil)},
		{name: "int", input: "1", want: (*IntLiteral)(nil)},
		{name: "double", input: "1.3e-9", want: (*DoubleLiteral)(nil)},
		{name: "str", input: "\"foo\"", want: (*StrLiteral)(nil)},
		{name: "nil", input: "nil", want: (*NilLiteral)(nil)},
		{name: "fld", input: "{a: 'foo'}.a", want: (*FieldAcc)(nil)},
		{name: "call0", input: "f()", want: (*CallExpr)(nil)},
		{name: "call1", input: "f(1)", want: (*CallExpr)(nil)},
		{name: "call2", input: "f(1, 2)", want: (*CallExpr)(nil)},
		{name: "func", input: "func (x, y) {x + y}", want: (*FuncExpr)(nil)},
		{name: "func", input: "func (x) {x}", want: (*FuncExpr)(nil)},
		{name: "func", input: "func () {42}", want: (*FuncExpr)(nil)},
		{name: "cond", input: "if 1 == 2 then 'foo' else 'bar'", want: (*ConditionalExpr)(nil)},
		{name: "merge", input: "{x: 1} @ {y: 2}", want: (*BinaryExpr)(nil)},
		{name: "list", input: "[1, 2, 3]", want: (*ListExpr)(nil)},
		// Format strings are desugared by the parser, so expect a str call:
		{name: "fstr", input: `"${1 + 2}"`, want: (*CallExpr)(nil)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Could not parse expression: %s", err)
			}
			if reflect.TypeOf(e) != reflect.TypeOf(test.want) {
				t.Fatalf("Expected expression of type %T, got type %T", test.want, e)
			}
		})
	}
}

func TestParseLetVar(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool // Whether want success or error
	}{
		{input: "let x: 7", wantErr: false},
		{input: "let x, y: 7", wantErr: true},
		{input: "let x(y): 7", wantErr: false},
		{input: "let template x() { a: 1 }", wantErr: false},
		{input: "let w(): { a: 1 }", wantErr: false},
		{input: "let w: func() { { a: 1 } }", wantErr: false},
		{input: "let w: func x() { a: 1 }", wantErr: true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			ts, err := scanTokens(test.input)
			if err != nil {
				t.Fatalf("Scan error: %s", err)
			}
			p := NewParser(ts)
			_, err = p.letVar()
			if !test.wantErr && !p.AtEnd() {
				t.Errorf("did not parse entire input")
			}
			if test.wantErr && err == nil {
				t.Errorf("Wanted error, but got success")
			} else if !test.wantErr && err != nil {
				t.Errorf("Wanted no error, but got %s", err)
			}
		})
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

// Test helpers to generate expressions.
func rec(fields ...*RecField) *RecExpr {
	fieldMap := make(map[string]RecField)
	for _, f := range fields {
		fieldMap[f.Name] = *f
	}
	return &RecExpr{LetVars: make(map[string]LetVar), Fields: fieldMap}
}
func reclet(letvars []*LetVar, fields []*RecField) *RecExpr {
	letvarMap := make(map[string]LetVar)
	for _, lv := range letvars {
		letvarMap[lv.Name] = *lv
	}
	fieldMap := make(map[string]RecField)
	for _, f := range fields {
		fieldMap[f.Name] = *f
	}
	return &RecExpr{LetVars: letvarMap, Fields: fieldMap}
}
func fld(name string, val Expr) *RecField {
	return &RecField{Name: name, Val: val}
}
func letv(name string, val Expr) *LetVar {
	return &LetVar{Name: name, Val: val}
}
func eint(i int64) *IntLiteral {
	return &IntLiteral{Val: i}
}
func estr(s string) *StrLiteral {
	return &StrLiteral{Val: s}
}
func ecall(name string, args ...Expr) Expr {
	return &CallExpr{Func: &VarExpr{Name: name}, Args: args}
}
func binexpr(x Expr, op token.TokenType, y Expr) Expr {
	return &BinaryExpr{X: x, Y: y, Op: op}
}
func eplus(x, y Expr) Expr {
	return binexpr(x, token.Plus, y)
}

func TestParseRecordExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Expr
	}{
		{
			name: "nested rec",
			input: `{
				x: 1 
				y: "a"
				z: {
					w: 0
				}
			}`,
			want: rec(fld("x", eint(1)),
				fld("y", estr("a")),
				fld("z", rec(fld("w", eint(0)))))},
		{
			name: "let vars",
			input: `{
				let x: 1 
				let y: 2
				z: 3
			}`,
			want: reclet(
				[]*LetVar{
					letv("x", eint(1)),
					letv("y", eint(2))},
				[]*RecField{
					fld("z", eint(3))})},
	}
	// Ignore token positions when comparing Exprs.
	opts := []cmp.Option{
		cmpopts.IgnoreTypes(token.Pos(0)),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts, err := scanTokens(test.input)
			if err != nil {
				t.Fatalf("Unexpected error while scanning the input: %s", err)
			}
			p := NewParser(ts)
			got, err := p.Expression()
			if err != nil {
				t.Fatalf("Could not parse expression: %s", err)
			}
			if diff := cmp.Diff(test.want, got, opts...); diff != "" {
				t.Fatalf("Record mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFormatString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Expr
	}{
		{
			name:  "simple",
			input: `"${0}"`,
			want:  ecall("str", eint(0)),
		},
		{
			name:  "double",
			input: `"${0}/${1}?"`,
			want:  eplus(eplus(eplus(ecall("str", eint(0)), estr("/")), ecall("str", eint(1))), estr("?")),
		},
		{
			name:  "nestedexpr",
			input: `"${ 1 + len(2) }"`,
			want:  ecall("str", eplus(eint(1), ecall("len", eint(2)))),
		},
	}
	// Ignore token positions when comparing Exprs.
	opts := []cmp.Option{
		cmpopts.IgnoreTypes(token.Pos(0)),
		cmpopts.IgnoreTypes(LiteralPos{}),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parse(test.input)
			if err != nil {
				t.Fatalf("Could not parse expression: %s", err)
			}
			if diff := cmp.Diff(test.want, got, opts...); diff != "" {
				t.Fatalf("Record mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		input    string
		errAtPos int
	}{
		{input: "{z}", errAtPos: 2},
		{input: "{z: 4, y: 3}", errAtPos: 5},
		{input: "{{}}", errAtPos: 1},
		{input: "{let x(7) { 7 }}", errAtPos: 7},
		{input: "{let x() { 7 }}", errAtPos: 9},
		{input: "[[]}", errAtPos: 3},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			got, err := parse(test.input)
			if err == nil {
				t.Errorf("Want error, got a successful parse: %T", got)
			} else if parseErr, ok := err.(*ParseError); !ok {
				t.Errorf("Want ParseError, got %T", err)
			} else if parseErr.tok.Pos != token.Pos(test.errAtPos) {
				t.Errorf("Got error at pos %d (%s), want at pos %d", parseErr.tok.Pos, parseErr, test.errAtPos)
			}
		})
	}
}
