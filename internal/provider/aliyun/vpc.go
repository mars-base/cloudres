package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type VPCFetcher struct{}

func (f *VPCFetcher) ResourceType() string { return "vpc" }

func (f *VPCFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchVPCRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("vpc region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchVPCRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type vpcResponse struct {
	Vpcs struct {
		VPC []vpcInstance `json:"Vpc"`
	} `json:"Vpcs"`
	TotalCount int `json:"TotalCount"`
}

type vpcInstance struct {
	VpcID        string `json:"VpcId"`
	VpcName      string `json:"VpcName"`
	CidrBlock    string `json:"CidrBlock"`
	Status       string `json:"Status"`
	RegionID     string `json:"RegionId"`
	CreationTime string `json:"CreationTime"`
}

// vpcPageSize is the page size requested per DescribeVpcs call. The aliyun
// API defaults to 10 per page, so without paging accounts with more than
// 10 VPCs in a region would silently lose results.
const vpcPageSize = 50

func fetchVPCRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"vpc", "DescribeVpcs",
			"--PageSize", fmt.Sprintf("%d", vpcPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp vpcResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse vpc response: %w", err)
		}

		for _, v := range resp.Vpcs.VPC {
			rawJSON, _ := json.Marshal(v)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "vpc",
				Region:       v.RegionID,
				ResourceID:   v.VpcID,
				ResourceName: v.VpcName,
				Status:       v.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.Vpcs.VPC) < vpcPageSize {
			break
		}
	}

	return allResources, nil
}
