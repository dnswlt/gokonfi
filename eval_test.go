package gokonfi

import (
	"fmt"
	"testing"
)

func TestEvalArithmeticExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "1", want: IntVal(1)},
		{input: "3 * 10 + 1", want: IntVal(31)},
		{input: "3 + 10 * 2", want: IntVal(23)},
		{input: "3 * (10 + 1)", want: IntVal(33)},
		{input: "10. / -2.", want: DoubleVal(-5.)},
		{input: "2 * 3 * 4 * 5 * 6", want: IntVal(720)},
		{input: "5 - 4 - 1", want: IntVal(0)},
		{input: "5 - (4 - 1)", want: IntVal(2)},
		{input: "(100 * 2 + 100) / -300", want: IntVal(-1)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, NewCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

func TestEvalComparisonExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "1 == 2", want: BoolVal(false)},
		{input: "nil == nil", want: BoolVal(true)},
		{input: "'foo' == 'foo'", want: BoolVal(true)},
		{input: "'foo' != 'bar'", want: BoolVal(true)},
		{input: "-0. == 0.", want: BoolVal(true)},
		{input: "1 < 2", want: BoolVal(true)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, NewCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

func TestEvalLogicalExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "true && false", want: BoolVal(false)},
		{input: "!(1==2)", want: BoolVal(true)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, NewCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

func TestEvalRecExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "{x: 1}.x", want: IntVal(1)},
		{input: "{x: 1 y: {a: 10 b: a}}.y.b", want: IntVal(10)},
		{input: "{x: 1 y: {a: 10 b: a + x}}.y.b", want: IntVal(11)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, NewCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

func TestEvalConditionalExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "if 1 < 2 then 'good' else 'bad'", want: StringVal("good")},
		{input: "if 1 < 2 then (if 2 < 3 then 'good' else 'bad') else 'verybad'", want: StringVal("good")},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, NewCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

func TestEvalBuiltins(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "len('')", want: IntVal(0)},
		{input: "len('foo' + 'bar')", want: IntVal(6)},
		{input: "len({a: 1 b: 2})", want: IntVal(2)},
		{input: "len({})", want: IntVal(0)},
		// Let variables are not fields, so don't add to record length:
		{input: "len({let x: 0 y: x - 1})", want: IntVal(1)},
		// contains
		{input: "{let s: 'affe' let t: 'ff' r: contains(s, t)}.r", want: BoolVal(true)},
		{input: "cond(1 == 2, 'insane', 'sane')", want: StringVal("sane")},
		// Conditional nil field. For now, we don't delete fields that are nil.
		{input: "{let enabled: false arg: cond(enabled, 'doSomething', nil)}.arg", want: NilVal{}},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}

/*
Some syntactic sugar ideas:

let func f(x) {}
  {same as}
let f: func (x) {}
let template f(x) {}
let f: template (x) {}
*/

func TestEvalFunc(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "{let f: func (x) { x * x } y: f(9)}.y", want: IntVal(81)},
		{input: `{
			let f: func (x) { x * x } 
			let g: func (x) { f(f(x)) + 1 }
			y: g(9)
			}.y`, want: IntVal((9*9)*(9*9) + 1)},
		{input: `{
			let f: func (x, y, z) { cond(x, z, y) } 
			y: f("string_as_bool", 10, 11)
			}.y`, want: IntVal(11)},
		{input: `{
			let f: template (x) { val: x } 
			y: f('a')
			}.y.val`, want: StringVal("a")},
		// Factorial, can't go without it:
		{input: `{
			let fac: func (n) { if n == 0 then 1 else n * fac(n-1) } 
			y: fac(10)
			}.y`, want: IntVal(3628800)},
		// Higher-order functions? Piece-o-cake:
		{input: `{
			let apply_n: func (f, n, v) { if n == 0 then v else apply_n(f, n-1, f(v)) } 
			y: apply_n(func (x) {x * x}, 4, 2)
			}.y`, want: IntVal(65536)},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if got != test.want {
				t.Errorf("Got %v, want %v", got, test.want)
			}

		})
	}
}
