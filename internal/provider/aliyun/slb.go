package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type SLBFetcher struct{}

func (f *SLBFetcher) ResourceType() string { return "slb" }

func (f *SLBFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchSLBRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("slb region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchSLBRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type slbResponse struct {
	LoadBalancers struct {
		LoadBalancer []slbInstance `json:"LoadBalancer"`
	} `json:"LoadBalancers"`
	TotalCount int `json:"TotalCount"`
}

type slbInstance struct {
	LoadBalancerId     string `json:"LoadBalancerId"`
	LoadBalancerName   string `json:"LoadBalancerName"`
	Address            string `json:"Address"`
	AddressType        string `json:"AddressType"`
	NetworkType        string `json:"NetworkType"`
	LoadBalancerStatus string `json:"LoadBalancerStatus"`
	LoadBalancerSpec   string `json:"LoadBalancerSpec"`
	Bandwidth          int    `json:"Bandwidth"`
	VpcId              string `json:"VpcId"`
	VSwitchId          string `json:"VSwitchId"`
	RegionId           string `json:"RegionId"`
	MasterZoneId       string `json:"MasterZoneId"`
	SlaveZoneId        string `json:"SlaveZoneId"`
	CreateTime         string `json:"CreateTime"`
	PayType            string `json:"PayType"`
	ResourceGroupId    string `json:"ResourceGroupId"`
}

const slbPageSize = 50

func fetchSLBRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"slb", "DescribeLoadBalancers",
			"--PageSize", fmt.Sprintf("%d", slbPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp slbResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse slb response: %w", err)
		}

		for _, lb := range resp.LoadBalancers.LoadBalancer {
			rawJSON, _ := json.Marshal(lb)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "slb",
				Region:       lb.RegionId,
				ResourceID:   lb.LoadBalancerId,
				ResourceName: lb.LoadBalancerName,
				Status:       lb.LoadBalancerStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.LoadBalancers.LoadBalancer) < slbPageSize {
			break
		}
	}

	return allResources, nil
}
