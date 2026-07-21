// Package cli implements the cobra command tree for cloudres.
package cli

import (
	"fmt"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
	"github.com/mars-base/cloudres/internal/provider/aliyun"
	"github.com/mars-base/cloudres/internal/provider/huawei"
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
	reg.Register(huawei.NewDetector())
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
	rootCmd.PersistentFlags().String("profile", "", "cloud provider profile (e.g. aliyun profile name)")
}

// resolveProfile picks the active profile from the --profile flag, falling
// back to Detect()'s ActiveProfile (the config's "current" field). Returns
// an error if neither is set so the user gets a clear message instead of
// silently empty credentials.
func resolveProfile(p *core.Provider) error {
	flagProfile, _ := rootCmd.PersistentFlags().GetString("profile")
	if flagProfile != "" {
		p.ActiveProfile = flagProfile
		return nil
	}
	if p.ActiveProfile != "" {
		return nil
	}
	return fmt.Errorf(
		"no profile specified: set --profile or configure \"current\" in %s",
		p.ConfigPath,
	)
}
