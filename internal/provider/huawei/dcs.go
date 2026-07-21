package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type DCSFetcher struct{}

func (f *DCSFetcher) ResourceType() string { return "dcs" }

func (f *DCSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchDCSProfile)
}

type dcsResponse struct {
	Instances   []dcsInstance `json:"instances"`
	InstanceNum int           `json:"instance_num"`
}

type dcsInstance struct {
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"name"`
	Status       string `json:"status"`
	Engine       string `json:"engine"`
	EngineVer    string `json:"engine_version"`
	Capacity     int    `json:"capacity"`
	MaxMemory    int    `json:"max_memory"`
	UsedMemory   int    `json:"used_memory"`
	IP           string `json:"ip"`
	Port         int    `json:"port"`
	VpcID        string `json:"vpc_id"`
	SubnetID     string `json:"subnet_id"`
	Zone         string `json:"zone"`
	ChargeMode   string `json:"charging_mode"`
	CreatedAt    string `json:"created_at"`
}

const dcsPageSize = 100

func fetchDCSProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for offset := 0; ; offset += dcsPageSize {
		args := []string{"DCS", "ListInstances", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", dcsPageSize),
			"--offset=" + fmt.Sprintf("%d", offset),
		}
		if region != "" {
			args = append(args, "--cli-region="+region)
		}

		out, err := runHuawei(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp dcsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse dcs response: %w", err)
		}

		for _, inst := range resp.Instances {
			rawJSON, _ := json.Marshal(inst)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "dcs",
				Region:       region,
				ResourceID:   inst.InstanceID,
				ResourceName: inst.InstanceName,
				Status:       inst.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Instances) < dcsPageSize {
			break
		}
	}

	return allResources, nil
}
