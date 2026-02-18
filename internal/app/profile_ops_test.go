package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParsePowerShellSymbols(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profile.ps1")
	content := `
# comment line
function z_last { }
function a_first { }
Set-Alias ll Get-ChildItem
New-Alias gs git-status
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	funcs, aliases, err := parsePowerShellSymbols(path)
	if err != nil {
		t.Fatal(err)
	}

	wantFuncs := []string{"a_first", "z_last"}
	if !reflect.DeepEqual(funcs, wantFuncs) {
		t.Fatalf("unexpected functions: got %v want %v", funcs, wantFuncs)
	}

	wantAliases := []string{"gs -> git-status", "ll -> Get-ChildItem"}
	if !reflect.DeepEqual(aliases, wantAliases) {
		t.Fatalf("unexpected aliases: got %v want %v", aliases, wantAliases)
	}
}
