package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// VSwitchFetcher fetches VSwitches (aliyun's equivalent of a subnet).
type VSwitchFetcher struct{}

func (f *VSwitchFetcher) ResourceType() string { return "vsw" }

func (f *VSwitchFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchVSwitchRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("vsw region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchVSwitchRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type vswResponse struct {
	VSwitches struct {
		VSwitch []vswInstance `json:"VSwitch"`
	} `json:"VSwitches"`
	TotalCount int `json:"TotalCount"`
}

type vswInstance struct {
	VSwitchID    string `json:"VSwitchId"`
	VSwitchName  string `json:"VSwitchName"`
	CidrBlock    string `json:"CidrBlock"`
	Status       string `json:"Status"`
	ZoneID       string `json:"ZoneId"`
	VpcID        string `json:"VpcId"`
	CreationTime string `json:"CreationTime"`
}

// vswPageSize is the page size requested per DescribeVSwitches call. The
// aliyun API defaults to 10 per page, so without paging accounts with more
// than 10 VSwitches in a region would silently lose results.
const vswPageSize = 50

func fetchVSwitchRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"vpc", "DescribeVSwitches",
			"--PageSize", fmt.Sprintf("%d", vswPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp vswResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse vsw response: %w", err)
		}

		for _, v := range resp.VSwitches.VSwitch {
			rawJSON, _ := json.Marshal(v)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "vsw",
				Region:       region,
				ResourceID:   v.VSwitchID,
				ResourceName: v.VSwitchName,
				Status:       v.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.VSwitches.VSwitch) < vswPageSize {
			break
		}
	}

	return allResources, nil
}
