package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ── Prometheus (ARMS) ───────────────────────────────────────────────────

// promView decodes the RawJSON written by the ARMS Prometheus fetcher: the
// list-instance fields sit at the top level (camelCase), the detail call's
// object under "detail" (PascalCase).
type promView struct {
	// list item
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

	Detail *promDetailView `json:"detail"`
}

type promDetailView struct {
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

func (r Resource) parseProm() promView {
	var d promView
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return d
}

func promYN(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// promPayLabel prefers the detail call's PaymentType and falls back to the
// list call's PayMethod (cloud-product instances leave the latter empty).
func (d *promView) promPayLabel() string {
	if d.Detail != nil && d.Detail.PaymentType != "" {
		return d.Detail.PaymentType
	}
	if d.PayMethod != "" {
		return d.PayMethod
	}
	return "-"
}

func (d *promView) promGrafana() string {
	if d.Detail != nil && d.Detail.GrafanaInstanceID != "" {
		return d.Detail.GrafanaInstanceID
	}
	if d.GrafanaInstanceID != "" {
		return d.GrafanaInstanceID
	}
	return "-"
}

func (r Resource) promRow() []string {
	d := r.parseProm()
	return []string{
		r.ResourceID, r.ResourceName, r.Status,
		d.ClusterType, d.promPayLabel(), d.promGrafana(),
		r.Region, formatMillis(d.CreateTime),
	}
}

func (r Resource) promDetail() [][2]string {
	d := r.parseProm()

	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Region", r.Region},
		{"Status", r.Status},
		{"Type", d.ClusterType},
		{"Workspace", d.Workspace},
		{"Access Type", d.AccessType},
		{"Pay Method", d.promPayLabel()},
		{"Free Instance", promYN(d.FreeInstance)},
		{"GB Fee", promYN(d.GbFee)},
		{"Grafana", d.promGrafana()},
		{"Created", formatMillis(d.CreateTime)},
		{"Updated", formatMillis(d.UpdateTime)},
	}

	det := d.Detail
	if det == nil {
		return pairs
	}

	if det.ClusterType != "" {
		pairs = append(pairs, [2]string{"Detail Type", det.ClusterType})
	}
	if det.ResourceType != "" {
		pairs = append(pairs, [2]string{"Resource Type", det.ResourceType})
	}
	if det.Version != "" {
		pairs = append(pairs, [2]string{"Version", det.Version})
	}
	if det.StorageDuration > 0 {
		pairs = append(pairs, [2]string{"Storage", fmt.Sprintf("%d days", det.StorageDuration)})
	}
	if det.ArchiveDuration > 0 {
		pairs = append(pairs, [2]string{"Archive", fmt.Sprintf("%d days", det.ArchiveDuration)})
	}
	if det.ResourceGroupID != "" {
		pairs = append(pairs, [2]string{"Resource Group", det.ResourceGroupID})
	}
	if d.Product != "" {
		pairs = append(pairs, [2]string{"Product", d.Product})
	}
	if d.UserID != "" {
		pairs = append(pairs, [2]string{"User ID", d.UserID})
	}

	// Storage backends (SLS / metric store engine).
	if d.SlsProject != "" {
		pairs = append(pairs, [2]string{"SLS Project", d.SlsProject})
	}
	if d.SlsStore != "" {
		pairs = append(pairs, [2]string{"SLS Store", d.SlsStore})
	}
	if d.StoreEngine != "" {
		pairs = append(pairs, [2]string{"Store Engine", d.StoreEngine})
	}
	pairs = append(pairs, [2]string{"Metric Store", promYN(d.MetricStoreEngine)})

	// Agent / integration flags.
	if d.ControllerID != "" {
		pairs = append(pairs, [2]string{"Controller", d.ControllerID})
	}
	pairs = append(pairs,
		[2]string{"Controller Installed", promYN(d.IsControllerInstalled)},
		[2]string{"Advanced Cluster", promYN(d.IsAdvancedClusterInstalled)},
		[2]string{"Integration Center", promYN(d.IsIntegrationCenter)},
	)
	if d.AgentStatus != "" {
		pairs = append(pairs, [2]string{"Agent Status", d.AgentStatus})
	}

	// Auth settings.
	pairs = append(pairs,
		[2]string{"Auth Free Read", promYN(det.EnableAuthFreeRead)},
		[2]string{"Auth Free Write", promYN(det.EnableAuthFreeWrite)},
		[2]string{"Auth Token", promYN(det.EnableAuthToken)},
	)
	if len(det.SupportAuthTypes) > 0 {
		pairs = append(pairs, [2]string{"Auth Types", strings.Join(det.SupportAuthTypes, ", ")})
	}

	// Access endpoints: public ("Inter") and VPC ("Intra") per protocol.
	for _, ep := range []struct {
		label    string
		interURL string
		intraURL string
	}{
		{"Remote Write", det.RemoteWriteInterURL, det.RemoteWriteIntraURL},
		{"Remote Read", det.RemoteReadInterURL, det.RemoteReadIntraURL},
		{"PushGateway", det.PushGatewayInterURL, det.PushGatewayIntraURL},
		{"OpenTelemetry", det.OpenTelemetryInterURL, det.OpenTelemetryIntraURL},
		{"HTTP API", det.HTTPAPIInterURL, det.HTTPAPIIntraURL},
	} {
		if ep.interURL != "" {
			pairs = append(pairs, [2]string{ep.label + " (public)", ep.interURL})
		}
		if ep.intraURL != "" {
			pairs = append(pairs, [2]string{ep.label + " (vpc)", ep.intraURL})
		}
	}

	return pairs
}
