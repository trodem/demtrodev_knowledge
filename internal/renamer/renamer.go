package renamer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type PlanItem struct {
	OldPath string
	NewPath string
}

type Options struct {
	BasePath      string
	NamePart      string
	From          string
	To            string
	Recursive     bool
	UseRegex      bool
	CaseSensitive bool
}

func BuildPlan(opts Options) ([]PlanItem, error) {
	base := opts.BasePath
	if base == "" {
		base = "."
	}
	namePart := strings.TrimSpace(opts.NamePart)
	from := opts.From
	to := opts.To

	var nameRe *regexp.Regexp
	var fromRe *regexp.Regexp
	var err error
	if opts.UseRegex {
		namePattern := namePart
		fromPattern := from
		if namePattern != "" {
			namePattern = "(?i)" + namePattern
		}
		if !opts.CaseSensitive {
			fromPattern = "(?i)" + fromPattern
		}
		if namePattern != "" {
			nameRe, err = regexp.Compile(namePattern)
			if err != nil {
				return nil, fmt.Errorf("invalid name regex: %w", err)
			}
		}
		fromRe, err = regexp.Compile(fromPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid replace regex: %w", err)
		}
	}

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
		if opts.UseRegex {
			if nameRe != nil && !nameRe.MatchString(name) {
				return nil
			}
			if !fromRe.MatchString(name) {
				return nil
			}
			newName := fromRe.ReplaceAllString(name, to)
			if newName == name {
				return nil
			}
			newPath := filepath.Join(filepath.Dir(path), newName)
			plan = append(plan, PlanItem{OldPath: path, NewPath: newPath})
			return nil
		}

		if namePart != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(namePart)) {
			return nil
		}
		if opts.CaseSensitive {
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
		newName := replaceInsensitive(name, from, to)
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

func replaceInsensitive(s, from, to string) string {
	if from == "" {
		return s
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(from))
	if err != nil {
		return s
	}
	return re.ReplaceAllString(s, to)
}
