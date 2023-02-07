package gokonfi

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dnswlt/gokonfi/token"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type sexpr interface {
	sexpr() string
}

func (e *BinaryExpr) sexpr() string {
	return fmt.Sprintf("(%s %s %s)", e.Op, e.X.(sexpr).sexpr(), e.Y.(sexpr).sexpr())
}
func (e *UnaryExpr) sexpr() string { return fmt.Sprintf("(%s %s)", e.Op, e.X.(sexpr).sexpr()) }
func (e *FieldAcc) sexpr() string {
	return fmt.Sprintf("(%s %s %s)", token.Dot, e.X.(sexpr).sexpr(), e.Name)
}
func (e *IntLiteral) sexpr() string    { return fmt.Sprintf("%d", e.Val) }
func (e *DoubleLiteral) sexpr() string { return fmt.Sprintf("%f", e.Val) }
func (e *BoolLiteral) sexpr() string   { return fmt.Sprintf("%t", e.Val) }
func (e *StrLiteral) sexpr() string    { return fmt.Sprintf("%q", e.Val) }
func (e *NilLiteral) sexpr() string    { return "nil" }
func (e *VarExpr) sexpr() string       { return e.Name }
func (e *RecExpr) sexpr() string {
	var b strings.Builder
	b.WriteString("(rec")
	for _, f := range e.Fields {
		if f.T != nil {
			b.WriteString(fmt.Sprintf(" ((%s %s) %s)", f.T.(sexpr).sexpr(), f.Name, f.X.(sexpr).sexpr()))
		} else {
			b.WriteString(fmt.Sprintf(" (%s %s)", f.Name, f.X.(sexpr).sexpr()))
		}
	}
	b.WriteString(")")
	return b.String()
}
func (e *ListExpr) sexpr() string {
	var b strings.Builder
	b.WriteString("(rec")
	for _, f := range e.Elements {
		b.WriteString(f.(sexpr).sexpr())
	}
	b.WriteString(")")
	return b.String()
}
func (e *TypedExpr) sexpr() string {
	return fmt.Sprintf("(%s %s %s)", token.OfType, e.X.(sexpr).sexpr(), e.T.(sexpr).sexpr())
}
func (e *NamedType) sexpr() string {
	return e.Name
}
func (e *ConditionalExpr) sexpr() string {
	return fmt.Sprintf("(if %s %s %s)", e.Cond.(sexpr).sexpr(), e.X.(sexpr).sexpr(), e.Y.(sexpr).sexpr())
}
func (e *CallExpr) sexpr() string {
	var b strings.Builder
	b.WriteString("(")
	b.WriteString(e.Func.(sexpr).sexpr())
	for _, arg := range e.Args {
		b.WriteString(" ")
		b.WriteString(arg.(sexpr).sexpr())
	}
	b.WriteString(")")
	return b.String()
}
func (e *FuncExpr) sexpr() string {
	var b strings.Builder
	b.WriteString("(func (")
	for i, p := range e.Params {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p.Name)
	}
	b.WriteString(")")
	b.WriteString(e.Body.(sexpr).sexpr())
	return b.String()
}

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
		{name: "type", input: "x::int", want: (*TypedExpr)(nil)},
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
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "rec", input: "{r: 1}.r", want: "(Dot (rec (r 1)) r)"},
		{name: "multiple", input: "a.b.c", want: "(Dot (Dot a b) c)"},
		{name: "call", input: "f().x", want: "(Dot (f) x)"},
		{name: "call2", input: "f().x().y", want: "(Dot ((Dot (f) x)) y)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Could not parse expression: %s", err)
			}
			if got := e.(sexpr).sexpr(); got != test.want {
				t.Errorf("Want %s, got %s", test.want, got)
			}
		})
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
	return &RecField{AnnotatedIdent: AnnotatedIdent{Name: name}, X: val}
}
func letv(name string, val Expr) *LetVar {
	return &LetVar{Name: name, X: val}
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

func TestParseTypedExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "variable",
			input: "x::int",
			want:  "(OfType x int)",
		},
		{
			name:  "brackets",
			input: "3 + (1 + 2)::int + 10",
			want:  "(Plus (Plus 3 (OfType (Plus 1 2) int)) 10)",
		},
		{
			name:  "rec",
			input: "{} :: int @ {} :: str",
			want:  "(Merge (OfType (rec) int) (OfType (rec) str))",
		},
		{
			name:  "recfield",
			input: "{x::int: 7}",
			want:  "(rec ((int x) 7))",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotExpr, err := parse(test.input)
			if err != nil {
				t.Fatalf("failed to parse: %s", err)
			}
			gotSexpr, ok := gotExpr.(sexpr)
			if !ok {
				t.Fatalf("Type %T does not implement sexpr", gotExpr)
			}
			if got := gotSexpr.sexpr(); got != test.want {
				t.Errorf("Want: %s, got: %s", test.want, got)
			}
		})
	}
}
