package aliyun

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// ACK Clusters (DescribeClusters + DescribeClusterNodes)
// ---------------------------------------------------------------------------

type ACKFetcher struct{}

func (f *ACKFetcher) ResourceType() string { return "ack" }

func (f *ACKFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchACKClusters(ctx, p)
}

type ackClusterListResponse []ackCluster

type ackCluster struct {
	ClusterID      string `json:"cluster_id"`
	Name           string `json:"name"`
	ClusterType    string `json:"cluster_type"`
	ClusterSpec    string `json:"cluster_spec"`
	State          string `json:"state"`
	CurrentVersion string `json:"current_version"`
	RegionID       string `json:"region_id"`
	Size           int    `json:"size"`
	Created        string `json:"created"`
}

type ackNodeListResponse struct {
	Nodes []ackNode `json:"nodes"`
}

type ackNode struct {
	NodeName     string `json:"node_name"`
	InstanceID   string `json:"instance_id"`
	IPAddress    []string `json:"ip_address"`
	NodeStatus   string `json:"node_status"`
	State        string `json:"state"`
	NodepoolID   string `json:"nodepool_id"`
	InstanceRole string `json:"instance_role"`
}

func fetchACKClusters(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	regions := p.ProfileRegions[p.ActiveProfile]
	if len(regions) == 0 {
		regions = []string{p.Regions[0]}
	}

	for _, region := range regions {
		args := []string{"cs", "DescribeClusters", "--region", region}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			continue
		}

		var clusters ackClusterListResponse
		if err := json.Unmarshal(out, &clusters); err != nil {
			continue
		}

		for _, c := range clusters {
			// Fetch node count for this cluster
			nodeCount := fetchACKNodeCount(ctx, c.ClusterID, p.ActiveProfile)
			c.Size = nodeCount

			rawJSON, _ := json.Marshal(c)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ack",
				Region:       c.RegionID,
				ResourceID:   c.ClusterID,
				ResourceName: c.Name,
				Status:       c.State,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}
	}

	return allResources, nil
}

func fetchACKNodeCount(ctx context.Context, clusterID, profile string) int {
	args := []string{"cs", "DescribeClusterNodes", "--ClusterId", clusterID}
	out, err := runAliyun(ctx, args, profile)
	if err != nil {
		return 0
	}

	var resp ackNodeListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0
	}

	return len(resp.Nodes)
}
