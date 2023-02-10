package gokonfi

import (
	"fmt"
	"strings"
	"testing"
	"unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func TestEvalFormatString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: `"/path/to/${'glory'}"`, want: "/path/to/glory"},
		{input: `"1 ${2} 3"`, want: "1 2 3"},
		{input: `"1 ${ 2 } 3"`, want: "1 2 3"},
		{input: `{let a: { b: 1 } r: "a.b=${a.b}"}.r`, want: "a.b=1"},
		{input: `{let f(x): x + 1 r: "x=${f(1)}"}.r`, want: "x=2"},
		{input: `"${'a' + 'b'}"`, want: "ab"},
		{input: `"$foo ${'bar'}"`, want: "$foo bar"},
		{input: `"${ {a: {b: 3} }.a.b }"`, want: "3"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			v, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			got, ok := v.(StringVal)
			if !ok {
				t.Fatalf("Expected StringVal, got %T", v)
			}
			if string(got) != test.want {
				t.Errorf("Want: %s, got: %s", test.want, got)
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
		{input: "len({x: 1} @ {y: 2})", want: IntVal(2)},
		{input: "({y: 1} @ {y: 2}).y", want: IntVal(2)},
		// Right overwrites left:
		{input: "({y: {z: 1}} @ {y: {z: 2}}).y.z", want: IntVal(2)},
		// Right overwrites left, different scalar types:
		{input: "({y: {z: 1}} @ {y: {z: 'a'}}).y.z", want: StringVal("a")},
		// Record overwrites number:
		{input: "({y: 1} @ {y: {z: 2}}).y.z", want: IntVal(2)},
		// Number overwrites record:
		{input: "({y: {z: 1}} @ {y: 2}).y", want: IntVal(2)},
		// Take left if right doesn't have the field:
		{input: "({y: {z: 1 w: 2}} @ {y: {z: 0}}).y.w", want: IntVal(2)},
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

func TestEvalListExpr(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "[1, 'a']", want: &ListVal{[]Val{IntVal(1), StringVal("a")}}},
		{input: "len([]) == 0", want: BoolVal(true)},
		{input: "len([1, 2, 3])", want: IntVal(3)},
		{input: "if [1] then 'good' else 'bad'", want: StringVal("good")},
		{input: "if [] then 'bad' else 'good'", want: StringVal("good")},
		{input: "[[1]]", want: &ListVal{[]Val{&ListVal{[]Val{IntVal(1)}}}}},
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
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("List mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvalBuiltins(t *testing.T) {
	tests := []struct {
		input string
		want  Val
	}{
		{input: "str(1)", want: StringVal("1")},
		{input: "len('')", want: IntVal(0)},
		{input: "len('foo' + 'bar')", want: IntVal(6)},
		{input: "len({a: 1 b: 2})", want: IntVal(2)},
		{input: "len({})", want: IntVal(0)},
		// Let variables are not fields, so don't add to record length:
		{input: "len({let x: 0 y: x - 1})", want: IntVal(1)},
		// contains
		{input: "{let s: 'affe' let t: 'ff' r: contains(s, t)}.r", want: BoolVal(true)},
		// cond
		{input: "cond(1 == 2, 'insane', 'sane')", want: StringVal("sane")},
		// Conditional nil field. For now, we don't delete fields that are nil.
		{input: "{let enabled: false arg: cond(enabled, 'doSomething', nil)}.arg", want: NilVal{}},
		// substr
		{input: `substr("", 0, 0)`, want: StringVal("")},
		{input: `substr("abc", 1, 2)`, want: StringVal("b")},
		{input: `substr("abc", 0, 3)`, want: StringVal("abc")},
		{input: `{s: "abc" r: substr(s, 0, len(s))}.r`, want: StringVal("abc")},
		// Unicode: characters can be longer than one byte. Umlaut-u is 2 bytes:
		{input: "substr('\u00fcber', 0, 2)", want: StringVal("\u00fc")},
		// Of course, len behaves accordingly:
		{input: "len('\u00fcber')", want: IntVal(5)},
		// typeof
		{input: "typeof('')", want: StringVal("string")},
		{input: "typeof(1)", want: StringVal("int")},
		{input: "typeof(3.)", want: StringVal("double")},
		{input: "typeof(len)", want: StringVal("builtin")},
		{input: "typeof(func(){nil})", want: StringVal("func")},
		{input: "typeof(true)", want: StringVal("bool")},
		{input: "typeof({})", want: StringVal("record")},
		{input: "typeof([1,2])", want: StringVal("list")},
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

// Almost an integration test (or a functional programming competition?):
// Tests higher-order functions using list builtins (flatmap, fold).
func TestEvalFunctional(t *testing.T) {
	const input = `{
		let map(f, xs): flatmap(func (x) { [f(x)] }, xs)
		let filter(p, xs): flatmap(func (x) { if p(x) then [x] else [] }, xs)
		let concat(xs, ys): flatmap(func (x) { x }, [xs, ys])
		let max(x, y): if x > y then x else y
		let sum(x, y): x + y
		let pos(x): x > 0
		let sqr(x): x * x
	
		xs: filter(pos, [-1, 2, -3, 4, -5, 6])
		ys: map(sqr, [1, -2, 3])
		z: fold(max, [1, 2, 3, 4, -5])
		w: fold(sum, 0, [1, 2, 3, 4, 5])
		// Folding an empty list yields nil.
		nil_field: fold(func(x){x}, [])
		cs: concat(['a'], [1])
	}`
	tests := []struct {
		field string
		want  Val
	}{
		{field: "xs", want: &ListVal{[]Val{IntVal(2), IntVal(4), IntVal(6)}}},
		{field: "ys", want: &ListVal{[]Val{IntVal(1), IntVal(4), IntVal(9)}}},
		{field: "z", want: IntVal(4)},
		{field: "w", want: IntVal(15)},
		{field: "nil_field", want: NilVal{}},
		{field: "cs", want: &ListVal{[]Val{StringVal("a"), IntVal(1)}}},
	}

	e, err := parse(input)
	if err != nil {
		t.Fatalf("Cannot parse expression: %s", err)
	}
	got, err := Eval(e, GlobalCtx())
	if err != nil {
		t.Fatalf("Failed to evaluate: %s", err)
	}
	record, ok := got.(*RecVal)
	if !ok {
		t.Fatalf("Got %T, want *RecVal", got)
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			if diff := cmp.Diff(test.want, record.Fields[test.field]); diff != "" {
				t.Errorf("Record field %s mismatch (-want +got):\n%s", test.field, diff)
			}
		})
	}
}

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
			let template f(x) { val: x } 
			y: f('a')
			}.y.val`, want: StringVal("a")},
		// Factorial, can't go without it:
		{input: `{
			let fac(n): if n == 0 then 1 else n * fac(n-1)
			y: fac(10)
			}.y`, want: IntVal(3628800)},
		// Higher-order functions? Piece-o-cake:
		{input: `{
			let apply_n: func (f, n, v) { if n == 0 then v else apply_n(f, n-1, f(v)) } 
			y: apply_n(func (x) {x * x}, 4, 2)
			}.y`, want: IntVal(65536)},
		// Lexical scoping:
		{input: `{
			let adder: func (n) { func (k) { n + k } }
			let add3: adder(3) 
			y: {
				// This n is not visible to add3 in the call below.
				n: 10
				a: add3(1)
			}
			}.y.a`, want: IntVal(4)},
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

func TestEvalErrors(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "len(1)", want: "invalid type"},
		{input: "len('a', 'b')", want: "wrong number of arguments"},
		{input: "{x: 1}.y", want: "no field"},
		{input: "'a'.y", want: "cannot access"},
		{input: "{let f: 'a' y: f(0) }", want: "not callable"},
		{input: "{x: y y: x}", want: "cyclic"},
		{input: "{x: { a: b b: y.c } y: { c: x.a } }", want: "cyclic"},
		{input: "'a' + 3", want: "incompatible types"},
		{input: "1 + 1.0", want: "incompatible types"},
		{input: "(func (x) {x}) + 3", want: "incompatible types"},
		{input: "-'a'", want: "incompatible type"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err == nil {
				t.Errorf("Want error, got: %s", got)
			} else if evalErr, ok := err.(*EvalError); !ok {
				t.Errorf("Want EvalError, got %T", err)
			} else if !strings.Contains(evalErr.msg, test.want) {
				t.Errorf("Got '%s', wanted it to contain '%v'", err, test.want)
			}
		})
	}
}

func TestSizeofVal(t *testing.T) {
	// Some tests showing that RecVal and ListVal are small enough
	// to be passed by value.
	if unsafe.Sizeof((*int)(nil)) != 8 {
		t.Skip("Skipping Sizeof tests on non-64bit architecture")
	}
	if got := unsafe.Sizeof(ListVal{}); got != 24 {
		t.Errorf("Want size 24 for ListVal, got %d", got)
	}
	if got := unsafe.Sizeof(RecVal{}); got != 8 {
		t.Errorf("Want size 8 for RecVal, got %d", got)
	}
}

func TestEvalTypedExpr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Val
	}{
		{name: "int", input: "1::int", want: IntVal(1)},
		{name: "intfield", input: "{x::int: 1}.x", want: IntVal(1)},
		// Casting truncates
		{name: "d2i-trunc", input: "1.9 :: int", want: IntVal(1)},
		{name: "d2i-neg", input: "(-1.9) :: int", want: IntVal(-1)},
		{name: "i2d", input: "1::double", want: DoubleVal(1)},
		{name: "rec", input: "{x: 1.::int y: 2::double r: x::double/y}.r :: string", want: StringVal("0.5")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDurationUnit(t *testing.T) {
	u := func(x float64, name string) UnitVal {
		if f, found := builtinTypeDuration.UnitFactor(name); found {
			return UnitVal{V: x, F: f, T: builtinTypeDuration}
		}
		t.Fatalf("invalid unit multiple name: %s", name)
		return UnitVal{}
	}
	tests := []struct {
		name  string
		input string
		want  Val
	}{
		{name: "minPlusSec", input: "(7::minutes + 3::seconds)", want: u(7*60+3, "seconds")},
		{name: "hourMinusMin", input: "(1::hours - 30::minutes)", want: u(30, "minutes")},
		{name: "dayMinusDay", input: "(365::days - 30::days)", want: u(335, "days")},
		{name: "dayMinusDay", input: "(365::days - (24 * 30)::hours)", want: u(335*24, "hours")},
		{name: "dayTimesN", input: "3 * 10::days", want: u(30, "days")},
		{name: "nTimesDay", input: "10::days * 3", want: u(30, "days")},
		// Division does not change the unit multiplier:
		{name: "millisDivN", input: "10::millis / 100", want: u(0.1, "millis")},
		{name: "millisDivN", input: "str(10::millis)", want: StringVal("10::millis")},
		// Casting to int yields the value in the given unit multiple:
		{name: "millisDivN", input: "(10::minutes)::int", want: IntVal(10)},
		{name: "plusd", input: "(7::minutes + 3::seconds)::double", want: DoubleVal(7*60 + 3)},
		{name: "plusb", input: "7::minutes + 3::seconds == (7*60+3)::seconds", want: BoolVal(true)},
		// Comparisons
		{name: "cmp.lt", input: "7::minutes < 7::hours", want: BoolVal(true)},
		{name: "cmp.lt2", input: "1::minutes < 61::seconds", want: BoolVal(true)},
		{name: "cmp.gt", input: "7::minutes > 8::seconds", want: BoolVal(true)},
		{name: "cmp.le", input: "1::minutes <= 60::seconds", want: BoolVal(true)},
		{name: "cmp.ge", input: "1::days >= 1::nanos", want: BoolVal(true)},
		{name: "cmp.eq", input: "1::days == (24::hours)::days", want: BoolVal(true)},
		// 1 day is not the same as 24 hours -- the multiples are different!
		{name: "cmp.neq", input: "1::days != 24::hours", want: BoolVal(true)},
	}
	opts := []cmp.Option{
		cmpopts.IgnoreFields(Typ{}, "Convert"),
		cmpopts.IgnoreFields(Typ{}, "Unwrap"),
		cmpopts.IgnoreFields(Typ{}, "Validate"),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Failed to evaluate: %s", err)
			}
			if diff := cmp.Diff(test.want, got, opts...); diff != "" {
				t.Errorf("Value mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvalTypedExprError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "s2i", input: `"foo"::int`, want: "cannot convert"},
		{name: "s2i-prefix", input: `"3i"::int`, want: "cannot convert"},
		{name: "nil-int", input: `nil::int`, want: "cannot convert"},
		{name: "rec-int", input: `{x: 0}::int`, want: "cannot convert"},
		{name: "invalid", input: `1::doesnotexist`, want: "unknown type"},
		{name: "duration", input: `1::seconds + 3`, want: "incompatible types"},
		{name: "daySquared", input: "3::days * 10::days", want: "incompatible types"},
		{name: "dayDivided", input: "3::days / 10::days", want: "incompatible types"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Cannot parse expression: %s", err)
			}
			got, err := Eval(e, GlobalCtx())
			if err == nil {
				t.Fatalf("Expected error, got %s", got)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("Wanted error message containing %q, got: %q", test.want, err)
			}
		})
	}
}
