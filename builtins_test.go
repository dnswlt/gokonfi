package gokonfi

import (
	"fmt"
	"testing"
)

func TestFormatSingleArg(t *testing.T) {
	tests := []struct {
		format string
		arg    Val
		want   string
	}{
		{format: "%d", arg: IntVal(3), want: "3"},
		{format: "%02d", arg: IntVal(3), want: "03"},
		{format: "%.2f", arg: DoubleVal(1.0 / 3), want: "0.33"},
		{format: "pre-%s-post", arg: StringVal("alpha"), want: "pre-alpha-post"},
		{format: "%v", arg: NilVal{}, want: "nil"},
		{format: "%t", arg: BoolVal(true), want: "true"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			args := []Val{
				StringVal(test.format),
				test.arg,
			}
			got, err := builtinFormat(args)
			if err != nil {
				t.Fatalf("Failed to format: %s", err)
			}
			if string(got.(StringVal)) != test.want {
				t.Errorf("Want: %s, got %s", test.want, got)
			}
		})
	}
}
