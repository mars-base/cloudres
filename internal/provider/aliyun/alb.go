package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type ALBFetcher struct{}

func (f *ALBFetcher) ResourceType() string { return "alb" }

func (f *ALBFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchALBRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("alb region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchALBRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type albResponse struct {
	LoadBalancers []albInstance `json:"LoadBalancers"`
	NextToken     string        `json:"NextToken"`
	MaxResults    int           `json:"MaxResults"`
	TotalCount    int           `json:"TotalCount"`
}

type albInstance struct {
	LoadBalancerId     string `json:"LoadBalancerId"`
	LoadBalancerName   string `json:"LoadBalancerName"`
	DNSName            string `json:"DNSName"`
	AddressType        string `json:"AddressType"`
	LoadBalancerStatus string `json:"LoadBalancerStatus"`
	LoadBalancerEdition string `json:"LoadBalancerEdition"`
	VpcId              string `json:"VpcId"`
	ResourceGroupId    string `json:"ResourceGroupId"`
	CreateTime         string `json:"CreateTime"`
}

const albPageSize = 50

func fetchALBRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	nextToken := ""

	for {
		args := []string{"alb", "ListLoadBalancers",
			"--MaxResults", fmt.Sprintf("%d", albPageSize),
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

		var resp albResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse alb response: %w", err)
		}

		for _, lb := range resp.LoadBalancers {
			rawJSON, _ := json.Marshal(lb)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "alb",
				Region:       region,
				ResourceID:   lb.LoadBalancerId,
				ResourceName: lb.LoadBalancerName,
				Status:       lb.LoadBalancerStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if resp.NextToken == "" || len(resp.LoadBalancers) < albPageSize {
			break
		}
		nextToken = resp.NextToken
	}

	return allResources, nil
}
