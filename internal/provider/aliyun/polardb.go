package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// PolarDBFetcher fetches PolarDB clusters.
type PolarDBFetcher struct{}

func (f *PolarDBFetcher) ResourceType() string { return "pdb" }

func (f *PolarDBFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchPolarDBRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("pdb region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	if len(p.Regions) == 0 {
		resources, err := fetchPolarDBRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type polarDBResponse struct {
	Items struct {
		DBCluster []polarDBCluster `json:"DBCluster"`
	} `json:"Items"`
	TotalRecordCount int `json:"TotalRecordCount"`
}

type polarDBCluster struct {
	DBClusterID          string `json:"DBClusterId"`
	DBClusterDescription string `json:"DBClusterDescription"`
	DBClusterStatus      string `json:"DBClusterStatus"`
	Engine               string `json:"Engine"`
	DBVersion            string `json:"DBVersion"`
	DBNodeClass          string `json:"DBNodeClass"`
	DBNodeNumber         string `json:"DBNodeNumber"`
	PayType              string `json:"PayType"`
	RegionID             string `json:"RegionId"`
	CreateTime           string `json:"CreateTime"`
	ExpireTime           string `json:"ExpireTime"`
	StorageUsed          int64  `json:"StorageUsed"`
	StorageSpace         int64  `json:"StorageSpace"`
	StorageType          string `json:"StorageType"`
	// StoragePayType tells whether StorageSpace is meaningful: it's the
	// prepaid (包年包月) purchased capacity when "Prepaid", but for
	// "Postpaid" (按量付费/弹性存储) clusters StorageSpace isn't a real cap —
	// StorageUsed can legitimately exceed it.
	StoragePayType string `json:"StoragePayType"`
}

// polarDBPageSize is the page size requested per DescribeDBClusters call.
// The polardb API accepts 30/50/100 per page; without paging accounts
// with more clusters than the default page size would silently lose
// results.
const polarDBPageSize = 100

func fetchPolarDBRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"polardb", "DescribeDBClusters",
			"--PageSize", fmt.Sprintf("%d", polarDBPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp polarDBResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse polardb response: %w", err)
		}

		for _, c := range resp.Items.DBCluster {
			rawJSON, _ := json.Marshal(c)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "pdb",
				Region:       c.RegionID,
				ResourceID:   c.DBClusterID,
				ResourceName: c.DBClusterDescription,
				Status:       c.DBClusterStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalRecordCount || len(resp.Items.DBCluster) < polarDBPageSize {
			break
		}
	}

	return allResources, nil
}
