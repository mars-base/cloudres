package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type ESSFetcher struct{}

func (f *ESSFetcher) ResourceType() string { return "ess" }

func (f *ESSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchESSRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("ess region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchESSRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type essResponse struct {
	ScalingGroups struct {
		ScalingGroup []essInstance `json:"ScalingGroup"`
	} `json:"ScalingGroups"`
	TotalCount int `json:"TotalCount"`
}

type essInstance struct {
	ScalingGroupId   string `json:"ScalingGroupId"`
	ScalingGroupName string `json:"ScalingGroupName"`
	LifecycleState   string `json:"LifecycleState"`
	ActiveCapacity   int    `json:"ActiveCapacity"`
	MaxSize          int    `json:"MaxSize"`
	MinSize          int    `json:"MinSize"`
	GroupType        string `json:"GroupType"`
	RegionId         string `json:"RegionId"`
	CreationTime     string `json:"CreationTime"`
	ResourceGroupId  string `json:"ResourceGroupId"`
}

const essPageSize = 50

func fetchESSRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"ess", "DescribeScalingGroups",
			"--PageSize", fmt.Sprintf("%d", essPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp essResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse ess response: %w", err)
		}

		for _, sg := range resp.ScalingGroups.ScalingGroup {
			rawJSON, _ := json.Marshal(sg)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ess",
				Region:       sg.RegionId,
				ResourceID:   sg.ScalingGroupId,
				ResourceName: sg.ScalingGroupName,
				Status:       sg.LifecycleState,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.ScalingGroups.ScalingGroup) < essPageSize {
			break
		}
	}

	return allResources, nil
}
