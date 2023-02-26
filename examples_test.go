package gokonfi

import (
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesDirectory(t *testing.T) {
	if testing.Short() {
		return // Don't read files in short mode.
	}
	const pathGlob = "./examples/*.konfi"
	konfiFiles, err := filepath.Glob(pathGlob)
	if err != nil {
		t.Fatalf("Cannot glob %s: %s", pathGlob, err)
	}
	for _, file := range konfiFiles {
		if strings.HasPrefix(path.Base(file), "_") {
			// Skip .konfi files starting with "_", they may intentionally contain invalid syntax.
			continue
		}
		ctx := GlobalCtx()
		_, err := LoadModule(file, ctx)
		if err != nil {
			t.Errorf("Failed to load %s:\n%s", file, FormattedError(err, ctx))
		}
	}
}
