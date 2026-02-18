package plugins

import "sync"

var (
	cacheMu        sync.RWMutex
	entryListCache = map[string][]Entry{}
	entryInfoCache = map[string]Info{}
)

func listEntriesCacheKey(dir string, includeFunctions bool) string {
	if includeFunctions {
		return dir + "|with-functions"
	}
	return dir + "|scripts-only"
}

func getCachedEntryList(key string) ([]Entry, bool) {
	cacheMu.RLock()
	items, ok := entryListCache[key]
	cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	out := make([]Entry, len(items))
	copy(out, items)
	return out, true
}

func setCachedEntryList(key string, items []Entry) {
	out := make([]Entry, len(items))
	copy(out, items)
	cacheMu.Lock()
	entryListCache[key] = out
	cacheMu.Unlock()
}

func infoCacheKey(dir, name string) string {
	return dir + "|" + name
}

func getCachedInfo(key string) (Info, bool) {
	cacheMu.RLock()
	info, ok := entryInfoCache[key]
	cacheMu.RUnlock()
	if !ok {
		return Info{}, false
	}
	return cloneInfo(info), true
}

func setCachedInfo(key string, info Info) {
	cacheMu.Lock()
	entryInfoCache[key] = cloneInfo(info)
	cacheMu.Unlock()
}

func cloneInfo(info Info) Info {
	out := info
	out.Sources = append([]string(nil), info.Sources...)
	out.Parameters = append([]string(nil), info.Parameters...)
	out.Examples = append([]string(nil), info.Examples...)
	return out
}
