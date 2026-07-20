package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
	"github.com/spf13/cobra"
)

// registerProviderCommands dynamically creates subcommands like:
//
//	cloudres aliyun ecs
//	cloudres aliyun vpc
func registerProviderCommands(reg *provider.Registry) {
	ctx := context.Background()

	for _, name := range reg.ProviderNames() {
		providerName := name // capture
		providerCmd := &cobra.Command{
			Use:   providerName,
			Short: fmt.Sprintf("Query %s resources", providerName),
		}

		fetchers := reg.FetchersFor(providerName)
		for _, fetcher := range fetchers {
			f := fetcher // capture
			rtype := f.ResourceType()
			resourceCmd := &cobra.Command{
				Use:   rtype,
				Short: fmt.Sprintf("List %s %s resources", providerName, rtype),
				RunE: func(cmd *cobra.Command, args []string) error {
					return runResourceCommand(cmd.Context(), reg, providerName, f)
				},
			}
			resourceCmd.Flags().String("region", "", "filter by region")
			resourceCmd.Flags().Bool("sync", false, "force sync before listing")
			providerCmd.AddCommand(resourceCmd)
		}

		_ = ctx
		rootCmd.AddCommand(providerCmd)
	}
}

func runResourceCommand(ctx context.Context, reg *provider.Registry, providerName string, fetcher core.ResourceFetcher) error {
	db, err := core.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	p, err := reg.Get(ctx, providerName)
	if err != nil {
		return err
	}

	// Pick first profile as the active one for CLI mode, so cached
	// results are scoped per-profile (matching TUI behavior) and
	// resources queried against the correct credentials.
	if p.ActiveProfile == "" && len(p.Profiles) > 0 {
		p.ActiveProfile = p.Profiles[0]
	}

	// Save detected provider
	db.UpsertProvider(*p)

	var resources []core.Resource
	if forceSync, _ := rootCmd.Flags().GetBool("sync"); forceSync {
		resources, err = core.SyncAndList(ctx, db, p, fetcher, "")
	} else {
		resources, err = core.Sync(ctx, db, p, fetcher, "")
	}
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		fmt.Printf("No %s resources found.\n", fetcher.ResourceType())
		return nil
	}

	// Render table
	columns := core.Columns(fetcher.ResourceType())
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Header
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, col)
	}
	fmt.Fprintln(w)

	// Rows
	for _, r := range resources {
		row := r.Row()
		for i, val := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, truncate(val, 40))
		}
		fmt.Fprintln(w)
	}
	w.Flush()

	fmt.Printf("\n%d resources\n", len(resources))
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
