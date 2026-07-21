package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type NLBFetcher struct{}

func (f *NLBFetcher) ResourceType() string { return "nlb" }

func (f *NLBFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchNLBRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("nlb region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchNLBRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type nlbResponse struct {
	LoadBalancers []nlbInstance `json:"LoadBalancers"`
	NextToken     string        `json:"NextToken"`
	MaxResults    int           `json:"MaxResults"`
	TotalCount    int           `json:"TotalCount"`
}

type nlbInstance struct {
	LoadBalancerId     string `json:"LoadBalancerId"`
	LoadBalancerName   string `json:"LoadBalancerName"`
	DNSName            string `json:"DNSName"`
	AddressType        string `json:"AddressType"`
	LoadBalancerStatus string `json:"LoadBalancerStatus"`
	LoadBalancerType   string `json:"LoadBalancerType"`
	VpcId              string `json:"VpcId"`
	RegionId           string `json:"RegionId"`
	ResourceGroupId    string `json:"ResourceGroupId"`
	CreateTime         string `json:"CreateTime"`
}

const nlbPageSize = 50

func fetchNLBRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	nextToken := ""

	for {
		args := []string{"nlb", "ListLoadBalancers",
			"--MaxResults", fmt.Sprintf("%d", nlbPageSize),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}
		if nextToken != "" {
			args = append(args, "--NextToken", nextToken)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp nlbResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse nlb response: %w", err)
		}

		for _, lb := range resp.LoadBalancers {
			rawJSON, _ := json.Marshal(lb)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "nlb",
				Region:       lb.RegionId,
				ResourceID:   lb.LoadBalancerId,
				ResourceName: lb.LoadBalancerName,
				Status:       lb.LoadBalancerStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if resp.NextToken == "" || len(resp.LoadBalancers) < nlbPageSize {
			break
		}
		nextToken = resp.NextToken
	}

	return allResources, nil
}
