package app

import (
	"fmt"
	"strings"

	"cli/internal/config"
	"cli/internal/runner"
	"cli/internal/search"
	"cli/internal/ui"
	"cli/tools"

	"github.com/spf13/cobra"
)

func addCobraSubcommands(root *cobra.Command, opts *flags) {
	root.AddCommand(&cobra.Command{
		Use:     "aliases",
		Aliases: []string{"a"},
		Short:   "Show aliases and projects",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			ui.PrintAliases(rt.Config)
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Show aliases and projects",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			ui.PrintAliases(rt.Config)
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List config entries",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runList(rt.Config, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "Add config entries",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runAdd(rt.BaseDir, *opts, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "pack",
		Short: "Manage packs",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runPack(rt.BaseDir, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runPlugin(rt.BaseDir, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:     "tools [tool]",
		Aliases: []string{"tool"},
		Short:   "Run tools menu or a specific tool",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			var code int
			if len(args) == 0 {
				code = tools.RunMenu(rt.BaseDir)
			} else {
				code = tools.RunByName(rt.BaseDir, args[0])
			}
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "find <query...>",
		Short: "Search knowledge markdown",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFindCommand(*opts, args)
		},
	})
	root.AddCommand(&cobra.Command{
		Use:     "search <query...>",
		Aliases: []string{"f"},
		Short:   "Search knowledge markdown",
		Hidden:  true,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFindCommand(*opts, args)
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "run <alias>",
		Short: "Run alias from config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			runner.RunAlias(rt.Config, args[0], "")
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runValidate(rt.Config)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "help [topic]",
		Short: "Show help",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(*opts)
			if err != nil {
				return err
			}
			code := runHelp(rt.BaseDir, rt.Config, args)
			if code != 0 {
				return exitCodeError{code: code}
			}
			return nil
		},
	})
}

func runFindCommand(opts flags, args []string) error {
	if len(args) < 1 {
		fmt.Println("Uso: dm find <query>")
		return nil
	}
	rt, err := loadRuntime(opts)
	if err != nil {
		return err
	}
	knowledgeDir := config.ResolvePath(rt.BaseDir, rt.Config.Search.Knowledge)
	search.InKnowledge(knowledgeDir, strings.Join(args, " "))
	return nil
}
