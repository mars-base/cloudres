package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type RDSFetcher struct{}

func (f *RDSFetcher) ResourceType() string { return "rds" }

func (f *RDSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchRDSProfile)
}

type rdsResponse struct {
	Instances  []rdsInstance `json:"instances"`
	TotalCount int           `json:"total_count"`
}

type rdsInstance struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	FlavorRef string `json:"flavor_ref"`
	Datastore struct {
		Type    string `json:"type"`
		Version string `json:"version"`
	} `json:"datastore"`
	Region     string   `json:"region"`
	VpcID      string   `json:"vpc_id"`
	SubnetID   string   `json:"subnet_id"`
	PrivateIPs []string `json:"private_ips"`
	PublicIPs  []string `json:"public_ips"`
	Port       int      `json:"port"`
	Volume     struct {
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"volume"`
	CPU   string `json:"cpu"`
	Mem   string `json:"mem"`
	Nodes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
		Zone string `json:"availability_zone"`
	} `json:"nodes"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

const rdsPageSize = 100

func fetchRDSProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for offset := 0; ; offset += rdsPageSize {
		args := []string{"RDS", "ListInstances", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", rdsPageSize),
			"--offset=" + fmt.Sprintf("%d", offset),
		}
		if region != "" {
			args = append(args, "--cli-region="+region)
		}

		out, err := runHuawei(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp rdsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse rds response: %w", err)
		}

		for _, inst := range resp.Instances {
			rawJSON, _ := json.Marshal(inst)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "rds",
				Region:       region,
				ResourceID:   inst.ID,
				ResourceName: inst.Name,
				Status:       inst.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Instances) < rdsPageSize {
			break
		}
	}

	return allResources, nil
}
