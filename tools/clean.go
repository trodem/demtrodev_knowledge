package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cli/internal/ui"
)

func RunCleanEmpty(r *bufio.Reader) int {
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

	dirs, code := showEmptyDirs(base)
	if code != 0 {
		return code
	}
	if len(dirs) == 0 {
		return 0
	}

	confirm := prompt(r, "Delete these folders? [y/N]", "N")
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println(ui.Warn("Canceled."))
		return 0
	}

	return removeEmptyDirs(dirs)
}

func RunCleanEmptyAuto(baseDir string, params map[string]string) int {
	base := strings.TrimSpace(params["base"])
	if base == "" {
		base = currentWorkingDir(baseDir)
	}
	base = normalizeAgentPath(base, baseDir)
	dirs, code := showEmptyDirs(base)
	if code != 0 {
		return code
	}
	if len(dirs) == 0 {
		return 0
	}
	apply := strings.ToLower(strings.TrimSpace(params["apply"]))
	if apply != "1" && apply != "true" && apply != "yes" && apply != "y" {
		fmt.Println(ui.Muted("Preview only. Set tool_args.apply=true to delete."))
		return 0
	}
	return removeEmptyDirs(dirs)
}

func showEmptyDirs(base string) ([]string, int) {
	dirs, err := findEmptyDirs(base)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, 1
	}
	if len(dirs) == 0 {
		fmt.Println("No empty folders found.")
		return nil, 0
	}
	fmt.Println("\nEmpty folders:")
	for _, d := range dirs {
		fmt.Println(d)
	}
	return dirs, 0
}

func removeEmptyDirs(dirs []string) int {
	for _, d := range dirs {
		_ = os.Remove(d)
	}
	fmt.Println("Done.")
	return 0
}

func findEmptyDirs(base string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if path == base {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		if len(entries) == 0 {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// remove deepest first
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})
	return dirs, nil
}
