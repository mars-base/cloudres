// Package cli implements the cobra command tree for cloudres.
package cli

import (
	"fmt"
	"path/filepath"

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

	// cfgPath/cfgVendor are populated by the root PersistentPreRunE when the
	// --config flag names a vendor CLI config file.
	cfgPath   string
	cfgVendor string
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
	if cfgVendor != "" {
		reg.RestrictTo(cfgVendor)
	}
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
	rootCmd.PersistentFlags().StringP("config", "c", "", "vendor CLI config file (auto-detects provider; aliyun only)")
	rootCmd.PersistentPreRunE = configureVendor
}

// configureVendor runs after flag parsing for every command (root/TUI, list,
// and dynamic provider subcommands). When --config is set it auto-detects the
// vendor from the file's contents, restricts the run to that vendor, and
// hands the path to the provider layer.
func configureVendor(cmd *cobra.Command, args []string) error {
	flagConfig, _ := rootCmd.PersistentFlags().GetString("config")
	if flagConfig == "" {
		return nil
	}

	abs, err := filepath.Abs(flagConfig)
	if err != nil {
		return fmt.Errorf("--config %q: %w", flagConfig, err)
	}
	vendor, err := provider.DetectConfigVendor(abs)
	if err != nil {
		return fmt.Errorf("--config %q: %w", flagConfig, err)
	}
	if vendor != "aliyun" {
		return fmt.Errorf("--config %q is a Huawei Cloud (hcloud) config, but hcloud has no custom config file option (it always reads ~/.hcloud/config.json); remove --config", flagConfig)
	}

	cfgPath = abs
	cfgVendor = vendor
	aliyun.SetConfigPath(abs)
	return nil
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
