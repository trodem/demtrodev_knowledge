package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkPluginDataset(b *testing.B, files, funcsPerFile int) string {
	b.Helper()
	base := b.TempDir()
	pluginsDir := filepath.Join(base, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < files; i++ {
		name := filepath.Join(pluginsDir, fmt.Sprintf("p%03d.ps1", i))
		content := ""
		for j := 0; j < funcsPerFile; j++ {
			content += fmt.Sprintf("function fn_%03d_%02d { }\n", i, j)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	return base
}

func clearPluginBenchCache() {
	cacheMu.Lock()
	entryListCache = map[string][]Entry{}
	entryInfoCache = map[string]Info{}
	cacheMu.Unlock()
}

func BenchmarkListEntriesWithFunctionsCold(b *testing.B) {
	base := benchmarkPluginDataset(b, 60, 6)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clearPluginBenchCache()
		items, err := ListEntries(base, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) == 0 {
			b.Fatal("expected entries")
		}
	}
}

func BenchmarkListEntriesWithFunctionsWarm(b *testing.B) {
	base := benchmarkPluginDataset(b, 60, 6)
	if _, err := ListEntries(base, true); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, err := ListEntries(base, true)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) == 0 {
			b.Fatal("expected entries")
		}
	}
}

func BenchmarkGetInfoFunctionCold(b *testing.B) {
	base := benchmarkPluginDataset(b, 30, 6)
	target := "fn_010_03"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clearPluginBenchCache()
		info, err := GetInfo(base, target)
		if err != nil {
			b.Fatal(err)
		}
		if info.Name != target {
			b.Fatalf("unexpected info name: %q", info.Name)
		}
	}
}

func BenchmarkGetInfoFunctionWarm(b *testing.B) {
	base := benchmarkPluginDataset(b, 30, 6)
	target := "fn_010_03"
	if _, err := GetInfo(base, target); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info, err := GetInfo(base, target)
		if err != nil {
			b.Fatal(err)
		}
		if info.Name != target {
			b.Fatalf("unexpected info name: %q", info.Name)
		}
	}
}
