package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type ECSFetcher struct{}

func (f *ECSFetcher) ResourceType() string { return "ecs" }

func (f *ECSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchECSProfile)
}

type ecsResponse struct {
	Servers []ecsServer `json:"servers"`
	Count   int         `json:"count"`
}

type ecsServer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Flavor struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Vcpus string `json:"vcpus"`
		RAM   string `json:"ram"`
		Disk  string `json:"disk"`
	} `json:"flavor"`
	Addresses map[string][]struct {
		Version string `json:"version"`
		Addr    string `json:"addr"`
		Type    string `json:"OS-EXT-IPS:type"`
	} `json:"addresses"`
	Metadata struct {
		OsType string `json:"os_type"`
		VpcID  string `json:"vpc_id"`
	} `json:"metadata"`
	AvailabilityZone string   `json:"OS-EXT-AZ:availability_zone"`
	KeyName          string   `json:"key_name"`
	Created          string   `json:"created"`
	Tags             []string `json:"tags"`
	Description      string   `json:"description"`
}

const ecsPageSize = 100

func fetchECSProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for offset := 0; ; offset += ecsPageSize {
		args := []string{"ECS", "ListServersDetails", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", ecsPageSize),
			"--offset=" + fmt.Sprintf("%d", offset),
		}
		if region != "" {
			args = append(args, "--cli-region="+region)
		}

		out, err := runHuawei(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp ecsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse ecs response: %w", err)
		}

		for _, s := range resp.Servers {
			rawJSON, _ := json.Marshal(s)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "ecs",
				Region:       region,
				ResourceID:   s.ID,
				ResourceName: s.Name,
				Status:       s.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Servers) < ecsPageSize {
			break
		}
	}

	return allResources, nil
}
