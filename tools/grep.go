package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"cli/internal/ui"
)

const (
	grepDefaultLimit   = 20
	grepMaxLimit       = 50
	grepMaxFileBytes   = 1024 * 1024 // 1 MB
	grepContextLines   = 0
	grepMaxLineLen     = 500
)

var grepSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"bin":          true,
	"obj":          true,
	"vendor":       true,
	"__pycache__":  true,
	".vs":          true,
	".idea":        true,
}

type grepMatch struct {
	File    string
	LineNum int
	Line    string
}

func RunGrep(r *bufio.Reader) int {
	pattern := prompt(r, "Search pattern", "")
	if strings.TrimSpace(pattern) == "" {
		fmt.Println(ui.Error("Error:"), "search pattern is required.")
		return 1
	}
	base := prompt(r, "Base path", currentWorkingDir("."))
	base = normalizeInputPath(base, currentWorkingDir("."))
	ext := prompt(r, "Extension filter (optional, e.g. go, ps1)", "")
	caseSensitive := strings.ToLower(prompt(r, "Case sensitive (y/N)", "n"))

	matches := grepFiles(base, pattern, ext, caseSensitive == "y" || caseSensitive == "yes", grepDefaultLimit)
	printGrepResults(matches, pattern)
	return 0
}

func RunGrepAuto(baseDir string, params map[string]string) int {
	return RunGrepAutoDetailed(baseDir, params).Code
}

func RunGrepAutoDetailed(baseDir string, params map[string]string) AutoRunResult {
	pattern := strings.TrimSpace(params["pattern"])
	if pattern == "" {
		fmt.Println("Error: pattern is required.")
		return AutoRunResult{Code: 1}
	}

	base := strings.TrimSpace(params["base"])
	if base == "" {
		base = currentWorkingDir(baseDir)
	}
	base = normalizeAgentPath(base, baseDir)

	ext := strings.TrimSpace(params["ext"])
	caseSensitive := false
	if v := strings.ToLower(strings.TrimSpace(params["case_sensitive"])); v == "true" || v == "yes" || v == "1" {
		caseSensitive = true
	}

	limit := grepDefaultLimit
	if v := strings.TrimSpace(params["limit"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			limit = n
		}
	}
	if limit > grepMaxLimit {
		limit = grepMaxLimit
	}

	matches := grepFiles(base, pattern, ext, caseSensitive, limit)
	printGrepResults(matches, pattern)
	return AutoRunResult{Code: 0}
}

func grepFiles(base, pattern, ext string, caseSensitive bool, limit int) []grepMatch {
	searchPattern := pattern
	if !caseSensitive {
		searchPattern = strings.ToLower(pattern)
	}

	extFilter := strings.TrimSpace(ext)
	if extFilter != "" && !strings.HasPrefix(extFilter, ".") {
		extFilter = "." + extFilter
	}
	if !caseSensitive {
		extFilter = strings.ToLower(extFilter)
	}

	var matches []grepMatch

	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if grepSkipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Size() > grepMaxFileBytes || info.Size() == 0 {
			return nil
		}
		if extFilter != "" {
			fileExt := filepath.Ext(info.Name())
			if !caseSensitive {
				fileExt = strings.ToLower(fileExt)
			}
			if fileExt != extFilter {
				return nil
			}
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !utf8.Valid(data) {
			return nil
		}

		relPath, _ := filepath.Rel(base, path)
		if relPath == "" {
			relPath = path
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			compareLine := line
			if !caseSensitive {
				compareLine = strings.ToLower(line)
			}
			if strings.Contains(compareLine, searchPattern) {
				trimmed := strings.TrimRight(line, "\r\n")
				if len(trimmed) > grepMaxLineLen {
					trimmed = trimmed[:grepMaxLineLen] + "..."
				}
				matches = append(matches, grepMatch{
					File:    relPath,
					LineNum: i + 1,
					Line:    trimmed,
				})
				if len(matches) >= limit {
					return fmt.Errorf("limit reached")
				}
			}
		}
		return nil
	})

	return matches
}

func printGrepResults(matches []grepMatch, pattern string) {
	if len(matches) == 0 {
		fmt.Printf("No matches found for '%s'.\n", pattern)
		return
	}

	fileGroups := make(map[string][]grepMatch)
	fileOrder := make([]string, 0)
	for _, m := range matches {
		if _, exists := fileGroups[m.File]; !exists {
			fileOrder = append(fileOrder, m.File)
		}
		fileGroups[m.File] = append(fileGroups[m.File], m)
	}

	fmt.Printf("Found %d matches in %d files\n\n", len(matches), len(fileOrder))
	for _, file := range fileOrder {
		fmt.Println(ui.Accent(file))
		for _, m := range fileGroups[file] {
			fmt.Printf("  %4d | %s\n", m.LineNum, m.Line)
		}
		fmt.Println()
	}

	if len(matches) >= grepMaxLimit {
		fmt.Println(ui.Muted("(results truncated, refine your search)"))
	}
}
