package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mars-base/cloudres/internal/core"
	"github.com/mars-base/cloudres/internal/provider"
)

// checkRegionGate blocks region-scoped queries when the active profile has no
// region configured. Without --region it prints a warning (including the
// built-in region catalog) to stderr and returns true so the caller exits
// without querying; with --region it pins fetching to that region and
// returns false.
func checkRegionGate(reg *provider.Registry, p *core.Provider, rtype, regionFlag string) bool {
	rs, ok := reg.Detector(p.Name).(core.RegionSupport)
	if !ok {
		return false
	}
	if len(p.ProfileRegions[p.ActiveProfile]) > 0 {
		return false
	}

	if !rs.RequiresRegion(rtype) {
		// Account-global APIs don't need a real region, but the aliyun CLI
		// still requires one for endpoint resolution.
		rs.SetRegion("")
		return false
	}
	if regionFlag == "" {
		fmt.Fprint(os.Stderr, formatRegionWarning(p, rtype, rs.RegionOptions()))
		return true
	}

	// Pin fetching to the requested region: most fetchers loop p.Regions,
	// while ack/acl/kfk/ck/prom read ProfileRegions[ActiveProfile].
	rs.SetRegion(regionFlag)
	if p.ProfileRegions == nil {
		p.ProfileRegions = map[string][]string{}
	}
	p.ProfileRegions[p.ActiveProfile] = []string{regionFlag}
	p.Regions = []string{regionFlag}
	return false
}

func formatRegionWarning(p *core.Provider, rtype string, opts []core.RegionOption) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Warning: %s profile %q has no region configured, and %q is queried per region.\n",
		p.Name, p.ActiveProfile, rtype)
	sb.WriteString("Specify one with --region <id>, e.g.:\n")
	fmt.Fprintf(&sb, "  cloudres %s %s --region cn-hangzhou\n", p.Name, rtype)
	fmt.Fprintf(&sb, "or set \"region_id\" for the profile in %s.\n\n", p.ConfigPath)
	sb.WriteString("Available regions:\n")

	idWidth := 0
	for _, o := range opts {
		if len(o.ID) > idWidth {
			idWidth = len(o.ID)
		}
	}
	colWidth := idWidth + 3
	const perLine = 4
	for i, o := range opts {
		if i%perLine == 0 {
			sb.WriteString("  ")
		}
		fmt.Fprintf(&sb, "%-*s", colWidth, o.ID)
		if i%perLine == perLine-1 || i == len(opts)-1 {
			sb.WriteByte('\n')
		}
	}

	sb.WriteString("\nNo resources were fetched.\n")
	return sb.String()
}
