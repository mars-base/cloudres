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

// polarDBEndpoints holds the connection strings we care about from
// DescribeDBClusterEndpoints — the cluster's primary (writer) address and
// its default cluster (read/write-split) address, if configured.
type polarDBEndpoints struct {
	PrimaryEndpoint       string
	PrimaryEndpointPublic string
	ClusterEndpoint       string
}

// polarDBNode holds per-node specs extracted from DescribeDBClusterAttribute.
// DBNodes[] in the response contains one entry per node — Writer nodes and
// Reader nodes — each with its own CPU, memory, class, and zone.
type polarDBNode struct {
	DBNodeRole  string `json:"DBNodeRole"`
	CpuCores    string `json:"CpuCores"`
	MemorySize  string `json:"MemorySize"`
	DBNodeClass string `json:"DBNodeClass"`
	ZoneId      string `json:"ZoneId"`
}

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
			endpoints, err := fetchPolarDBEndpoints(ctx, p, c.DBClusterID)
			if err != nil {
				// Best-effort: don't fail the whole sync over a single
				// cluster's endpoint lookup (e.g. transient API error).
				endpoints = &polarDBEndpoints{}
			}

			nodes, err := fetchPolarDBNodes(ctx, p, c.DBClusterID)
			if err != nil {
				// Best-effort, same as endpoints.
				nodes = nil
			}

			rawJSON, _ := json.Marshal(struct {
				polarDBCluster
				PrimaryEndpoint       string         `json:"PrimaryEndpoint"`
				PrimaryEndpointPublic string         `json:"PrimaryEndpointPublic"`
				ClusterEndpoint       string         `json:"ClusterEndpoint"`
				DBNodes               []polarDBNode  `json:"DBNodes"`
			}{
				polarDBCluster:        c,
				PrimaryEndpoint:       endpoints.PrimaryEndpoint,
				PrimaryEndpointPublic: endpoints.PrimaryEndpointPublic,
				ClusterEndpoint:       endpoints.ClusterEndpoint,
				DBNodes:               nodes,
			})
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

// fetchPolarDBEndpoints calls DescribeDBClusterEndpoints for a single
// cluster. There's no batch/list variant of this API, so it's one extra CLI
// call per cluster — same tradeoff as RDS's DescribeResourceUsage.
func fetchPolarDBEndpoints(ctx context.Context, p *core.Provider, clusterID string) (*polarDBEndpoints, error) {
	out, err := runAliyun(ctx, []string{"polardb", "DescribeDBClusterEndpoints", "--DBClusterId", clusterID}, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Items []struct {
			EndpointType string `json:"EndpointType"`
			AddressItems []struct {
				ConnectionString string `json:"ConnectionString"`
				NetType          string `json:"NetType"`
			} `json:"AddressItems"`
		} `json:"Items"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse polardb endpoints response: %w", err)
	}

	var eps polarDBEndpoints
	for _, item := range resp.Items {
		for _, addr := range item.AddressItems {
			switch {
			case item.EndpointType == "Primary" && addr.NetType == "Private":
				eps.PrimaryEndpoint = addr.ConnectionString
			case item.EndpointType == "Primary" && addr.NetType == "Public":
				eps.PrimaryEndpointPublic = addr.ConnectionString
			case item.EndpointType == "Cluster" && addr.NetType == "Private":
				eps.ClusterEndpoint = addr.ConnectionString
			}
		}
	}
	return &eps, nil
}

// fetchPolarDBNodes calls DescribeDBClusterAttribute for a single cluster to
// extract per-node specs (role, CPU, memory, class, zone). Like
// fetchPolarDBEndpoints, there's no batch variant — one extra CLI call per
// cluster.
func fetchPolarDBNodes(ctx context.Context, p *core.Provider, clusterID string) ([]polarDBNode, error) {
	out, err := runAliyun(ctx, []string{"polardb", "DescribeDBClusterAttribute", "--DBClusterId", clusterID}, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp struct {
		DBNodes []polarDBNode `json:"DBNodes"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse polardb attribute response: %w", err)
	}
	return resp.DBNodes, nil
}
