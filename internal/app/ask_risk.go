package app

import (
	"bufio"
	"fmt"
	"strings"

	"cli/internal/agent"
	"cli/internal/plugins"
	"cli/internal/ui"
	"cli/tools"
)

var askRiskBaseDir string

func normalizeRiskPolicy(raw string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(raw))
	switch p {
	case "", riskPolicyNormal:
		return riskPolicyNormal, nil
	case riskPolicyStrict, riskPolicyOff:
		return p, nil
	default:
		return "", fmt.Errorf("invalid --risk-policy %q (use strict|normal|off)", raw)
	}
}

func shouldConfirmAction(confirmTools bool, riskPolicy, risk string) bool {
	switch riskPolicy {
	case riskPolicyOff:
		return confirmTools
	case riskPolicyStrict:
		return true
	default:
		return confirmTools || risk == "high"
	}
}

func confirmAgentAction(reader *bufio.Reader, risk string) bool {
	if risk == "high" {
		fmt.Print(ui.Prompt("Confirm HIGH risk action? [y/N]: "))
		confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
		return confirm == "y" || confirm == "yes"
	}
	fmt.Print(ui.Prompt("Confirm agent action? [Y/n]: "))
	confirm := strings.ToLower(strings.TrimSpace(readLine(reader)))
	return !(confirm == "n" || confirm == "no")
}

func assessDecisionRisk(decision agent.DecisionResult) (string, string) {
	if decision.Action == "run_tool" {
		return tools.ToolRisk(decision.Tool, decision.ToolArgs)
	}
	if decision.Action == "run_plugin" {
		name := strings.ToLower(strings.TrimSpace(decision.Plugin))
		if strings.Contains(name, "reset") || strings.Contains(name, "delete") || strings.Contains(name, "drop") || strings.Contains(name, "rm") {
			return "high", "plugin may perform destructive operations"
		}
		if info, err := plugins.GetInfo(askRiskBaseDir, decision.Plugin); err == nil {
			safety := plugins.ParseToolkitSafety(info.Path)
			if safety != "" {
				risk := plugins.ToolkitRiskLevel(safety)
				return risk, safety
			}
		}
		return "medium", "external plugin execution"
	}
	return "low", "response only"
}
