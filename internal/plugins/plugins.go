package plugins

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type Plugin struct {
	Name string
	Path string
}

var ErrNotFound = errors.New("plugin not found")
var psFunctionLine = regexp.MustCompile(`(?i)^\s*function\s+([a-z0-9_-]+)\b`)

const profileFunctionsFile = "Import-Module PSReadLine.txt"

func List(baseDir string) ([]Plugin, error) {
	dir := filepath.Join(baseDir, "plugins")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Plugin{}, nil
		}
		return nil, err
	}

	bestByName := map[string]Plugin{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isSupportedPlugin(name) {
			continue
		}
		baseName := pluginName(name)
		candidate := Plugin{
			Name: baseName,
			Path: filepath.Join(dir, name),
		}
		current, ok := bestByName[baseName]
		if !ok || pluginScore(candidate.Path) < pluginScore(current.Path) {
			bestByName[baseName] = candidate
		}
	}

	plugins := make([]Plugin, 0, len(bestByName))
	for _, p := range bestByName {
		plugins = append(plugins, p)
	}
	functionPath := filepath.Join(dir, profileFunctionsFile)
	functionNames, err := readPowerShellFunctionNames(functionPath)
	if err != nil {
		return nil, err
	}
	for _, name := range functionNames {
		if _, ok := bestByName[name]; ok {
			continue
		}
		plugins = append(plugins, Plugin{
			Name: name,
			Path: functionPath,
		})
	}

	sort.Slice(plugins, func(i, j int) bool { return plugins[i].Name < plugins[j].Name })
	return plugins, nil
}

func Run(baseDir, name string, args []string) error {
	dir := filepath.Join(baseDir, "plugins")
	candidate, err := findPlugin(dir, name)
	if err != nil {
		return err
	}
	if candidate == "" {
		functionPath, found, fErr := findPowerShellFunction(dir, name)
		if fErr != nil {
			return fErr
		}
		if !found {
			return fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return runPowerShellFunction(functionPath, name, args)
	}
	return execPlugin(candidate, args)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func findPlugin(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	var matches []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !isSupportedPlugin(e.Name()) {
			continue
		}
		if pluginName(e.Name()) == name {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}
	if len(matches) == 0 {
		return "", nil
	}

	sort.Slice(matches, func(i, j int) bool {
		si := pluginScore(matches[i])
		sj := pluginScore(matches[j])
		if si == sj {
			return strings.ToLower(matches[i]) < strings.ToLower(matches[j])
		}
		return si < sj
	})
	return matches[0], nil
}

func pluginScore(path string) int {
	ext := strings.ToLower(filepath.Ext(path))
	order := preferredPluginExtOrder()
	for i, v := range order {
		if ext == v {
			return i
		}
	}
	return len(order) + 1
}

func findPowerShellFunction(pluginsDir, name string) (string, bool, error) {
	functionPath := filepath.Join(pluginsDir, profileFunctionsFile)
	names, err := readPowerShellFunctionNames(functionPath)
	if err != nil {
		return "", false, err
	}
	for _, fn := range names {
		if fn == name {
			return functionPath, true, nil
		}
	}
	return "", false, nil
}

func readPowerShellFunctionNames(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var out []string
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		m := psFunctionLine.FindStringSubmatch(line)
		if len(m) != 2 {
			continue
		}
		name := strings.TrimSpace(m[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
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

func runPowerShellFunction(profilePath, functionName string, args []string) error {
	ps := firstAvailableBinary("pwsh", "powershell")
	if ps == "" {
		return errors.New("pwsh/powershell executable not found")
	}
	script := "$dmProfilePath=" + quotePowerShellArg(profilePath) + "; $oldEap=$ErrorActionPreference; $ErrorActionPreference='SilentlyContinue'; Invoke-Expression (Get-Content -Raw $dmProfilePath); $ErrorActionPreference=$oldEap; " + functionName
	if len(args) > 0 {
		quoted := make([]string, 0, len(args))
		for _, a := range args {
			quoted = append(quoted, quotePowerShellArg(a))
		}
		script += " " + strings.Join(quoted, " ")
	}
	cmd := exec.Command(ps, "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func execPlugin(path string, args []string) error {
	ext := strings.ToLower(filepath.Ext(path))
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("powershell", "pwsh")
			if ps == "" {
				return errors.New("powershell executable not found")
			}
			cmd = exec.Command(ps, "-ExecutionPolicy", "Bypass", "-File", path)
		case ".sh":
			sh := firstAvailableBinary("sh", "bash")
			if sh == "" {
				return errors.New("sh/bash executable not found")
			}
			cmd = exec.Command(sh, path)
		case ".cmd", ".bat":
			cmd = exec.Command("cmd", "/C", path)
		case ".exe", "", ".out":
			cmd = exec.Command(path)
		default:
			return errors.New("unsupported plugin type on windows")
		}
	default:
		switch ext {
		case ".ps1":
			ps := firstAvailableBinary("pwsh", "powershell")
			if ps == "" {
				return errors.New("pwsh/powershell executable not found")
			}
			cmd = exec.Command(ps, "-File", path)
		case ".sh":
			cmd = exec.Command("sh", path)
		default:
			cmd = exec.Command(path)
		}
	}

	if len(args) > 0 {
		cmd.Args = append(cmd.Args, args...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func isSupportedPlugin(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ps1" || ext == ".cmd" || ext == ".bat" || ext == ".exe" || ext == ".sh" || ext == "" || ext == ".out"
}

func pluginName(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
