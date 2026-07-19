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
}

type vpcInstance struct {
	VpcID        string `json:"VpcId"`
	VpcName      string `json:"VpcName"`
	CidrBlock    string `json:"CidrBlock"`
	Status       string `json:"Status"`
	RegionID     string `json:"RegionId"`
	CreationTime string `json:"CreationTime"`
}

func fetchVPCRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	args := []string{"vpc", "DescribeVpcs"}
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

	now := time.Now()
	resources := make([]core.Resource, 0, len(resp.Vpcs.VPC))
	for _, v := range resp.Vpcs.VPC {
		rawJSON, _ := json.Marshal(v)
		resources = append(resources, core.Resource{
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
	return resources, nil
}
