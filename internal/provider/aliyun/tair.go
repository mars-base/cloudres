package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// tairAttr holds per-instance specs from DescribeInstanceAttribute.
// These fields (shard count, real class, bandwidth, etc.) aren't
// included in the list response and require a per-instance call.
type tairAttr struct {
	ShardCount        int    `json:"ShardCount"`
	RealInstanceClass string `json:"RealInstanceClass"`
	Bandwidth         int    `json:"Bandwidth"`
	Connections       int64  `json:"Connections"`
	QPS               int64  `json:"QPS"`
}

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

// tairMemoryUsage mirrors the (only, non-configurable) default response of
// DescribeHistoryMonitorValues when MonitorKeys is omitted: used/quota
// memory in bytes, as strings with decimal fractions.
type tairMemoryUsage struct {
	UsedMemory  int64
	QuotaMemory int64
}

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
			usage, err := fetchTairMemoryUsage(ctx, p, inst.InstanceID)
			if err != nil {
				usage = &tairMemoryUsage{}
			}

			attr, err := fetchTairAttribute(ctx, p, inst.InstanceID)
			if err != nil {
				attr = &tairAttr{}
			}

			rawJSON, _ := json.Marshal(struct {
				tairInstance
				UsedMemory        int64  `json:"UsedMemory"`
				QuotaMemory       int64  `json:"QuotaMemory"`
				ShardCount        int    `json:"ShardCount"`
				RealInstanceClass string `json:"RealInstanceClass"`
				Bandwidth         int    `json:"Bandwidth"`
				Connections       int64  `json:"Connections"`
				QPS               int64  `json:"QPS"`
			}{
				tairInstance:      inst,
				UsedMemory:        usage.UsedMemory,
				QuotaMemory:       usage.QuotaMemory,
				ShardCount:        attr.ShardCount,
				RealInstanceClass: attr.RealInstanceClass,
				Bandwidth:         attr.Bandwidth,
				Connections:       attr.Connections,
				QPS:               attr.QPS,
			})
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

// fetchTairMemoryUsage calls DescribeHistoryMonitorValues for a single
// instance, requesting just the most recent data point in a narrow window
// (there's no "current usage" API — only historical monitoring — so we ask
// for the last few minutes and take the latest sample). Returns zero values
// if the instance has no recent monitoring data yet (e.g. newly created).
func fetchTairMemoryUsage(ctx context.Context, p *core.Provider, instanceID string) (*tairMemoryUsage, error) {
	now := time.Now().UTC()
	start := now.Add(-10 * time.Minute)
	args := []string{"r-kvstore", "DescribeHistoryMonitorValues",
		"--InstanceId", instanceID,
		"--StartTime", start.Format("2006-01-02T15:04:05Z"),
		"--EndTime", now.Format("2006-01-02T15:04:05Z"),
		"--IntervalForHistory", "01m",
	}
	out, err := runAliyun(ctx, args, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp struct {
		MonitorHistory string `json:"MonitorHistory"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse tair monitor response: %w", err)
	}
	if resp.MonitorHistory == "" {
		return &tairMemoryUsage{}, nil
	}

	// MonitorHistory is itself a JSON-encoded string, keyed by timestamp:
	// {"2026-07-20T06:47:55Z":{"UsedMemory":"6030825520","quotaMemory":"34359738368"}}
	var byTime map[string]struct {
		UsedMemory  string `json:"UsedMemory"`
		QuotaMemory string `json:"quotaMemory"`
	}
	if err := json.Unmarshal([]byte(resp.MonitorHistory), &byTime); err != nil {
		return nil, fmt.Errorf("parse tair monitor history: %w", err)
	}

	var latestKey string
	for k := range byTime {
		if k > latestKey {
			latestKey = k
		}
	}
	if latestKey == "" {
		return &tairMemoryUsage{}, nil
	}

	sample := byTime[latestKey]
	used, _ := strconv.ParseFloat(sample.UsedMemory, 64)
	quota, _ := strconv.ParseFloat(sample.QuotaMemory, 64)
	return &tairMemoryUsage{UsedMemory: int64(used), QuotaMemory: int64(quota)}, nil
}

// fetchTairAttribute calls DescribeInstanceAttribute for a single instance
// to extract shard count, real instance class, bandwidth, etc. Like other
// per-instance enrichment calls, this is best-effort.
func fetchTairAttribute(ctx context.Context, p *core.Provider, instanceID string) (*tairAttr, error) {
	out, err := runAliyun(ctx, []string{"r-kvstore", "DescribeInstanceAttribute", "--InstanceId", instanceID}, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Instances struct {
			DBInstanceAttribute []tairAttr `json:"DBInstanceAttribute"`
		} `json:"Instances"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse tair attribute response: %w", err)
	}
	if len(resp.Instances.DBInstanceAttribute) == 0 {
		return &tairAttr{}, nil
	}
	attr := resp.Instances.DBInstanceAttribute[0]
	return &attr, nil
}
