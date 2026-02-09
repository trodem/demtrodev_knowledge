package tools

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"cli/internal/filesearch"
	"cli/internal/platform"
	"cli/internal/ui"
)

func RunSearch(r *bufio.Reader) int {
	base := prompt(r, "Base path", currentWorkingDir("."))
	base = normalizeInputPath(base, currentWorkingDir("."))
	if strings.TrimSpace(base) == "" {
		fmt.Println("Error: base path is required.")
		return 1
	}
	if err := validateExistingDir(base, "base path"); err != nil {
		fmt.Println(ui.Error("Error:"), err)
		fmt.Println(ui.Muted("Hint: use '.' for current dir or '..' for parent dir."))
		return 1
	}
	name := prompt(r, "Name contains", "")
	ext := prompt(r, "Extension (optional)", "")
	sortBy := prompt(r, "Sort (name|date|size)", "name")

	results, err := filesearch.Find(filesearch.Options{
		BasePath: base,
		NamePart: name,
		Ext:      ext,
		SortBy:   sortBy,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	if len(results) == 0 {
		fmt.Println("No files found.")
		return 0
	}

	for i, item := range results {
		idx := ui.Warn(fmt.Sprintf("%2d)", i+1))
		fmt.Printf("%s %s | %s | %s\n", idx, item.ModTime.Format("2006-01-02 15:04"), filesearch.FormatSize(item.Size), item.Path)
	}

	selection := prompt(r, "Select result to open (number, Enter to skip)", "")
	if strings.TrimSpace(selection) == "" {
		return 0
	}
	idx, ok := parseSelectionIndex(selection, len(results))
	if !ok {
		fmt.Println(ui.Error("Invalid selection."))
		return 1
	}
	platform.OpenFile(results[idx].Path)
	return 0
}

func parseSelectionIndex(raw string, max int) (int, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return -1, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return -1, false
	}
	if n < 1 || n > max {
		return -1, false
	}
	return n - 1, true
}
