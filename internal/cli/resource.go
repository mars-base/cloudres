package cli

import (
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
	for _, name := range reg.ProviderNames() {
		providerName := name
		providerCmd := &cobra.Command{
			Use:   providerName,
			Short: fmt.Sprintf("Query %s resources", providerName),
		}

		fetchers := reg.FetchersFor(providerName)
		for _, fetcher := range fetchers {
			f := fetcher
			rtype := f.ResourceType()
			resourceCmd := &cobra.Command{
				Use:   rtype,
				Short: fmt.Sprintf("List %s %s resources", providerName, rtype),
				RunE: func(cmd *cobra.Command, args []string) error {
					return runResourceCommand(cmd, reg, providerName, f)
				},
			}
			resourceCmd.Flags().String("region", "", "filter by region")
			resourceCmd.Flags().Bool("sync", false, "force sync before listing")
			providerCmd.AddCommand(resourceCmd)
		}

		rootCmd.AddCommand(providerCmd)
	}
}

func runResourceCommand(cmd *cobra.Command, reg *provider.Registry, providerName string, fetcher core.ResourceFetcher) error {
	db, err := core.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	p, err := reg.Get(cmd.Context(), providerName)
	if err != nil {
		return err
	}

	if err := resolveProfile(p); err != nil {
		return err
	}

	db.UpsertProvider(*p)

	forceSync, _ := cmd.Flags().GetBool("sync")
	region, _ := cmd.Flags().GetString("region")

	var resources []core.Resource
	if forceSync {
		resources, err = core.SyncAndList(cmd.Context(), db, p, fetcher, region)
	} else {
		resources, err = core.Sync(cmd.Context(), db, p, fetcher, region)
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
