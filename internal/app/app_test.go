package app

import (
	"reflect"
	"testing"
)

func TestParseFlagsToolsShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-t"})
	want := []string{"tools"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithTarget(t *testing.T) {
	_, out := parseFlags([]string{"-t", "search"})
	want := []string{"tools", "search"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsToolsShortcutWithAliasAndPack(t *testing.T) {
	f, out := parseFlags([]string{"-p", "git", "-t", "s"})
	want := []string{"tools", "s"}
	if f.Pack != "git" {
		t.Fatalf("expected pack git, got %q", f.Pack)
	}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsPacksShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-k", "list"})
	want := []string{"pack", "list"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}

func TestParseFlagsPluginsShortcut(t *testing.T) {
	_, out := parseFlags([]string{"-g", "list"})
	want := []string{"plugin", "list"}
	if !reflect.DeepEqual(out, want) {
		t.Fatalf("expected %v, got %v", want, out)
	}
}
