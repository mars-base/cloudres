package aliyun

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// ARMS Prometheus (ListPrometheusInstances / GetPrometheusInstance)
//
// Both APIs are legacy (2019-08-08, kebab-case). The list API returns its
// instances as a JSON *string* in Data, so the payload needs a second
// unmarshal; the detail API returns Data as a normal object.
// ---------------------------------------------------------------------------

const promAPIVersion = "2019-08-08"

type PrometheusFetcher struct{}

func (f *PrometheusFetcher) ResourceType() string { return "prom" }

func (f *PrometheusFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchPrometheus(ctx, p)
}

type promInstance struct {
	AccessType                 string `json:"accessType"`
	AgentStatus                string `json:"agentStatus"`
	ClusterID                  string `json:"clusterId"`
	ClusterName                string `json:"clusterName"`
	ClusterType                string `json:"clusterType"`
	ControllerID               string `json:"controllerId"`
	CreateTime                 int64  `json:"createTime"`
	FreeInstance               bool   `json:"freeInstance"`
	GbFee                      bool   `json:"gbFee"`
	GrafanaInstanceID          string `json:"grafanaInstanceId"`
	ID                         int64  `json:"id"`
	IsAdvancedClusterInstalled bool   `json:"isAdvancedClusterInstalled"`
	IsClusterRunning           bool   `json:"isClusterRunning"`
	IsControllerInstalled      bool   `json:"isControllerInstalled"`
	IsIntegrationCenter        bool   `json:"isIntegrationCenter"`
	MetricStoreEngine          bool   `json:"metricStoreEngine"`
	PayMethod                  string `json:"payMethod"`
	Product                    string `json:"product"`
	RegionID                   string `json:"regionId"`
	SlsProject                 string `json:"slsProject"`
	SlsStore                   string `json:"slsStore"`
	StoreEngine                string `json:"storeEngine"`
	UpdateTime                 int64  `json:"updateTime"`
	UserID                     string `json:"userId"`
	Workspace                  string `json:"workspace"`
}

type promDetail struct {
	AccessType            string   `json:"AccessType"`
	ArchiveDuration       int      `json:"ArchiveDuration"`
	ClusterID             string   `json:"ClusterId"`
	ClusterName           string   `json:"ClusterName"`
	ClusterType           string   `json:"ClusterType"`
	DBInstanceStatus      string   `json:"DbInstanceStatus"`
	EnableAuthFreeRead    bool     `json:"EnableAuthFreeRead"`
	EnableAuthFreeWrite   bool     `json:"EnableAuthFreeWrite"`
	EnableAuthToken       bool     `json:"EnableAuthToken"`
	GrafanaInstanceID     string   `json:"GrafanaInstanceId"`
	HTTPAPIInterURL       string   `json:"HttpApiInterUrl"`
	HTTPAPIIntraURL       string   `json:"HttpApiIntraUrl"`
	OpenTelemetryInterURL string   `json:"OpenTelemetryInterUrl"`
	OpenTelemetryIntraURL string   `json:"OpenTelemetryIntraUrl"`
	PaymentType           string   `json:"PaymentType"`
	PushGatewayInterURL   string   `json:"PushGatewayInterUrl"`
	PushGatewayIntraURL   string   `json:"PushGatewayIntraUrl"`
	RegionID              string   `json:"RegionId"`
	RemoteReadInterURL    string   `json:"RemoteReadInterUrl"`
	RemoteReadIntraURL    string   `json:"RemoteReadIntraUrl"`
	RemoteWriteInterURL   string   `json:"RemoteWriteInterUrl"`
	RemoteWriteIntraURL   string   `json:"RemoteWriteIntraUrl"`
	ResourceGroupID       string   `json:"ResourceGroupId"`
	ResourceType          string   `json:"ResourceType"`
	StorageDuration       int      `json:"StorageDuration"`
	SupportAuthTypes      []string `json:"SupportAuthTypes"`
	UserID                string   `json:"UserId"`
	Version               string   `json:"Version"`
}

type promEnriched struct {
	promInstance
	Detail *promDetail `json:"detail,omitempty"`
}

func fetchPrometheus(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var all []core.Resource
	now := time.Now()

	regions := p.ProfileRegions[p.ActiveProfile]
	if len(regions) == 0 && len(p.Regions) > 0 {
		regions = []string{p.Regions[0]}
	}

	for _, region := range regions {
		all = append(all, fetchPrometheusRegion(ctx, p.ActiveProfile, region, now)...)
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].ResourceName != all[j].ResourceName {
			return all[i].ResourceName < all[j].ResourceName
		}
		return all[i].ResourceID < all[j].ResourceID
	})

	return all, nil
}

func fetchPrometheusRegion(ctx context.Context, profile, region string, now time.Time) []core.Resource {
	out, err := runAliyun(ctx, []string{
		"arms", "list-prometheus-instances",
		"--biz-region-id", region,
		"--show-global-view", "false",
		"--api-version", promAPIVersion,
		"--region", region,
	}, profile)
	if err != nil {
		return nil
	}

	var resp struct {
		Data string `json:"Data"`
	}
	if json.Unmarshal(out, &resp) != nil || resp.Data == "" {
		return nil
	}

	var items []promInstance
	if json.Unmarshal([]byte(resp.Data), &items) != nil {
		return nil
	}

	var resources []core.Resource
	for _, inst := range items {
		enriched := promEnriched{promInstance: inst}
		enriched.Detail = fetchPrometheusDetail(ctx, profile, region, inst.ClusterID)

		rawJSON, _ := json.Marshal(enriched)

		name := inst.ClusterName
		if name == "" {
			name = inst.ClusterID
		}
		regionID := inst.RegionID
		if regionID == "" {
			regionID = region
		}
		resources = append(resources, core.Resource{
			Provider:     "aliyun",
			ResourceType: "prom",
			Region:       regionID,
			ResourceID:   inst.ClusterID,
			ResourceName: name,
			Status:       promStatus(inst, enriched.Detail),
			RawJSON:      string(rawJSON),
			SyncedAt:     now,
		})
	}

	return resources
}

func fetchPrometheusDetail(ctx context.Context, profile, region, clusterID string) *promDetail {
	out, err := runAliyun(ctx, []string{
		"arms", "get-prometheus-instance",
		"--biz-region-id", region,
		"--cluster-id", clusterID,
		"--api-version", promAPIVersion,
		"--region", region,
	}, profile)
	if err != nil {
		return nil
	}

	var resp struct {
		Data promDetail `json:"Data"`
	}
	if json.Unmarshal(out, &resp) != nil {
		return nil
	}
	return &resp.Data
}

func promStatus(inst promInstance, detail *promDetail) string {
	if detail != nil && detail.DBInstanceStatus != "" {
		return detail.DBInstanceStatus
	}
	if inst.IsClusterRunning {
		return "Running"
	}
	return "Stopped"
}
