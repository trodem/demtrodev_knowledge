package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"cli/internal/ui"
)

func RunMenu(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	for {
		ui.PrintSection("Tools")
		ui.PrintMenuLine("1", "[s] Search files", false)
		ui.PrintMenuLine("2", "[r] Rename files", false)
		ui.PrintMenuLine("3", "[n] Quick note", false)
		ui.PrintMenuLine("4", "[e] Recent files", false)
		ui.PrintMenuLine("5", "[b] Pack backup", false)
		ui.PrintMenuLine("6", "[c] Clean empty folders", false)
		ui.PrintMenuLine("7", "[y] System snapshot", false)
		ui.PrintMenuLine("0", "[x] Exit", true)
		fmt.Print("\nSelect option > ")

		choice := readLine(reader)
		switch choice {
		case "0", "x", "X", "exit", "Exit", "":
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
	case "system":
		return RunSystem(reader)
	default:
		fmt.Println(ui.Error("Invalid tool:"), name)
		fmt.Println(ui.Muted("Use: search|rename|note|recent|backup|clean|system"))
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
	case "e":
		return "recent"
	case "5", "backup", "b":
		return "backup"
	case "6", "clean", "c":
		return "clean"
	case "7", "system", "sys", "htop":
		return "system"
	case "y":
		return "system"
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
