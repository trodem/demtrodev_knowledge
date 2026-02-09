package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	fnRe       = regexp.MustCompile(`(?i)^\s*function\s+(g_[a-z0-9_]+)\b`)
	synopsisRe = regexp.MustCompile(`(?i)^\s*\.SYNOPSIS\s*$`)
	exampleRe  = regexp.MustCompile(`(?i)^\s*\.EXAMPLE\s*$`)
)

type item struct {
	Name     string
	Synopsis string
	Example  string
}

func main() {
	path := "plugins/functions/git.ps1"
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open git plugin:", err)
		os.Exit(1)
	}
	defer f.Close()

	lines := make([]string, 0, 1024)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan git plugin:", err)
		os.Exit(1)
	}

	items := parse(lines)
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	out := "docs/git-cheatsheet.md"
	if err := os.MkdirAll("docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir docs:", err)
		os.Exit(1)
	}
	fd, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create cheatsheet:", err)
		os.Exit(1)
	}
	defer fd.Close()

	fmt.Fprintln(fd, "# Git Functions Cheat Sheet")
	fmt.Fprintln(fd)
	fmt.Fprintln(fd, "Generated from `plugins/functions/git.ps1`.")
	fmt.Fprintln(fd)
	for _, it := range items {
		fmt.Fprintf(fd, "## %s\n", it.Name)
		if it.Synopsis != "" {
			fmt.Fprintln(fd, it.Synopsis)
		}
		fmt.Fprintln(fd)
		if it.Example != "" {
			fmt.Fprintln(fd, "```powershell")
			fmt.Fprintln(fd, it.Example)
			fmt.Fprintln(fd, "```")
		}
		fmt.Fprintln(fd)
	}

	fmt.Println("OK: generated", out)
}

func parse(lines []string) []item {
	out := make([]item, 0)
	for i := 0; i < len(lines); i++ {
		m := fnRe.FindStringSubmatch(lines[i])
		if len(m) != 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		synopsis, example := findHelpForFunction(lines, i)
		out = append(out, item{Name: name, Synopsis: synopsis, Example: example})
	}
	return out
}

func findHelpForFunction(lines []string, fnIndex int) (string, string) {
	end := fnIndex - 1
	for end >= 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if end < 0 || strings.TrimSpace(lines[end]) != "#>" {
		return "", ""
	}
	start := end - 1
	for start >= 0 && strings.TrimSpace(lines[start]) != "<#" {
		start--
	}
	if start < 0 {
		return "", ""
	}
	block := lines[start+1 : end]
	return parseBlock(block)
}

func parseBlock(block []string) (string, string) {
	mode := ""
	synopsis := ""
	example := ""
	for _, raw := range block {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case synopsisRe.MatchString(line):
			mode = "synopsis"
			continue
		case exampleRe.MatchString(line):
			mode = "example"
			continue
		case strings.HasPrefix(line, "."):
			mode = ""
			continue
		}

		switch mode {
		case "synopsis":
			if synopsis == "" {
				synopsis = line
			} else {
				synopsis += " " + line
			}
		case "example":
			if example == "" {
				example = line
			}
		}
	}
	return synopsis, example
}
