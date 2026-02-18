package plugins

import (
	"os"
	"sync"
)

var (
	cacheMu        sync.RWMutex
	entryListCache = map[string]entryListCacheValue{}
	entryInfoCache = map[string]entryInfoCacheValue{}
)

type entryListCacheValue struct {
	DirPath    string
	Items      []Entry
	DirStamp   int64
	FileStamps map[string]int64
}

type entryInfoCacheValue struct {
	DirPath    string
	Info       Info
	DirStamp   int64
	FileStamps map[string]int64
}

func listEntriesCacheKey(dir string, includeFunctions bool) string {
	if includeFunctions {
		return dir + "|with-functions"
	}
	return dir + "|scripts-only"
}

func getCachedEntryList(key string) ([]Entry, bool) {
	cacheMu.RLock()
	value, ok := entryListCache[key]
	cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if !fingerprintsValid(value.DirPath, value.DirStamp, value.FileStamps) {
		return nil, false
	}
	out := make([]Entry, len(value.Items))
	copy(out, value.Items)
	return out, true
}

func setCachedEntryList(key string, dirPath string, items []Entry, dirStamp int64, fileStamps map[string]int64) {
	out := make([]Entry, len(items))
	copy(out, items)
	fsCopy := cloneFileStamps(fileStamps)
	cacheMu.Lock()
	entryListCache[key] = entryListCacheValue{
		DirPath:    dirPath,
		Items:      out,
		DirStamp:   dirStamp,
		FileStamps: fsCopy,
	}
	cacheMu.Unlock()
}

func infoCacheKey(dir, name string) string {
	return dir + "|" + name
}

func getCachedInfo(key string) (Info, bool) {
	cacheMu.RLock()
	value, ok := entryInfoCache[key]
	cacheMu.RUnlock()
	if !ok {
		return Info{}, false
	}
	if !fingerprintsValid(value.DirPath, value.DirStamp, value.FileStamps) {
		return Info{}, false
	}
	return cloneInfo(value.Info), true
}

func setCachedInfo(key string, dirPath string, info Info, dirStamp int64, fileStamps map[string]int64) {
	fsCopy := cloneFileStamps(fileStamps)
	cacheMu.Lock()
	entryInfoCache[key] = entryInfoCacheValue{
		DirPath:    dirPath,
		Info:       cloneInfo(info),
		DirStamp:   dirStamp,
		FileStamps: fsCopy,
	}
	cacheMu.Unlock()
}

func cloneInfo(info Info) Info {
	out := info
	out.Sources = append([]string(nil), info.Sources...)
	out.Parameters = append([]string(nil), info.Parameters...)
	out.Examples = append([]string(nil), info.Examples...)
	return out
}

func cloneFileStamps(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func fingerprintsValid(dirPath string, dirStamp int64, fileStamps map[string]int64) bool {
	for path, want := range fileStamps {
		got := statStamp(path)
		if got != want {
			return false
		}
	}
	if dirStamp >= 0 {
		return statStamp(dirPath) == dirStamp
	}
	return true
}

func statStamp(path string) int64 {
	if path == "" {
		return -1
	}
	info, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return info.ModTime().UnixNano()
}
