package aliyun

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
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchRDSRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("rds region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchRDSRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type rdsResponse struct {
	Items struct {
		DBInstance []rdsInstance `json:"DBInstance"`
	} `json:"Items"`
}

type rdsInstance struct {
	DBInstanceID          string `json:"DBInstanceId"`
	DBInstanceDescription string `json:"DBInstanceDescription"`
	DBInstanceStatus      string `json:"DBInstanceStatus"`
	Engine                string `json:"Engine"`
	EngineVersion         string `json:"EngineVersion"`
	RegionID              string `json:"RegionId"`
	DBInstanceClass       string `json:"DBInstanceClass"`
	PayType               string `json:"PayType"`
	CreationTime          string `json:"CreateTime"`
}

func fetchRDSRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	args := []string{"rds", "DescribeDBInstances"}
	if region != "" {
		args = append(args, "--RegionId", region)
	}

	out, err := runAliyun(ctx, args, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp rdsResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse rds response: %w", err)
	}

	now := time.Now()
	resources := make([]core.Resource, 0, len(resp.Items.DBInstance))
	for _, inst := range resp.Items.DBInstance {
		rawJSON, _ := json.Marshal(inst)
		resources = append(resources, core.Resource{
			Provider:     "aliyun",
			ResourceType: "rds",
			Region:       inst.RegionID,
			ResourceID:   inst.DBInstanceID,
			ResourceName: inst.DBInstanceDescription,
			Status:       inst.DBInstanceStatus,
			RawJSON:      string(rawJSON),
			SyncedAt:     now,
		})
	}
	return resources, nil
}
