package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\nTools:")
		fmt.Println("  1) Search files")
		fmt.Println("  2) Rename files")
		fmt.Println("  3) Quick note")
		fmt.Println("  4) Recent files")
		fmt.Println("  5) Pack backup")
		fmt.Println("  6) Clean empty folders")
		fmt.Println("  0) Exit")
		fmt.Print("\nSelect option: ")

		choice := readLine(reader)
		switch choice {
		case "0", "exit", "Exit", "":
			return 0
		default:
			_ = RunByNameWithReader(baseDir, choice, reader)
		}
	}
}

func RunByName(baseDir, name string) int {
	return RunByNameWithReader(baseDir, name, bufio.NewReader(os.Stdin))
}

func RunByNameWithReader(baseDir, name string, reader *bufio.Reader) int {
	switch normalizeToolName(name) {
	case "search":
		return RunSearch(reader)
	case "rename":
		return RunRename(baseDir, reader)
	case "note":
		return RunQuickNote(baseDir, reader)
	case "recent":
		return RunRecent(reader)
	case "backup":
		return RunPackBackup(baseDir, reader)
	case "clean":
		return RunCleanEmpty(reader)
	default:
		fmt.Println("Invalid tool:", name)
		fmt.Println("Use: search|rename|note|recent|backup|clean")
		return 1
	}
}

func normalizeToolName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "1", "search", "s":
		return "search"
	case "2", "rename", "r":
		return "rename"
	case "3", "note", "n":
		return "note"
	case "4", "recent", "rec":
		return "recent"
	case "5", "backup", "b":
		return "backup"
	case "6", "clean", "c":
		return "clean"
	default:
		return ""
	}
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := r.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func readLine(r *bufio.Reader) string {
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s)
}

func currentWorkingDir(fallback string) string {
	wd, err := os.Getwd()
	if err != nil || strings.TrimSpace(wd) == "" {
		return fallback
	}
	return wd
}
