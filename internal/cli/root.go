// Package cli implements the cobra command tree for cloudres.
package cli

import (
	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
	"github.com/mars-base/cloudres/internal/provider/aliyun"
	"github.com/mars-base/cloudres/internal/tui"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "cloudres",
	Short: "Cloud resource query and display tool",
	Long: `cloudres queries cloud provider CLIs and caches resources in a local SQLite database.
Run without arguments to enter the interactive TUI mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// No subcommand → enter TUI
		db, err := core.Open()
		if err != nil {
			return err
		}
		defer db.Close()

		reg := newRegistry()
		return tui.Run(cmd.Context(), db, reg)
	},
}

func newRegistry() *provider.Registry {
	reg := provider.NewRegistry()
	reg.Register(aliyun.NewDetector())
	return reg
}

func Execute() error {
	// Register dynamic provider subcommands before executing
	reg := newRegistry()
	registerProviderCommands(reg)

	return rootCmd.Execute()
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
