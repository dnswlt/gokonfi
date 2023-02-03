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
