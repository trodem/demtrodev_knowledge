package app

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"cli/internal/plugins"
	"cli/tools"
)

const catalogTokenBudget = 6000

func buildPluginCatalogScoped(baseDir, scope string) string {
	items, err := plugins.ListEntries(baseDir, true)
	if err != nil || len(items) == 0 {
		return "(none)"
	}

	scopeLower := strings.ToLower(strings.TrimSpace(scope))

	type catalogEntry struct {
		item plugins.Entry
		line string
	}

	groups := map[string][]catalogEntry{}
	groupOrder := []string{}

	for _, item := range items {
		key := toolkitGroupKey(item.Path)
		label := toolkitLabel(key)

		if scopeLower != "" && !scopeMatches(item.Name, label, scopeLower) {
			continue
		}

		info, _ := plugins.GetInfo(baseDir, item.Name)

		var paramsPart string
		if len(info.ParamDetails) > 0 {
			paramsPart = formatParamDetailsForCatalog(info.ParamDetails)
		} else if len(info.Parameters) > 0 {
			paramsPart = strings.Join(info.Parameters, ", ")
		}

		synopsis := strings.TrimSpace(info.Synopsis)
		synopsis = stripGroupWords(synopsis, label)

		var line string
		if paramsPart != "" {
			line = fmt.Sprintf("- %s(%s): %s", item.Name, paramsPart, synopsis)
		} else if synopsis != "" {
			line = fmt.Sprintf("- %s: %s", item.Name, synopsis)
		} else {
			line = fmt.Sprintf("- %s", item.Name)
		}

		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], catalogEntry{item: item, line: line})
	}

	sort.Strings(groupOrder)

	var out []string
	for _, key := range groupOrder {
		label := toolkitLabel(key)
		out = append(out, fmt.Sprintf("\n[%s]", label))
		for _, entry := range groups[key] {
			out = append(out, entry.line)
		}
	}
	catalog := strings.Join(out, "\n")

	tokens := estimateTokens(catalog)
	slog.Debug("plugin catalog built", "tokens", tokens, "functions", countCatalogFunctions(out), "scope", scope)
	if tokens > catalogTokenBudget {
		slog.Warn("plugin catalog exceeds token budget",
			"tokens", tokens, "budget", catalogTokenBudget,
			"hint", "use --scope to reduce catalog size")
	}

	return catalog
}

func buildPluginCatalog(baseDir string) string {
	return buildPluginCatalogScoped(baseDir, "")
}

func scopeMatches(funcName, groupLabel, scope string) bool {
	if strings.HasPrefix(strings.ToLower(funcName), scope+"_") {
		return true
	}
	return strings.Contains(strings.ToLower(groupLabel), scope)
}

func countCatalogFunctions(lines []string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "- ") {
			n++
		}
	}
	return n
}

func toolkitGroupKey(filePath string) string {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	base := filepath.Base(normalized)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func toolkitLabel(groupKey string) string {
	name := groupKey
	if len(name) >= 2 && name[0] >= '0' && name[0] <= '9' && name[1] == '_' {
		name = name[2:]
	}
	name = strings.TrimSuffix(name, "_Toolkit")
	return strings.ReplaceAll(name, "_", " ")
}

func stripGroupWords(synopsis, groupLabel string) string {
	if synopsis == "" || groupLabel == "" {
		return synopsis
	}
	words := strings.Fields(groupLabel)
	result := synopsis
	for _, w := range words {
		if len(w) < 3 {
			continue
		}
		re := "(?i)\\b" + regexp.QuoteMeta(w) + "\\b\\s*"
		result = regexp.MustCompile(re).ReplaceAllString(result, "")
	}
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "- ")
	result = strings.TrimSpace(result)
	if result == "" {
		return synopsis
	}
	return result
}

func formatParamDetailsForCatalog(details []plugins.ParamDetail) string {
	parts := make([]string, 0, len(details))
	for _, d := range details {
		s := d.Name
		if d.Mandatory {
			s += "*"
		}
		if d.Switch {
			s += "?"
		}
		if len(d.ValidateSet) > 0 {
			s += "=" + strings.Join(d.ValidateSet, "|")
		} else if d.Default != "" {
			s += "=" + d.Default
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ", ")
}

func buildToolsCatalog() string {
	return tools.BuildAgentCatalog()
}

func isKnownTool(name string) bool {
	return tools.IsKnownTool(name)
}
