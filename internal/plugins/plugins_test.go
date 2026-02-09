package plugins

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRunNotFound(t *testing.T) {
	baseDir := t.TempDir()
	err := Run(baseDir, "missing_plugin", nil)
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListDeduplicatesByPluginName(t *testing.T) {
	baseDir := t.TempDir()
	pluginsDir := filepath.Join(baseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ps1 := filepath.Join(pluginsDir, "hello.ps1")
	sh := filepath.Join(pluginsDir, "hello.sh")
	if err := os.WriteFile(ps1, []byte("Write-Host hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sh, []byte("echo hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	items, err := List(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one plugin entry, got %d", len(items))
	}
	if items[0].Name != "hello" {
		t.Fatalf("expected plugin name hello, got %q", items[0].Name)
	}
}

func TestReadPowerShellFunctionNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), profileFunctionsFile)
	content := "function stibs_restart_backend { }\nfunction test_one { }\n# function ignored\nfunction stibs_restart_backend { }\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readPowerShellFunctionNames(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"stibs_restart_backend", "test_one"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
