package token

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPositionFor(t *testing.T) {
	tests := []struct {
		name string
		pos  Pos
		want Position
	}{
		{"zero", Pos(0), Position{line: 1, col: 1, file: "foo"}},
		{"f1-l2", Pos(101), Position{line: 2, col: 2, file: "foo"}},
		{"f1-l1", Pos(1000), Position{line: 1, col: 1, file: "bar"}},
		{"f2-l2", Pos(1099), Position{line: 2, col: 90, file: "bar"}},
	}
	fs := NewFileSet()
	f1 := fs.AddFile("foo", 1000)
	f1.AddLine(100)
	f1.AddLine(200)
	f2 := fs.AddFile("bar", 100)
	f2.AddLine(10)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := fs.Position(test.pos)
			if !ok {
				t.Fatalf("Wanted Position, got none")
			}
			if diff := cmp.Diff(test.want, got, cmp.AllowUnexported(Position{})); diff != "" {
				t.Errorf("Value mismatch (-want +got):\n%s", diff)
			}
		})
	}

}

func TestPositionForError(t *testing.T) {
	tests := []struct {
		name string
		pos  Pos
	}{
		{"neg", Pos(-1)},
		{"large", Pos(10000)},
	}
	fs := NewFileSet()
	f1 := fs.AddFile("foo", 1000)
	f1.AddLine(100)
	f1.AddLine(200)
	f2 := fs.AddFile("bar", 100)
	f2.AddLine(10)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := fs.Position(test.pos)
			if ok {
				t.Errorf("Wanted error, got %v", got)
			}
		})
	}

}
