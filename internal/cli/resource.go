package cli

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
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

	// Render table with proper CJK alignment
	columns := core.Columns(providerName, fetcher.ResourceType())

	// Collect all rows including header
	allRows := make([][]string, 0, len(resources)+1)
	allRows = append(allRows, columns)
	for _, r := range resources {
		allRows = append(allRows, r.Row())
	}

	// Calculate display widths
	ncols := len(columns)
	widths := make([]int, ncols)
	for _, row := range allRows {
		for i, v := range row {
			if i >= ncols {
				break
			}
			if w := runewidth.StringWidth(v); w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i := range widths {
		if widths[i] > 40 {
			widths[i] = 40
		}
	}

	// Print header
	for i, col := range columns {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(padRight(cliTruncate(col, widths[i]), widths[i]))
	}
	fmt.Println()

	// Print rows
	for _, r := range resources {
		row := r.Row()
		for i, val := range row {
			if i > 0 {
				fmt.Print("  ")
			}
			if i >= ncols {
				break
			}
			fmt.Print(padRight(cliTruncate(val, widths[i]), widths[i]))
		}
		fmt.Println()
	}

	fmt.Printf("\n%d resources\n", len(resources))
	return nil
}

func cliTruncate(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	if max <= 3 {
		return runewidth.Truncate(s, max, "")
	}
	return runewidth.Truncate(s, max-3, "") + "..."
}

func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
