package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type SubnetFetcher struct{}

func (f *SubnetFetcher) ResourceType() string { return "subnet" }

func (f *SubnetFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchSubnetProfile)
}

type subnetResponse struct {
	Subnets []subnetItem `json:"subnets"`
}

type subnetItem struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Cidr             string `json:"cidr"`
	Status           string `json:"status"`
	VpcID            string `json:"vpc_id"`
	GatewayIP        string `json:"gateway_ip"`
	DhcpEnable       bool   `json:"dhcp_enable"`
	AvailabilityZone string `json:"availability_zone"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

const subnetPageSize = 200

func fetchSubnetProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	marker := ""

	for {
		args := []string{"VPC", "ListSubnets", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", subnetPageSize),
		}
		if region != "" {
			args = append(args, "--cli-region="+region)
		}
		if marker != "" {
			args = append(args, "--marker="+marker)
		}

		out, err := runHuawei(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp subnetResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse subnet response: %w", err)
		}

		for _, s := range resp.Subnets {
			rawJSON, _ := json.Marshal(s)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "subnet",
				Region:       region,
				ResourceID:   s.ID,
				ResourceName: s.Name,
				Status:       s.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Subnets) < subnetPageSize {
			break
		}
		if len(resp.Subnets) > 0 {
			marker = resp.Subnets[len(resp.Subnets)-1].ID
		} else {
			break
		}
	}

	return allResources, nil
}
