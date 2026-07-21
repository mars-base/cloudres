package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type EIPFetcher struct{}

func (f *EIPFetcher) ResourceType() string { return "eip" }

func (f *EIPFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchEIPProfile)
}

type eipResponse struct {
	Publicips []eipItem `json:"publicips"`
}

type eipItem struct {
	ID                string `json:"id"`
	PublicIPAddress   string `json:"public_ip_address"`
	PublicIPv6Address string `json:"public_ipv6_address"`
	Status            string `json:"status"`
	Type              string `json:"type"`
	IPVersion         int    `json:"ip_version"`
	Description       string `json:"description"`
	CreatedAt         string `json:"created_at"`
	Bandwidth         struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Size       int    `json:"size"`
		ShareType  string `json:"share_type"`
		ChargeMode string `json:"charge_mode"`
	} `json:"bandwidth"`
	Vnic struct {
		PrivateIPAddress string `json:"private_ip_address"`
		DeviceID         string `json:"device_id"`
		VpcID            string `json:"vpc_id"`
		PortID           string `json:"port_id"`
		InstanceID       string `json:"instance_id"`
		InstanceType     string `json:"instance_type"`
	} `json:"vnic"`
}

const eipPageSize = 200

func fetchEIPProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	marker := ""

	for {
		args := []string{"EIP", "ListPublicips", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", eipPageSize),
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

		var resp eipResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse eip response: %w", err)
		}

		for _, e := range resp.Publicips {
			rawJSON, _ := json.Marshal(e)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "eip",
				Region:       region,
				ResourceID:   e.ID,
				ResourceName: e.PublicIPAddress,
				Status:       e.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Publicips) < eipPageSize {
			break
		}
		if len(resp.Publicips) > 0 {
			marker = resp.Publicips[len(resp.Publicips)-1].ID
		} else {
			break
		}
	}

	return allResources, nil
}
