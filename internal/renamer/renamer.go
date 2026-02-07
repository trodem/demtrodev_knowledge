package renamer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type PlanItem struct {
	OldPath string
	NewPath string
}

type Options struct {
	BasePath  string
	NamePart  string
	From      string
	To        string
	Recursive bool
}

func RunInteractive(baseDir string) int {
	reader := bufio.NewReader(os.Stdin)

	basePath := prompt(reader, "Base path", baseDir)
	if basePath == "" {
		basePath = baseDir
	}
	namePart := prompt(reader, "Name contains", "")
	from := prompt(reader, "Replace from", "")
	to := prompt(reader, "Replace to", "")

	if strings.TrimSpace(from) == "" {
		fmt.Println("Error: replace-from is required.")
		return 1
	}

	plan, err := BuildPlan(Options{
		BasePath:  basePath,
		NamePart:  namePart,
		From:      from,
		To:        to,
		Recursive: true,
	})
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}

	if len(plan) == 0 {
		fmt.Println("No files to rename.")
		return 0
	}

	fmt.Println("\nPreview:")
	for _, item := range plan {
		fmt.Printf("%s -> %s\n", item.OldPath, item.NewPath)
	}

	confirm := prompt(reader, "Proceed? [y/N]", "N")
	if strings.ToLower(strings.TrimSpace(confirm)) != "y" {
		fmt.Println("Canceled.")
		return 0
	}

	if err := ApplyPlan(plan); err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	fmt.Println("Done.")
	return 0
}

func BuildPlan(opts Options) ([]PlanItem, error) {
	base := opts.BasePath
	if base == "" {
		base = "."
	}
	namePart := strings.ToLower(strings.TrimSpace(opts.NamePart))
	from := opts.From
	to := opts.To

	var plan []PlanItem
	walk := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if !opts.Recursive && path != base {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if namePart != "" && !strings.Contains(strings.ToLower(name), namePart) {
			return nil
		}

		if !strings.Contains(name, from) {
			return nil
		}
		newName := strings.ReplaceAll(name, from, to)
		if newName == name {
			return nil
		}
		newPath := filepath.Join(filepath.Dir(path), newName)
		plan = append(plan, PlanItem{OldPath: path, NewPath: newPath})
		return nil
	}

	if err := filepath.WalkDir(base, walk); err != nil {
		return nil, err
	}

	return dedupe(plan), nil
}

func ApplyPlan(plan []PlanItem) error {
	seen := map[string]struct{}{}
	for _, item := range plan {
		if _, ok := seen[item.NewPath]; ok {
			return fmt.Errorf("duplicate target path: %s", item.NewPath)
		}
		seen[item.NewPath] = struct{}{}
		if _, err := os.Stat(item.NewPath); err == nil {
			return fmt.Errorf("target already exists: %s", item.NewPath)
		}
	}
	for _, item := range plan {
		if err := os.Rename(item.OldPath, item.NewPath); err != nil {
			return err
		}
	}
	return nil
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

func dedupe(items []PlanItem) []PlanItem {
	out := make([]PlanItem, 0, len(items))
	seen := map[string]struct{}{}
	for _, it := range items {
		key := it.OldPath + "->" + it.NewPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, it)
	}
	return out
}
