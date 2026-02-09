package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cli/internal/config"
	"cli/internal/plugins"
	"cli/internal/runner"
	"cli/internal/search"
	"cli/internal/store"
	"cli/internal/ui"
	"cli/tools"
)

func runLegacy(args []string) int {
	opts, rest := parseFlags(args)
	rt, err := loadRuntime(opts)
	if err != nil {
		fmt.Println("Error:", err)
		return 1
	}
	baseDir := rt.BaseDir
	cfg := rt.Config

	if len(rest) == 0 {
		packs, _ := store.ListPacks(baseDir)
		active, _ := store.GetActivePack(baseDir)
		cfgPath := filepath.Join(baseDir, "dm.json")
		cfgExists := fileExists(cfgPath)
		ui.PrintSplash(ui.SplashData{
			BaseDir:    baseDir,
			PackCount:  len(packs),
			ActivePack: active,
			ConfigPath: cfgPath,
			ConfigUsed: cfgExists,
		})
		return 0
	}

	args = rest

	// global commands
	switch args[0] {
	case "aliases", "config":
		ui.PrintAliases(cfg)
		return 0
	case "list":
		return runList(cfg, args[1:])
	case "add":
		return runAdd(baseDir, opts, args[1:])
	case "pack":
		return runPack(baseDir, args[1:])
	case "validate":
		return runValidate(baseDir, cfg)
	case "plugin":
		return runPlugin(baseDir, args[1:])
	case "tools":
		if len(args) == 1 {
			return tools.RunMenu(baseDir)
		}
		return tools.RunByName(baseDir, args[1])
	case "find", "search":
		if len(args) < 2 {
			fmt.Println("Usage: dm find <query>")
			return 0
		}
		knowledgeDir := config.ResolvePath(baseDir, cfg.Search.Knowledge)
		query := strings.Join(args[1:], " ")
		search.InKnowledge(knowledgeDir, query)
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: dm run <alias>")
			return 0
		}
		name := args[1]
		runner.RunAlias(cfg, name, "")
		return 0
	}

	// PROJECT MODE: dm <project> <action>
	return runTargetOrSearch(baseDir, cfg, args)
}

func runTargetOrSearch(baseDir string, cfg config.Config, args []string) int {
	if len(args) == 0 {
		return 0
	}

	// PROJECT MODE: dm <project> <action>
	if _, ok := cfg.Projects[args[0]]; ok && len(args) >= 2 {
		action := args[1]
		runner.RunProjectCommand(cfg, args[0], action, baseDir)
		return 0
	}

	// INTERACTIVE TARGET: dm <name>
	name := args[0]

	// target can be jump or project
	targetPath, isJump := cfg.Jump[name]
	_, isProject := cfg.Projects[name]

	if !isJump && !isProject {
		err := plugins.Run(baseDir, args[0], args[1:])
		if err != nil {
			if plugins.IsNotFound(err) {
				fmt.Println("Error:", err)
				return 1
			}
			fmt.Println("Error:", err)
			return 1
		}
		return 0
	}

	if isProject {
		targetPath = cfg.Projects[name].Path
	}

	targetPath = config.ResolvePath(baseDir, targetPath)
	ui.ShowMenu(cfg, name, targetPath, baseDir)
	return 0
}

func exeDir() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

type runtimeContext struct {
	BaseDir string
	Config  config.Config
}

func loadRuntime(opts flags) (runtimeContext, error) {
	baseDir, err := exeDir()
	if err != nil {
		return runtimeContext{}, fmt.Errorf("cannot determine executable directory: %w", err)
	}
	cfgPath := filepath.Join(baseDir, "dm.json")
	if opts.Pack == "" {
		if active, err := store.GetActivePack(baseDir); err == nil && active != "" {
			opts.Pack = active
		}
	}
	cfg, err := config.Load(cfgPath, config.Options{
		Profile:  opts.Profile,
		UseCache: !opts.NoCache,
		Pack:     opts.Pack,
	})
	if err != nil {
		return runtimeContext{}, fmt.Errorf("loading config: %w", err)
	}
	return runtimeContext{
		BaseDir: baseDir,
		Config:  cfg,
	}, nil
}

type flags struct {
	Profile string
	NoCache bool
	Pack    string
}

func parseFlags(args []string) (flags, []string) {
	var out []string
	var f flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if group, ok := mapGroupShortcut(arg); ok {
			out = append(out, group)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				out = append(out, args[i+1])
				i++
			}
			continue
		}
		if arg == "--no-cache" {
			f.NoCache = true
			continue
		}
		if arg == "--profile" && i+1 < len(args) {
			f.Profile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--profile=") {
			f.Profile = strings.TrimPrefix(arg, "--profile=")
			continue
		}
		if arg == "--pack" && i+1 < len(args) {
			f.Pack = args[i+1]
			i++
			continue
		}
		if arg == "-p" && i+1 < len(args) {
			f.Pack = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--pack=") {
			f.Pack = strings.TrimPrefix(arg, "--pack=")
			continue
		}
		out = append(out, arg)
	}
	return f, out
}

func runValidate(baseDir string, cfg config.Config) int {
	issues := config.Validate(cfg)
	issues = append(issues, validatePackMetadata(baseDir)...)
	if len(issues) == 0 {
		fmt.Println("OK: valid configuration")
		return 0
	}
	for _, issue := range issues {
		fmt.Printf("%s: %s\n", issue.Level, issue.Message)
	}
	return 1
}

func validatePackMetadata(baseDir string) []config.Issue {
	var issues []config.Issue
	packs, err := store.ListPacks(baseDir)
	if err != nil {
		return issues
	}
	for _, name := range packs {
		path := filepath.Join(baseDir, "packs", name, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			issues = append(issues, config.Issue{
				Level:   "warn",
				Message: fmt.Sprintf("pack '%s': cannot read pack.json (%v)", name, err),
			})
			continue
		}
		if strings.TrimSpace(pf.Description) == "" {
			issues = append(issues, config.Issue{
				Level:   "warn",
				Message: fmt.Sprintf("pack '%s': description is empty", name),
			})
		}
		if pf.SchemaVersion != 1 {
			issues = append(issues, config.Issue{
				Level:   "warn",
				Message: fmt.Sprintf("pack '%s': schema_version %d is not supported (expected 1)", name, pf.SchemaVersion),
			})
		}
		if len(pf.Examples) == 0 {
			issues = append(issues, config.Issue{
				Level:   "warn",
				Message: fmt.Sprintf("pack '%s': examples is empty", name),
			})
		}
	}
	return issues
}

func runList(cfg config.Config, args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: dm list <jumps|runs|projects|actions>")
		return 0
	}
	switch args[0] {
	case "jumps":
		ui.PrintMap(cfg.Jump)
	case "runs":
		ui.PrintMap(cfg.Run)
	case "projects":
		ui.PrintProjects(cfg.Projects)
	case "actions":
		if len(args) < 2 {
			fmt.Println("Usage: dm list actions <project>")
			return 0
		}
		p, ok := cfg.Projects[args[1]]
		if !ok {
			fmt.Println("Project not found:", args[1])
			return 0
		}
		ui.PrintMap(p.Commands)
	default:
		fmt.Println("Usage: dm list <jumps|runs|projects|actions>")
	}
	return 0
}

func runAdd(baseDir string, opts flags, args []string) int {
	if len(args) < 1 {
		fmt.Println("Usage: dm add <jump|run|project|action> ...")
		return 0
	}
	pack := opts.Pack
	if pack == "" {
		fmt.Println("Error: no active pack. Use -p <pack> or dm pack use <name>.")
		return 1
	}
	switch args[0] {
	case "jump":
		if len(args) < 3 {
			fmt.Println("Usage: dm add jump <name> <path>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		pf.Jump[args[1]] = args[2]
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: jump added")
		return 0
	case "run":
		if len(args) < 3 {
			fmt.Println("Usage: dm add run <name> <command>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		pf.Run[args[1]] = strings.Join(args[2:], " ")
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: run added")
		return 0
	case "project":
		if len(args) < 3 {
			fmt.Println("Usage: dm add project <name> <path>")
			return 0
		}
		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		if pf.Projects == nil {
			pf.Projects = map[string]store.Project{}
		}
		pf.Projects[args[1]] = store.Project{
			Path:     args[2],
			Commands: map[string]string{},
		}
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: project added")
		return 0
	case "action":
		if len(args) < 4 {
			fmt.Println("Usage: dm add action <project> <name> <command>")
			return 0
		}
		project := args[1]
		action := args[2]
		cmd := strings.Join(args[3:], " ")

		path := filepath.Join(baseDir, "packs", pack, "pack.json")
		pf, err := store.LoadPackFile(path)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		p, ok := pf.Projects[project]
		if !ok {
			fmt.Println("Project not found. Add the project first.")
			return 1
		}
		if p.Commands == nil {
			p.Commands = map[string]string{}
		}
		p.Commands[action] = cmd
		pf.Projects[project] = p
		if err := store.SavePackFile(path, pf); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: action added")
		return 0
	default:
		fmt.Println("Usage: dm add <jump|run|project|action> ...")
		return 0
	}
}

func runPlugin(baseDir string, args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: dm plugin <list|info|run> ...")
		return 0
	}
	switch args[0] {
	case "list":
		includeFunctions := false
		for _, arg := range args[1:] {
			if arg == "--functions" || arg == "-f" {
				includeFunctions = true
			}
		}
		items, err := plugins.ListEntries(baseDir, includeFunctions)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("No plugins/functions found.")
			return 0
		}
		for _, item := range items {
			if includeFunctions {
				if item.Kind == "function" {
					fmt.Println(item.Name)
				}
				continue
			}
			if item.Kind == "script" {
				fmt.Println(item.Name)
			}
		}
		return 0
	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: dm plugin info <name>")
			return 0
		}
		info, err := plugins.GetInfo(baseDir, args[1])
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("Name      :", info.Name)
		fmt.Println("Kind      :", info.Kind)
		fmt.Println("Path      :", info.Path)
		fmt.Println("Runner    :", info.Runner)
		if len(info.Sources) > 1 {
			fmt.Println("Sources   :", strings.Join(info.Sources, ", "))
		}
		if strings.TrimSpace(info.Synopsis) != "" {
			fmt.Println("Synopsis  :", info.Synopsis)
		}
		if strings.TrimSpace(info.Description) != "" {
			fmt.Println("Description:", info.Description)
		}
		if len(info.Parameters) > 0 {
			fmt.Println("Parameters:")
			for _, p := range info.Parameters {
				fmt.Println("-", p)
			}
		}
		if len(info.Examples) > 0 {
			fmt.Println("Examples:")
			for _, ex := range info.Examples {
				fmt.Println("-", ex)
			}
		}
		return 0
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: dm plugin run <name> [args...]")
			return 0
		}
		if err := plugins.Run(baseDir, args[1], args[2:]); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		return 0
	default:
		fmt.Println("Usage: dm plugin <list|info|run> ...")
		return 0
	}
}

func runPack(baseDir string, args []string) int {
	if len(args) < 1 {
		fmt.Println("Usage: dm pack <new|clone|list|info|use|current|unset> [name]")
		return 0
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			fmt.Println("Usage: dm pack new <name>")
			return 0
		}
		name := args[1]
		if err := store.CreatePack(baseDir, name); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: pack created")
		return 0
	case "clone":
		if len(args) < 3 {
			fmt.Println("Usage: dm pack clone <src> <dst>")
			return 0
		}
		if err := store.ClonePack(baseDir, args[1], args[2]); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Printf("OK: pack cloned %s -> %s\n", args[1], args[2])
		return 0
	case "list":
		items, err := store.ListPacks(baseDir)
		if err != nil {
			fmt.Println(ui.Error("Error:"), err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println(ui.Warn("No packs found."))
			return 0
		}
		for _, name := range items {
			fmt.Println(name)
		}
		return 0
	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: dm pack info <name>")
			return 0
		}
		info, err := store.GetPackInfo(baseDir, args[1])
		if err != nil {
			fmt.Println(ui.Error("Error:"), err)
			return 1
		}
		ui.PrintSection("Pack Info")
		ui.PrintKV("Pack", info.Name)
		ui.PrintKV("Path", info.Path)
		if strings.TrimSpace(info.Description) != "" {
			ui.PrintKV("Description", info.Description)
		}
		if strings.TrimSpace(info.Summary) != "" {
			ui.PrintKV("Summary", info.Summary)
		}
		if strings.TrimSpace(info.Owner) != "" {
			ui.PrintKV("Owner", info.Owner)
		}
		if len(info.Tags) > 0 {
			ui.PrintKV("Tags", strings.Join(info.Tags, ", "))
		}
		if info.Knowledge != "" {
			ui.PrintKV("Knowledge", info.Knowledge)
		}
		ui.PrintSection("Counts")
		ui.PrintKV("Jumps", fmt.Sprintf("%d", info.Jumps))
		ui.PrintKV("Runs", fmt.Sprintf("%d", info.Runs))
		ui.PrintKV("Projects", fmt.Sprintf("%d", info.Projects))
		ui.PrintKV("Actions", fmt.Sprintf("%d", info.Actions))
		if len(info.Examples) > 0 {
			ui.PrintSection("Examples")
			for _, ex := range info.Examples {
				fmt.Printf("- %s\n", ex)
			}
		}
		return 0
	case "use":
		if len(args) < 2 {
			fmt.Println("Usage: dm pack use <name>")
			return 0
		}
		name := args[1]
		if !store.PackExists(baseDir, name) {
			fmt.Println("Pack not found:", name)
			return 1
		}
		if err := store.SetActivePack(baseDir, name); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: active pack ->", name)
		return 0
	case "current":
		name, err := store.GetActivePack(baseDir)
		if err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		if name == "" {
			fmt.Println("No active pack.")
			return 0
		}
		fmt.Println(name)
		return 0
	case "unset":
		if err := store.ClearActivePack(baseDir); err != nil {
			fmt.Println("Error:", err)
			return 1
		}
		fmt.Println("OK: active pack cleared")
		return 0
	default:
		fmt.Println("Usage: dm pack <new|clone|list|info|use|current|unset> [name]")
		return 0
	}
}
