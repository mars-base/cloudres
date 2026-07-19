package cli

import (
	"context"
	"fmt"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
	"github.com/mars-base/cloudres/internal/provider/aliyun"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync [provider]",
	Short: "Force sync resources from provider(s)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := core.Open()
		if err != nil {
			return err
		}
		defer db.Close()

		reg := provider.NewRegistry()
		reg.Register(aliyun.NewDetector())

		ctx := context.Background()
		providers := reg.DetectAll(ctx)

		if len(args) > 0 {
			// Sync specific provider
			target := args[0]
			var found *core.Provider
			for _, p := range providers {
				if p.Name == target {
					found = p
					break
				}
			}
			if found == nil {
				return fmt.Errorf("provider %q not detected", target)
			}
			return syncProvider(ctx, db, reg, found)
		}

		// Sync all providers
		for _, p := range providers {
			if err := syncProvider(ctx, db, reg, p); err != nil {
				fmt.Printf("Error syncing %s: %v\n", p.Name, err)
			}
		}
		return nil
	},
}

func syncProvider(ctx context.Context, db *core.DB, reg *provider.Registry, p *core.Provider) error {
	fmt.Printf("Syncing %s...\n", p.Name)
	db.UpsertProvider(*p)

	for _, fetcher := range reg.FetchersFor(p.Name) {
		fmt.Printf("  %-6s ... ", fetcher.ResourceType())
		resources, err := core.SyncAndList(ctx, db, p, fetcher, "")
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}
		fmt.Printf("%d resources\n", len(resources))
	}
	return nil
}
