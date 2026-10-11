package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/shyim/agm/internal/aliases"
	"github.com/shyim/agm/internal/db"
)

var RootCmd = &cobra.Command{
	Use:   "agm",
	Short: "agm — multi-account CLI for Antigravity (agy / IDE / desktop)",
	Long: `agm — multi-account CLI for Antigravity (agy / IDE / desktop)

Data: ~/.antigravity-agent/`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Do not auto-initialize database if help command is invoked
		if cmd.Name() == "help" {
			return nil
		}
		// Ensure that the help flags do not trigger db initialization
		if flag, err := cmd.Flags().GetBool("help"); err == nil && flag {
			return nil
		}
		return db.EnsureStore()
	},
}

func Execute() {
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		first := os.Args[1]
		if !isKnownCommand(first) {
			if isAccountOrAlias(first) {
				// Rewrite os.Args: agm acc1 [flags...] -> agm run acc1 [flags...]
				os.Args = append([]string{os.Args[0], "run", first}, os.Args[2:]...)
			}
		}
	}
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func isKnownCommand(name string) bool {
	for _, c := range RootCmd.Commands() {
		if c.Name() == name {
			return true
		}
		for _, a := range c.Aliases {
			if a == name {
				return true
			}
		}
	}
	return false
}

func isAccountOrAlias(name string) bool {
	m := aliases.Load()
	if _, ok := m[name]; ok {
		return true
	}
	acc, err := db.FindAccount(name)
	return err == nil && acc != nil
}

func resolve(pattern string) string {
	return aliases.Resolve(pattern)
}
