package tools

import (
	"errors"
	"testing"
	"time"

	"cli/internal/filesearch"
)

func withPagingTestClock(t *testing.T, at time.Time) func() {
	t.Helper()
	prev := nowFunc
	now := at
	nowFunc = func() time.Time { return now }
	return func() {
		nowFunc = prev
	}
}

func TestSearchPagingCacheHitSkipsLoader(t *testing.T) {
	resetPagingCachesForTest()
	restore := withPagingTestClock(t, time.Unix(1000, 0))
	defer restore()

	loaderCalls := 0
	_, err := getOrLoadSearchPageResults("k1", func() ([]filesearch.Result, error) {
		loaderCalls++
		return []filesearch.Result{{Path: "a"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = getOrLoadSearchPageResults("k1", func() ([]filesearch.Result, error) {
		loaderCalls++
		return []filesearch.Result{{Path: "b"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaderCalls != 1 {
		t.Fatalf("expected loader to run once, got %d", loaderCalls)
	}
}

func TestSearchPagingCacheTTLExpiryReloads(t *testing.T) {
	resetPagingCachesForTest()
	base := time.Unix(2000, 0)
	restore := withPagingTestClock(t, base)
	defer restore()

	loaderCalls := 0
	_, err := getOrLoadSearchPageResults("ttl", func() ([]filesearch.Result, error) {
		loaderCalls++
		return []filesearch.Result{{Path: "a"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	nowFunc = func() time.Time { return base.Add(pagingCacheTTL + time.Second) }
	_, err = getOrLoadSearchPageResults("ttl", func() ([]filesearch.Result, error) {
		loaderCalls++
		return []filesearch.Result{{Path: "b"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaderCalls != 2 {
		t.Fatalf("expected loader to run twice after TTL expiry, got %d", loaderCalls)
	}
}

func TestRecentPagingCacheEvictsOldest(t *testing.T) {
	resetPagingCachesForTest()
	base := time.Unix(3000, 0)
	restore := withPagingTestClock(t, base)
	defer restore()

	for i := 0; i < pagingCacheMaxEntries+1; i++ {
		idx := i
		now := base.Add(time.Duration(i) * time.Second)
		nowFunc = func() time.Time { return now }
		key := "k" + string(rune('a'+i))
		_, err := getOrLoadRecentPageResults(key, func() ([]recentItem, error) {
			return []recentItem{{Path: string(rune('a' + idx))}}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	pagingCacheMu.Lock()
	_, hasOldest := recentPageCache["ka"]
	pagingCacheMu.Unlock()
	if hasOldest {
		t.Fatal("expected oldest recent cache entry to be evicted")
	}
}

func TestSearchPagingCacheLoaderError(t *testing.T) {
	resetPagingCachesForTest()
	restore := withPagingTestClock(t, time.Unix(4000, 0))
	defer restore()

	wantErr := errors.New("boom")
	_, err := getOrLoadSearchPageResults("err", func() ([]filesearch.Result, error) {
		return nil, wantErr
	})
	if err == nil {
		t.Fatal("expected loader error")
	}
}
