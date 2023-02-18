package gokonfi

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
)

func TestLoadModuleSameDir(t *testing.T) {
	// load('util') should work from sibling 'root' module.
	if testing.Short() {
		// Don't run tests writing to disk in -short mode.
		return
	}
	// Write modules to disk.
	d := t.TempDir()
	rootPath := path.Join(d, "root.konfi")
	rootModule := []byte(`
	{
		let m: load('util')
		x: m.one
	}
	`)
	os.WriteFile(rootPath, rootModule, 0644)
	utilPath := path.Join(d, "util.konfi")
	utilModule := []byte("{ one: 1 }")
	os.WriteFile(utilPath, utilModule, 0644)
	// Load module and check result.
	m, err := LoadModule(rootPath, GlobalCtx())
	if err != nil {
		t.Fatalf("failed to load module: %s", err)
	}
	r, ok := m.body.(*RecVal)
	if !ok {
		t.Fatalf("expected *RecVal, got %T", m.body)
	}
	got, ok := r.Fields["x"]
	if got != IntVal(1) {
		t.Errorf("want 1, got: %v", got)
	}
}

func TestLoadModuleKonfipath(t *testing.T) {
	// load('util') should work when it's on KONFIPATH.
	if testing.Short() {
		// Don't run tests writing to disk in -short mode.
		return
	}
	d1 := t.TempDir()
	d2 := t.TempDir() // Contains the loaded module.
	d3 := t.TempDir() // Empty, but added to KONFIPATH.
	os.Setenv(konfiPathEnv, fmt.Sprintf("%s:%s", d2, d3))
	// Write modules to disk.
	rootPath := path.Join(d1, "root.konfi")
	rootModule := []byte(`
	{
		let m: load('util')
		x: m.one
	}
	`)
	os.WriteFile(rootPath, rootModule, 0644)
	utilPath := path.Join(d2, "util.konfi")
	utilModule := []byte("{ one: 1 }")
	os.WriteFile(utilPath, utilModule, 0644)
	// Load module and check result.
	m, err := LoadModule(rootPath, GlobalCtx())
	if err != nil {
		t.Fatalf("failed to load module: %s", err)
	}
	r, ok := m.body.(*RecVal)
	if !ok {
		t.Fatalf("expected *RecVal, got %T", m.body)
	}
	got, ok := r.Fields["x"]
	if got != IntVal(1) {
		t.Errorf("want 1, got: %v", got)
	}
}

func TestLoadModuleSubdir(t *testing.T) {
	// load('sub/util') should work.
	if testing.Short() {
		// Don't run tests writing to disk in -short mode.
		return
	}
	// Write modules to disk.
	d := t.TempDir()
	subd := path.Join(d, "sub")
	os.Mkdir(subd, 0755)
	rootPath := path.Join(d, "root.konfi")
	rootModule := []byte(`
	{
		let m: load('sub/util')
		x: m.one
	}
	`)
	os.WriteFile(rootPath, rootModule, 0644)
	utilPath := path.Join(subd, "util.konfi")
	utilModule := []byte("{ one: 1 }")
	os.WriteFile(utilPath, utilModule, 0644)
	// Load module and check result.
	m, err := LoadModule(rootPath, GlobalCtx())
	if err != nil {
		t.Fatalf("failed to load module: %s", err)
	}
	r, ok := m.body.(*RecVal)
	if !ok {
		t.Fatalf("expected *RecVal, got %T", m.body)
	}
	got, ok := r.Fields["x"]
	if got != IntVal(1) {
		t.Errorf("want 1, got: %v", got)
	}
}

func TestLoadModuleNotFound(t *testing.T) {
	if testing.Short() {
		// Don't run tests writing to disk in -short mode.
		return
	}
	// Write module to disk.
	d := t.TempDir()
	rootPath := path.Join(d, "root.konfi")
	rootModule := []byte(`
	{
		m: load('doesnotexist')
	}
	`)
	os.WriteFile(rootPath, rootModule, 0644)
	// Load module and check result.
	m, gotErr := LoadModule(rootPath, GlobalCtx())
	if gotErr == nil {
		t.Fatalf("wanted error, got: %v", m)
	}
	want := "not found"
	if !strings.Contains(gotErr.Error(), want) {
		t.Errorf("wanted error containing '%s', got: %s", want, gotErr)
	}
}

func TestLoadModuleCycle(t *testing.T) {
	// Cycle detection for two modules trying to load each other.
	if testing.Short() {
		// Don't run tests writing to disk in -short mode.
		return
	}
	d := t.TempDir()
	// Write modules to disk.
	m1Path := path.Join(d, "m1.konfi")
	m1Module := []byte(`{ m: load('m2') }`)
	os.WriteFile(m1Path, m1Module, 0644)
	m2Path := path.Join(d, "m2.konfi")
	m2Module := []byte("{ m: load('m3') }")
	os.WriteFile(m2Path, m2Module, 0644)
	m3Path := path.Join(d, "m3.konfi")
	m3Module := []byte("{ m: load('m1') }")
	os.WriteFile(m3Path, m3Module, 0644)
	// Loading the module should fail.
	m, err := LoadModule(m1Path, GlobalCtx())
	if err == nil {
		t.Fatalf("expected error, got: %v", m)
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected 'cycle' error, got: %s", err)
	}
}
