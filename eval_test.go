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
		{input: "true && false", want: BoolVal(false)},
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
