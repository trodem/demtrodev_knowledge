package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var fnLine = regexp.MustCompile(`(?i)^\s*function\s+([a-z0-9_-]+)\b`)

func main() {
	pluginsDir := "plugins"
	var problems []string
	err := filepath.WalkDir(pluginsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".ps1" && ext != ".psm1" && ext != ".txt" {
			return nil
		}
		lines, err := readLines(path)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", path, err))
			return nil
		}
		for i, line := range lines {
			if !fnLine.MatchString(line) {
				continue
			}
			j := i - 1
			for j >= 0 && strings.TrimSpace(lines[j]) == "" {
				j--
			}
			if j < 0 || strings.TrimSpace(lines[j]) != "#>" {
				problems = append(problems, fmt.Sprintf("%s:%d: missing comment-based help block before function", path, i+1))
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading plugins directory:", err)
		os.Exit(1)
	}

	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, p)
		}
		os.Exit(1)
	}

	fmt.Println("OK: plugin function help blocks are valid")
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
