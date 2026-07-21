package huawei

import (
	"context"

	"github.com/mars-base/cloudres/internal/core"
)

// fetchByProfiles iterates over each profile→region mapping and calls fn for
// each. Huawei Cloud differs from Aliyun: each profile is bound to a single
// region+project, so we can't iterate all regions with one profile.
// If p.ActiveProfile is set (via --profile flag or config "current"), only
// that profile is used.
func fetchByProfiles(ctx context.Context, p *core.Provider, fn func(ctx context.Context, profile, region string) ([]core.Resource, error)) ([]core.Resource, error) {
	var allResources []core.Resource

	profiles := p.Profiles
	if p.ActiveProfile != "" {
		profiles = []string{p.ActiveProfile}
	}

	for _, profile := range profiles {
		regions, ok := p.ProfileRegions[profile]
		if !ok || len(regions) == 0 {
			resources, err := fn(ctx, profile, "")
			if err != nil {
				return nil, err
			}
			allResources = append(allResources, resources...)
			continue
		}
		for _, region := range regions {
			resources, err := fn(ctx, profile, region)
			if err != nil {
				return nil, err
			}
			allResources = append(allResources, resources...)
		}
	}

	if len(profiles) == 0 {
		resources, err := fn(ctx, "", "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}
