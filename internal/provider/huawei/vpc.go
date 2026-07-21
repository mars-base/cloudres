package huawei

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
	return fetchByProfiles(ctx, p, fetchVPCProfile)
}

type vpcResponse struct {
	VPCs     []vpcItem `json:"vpcs"`
	PageInfo struct {
		PreviousMarker string `json:"previous_marker"`
		CurrentCount   int    `json:"current_count"`
	} `json:"page_info"`
}

type vpcItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Cidr        string `json:"cidr"`
	Status      string `json:"status"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

const vpcPageSize = 200

func fetchVPCProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	marker := ""

	for {
		args := []string{"VPC", "ListVpcs", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", vpcPageSize),
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

		var resp vpcResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse vpc response: %w", err)
		}

		for _, v := range resp.VPCs {
			rawJSON, _ := json.Marshal(v)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "vpc",
				Region:       region,
				ResourceID:   v.ID,
				ResourceName: v.Name,
				Status:       v.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if resp.PageInfo.CurrentCount < vpcPageSize || resp.PageInfo.PreviousMarker == marker {
			break
		}
		marker = resp.PageInfo.PreviousMarker
	}

	return allResources, nil
}
