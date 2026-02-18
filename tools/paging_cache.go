package tools

import (
	"sync"
	"time"

	"cli/internal/filesearch"
)

const (
	pagingCacheTTL        = 15 * time.Second
	pagingCacheMaxEntries = 8
)

type searchPageCacheEntry struct {
	Results []filesearch.Result
	Stored  time.Time
	LastUse time.Time
}

type recentPageCacheEntry struct {
	Results []recentItem
	Stored  time.Time
	LastUse time.Time
}

var (
	pagingCacheMu   sync.Mutex
	searchPageCache = map[string]searchPageCacheEntry{}
	recentPageCache = map[string]recentPageCacheEntry{}
	nowFunc         = time.Now
)

func getOrLoadSearchPageResults(key string, loader func() ([]filesearch.Result, error)) ([]filesearch.Result, error) {
	now := nowFunc()
	pagingCacheMu.Lock()
	if entry, ok := searchPageCache[key]; ok && now.Sub(entry.Stored) <= pagingCacheTTL {
		entry.LastUse = now
		searchPageCache[key] = entry
		out := make([]filesearch.Result, len(entry.Results))
		copy(out, entry.Results)
		pagingCacheMu.Unlock()
		return out, nil
	}
	pagingCacheMu.Unlock()

	results, err := loader()
	if err != nil {
		return nil, err
	}
	out := make([]filesearch.Result, len(results))
	copy(out, results)

	pagingCacheMu.Lock()
	searchPageCache[key] = searchPageCacheEntry{
		Results: out,
		Stored:  now,
		LastUse: now,
	}
	pruneSearchPageCache()
	pagingCacheMu.Unlock()
	return out, nil
}

func getOrLoadRecentPageResults(key string, loader func() ([]recentItem, error)) ([]recentItem, error) {
	now := nowFunc()
	pagingCacheMu.Lock()
	if entry, ok := recentPageCache[key]; ok && now.Sub(entry.Stored) <= pagingCacheTTL {
		entry.LastUse = now
		recentPageCache[key] = entry
		out := make([]recentItem, len(entry.Results))
		copy(out, entry.Results)
		pagingCacheMu.Unlock()
		return out, nil
	}
	pagingCacheMu.Unlock()

	results, err := loader()
	if err != nil {
		return nil, err
	}
	out := make([]recentItem, len(results))
	copy(out, results)

	pagingCacheMu.Lock()
	recentPageCache[key] = recentPageCacheEntry{
		Results: out,
		Stored:  now,
		LastUse: now,
	}
	pruneRecentPageCache()
	pagingCacheMu.Unlock()
	return out, nil
}

func resetPagingCachesForTest() {
	pagingCacheMu.Lock()
	searchPageCache = map[string]searchPageCacheEntry{}
	recentPageCache = map[string]recentPageCacheEntry{}
	pagingCacheMu.Unlock()
}

func pruneSearchPageCache() {
	if len(searchPageCache) <= pagingCacheMaxEntries {
		return
	}
	var (
		oldestKey string
		oldestUse time.Time
		first     = true
	)
	for k, v := range searchPageCache {
		if first || v.LastUse.Before(oldestUse) {
			oldestKey = k
			oldestUse = v.LastUse
			first = false
		}
	}
	if oldestKey != "" {
		delete(searchPageCache, oldestKey)
	}
}

func pruneRecentPageCache() {
	if len(recentPageCache) <= pagingCacheMaxEntries {
		return
	}
	var (
		oldestKey string
		oldestUse time.Time
		first     = true
	)
	for k, v := range recentPageCache {
		if first || v.LastUse.Before(oldestUse) {
			oldestKey = k
			oldestUse = v.LastUse
			first = false
		}
	}
	if oldestKey != "" {
		delete(recentPageCache, oldestKey)
	}
}
