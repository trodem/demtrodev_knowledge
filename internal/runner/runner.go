package runner

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"cli/internal/config"
)

func RunAlias(cfg config.Config, name, workDir string) {
	cmdStr, ok := cfg.Run[name]
	if !ok {
		fmt.Println("Run alias not found:", name)
		return
	}
	ExecShell(cmdStr, workDir)
}

func RunProjectCommand(cfg config.Config, project, action string, baseDir string) {
	p, ok := cfg.Projects[project]
	if !ok {
		fmt.Println("Project not found:", project)
		return
	}
	cmdStr, ok := p.Commands[action]
	if !ok {
		fmt.Printf("Action '%s' not found for project '%s'\n", action, project)
		return
	}
	workDir := config.ResolvePath(baseDir, p.Path)
	ExecShell(cmdStr, workDir)
}

func ExecShell(command string, workDir string) {
	fmt.Println(">", command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-lc", command)
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	_ = cmd.Run()
}
