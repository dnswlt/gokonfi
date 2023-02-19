package gokonfi

import (
	"fmt"
	"testing"
)

func TestEncodeAsJson(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "1 + 3", want: "4"},
		{input: "{x: 1}", want: "{\"x\":1}"},
		{input: "{x: 1 y: 'a' z: false w: 1e6}", want: `{"w":1000000,"x":1,"y":"a","z":false}`},
		{input: "{x: {y: {z: 0}}}", want: `{"x":{"y":{"z":0}}}`},
		{input: "{x: nil}", want: `{"x":null}`},
		{input: "{let f(x): x + '.exe' y: f('konfi')}", want: `{"y":"konfi.exe"}`},
		{input: "{x: [1, 2]}", want: `{"x":[1,2]}`},
		// Don't want any pesky HTML escaping for < and >
		{input: "{x: '<>'}", want: `{"x":"<>"}`},
		{input: "['<>']", want: `["<>"]`},
		{input: `{x: 7::minutes y: 7::hours}`, want: `{"x":7,"y":7}`},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			e, err := parse(test.input)
			if err != nil {
				t.Fatalf("Could not parse expression: %s", err)
			}
			v, err := Eval(e, GlobalCtx())
			if err != nil {
				t.Fatalf("Could not evaluate expression: %s", err)
			}
			got, err := EncodeAsJson(v)
			if err != nil {
				t.Fatalf("Could not encode value as JSON: %s", err)
			}
			if got != test.want {
				t.Errorf("Got: %s, want: %s", got, test.want)
			}
		})
	}
}
