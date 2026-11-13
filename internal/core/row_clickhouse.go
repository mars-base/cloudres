package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ── ClickHouse (enterprise + community-compatible editions) ─────────────

// ckMetricsView decodes the "metrics" block written by the fetcher.
type ckMetricsView struct {
	DiskUsagePct  *float64 `json:"disk_usage_pct"`
	DiskUsedBytes *int64   `json:"disk_used_bytes"`
	CPUUsagePct   *float64 `json:"cpu_usage_pct"`
	Connections   *float64 `json:"connections"`
	Source        string   `json:"source"`
}

type ckWhitelistView struct {
	Name   string `json:"name"`
	IPs    string `json:"ips"`
	Hidden bool   `json:"hidden"`
}

type ckAccountView struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

type ckBackupView struct {
	Switch        bool   `json:"switch"`
	Period        string `json:"period"`
	Time          string `json:"time"`
	RetentionDays int    `json:"retention_days"`
	Size          string `json:"size"`
}

type ckEndpointView struct {
	Name   string `json:"name"`
	Conn   string `json:"connection_string"`
	IP     string `json:"ip_address"`
	Net    string `json:"net_type"`
	Ports  string `json:"ports"`
	Status string `json:"status"`
}

// ckAttrView is the union of the community DescribeDBClusterAttribute and the
// enterprise DescribeDBInstanceAttribute fields the detail view needs — the
// two editions share enough names (EngineVersion, StorageType, …) that one
// combined shape is simpler than two embedded ones.
type ckAttrView struct {
	// shared
	Category           string `json:"Category"`
	EngineVersion      string `json:"EngineVersion"`
	EngineMinorVersion string `json:"EngineMinorVersion"`
	StorageType        string `json:"StorageType"`

	// community
	DBClusterType            string `json:"DBClusterType"`
	EngineLatestMinorVersion string `json:"EngineLatestMinorVersion"`
	DBNodeCount              int    `json:"DBNodeCount"`
	DBNodeStorage            int64  `json:"DBNodeStorage"`
	ZookeeperClass           string `json:"ZookeeperClass"`
	VpcIPAddr                string `json:"VpcIpAddr"`
	PublicIPAddr             string `json:"PublicIpAddr"`
	Port                     string `json:"Port"`
	PublicPort               string `json:"PublicPort"`
	PublicConnectionString   string `json:"PublicConnectionString"`
	MaintainTime             string `json:"MaintainTime"`
	EncryptionType           string `json:"EncryptionType"`
	LbKind                   string `json:"LbKind"`

	// enterprise
	DeploySchema             string `json:"DeploySchema"`
	Engine                   string `json:"Engine"`
	LatestEngineMinorVersion string `json:"LatestEngineMinorVersion"`
	NodeCount                string `json:"NodeCount"`
	StorageQuota             string `json:"StorageQuota"`
	StorageSize              int64  `json:"StorageSize"`
	ObjectStoreSize          string `json:"ObjectStoreSize"`
	MaintainStart            string `json:"MaintainStartTime"`
	MaintainEnd              string `json:"MaintainEndTime"`
	Nodes                    struct {
		Node []struct {
			NodeStatus string `json:"NodeStatus"`
			ZoneID     string `json:"ZoneId"`
		} `json:"Node"`
	} `json:"Nodes"`
}

type ckNodeView struct {
	ShardID   string `json:"ShardId"`
	NodeIP    string `json:"NodeIp"`
	NodeName  string `json:"NodeName"`
	ReplicaID string `json:"ReplicaId"`
}

type ckZkNodeView struct {
	NodeName  string `json:"NodeName"`
	ReplicaID string `json:"ReplicaId"`
}

type ckNodesView struct {
	TotalNodeCount int            `json:"TotalNodeCount"`
	NodeInfos      []ckNodeView   `json:"NodeInfos"`
	ZkNodeInfos    []ckZkNodeView `json:"ZkNodeInfos"`
}

// ckView is the combined decode target for either edition's RawJSON — the
// community cluster fields sit at the top level of its JSON, the enterprise
// instance fields likewise, so unused fields simply stay zero.
type ckView struct {
	Edition string `json:"edition"`

	// community cluster
	DBClusterID          string `json:"DBClusterId"`
	DBClusterDescription string `json:"DBClusterDescription"`
	Category             string `json:"Category"`
	DbVersion            string `json:"DbVersion"`
	DBNodeClass          string `json:"DBNodeClass"`
	DBNodeCount          int    `json:"DBNodeCount"`
	DBNodeStorage        int64  `json:"DBNodeStorage"`
	CreationTime         string `json:"CreateTime"`
	ExpireTime           string `json:"ExpireTime"`
	PayType              string `json:"PayType"`
	VpcID                string `json:"VpcId"`
	VSwitchID            string `json:"VSwitchId"`
	ZoneID               string `json:"ZoneId"`
	ConnectionString     string `json:"ConnectionString"`
	Port                 string `json:"Port"`
	StorageType          string `json:"StorageType"`
	LockMode             string `json:"LockMode"`

	// enterprise instance
	DBInstanceID  string `json:"DBInstanceId"`
	Description   string `json:"Description"`
	EngineVersion string `json:"EngineVersion"`
	ChargeType    string `json:"ChargeType"`

	Attribute *ckAttrView       `json:"attribute"`
	Nodes     *ckNodesView      `json:"nodes"`
	Endpoints []ckEndpointView  `json:"endpoints"`
	Whitelist []ckWhitelistView `json:"whitelist_groups"`
	Accounts  []ckAccountView   `json:"accounts"`
	Backup    *ckBackupView     `json:"backup_policy"`
	Metrics   *ckMetricsView    `json:"metrics"`
}

func (r Resource) parseCK() ckView {
	var d ckView
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return d
}

// ckCapacity derives per-node GB, node count and total GB from the edition's
// fields. Community storage is per-node (× replicas); enterprise
// StorageQuota is the cluster total, so per-node is quota/count.
func (d *ckView) ckCapacity() (perNodeGB, totalGB int64, nodes int) {
	if d.Edition == "community" {
		nodes = 0
		if d.Nodes != nil && len(d.Nodes.NodeInfos) > 0 {
			nodes = len(d.Nodes.NodeInfos)
		}
		if nodes == 0 {
			nodes = d.DBNodeCount
		}
		if nodes == 0 {
			nodes = 1
		}
		perNode := d.DBNodeStorage
		if d.Attribute != nil && d.Attribute.DBNodeStorage > 0 {
			perNode = d.Attribute.DBNodeStorage
		}
		return perNode, perNode * int64(nodes), nodes
	}

	if d.Attribute != nil {
		if n, err := strconv.Atoi(d.Attribute.NodeCount); err == nil && n > 0 {
			nodes = n
		} else if len(d.Attribute.Nodes.Node) > 0 {
			nodes = len(d.Attribute.Nodes.Node)
		}
		if q, err := strconv.ParseInt(d.Attribute.StorageQuota, 10, 64); err == nil {
			totalGB = q
		}
		if nodes > 0 && totalGB > 0 {
			perNodeGB = totalGB / int64(nodes)
		}
	}
	if nodes == 0 {
		nodes = len(d.Attribute.Nodes.Node)
	}
	return perNodeGB, totalGB, nodes
}

// ckShardCount counts distinct shards from the node topology, falling back to
// the cluster's DBNodeCount (the API's node-group count, i.e. shards).
func (d *ckView) ckShardCount() int {
	seen := make(map[string]bool)
	if d.Nodes != nil {
		for _, n := range d.Nodes.NodeInfos {
			if n.ShardID != "" {
				seen[n.ShardID] = true
			}
		}
	}
	if len(seen) > 0 {
		return len(seen)
	}
	if d.Attribute != nil && d.Attribute.DBNodeCount > 0 {
		return d.Attribute.DBNodeCount
	}
	return d.DBNodeCount
}

func ckDiskLabel(m *ckMetricsView) string {
	if m == nil || m.DiskUsagePct == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", *m.DiskUsagePct)
}

func ckOptPct(p *float64) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", *p)
}

func (r Resource) ckRow() []string {
	d := r.parseCK()

	version := d.DbVersion
	arch := d.Category
	if d.Edition == "enterprise" {
		version = d.EngineVersion
		if d.Attribute != nil && d.Attribute.EngineVersion != "" {
			version = d.Attribute.EngineVersion
		}
		arch = ""
		if d.Attribute != nil {
			arch = d.Attribute.DeploySchema
		}
	} else if d.Attribute != nil && d.Attribute.EngineVersion != "" {
		version = d.Attribute.EngineVersion
	}

	perNode, total, nodes := d.ckCapacity()
	storage := "-"
	if perNode > 0 {
		storage = fmt.Sprintf("%dGB×%d=%dGB", perNode, nodes, total)
	}

	edition := "Community"
	if d.Edition == "enterprise" {
		edition = "Enterprise"
	}

	return []string{
		r.ResourceID, r.ResourceName, r.Status, edition, version, arch,
		itoa(nodes), storage, ckDiskLabel(d.Metrics), r.Region,
	}
}

func (r Resource) ckDetail() [][2]string {
	d := r.parseCK()

	var pairs [][2]string
	if d.Edition == "enterprise" {
		pairs = d.ckEnterpriseDetail(r)
	} else {
		pairs = d.ckCommunityDetail(r)
	}

	if d.Backup != nil {
		on := "off"
		if d.Backup.Switch {
			on = "on"
		}
		window := strings.TrimSpace(d.Backup.Period + " " + d.Backup.Time)
		pairs = append(pairs, [2]string{"Backup", on + "  " + window})
		if d.Backup.RetentionDays > 0 {
			pairs = append(pairs, [2]string{"Backup Retention", itoa(d.Backup.RetentionDays) + " days"})
		}
		if d.Backup.Size != "" {
			pairs = append(pairs, [2]string{"Backup Size", d.Backup.Size})
		}
	}

	for _, g := range d.Whitelist {
		name := g.Name
		if g.Hidden {
			name += " (hidden)"
		}
		pairs = append(pairs, [2]string{"ACL: " + name, g.IPs})
	}

	for _, a := range d.Accounts {
		line := a.Name
		if a.Type != "" || a.Status != "" {
			line = fmt.Sprintf("%s  (%s/%s)", a.Name, a.Type, a.Status)
		}
		if a.Description != "" {
			line += "  " + a.Description
		}
		pairs = append(pairs, [2]string{"Account", line})
	}

	return pairs
}

func (d *ckView) ckMetricPairs() [][2]string {
	m := d.Metrics
	if m == nil {
		return nil
	}
	var pairs [][2]string
	if m.DiskUsagePct != nil {
		label := ckOptPct(m.DiskUsagePct)
		if m.Source == "calculated" {
			label += " (calculated)"
		}
		pairs = append(pairs, [2]string{"Disk Usage", label})
	}
	if m.DiskUsedBytes != nil {
		pairs = append(pairs, [2]string{"Disk Used", formatBytes(*m.DiskUsedBytes)})
	}
	if m.CPUUsagePct != nil {
		pairs = append(pairs, [2]string{"CPU Usage", ckOptPct(m.CPUUsagePct)})
	}
	if m.Connections != nil {
		pairs = append(pairs, [2]string{"Connections", fmt.Sprintf("%.0f", *m.Connections)})
	}
	return pairs
}

func (d *ckView) ckCommunityDetail(r Resource) [][2]string {
	attr := d.Attribute
	if attr == nil {
		attr = &ckAttrView{}
	}

	version := d.DbVersion
	if attr.EngineVersion != "" {
		version = attr.EngineVersion
	}
	minor := attr.EngineMinorVersion
	if minor != "" && attr.EngineLatestMinorVersion != "" && attr.EngineMinorVersion != attr.EngineLatestMinorVersion {
		minor = fmt.Sprintf("%s (latest %s)", attr.EngineMinorVersion, attr.EngineLatestMinorVersion)
	}

	perNode, total, nodes := d.ckCapacity()

	pairs := [][2]string{
		{"ID", d.DBClusterID},
		{"Name", d.DBClusterDescription},
		{"Edition", "Community-compatible"},
		{"Region", r.Region},
		{"Zone", d.ZoneID},
		{"Status", r.Status},
		{"Version", version},
	}
	if minor != "" {
		pairs = append(pairs, [2]string{"Engine Minor", minor})
	}
	pairs = append(pairs,
		[2]string{"Architecture", d.Category},
		[2]string{"Cluster Type", attr.DBClusterType},
		[2]string{"Node Class", d.DBNodeClass},
		[2]string{"ZooKeeper Class", attr.ZookeeperClass},
		[2]string{"Shards", itoa(d.ckShardCount())},
		[2]string{"Nodes", itoa(nodes)},
		[2]string{"Storage / Node", fmt.Sprintf("%dGB", perNode)},
		[2]string{"Total Storage", fmt.Sprintf("%dGB", total)},
		[2]string{"Storage Type", d.StorageType},
	)
	pairs = append(pairs, d.ckMetricPairs()...)
	pairs = append(pairs,
		[2]string{"VPC", d.VpcID},
		[2]string{"VSwitch", d.VSwitchID},
		[2]string{"VPC IP", attr.VpcIPAddr},
		[2]string{"Public IP", attr.PublicIPAddr},
		[2]string{"Port", attr.Port},
		[2]string{"Public Port", attr.PublicPort},
		[2]string{"Endpoint", d.ConnectionString},
		[2]string{"Public Endpoint", attr.PublicConnectionString},
		[2]string{"Pay Type", d.PayType},
		[2]string{"Created", d.CreationTime},
		[2]string{"Expires", d.ExpireTime},
		[2]string{"Maintain Window", attr.MaintainTime},
		[2]string{"Encryption", attr.EncryptionType},
		[2]string{"LB Kind", attr.LbKind},
	)

	if d.Nodes != nil {
		for _, n := range d.Nodes.NodeInfos {
			line := fmt.Sprintf("%s  shard=%s replica=%s", n.NodeIP, n.ShardID, n.ReplicaID)
			pairs = append(pairs, [2]string{"CH Node", line})
		}
		for _, n := range d.Nodes.ZkNodeInfos {
			pairs = append(pairs, [2]string{"ZK Node", n.NodeName})
		}
	}

	return pairs
}

func (d *ckView) ckEnterpriseDetail(r Resource) [][2]string {
	attr := d.Attribute
	if attr == nil {
		attr = &ckAttrView{}
	}

	version := d.EngineVersion
	if attr.EngineVersion != "" {
		version = attr.EngineVersion
	}
	minor := attr.EngineMinorVersion
	if minor != "" && attr.LatestEngineMinorVersion != "" && attr.EngineMinorVersion != attr.LatestEngineMinorVersion {
		minor = fmt.Sprintf("%s (latest %s)", attr.EngineMinorVersion, attr.LatestEngineMinorVersion)
	}

	perNode, total, nodes := d.ckCapacity()

	pairs := [][2]string{
		{"ID", d.DBInstanceID},
		{"Name", d.Description},
		{"Edition", "Enterprise"},
		{"Region", r.Region},
		{"Zone", d.ZoneID},
		{"Status", r.Status},
		{"Version", version},
	}
	if minor != "" {
		pairs = append(pairs, [2]string{"Engine Minor", minor})
	}
	pairs = append(pairs,
		[2]string{"Deploy Schema", attr.DeploySchema},
		[2]string{"Category", attr.Category},
		[2]string{"Nodes", itoa(nodes)},
	)
	if perNode > 0 {
		pairs = append(pairs, [2]string{"Storage / Node", fmt.Sprintf("%dGB", perNode)})
	}
	pairs = append(pairs,
		[2]string{"Total Storage", fmt.Sprintf("%dGB", total)},
		[2]string{"Storage Used", fmt.Sprintf("%dGB", attr.StorageSize)},
		[2]string{"Object Store", attr.ObjectStoreSize},
		[2]string{"Storage Type", attr.StorageType},
	)
	pairs = append(pairs, d.ckMetricPairs()...)
	pairs = append(pairs,
		[2]string{"VPC", d.VpcID},
		[2]string{"VSwitch", d.VSwitchID},
		[2]string{"Pay Type", d.ChargeType},
		[2]string{"Created", d.CreationTime},
		[2]string{"Expires", d.ExpireTime},
		[2]string{"Maintain Window", strings.TrimSpace(attr.MaintainStart + " - " + attr.MaintainEnd)},
	)

	for _, ep := range d.Endpoints {
		line := ep.Conn
		if ep.IP != "" {
			line += "  " + ep.IP
		}
		if ep.Ports != "" {
			line += "  " + ep.Ports
		}
		if ep.Net != "" || ep.Status != "" {
			line += fmt.Sprintf("  (%s/%s)", ep.Net, ep.Status)
		}
		pairs = append(pairs, [2]string{"Endpoint", line})
	}

	for i, n := range attr.Nodes.Node {
		pairs = append(pairs, [2]string{fmt.Sprintf("Node %d", i+1), fmt.Sprintf("%s  %s", n.ZoneID, n.NodeStatus)})
	}

	return pairs
}
