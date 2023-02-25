package gokonfi

import (
	"fmt"
	"os"
	"path"
	"strings"
)

const (
	konfiFileExtension = ".konfi"
	konfiPathEnv       = "KONFIPATH"
)

// LoadModule loads a module specified by a file path or module name.
//
// A module name gets resolved to a filename by checking for files of the
// given name (with konfiFileExtension appended) in ctx's current working directory
// and directories specified in konfiPathEnv.
//
// The module gets evaluated in the given ctx.
//
// If the module is loaded successfully, it is stored in ctx.
func LoadModule(name string, ctx *Ctx) (*loadedModule, error) {
	filename, ok := fileForModule(name, ctx.cwd())
	if !ok {
		return nil, fmt.Errorf("LoadModule: module %q not found in %q or %s", name, ctx.cwd(), konfiPathEnv)
	}
	// Check if module has already been loaded.
	if m := ctx.LookupModule(filename); m != nil {
		return m, nil
	}
	// Check for load dependency cycle.
	if ctx.isActiveFile(filename) {
		return nil, fmt.Errorf("LoadModule: load cycle detected while loading %q", filename)
	}
	// Read and parse file.
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("LoadModule: error reading module file: %w", err)
	}
	input := string(data)
	file := ctx.addFile(filename, len(input))
	mod, err := ParseModule(input, file)
	if err != nil {
		return nil, chainError(err, "LoadModule: failed to parse module")
	}
	// Evaluate module and store it in context.
	ctx.pushFile(filename)
	defer ctx.popFile()
	m, err := EvalModule(mod, ctx)
	if err != nil {
		return nil, chainError(err, "LoadModule: failed to evaluate module")
	}
	ctx.storeModule(m)
	return m, nil
}

func fileForModule(name string, cwd string) (string, bool) {
	filename := name
	if !strings.HasSuffix(filename, konfiFileExtension) {
		filename = filename + konfiFileExtension
	}
	if path.IsAbs(filename) {
		if s, err := os.Stat(name); err == nil && !s.IsDir() {
			return name, true
		}
		return "", false
	}
	// Relative path or module name: check in all configured directories.
	kpath, ok := os.LookupEnv(konfiPathEnv)
	dirs := []string{cwd}
	if ok {
		dirs = append(strings.Split(kpath, ":"), dirs...)
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		p := path.Join(dirs[i], filename)
		if s, err := os.Stat(p); err == nil && !s.IsDir() {
			return p, true
		}
	}
	return "", false
}
