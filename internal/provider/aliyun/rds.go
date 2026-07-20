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
	TotalRecordCount int `json:"TotalRecordCount"`
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

// rdsResourceUsage mirrors the fields we care about from
// DescribeResourceUsage — actual space usage, as opposed to the allocated
// storage size returned by the instance list/attribute APIs.
type rdsResourceUsage struct {
	DataSize   int64 `json:"DataSize"`
	DiskUsed   int64 `json:"DiskUsed"`
	BackupSize int64 `json:"BackupSize"`
	LogSize    int64 `json:"LogSize"`
}

// rdsPageSize is the page size requested per DescribeDBInstances call.
// Without paging, accounts with more instances than the API's default
// page size would silently lose results.
const rdsPageSize = 100

func fetchRDSRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"rds", "DescribeDBInstances",
			"--PageSize", fmt.Sprintf("%d", rdsPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
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

		for _, inst := range resp.Items.DBInstance {
			usage, err := fetchRDSResourceUsage(ctx, p, inst.DBInstanceID)
			if err != nil {
				// Usage is best-effort: don't fail the whole sync over a
				// single instance's usage lookup (e.g. transient API error).
				usage = &rdsResourceUsage{}
			}

			rawJSON, _ := json.Marshal(struct {
				rdsInstance
				DataSize   int64 `json:"DataSize"`
				DiskUsed   int64 `json:"DiskUsed"`
				BackupSize int64 `json:"BackupSize"`
			}{
				rdsInstance: inst,
				DataSize:    usage.DataSize,
				DiskUsed:    usage.DiskUsed,
				BackupSize:  usage.BackupSize,
			})
			allResources = append(allResources, core.Resource{
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

		if len(allResources) >= resp.TotalRecordCount || len(resp.Items.DBInstance) < rdsPageSize {
			break
		}
	}

	return allResources, nil
}

// fetchRDSResourceUsage calls DescribeResourceUsage for a single instance.
// There's no batch/list variant of this API, so it's one extra CLI call
// per RDS instance — acceptable given RDS instance counts are typically
// small, but keep an eye on this if that ever changes.
func fetchRDSResourceUsage(ctx context.Context, p *core.Provider, instanceID string) (*rdsResourceUsage, error) {
	out, err := runAliyun(ctx, []string{"rds", "DescribeResourceUsage", "--DBInstanceId", instanceID}, p.ActiveProfile)
	if err != nil {
		return nil, err
	}
	var usage rdsResourceUsage
	if err := json.Unmarshal(out, &usage); err != nil {
		return nil, fmt.Errorf("parse rds resource usage: %w", err)
	}
	return &usage, nil
}
