package gokonfi

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			got, err := builtinFormat(args, nil)
			if err != nil {
				t.Fatalf("Failed to format: %s", err)
			}
			if string(got.(StringVal)) != test.want {
				t.Errorf("Want: %s, got %s", test.want, got)
			}
		})
	}
}

func TestIsnil(t *testing.T) {
	tests := []struct {
		input Val
		want  bool
	}{
		{input: BoolVal(true), want: false},
		{input: IntVal(1), want: false},
		{input: NilVal{}, want: true},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := builtinIsnil([]Val{test.input}, nil)
			if err != nil {
				t.Fatalf("Error calling isnil: %s", err)
			}
			if got != BoolVal(test.want) {
				t.Errorf("Want: %v, got %v", test.want, got)
			}
		})
	}
}

func TestLenientParseTime(t *testing.T) {
	r := func(vals []int) *RecVal {
		fields := []string{"year", "month", "day", "hour", "minute", "second", "nanosecond", "offset"}
		rec := NewRec()
		for i, f := range fields {
			v := 0
			if i < len(vals) {
				v = vals[i]
			}
			rec.setField(f, IntVal(v), nil)
		}
		return rec
	}
	tests := []struct {
		name  string
		input string
		want  []int // year, month, day, hour, minute, second, nanosecond, offset(seconds)
	}{
		{
			name:  "local_datetime",
			input: "2022-02-03 17:55:10",
			want:  []int{2022, 2, 3, 17, 55, 10},
		},
		{
			name:  "datetime_with_offset",
			input: "2022-02-03 17:55:10.1001 -0700",
			want:  []int{2022, 2, 3, 17, 55, 10, 100100000, -7 * 60 * 60},
		},
		{
			name:  "iso_8601",
			input: "2022-02-03T17:55:10.1001-07:00",
			want:  []int{2022, 2, 3, 17, 55, 10, 100100000, -7 * 60 * 60},
		},
		{
			name:  "iso_date",
			input: "2022-02-03",
			want:  []int{2022, 2, 3},
		},
		{
			name:  "rfc1123z",
			input: "Wed, 22 Feb 2023 08:39:36 +0100",
			want:  []int{2023, 2, 22, 8, 39, 36, 0, 60 * 60},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := builtinLenientParseTime([]Val{StringVal(test.input)}, nil)
			if err != nil {
				t.Fatalf("Error calling lenient_parse_time: %s", err)
			}
			if diff := cmp.Diff(r(test.want), got); diff != "" {
				t.Errorf("record mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRegexpExtract(t *testing.T) {
	tests := []struct {
		s    string
		re   string
		gi   int // group index. Set to -1 if you don't want to pass it to regexp_extract.
		want string
	}{
		{s: "aabbcc", re: "ab+", gi: -1, want: "abb"},
		// No match returns "".
		{s: "abc", re: "z", gi: -1, want: ""},
		{s: "abc", re: "a(b)?d", gi: 1, want: ""},
		// Extract 1st group.
		{s: "name: foo", re: "name: (\\w+)", gi: 1, want: "foo"},
		{s: "https://www2.example.com/path/to", re: "^https?://([^/]*)/.*", gi: 1, want: "www2.example.com"},
		// Extract 2nd group, no match.
		{s: "xxx2", re: "(x)(1)?(2)", gi: 3, want: "2"},
		// Group index out of bounds returns ""
		{s: "a", re: "a", gi: 100, want: ""},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
			args := []Val{StringVal(test.s), StringVal(test.re)}
			if test.gi >= 0 {
				args = append(args, IntVal(test.gi))
			}
			got, err := builtinRegexpExtract(args, nil)
			if err != nil {
				t.Fatalf("Error calling regexp_extract: %s", err)
			}
			if got != StringVal(test.want) {
				t.Errorf("Want: %q, got %v", test.want, got)
			}
		})
	}

}

func TestRegexpExtractError(t *testing.T) {
	tests := []struct {
		name string
		s    string
		re   string
		gi   int
		args []Val // If set, passes those as args to regexp_extract
	}{
		{name: "syntax", s: "xxx", re: "+", gi: 0},
		{name: "syntax2", s: "ab", re: "(a)((b)", gi: 0},
		{name: "neg", s: "a", re: "a", gi: -1},
		{name: "1arg", args: []Val{StringVal("foo")}},
		{name: "types", args: []Val{StringVal("foo"), StringVal("bar"), DoubleVal(3.)}},
		{name: "nargs", args: []Val{StringVal("foo"), StringVal("bar"), IntVal(3), NilVal{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var args []Val
			if test.args != nil {
				args = test.args
			} else {
				args = []Val{StringVal(test.s), StringVal(test.re), IntVal(test.gi)}
			}
			got, err := builtinRegexpExtract(args, nil)
			if err == nil {
				t.Errorf("Wanted error, got match: %s", got)
			}
		})
	}

}

func TestTypeof(t *testing.T) {
	tests := []struct {
		input Val
		want  string
	}{
		{input: BoolVal(true), want: "bool"},
		{input: IntVal(1), want: "int"},
		{input: DoubleVal(1.), want: "double"},
		{input: UnitVal{V: 1., F: 1., T: builtinTypeDuration}, want: "duration"},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := builtinTypeof([]Val{test.input}, nil)
			if err != nil {
				t.Fatalf("Error calling typeof: %s", err)
			}
			if got != StringVal(test.want) {
				t.Errorf("Want: %q, got %v", test.want, got)
			}
		})
	}
}
