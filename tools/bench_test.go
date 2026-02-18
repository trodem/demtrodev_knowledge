package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cli/internal/filesearch"
)

func benchmarkDataset(b *testing.B, files int) string {
	b.Helper()
	base := b.TempDir()
	for i := 0; i < files; i++ {
		sub := filepath.Join(base, fmt.Sprintf("dir-%03d", i%40))
		if err := os.MkdirAll(sub, 0o755); err != nil {
			b.Fatal(err)
		}
		ext := ".txt"
		if i%5 == 0 {
			ext = ".md"
		}
		name := fmt.Sprintf("note-%05d%s", i, ext)
		p := filepath.Join(sub, name)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			b.Fatal(err)
		}
	}
	return base
}

func BenchmarkSearchFind(b *testing.B) {
	base := benchmarkDataset(b, 4000)
	opts := filesearch.Options{
		BasePath: base,
		NamePart: "note",
		Ext:      ".md",
		SortBy:   "name",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := filesearch.Find(opts)
		if err != nil {
			b.Fatal(err)
		}
		if len(results) == 0 {
			b.Fatal("expected non-empty results")
		}
	}
}

func BenchmarkSearchPagingCacheHit(b *testing.B) {
	base := benchmarkDataset(b, 4000)
	key := "bench-search"
	results, err := filesearch.Find(filesearch.Options{
		BasePath: base,
		NamePart: "note",
		Ext:      ".md",
		SortBy:   "name",
	})
	if err != nil {
		b.Fatal(err)
	}
	pagingCacheMu.Lock()
	searchPageCache[key] = searchPageCacheEntry{
		Results: results,
		Stored:  time.Now(),
		LastUse: time.Now(),
	}
	pagingCacheMu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := getOrLoadSearchPageResults(key, func() ([]filesearch.Result, error) {
			return filesearch.Find(filesearch.Options{
				BasePath: base,
				NamePart: "note",
				Ext:      ".md",
				SortBy:   "name",
			})
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(got) == 0 {
			b.Fatal("expected cached results")
		}
	}
}

func BenchmarkRecentCollectSorted(b *testing.B) {
	base := benchmarkDataset(b, 4000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items, err := collectRecentSorted(base)
		if err != nil {
			b.Fatal(err)
		}
		if len(items) == 0 {
			b.Fatal("expected non-empty items")
		}
	}
}

func BenchmarkRecentPagingCacheHit(b *testing.B) {
	base := benchmarkDataset(b, 4000)
	key := "bench-recent"
	items, err := collectRecentSorted(base)
	if err != nil {
		b.Fatal(err)
	}
	pagingCacheMu.Lock()
	recentPageCache[key] = recentPageCacheEntry{
		Results: items,
		Stored:  time.Now(),
		LastUse: time.Now(),
	}
	pagingCacheMu.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got, err := getOrLoadRecentPageResults(key, func() ([]recentItem, error) {
			return collectRecentSorted(base)
		})
		if err != nil {
			b.Fatal(err)
		}
		if len(got) == 0 {
			b.Fatal("expected cached items")
		}
	}
}
