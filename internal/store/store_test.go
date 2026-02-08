package store

import (
	"path/filepath"
	"testing"
)

func TestCreatePackSetsDefaultMetadata(t *testing.T) {
	baseDir := t.TempDir()
	name := "work"

	if err := CreatePack(baseDir, name); err != nil {
		t.Fatal(err)
	}

	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	pf, err := LoadPackFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	if pf.Description != "Pack "+name {
		t.Fatalf("expected default description, got %q", pf.Description)
	}
	if pf.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", pf.SchemaVersion)
	}
	if pf.Summary == "" {
		t.Fatalf("expected default summary")
	}
	if len(pf.Examples) == 0 {
		t.Fatalf("expected default examples")
	}

	info, err := GetPackInfo(baseDir, name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Description != "Pack "+name {
		t.Fatalf("expected info description, got %q", info.Description)
	}
	if info.Name != name {
		t.Fatalf("expected info name %q, got %q", name, info.Name)
	}
	if info.Summary == "" {
		t.Fatalf("expected info summary")
	}
	if len(info.Examples) == 0 {
		t.Fatalf("expected info examples")
	}
}

func TestClonePack(t *testing.T) {
	baseDir := t.TempDir()
	if err := CreatePack(baseDir, "src"); err != nil {
		t.Fatal(err)
	}

	srcPath := filepath.Join(baseDir, "packs", "src", "pack.json")
	pf, err := LoadPackFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	pf.Description = "Pack src"
	pf.Summary = "Commands and knowledge for src"
	pf.Search.Knowledge = filepath.Join("packs", "src", "knowledge")
	if err := SavePackFile(srcPath, pf); err != nil {
		t.Fatal(err)
	}

	if err := ClonePack(baseDir, "src", "dst"); err != nil {
		t.Fatal(err)
	}
	dstPath := filepath.Join(baseDir, "packs", "dst", "pack.json")
	cloned, err := LoadPackFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if cloned.SchemaVersion != 1 {
		t.Fatalf("expected schema version 1, got %d", cloned.SchemaVersion)
	}
	if cloned.Description != "Pack dst" {
		t.Fatalf("expected cloned description to adapt, got %q", cloned.Description)
	}
	if cloned.Summary != "Commands and knowledge for dst" {
		t.Fatalf("expected cloned summary to adapt, got %q", cloned.Summary)
	}
	if filepath.Clean(filepath.FromSlash(cloned.Search.Knowledge)) != filepath.Clean(filepath.Join("packs", "dst", "knowledge")) {
		t.Fatalf("expected cloned knowledge path to adapt, got %q", cloned.Search.Knowledge)
	}
}
