package plugins

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const pluginExecTimeout = 5 * time.Minute

type psNamedArg struct {
	Name     string
	Value    string
	IsSwitch bool
}

func shellLooksLikeBash() bool {
	shell := strings.ToLower(strings.TrimSpace(os.Getenv("SHELL")))
	return strings.Contains(shell, "bash") || strings.Contains(shell, "zsh") || strings.Contains(shell, "fish")
}

func firstAvailableBinary(names ...string) string {
	for _, n := range names {
		if _, err := exec.LookPath(n); err == nil {
			return n
		}
	}
	return ""
}

func quotePowerShellArg(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func looksLikePowerShellNamedToken(v string) bool {
	token := strings.TrimSpace(v)
	if !strings.HasPrefix(token, "-") || token == "-" {
		return false
	}
	// Treat negative numbers (for example -1, -0.5) as values, not parameter names.
	if len(token) > 1 {
		ch := token[1]
		if (ch >= '0' && ch <= '9') || ch == '.' {
			return false
		}
	}
	return true
}

func splitPowerShellSplatArgs(args []string) ([]psNamedArg, []string) {
	named := make([]psNamedArg, 0, len(args))
	positional := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		if !looksLikePowerShellNamedToken(current) {
			positional = append(positional, args[i])
			continue
		}
		name := strings.TrimLeft(current, "-")
		if name == "" {
			positional = append(positional, args[i])
			continue
		}
		if i+1 < len(args) && !looksLikePowerShellNamedToken(args[i+1]) {
			named = append(named, psNamedArg{Name: name, Value: args[i+1]})
			i++
			continue
		}
		named = append(named, psNamedArg{Name: name, IsSwitch: true})
	}
	return named, positional
}

func buildPowerShellFunctionScript(profilePaths []string, functionName string, args []string) string {
	quotedPaths := make([]string, 0, len(profilePaths))
	for _, p := range profilePaths {
		quotedPaths = append(quotedPaths, quotePowerShellArg(p))
	}
	namedArgs, positionalArgs := splitPowerShellSplatArgs(args)

	lines := []string{
		"[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new()",
		"Set-StrictMode -Version Latest",
		"$ErrorActionPreference='Stop'",
		"$dmProfilePaths=@(" + strings.Join(quotedPaths, ",") + ")",
		"$dmNamedArgs=@{}",
		"$dmPositionalArgs=@()",
	}
	for _, a := range namedArgs {
		valueExpr := "$true"
		if !a.IsSwitch {
			valueExpr = quotePowerShellArg(a.Value)
		}
		lines = append(lines, "$dmNamedArgs["+quotePowerShellArg(a.Name)+"]="+valueExpr)
	}
	for _, a := range positionalArgs {
		lines = append(lines, "$dmPositionalArgs+="+quotePowerShellArg(a))
	}
	lines = append(lines,
		"foreach($dmProfilePath in $dmProfilePaths){ if(Test-Path -LiteralPath $dmProfilePath){ . $dmProfilePath } }",
		"if(-not(Get-Command -Name "+quotePowerShellArg(functionName)+" -CommandType Function -ErrorAction SilentlyContinue)){",
		"  throw \"Function '"+functionName+"' was not loaded from plugin sources.\"",
		"}",
		"& "+quotePowerShellArg(functionName)+" @dmNamedArgs @dmPositionalArgs",
	)
	return strings.Join(lines, "\n") + "\n"
}

func runPowerShellFunctionCapture(profilePaths []string, functionName string, args []string, interactive bool) (string, error) {
	ps := firstAvailableBinary("pwsh", "powershell")
	if ps == "" {
		return "", errors.New("pwsh/powershell executable not found")
	}

	scriptBody := buildPowerShellFunctionScript(profilePaths, functionName, args)

	tmp, tmpErr := os.CreateTemp("", "dm-plugin-*.ps1")
	if tmpErr != nil {
		return "", tmpErr
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if writeErr := os.WriteFile(tmpPath, []byte(scriptBody), 0600); writeErr != nil {
		return "", writeErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), pluginExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ps, "-NoProfile", "-NonInteractive", "-File", tmpPath)
	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	if interactive {
		cmd.Stdin = os.Stdin
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output.String(), &RunError{
				Err:    errors.New("plugin execution timed out after " + pluginExecTimeout.String()),
				Output: output.String(),
			}
		}
		return output.String(), &RunError{Err: err, Output: output.String()}
	}
	return output.String(), nil
}

func execPluginCapture(path string, args []string, interactive bool) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))

	ctx, cancel := context.WithTimeout(context.Background(), pluginExecTimeout)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return "", errors.New("powershell executable not found")
			}
			cmd = exec.CommandContext(ctx, ps, "-NoProfile", "-NonInteractive", "-File", path)
		case ".sh":
			sh := firstAvailableBinary("sh", "bash")
			if sh == "" {
				return "", errors.New("sh/bash executable not found")
			}
			cmd = exec.CommandContext(ctx, sh, path)
		case ".cmd", ".bat":
			cmd = exec.CommandContext(ctx, "cmd", "/C", path)
		case ".exe", "", ".out":
			cmd = exec.CommandContext(ctx, path)
		default:
			return "", errors.New("unsupported plugin type on windows")
		}
	default:
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return "", errors.New("pwsh/powershell executable not found")
			}
			cmd = exec.CommandContext(ctx, ps, "-File", path)
		case ".sh":
			cmd = exec.CommandContext(ctx, "sh", path)
		default:
			cmd = exec.CommandContext(ctx, path)
		}
	}

	if len(args) > 0 {
		cmd.Args = append(cmd.Args, args...)
	}

	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)
	if interactive {
		cmd.Stdin = os.Stdin
	}
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return output.String(), &RunError{
				Err:    errors.New("plugin execution timed out after " + pluginExecTimeout.String()),
				Output: output.String(),
			}
		}
		return output.String(), &RunError{Err: err, Output: output.String()}
	}
	return output.String(), nil
}

func runnerForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			return "powershell -File"
		case ".sh":
			return "sh"
		case ".cmd", ".bat":
			return "cmd /C"
		case ".exe", "", ".out":
			return "direct"
		}
	default:
		switch ext {
		case ".ps1":
			return "pwsh -File"
		case ".sh":
			return "sh"
		default:
			return "direct"
		}
	}
	return "unknown"
}

func preferredPluginExtOrder() []string {
	if shellLooksLikeBash() {
		if runtime.GOOS == "windows" {
			return []string{".sh", ".ps1", ".cmd", ".bat", ".exe", "", ".out"}
		}
		return []string{".sh", "", ".out", ".ps1"}
	}
	if runtime.GOOS == "windows" {
		return []string{".ps1", ".cmd", ".bat", ".exe", ".sh", "", ".out"}
	}
	return []string{".sh", "", ".out", ".ps1"}
}
