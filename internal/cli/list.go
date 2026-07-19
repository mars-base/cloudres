package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
	"github.com/mars-base/cloudres/internal/provider/aliyun"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Detect and list cloud providers",
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

		if len(providers) == 0 {
			fmt.Println("No cloud providers detected.")
			fmt.Println("\nSupported providers and their required CLI tools:")
			fmt.Println("  aliyun  → aliyun CLI + ~/.aliyun/config.json")
			fmt.Println("  aws     → aws CLI + ~/.aws/credentials")
			fmt.Println("  huawei  → hcloud CLI + ~/.hcloud/")
			return nil
		}

		// Persist detected providers
		for _, p := range providers {
			db.UpsertProvider(*p)
		}

		// Build rows: one per profile
		type row struct {
			provider string
			profile  string
			regions  string
		}
		var rows []row
		for _, p := range providers {
			if len(p.Profiles) > 0 {
				for _, prof := range p.Profiles {
					regions := p.Regions
					if pr, ok := p.ProfileRegions[prof]; ok {
						regions = pr
					}
					rows = append(rows, row{p.Name, prof, strings.Join(regions, ", ")})
				}
			} else {
				rows = append(rows, row{p.Name, "", strings.Join(p.Regions, ", ")})
			}
		}

		// Calculate column widths
		wProv, wProf, wReg := 8, 7, 7
		for _, r := range rows {
			if l := len(r.provider); l > wProv {
				wProv = l
			}
			if l := len(r.profile); l > wProf {
				wProf = l
			}
			if l := len(r.regions); l > wReg {
				wReg = l
			}
		}

		// Print table
		hl := strings.Repeat("─", wProv+2) + strings.Repeat("─", wProf+2) + strings.Repeat("─", wReg+2)
		fmt.Printf("  %-*s  %-*s  %-*s\n", wProv, "Provider", wProf, "Profile", wReg, "Regions")
		fmt.Println(hl)
		for _, r := range rows {
			fmt.Printf("  %-*s  %-*s  %-*s\n", wProv, r.provider, wProf, r.profile, wReg, r.regions)
		}
		return nil
	},
}
