package app

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
)

func pluginArgsToPS(pluginArgs map[string]string) []string {
	if len(pluginArgs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(pluginArgs))
	for k := range pluginArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var args []string
	for _, k := range keys {
		v := strings.TrimSpace(pluginArgs[k])
		paramName := k
		if !strings.HasPrefix(paramName, "-") {
			paramName = "-" + paramName
		}
		lv := strings.ToLower(v)
		if lv == "true" || lv == "" {
			args = append(args, paramName)
			continue
		}
		if lv == "false" {
			continue
		}
		args = append(args, paramName, v)
	}
	return args
}

func formatPluginArgs(pluginArgs map[string]string) string {
	if len(pluginArgs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(pluginArgs))
	for k := range pluginArgs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("-%s %s", k, pluginArgs[k]))
	}
	return strings.Join(parts, " ")
}

func formatToolArgs(args map[string]string) string {
	if len(args) == 0 {
		return ""
	}
	keys := make([]string, 0, len(args))
	for k := range args {
		v := strings.TrimSpace(args[k])
		lc := strings.ToLower(v)
		if v == "" || lc == "<nil>" || lc == "null" {
			continue
		}
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, args[k]))
	}
	return strings.Join(parts, ", ")
}

func plannedActionSummary(decision agent.DecisionResult) string {
	switch strings.ToLower(strings.TrimSpace(decision.Action)) {
	case "run_plugin":
		s := "plugin " + strings.TrimSpace(decision.Plugin)
		if a := formatPluginArgs(decision.PluginArgs); a != "" {
			s += " " + a
		} else if len(decision.Args) > 0 {
			s += " " + strings.Join(decision.Args, " ")
		}
		return s
	case "run_tool":
		s := "tool " + strings.TrimSpace(decision.Tool)
		if args := formatToolArgs(decision.ToolArgs); strings.TrimSpace(args) != "" {
			s += " (" + args + ")"
		}
		return s
	case "create_function":
		desc := strings.TrimSpace(decision.FunctionDescription)
		if len(desc) > askDescMaxLen {
			desc = desc[:askDescMaxLen] + "..."
		}
		return "create function: " + desc
	default:
		if strings.TrimSpace(decision.Answer) != "" {
			return "answer"
		}
		return "noop"
	}
}

func missingMandatoryParams(info plugins.Info, pluginArgs map[string]string) []string {
	var missing []string
	for _, p := range info.ParamDetails {
		if !p.Mandatory {
			continue
		}
		found := false
		for k, v := range pluginArgs {
			if strings.EqualFold(k, p.Name) && strings.TrimSpace(v) != "" {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, p.Name)
		}
	}
	return missing
}

var missingPathErr = regexp.MustCompile(`(?i)required path '([^']+)' does not exist`)
var psMandatoryParam = regexp.MustCompile(`(?i)missing mandatory parameters?:\s*(.+)`)
var psParamNotFound = regexp.MustCompile(`(?i)cannot be found that matches parameter name '([^']+)'`)
var psGeneralError = regexp.MustCompile(`(?m)^\s*\|\s+(.+)$`)

const promptTokenBudget = 20000

func estimateTokens(s string) int {
	return len(s) / 4
}

func trimToTokenBudget(prompt, sessionBlock, previousBlock string, budget int) (string, string) {
	total := estimateTokens(prompt) + estimateTokens(sessionBlock) + estimateTokens(previousBlock)
	if total <= budget {
		return sessionBlock, previousBlock
	}
	if estimateTokens(prompt)+estimateTokens(previousBlock) > budget {
		sessionBlock = ""
		for estimateTokens(prompt)+estimateTokens(previousBlock) > budget && len(previousBlock) > 0 {
			idx := strings.Index(previousBlock, "\n")
			if idx < 0 {
				previousBlock = ""
				break
			}
			previousBlock = previousBlock[idx+1:]
		}
	} else {
		excess := total - budget
		charsToDrop := excess * 4
		if charsToDrop >= len(sessionBlock) {
			sessionBlock = ""
		} else {
			sessionBlock = sessionBlock[charsToDrop:]
			if idx := strings.Index(sessionBlock, "\n"); idx >= 0 {
				sessionBlock = sessionBlock[idx+1:]
			}
		}
	}
	return sessionBlock, previousBlock
}

func truncateForHistory(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

func printAgentActionError(err error) {
	raw := plugins.ErrorOutput(err)
	combined := strings.TrimSpace(err.Error() + "\n" + raw)

	slog.Debug("plugin error trace", "raw", combined)

	friendly := extractFriendlyError(combined)
	if friendly != "" {
		fmt.Fprintln(os.Stderr, "  "+ui.Error("Error:")+" "+friendly)
	} else {
		fmt.Fprintln(os.Stderr, "  "+ui.Error("Error:")+" plugin execution failed")
	}

	if m := missingPathErr.FindStringSubmatch(combined); len(m) == 2 {
		fmt.Println("  " + ui.Warn("Missing path: "+m[1]))
		fmt.Println("  " + ui.Muted("Check plugin config, then retry."))
	}
}

const fileContextMaxBytes = 32 * 1024

func buildFileContext(paths []string) (string, error) {
	var parts []string
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("cannot read file %q: %w", p, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("%q is a directory, not a file", p)
		}
		if info.Size() > fileContextMaxBytes {
			return "", fmt.Errorf("file %q too large (%d bytes, max %d)", p, info.Size(), fileContextMaxBytes)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("cannot read file %q: %w", p, err)
		}
		parts = append(parts, fmt.Sprintf("--- file: %s ---\n%s\n--- end ---", p, string(data)))
	}
	return "Attached file context:\n" + strings.Join(parts, "\n"), nil
}

func extractFriendlyError(raw string) string {
	if m := psMandatoryParam.FindStringSubmatch(raw); len(m) == 2 {
		return "missing required parameters: " + strings.TrimSpace(m[1])
	}
	if m := psParamNotFound.FindStringSubmatch(raw); len(m) == 2 {
		return "unknown parameter '" + m[1] + "'"
	}
	if m := missingPathErr.FindStringSubmatch(raw); len(m) == 2 {
		return "path not found: " + m[1]
	}
	matches := psGeneralError.FindAllStringSubmatch(raw, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		msg := strings.TrimSpace(matches[i][1])
		if msg != "" && !strings.HasPrefix(msg, "~") && !strings.HasPrefix(msg, "&") {
			return msg
		}
	}
	errMsg := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "exit status 1"))
	if errMsg != "" && len(errMsg) < 200 {
		return errMsg
	}
	return ""
}
