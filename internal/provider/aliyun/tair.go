package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// TairFetcher fetches Tair (Redis-compatible) instances via the r-kvstore API.
type TairFetcher struct{}

func (f *TairFetcher) ResourceType() string { return "tair" }

func (f *TairFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchTairRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("tair region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchTairRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type tairResponse struct {
	Instances struct {
		KVStoreInstance []tairInstance `json:"KVStoreInstance"`
	} `json:"Instances"`
	TotalCount int `json:"TotalCount"`
}

type tairInstance struct {
	InstanceID     string `json:"InstanceId"`
	InstanceName   string `json:"InstanceName"`
	InstanceStatus string `json:"InstanceStatus"`
	InstanceType   string `json:"InstanceType"`
	EditionType    string `json:"EditionType"`
	EngineVersion  string `json:"EngineVersion"`
	InstanceClass  string `json:"InstanceClass"`
	RegionID       string `json:"RegionId"`
	ZoneID         string `json:"ZoneId"`
	VpcID          string `json:"VpcId"`
	VSwitchID      string `json:"VSwitchId"`
	ChargeType     string `json:"ChargeType"`
	CreateTime     string `json:"CreateTime"`
	EndTime        string `json:"EndTime"`
}

// tairPageSize is the page size requested per DescribeInstances call. The
// r-kvstore API defaults to 30 per page (max 50), so without paging
// accounts with more instances than that would silently lose results.
const tairPageSize = 50

func fetchTairRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"r-kvstore", "DescribeInstances",
			"--PageSize", fmt.Sprintf("%d", tairPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp tairResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse tair response: %w", err)
		}

		for _, inst := range resp.Instances.KVStoreInstance {
			rawJSON, _ := json.Marshal(inst)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "tair",
				Region:       inst.RegionID,
				ResourceID:   inst.InstanceID,
				ResourceName: inst.InstanceName,
				Status:       inst.InstanceStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.Instances.KVStoreInstance) < tairPageSize {
			break
		}
	}

	return allResources, nil
}
