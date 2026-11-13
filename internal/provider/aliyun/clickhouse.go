package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// ClickHouse — enterprise edition and community-compatible edition are two
// separate API versions of the same product:
//
//   Enterprise (2023-05-22, PascalCase):
//     aliyun clickhouse DescribeDBInstances --RegionId <r>
//   Community (2019-11-11, kebab-case only — the PascalCase form rejects the
//   legacy version in CLI v3):
//     aliyun clickhouse describe-db-clusters --api-version 2019-11-11 \
//       --biz-region-id <r>
//
// Both are queried and merged under the "ck" resource type; the stored JSON
// carries an "edition" field so the row/detail views can decode accordingly.
// ---------------------------------------------------------------------------

type ClickHouseFetcher struct{}

func (f *ClickHouseFetcher) ResourceType() string { return "ck" }

func (f *ClickHouseFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchClickHouse(ctx, p)
}

const (
	ckCommunityAPIVersion = "2019-11-11"
	ckPageSize            = 30
)

var ckEnterpriseStatus = map[string]string{
	"ACTIVATION":             "Running",
	"CREATING":               "Creating",
	"STOPPED":                "Stopped",
	"RESOURCE_PREPARING":     "Preparing",
	"MINOR_VERSION_TRANSING": "Upgrading",
	"CLASS_CHANGING":         "Resizing",
	"DELETING":               "Deleting",
	"DELETED":                "Deleted",
}

var ckCommunityStatus = map[string]string{
	"Running":                 "Running",
	"Preparing":               "Preparing",
	"ResourcePreparing":       "Preparing",
	"Creating":                "Creating",
	"Restoring":               "Restoring",
	"Deleting":                "Deleting",
	"ClassChanging":           "Resizing",
	"MultiAZChanging":         "Resizing",
	"Scale_Out":               "ScalingOut",
	"READINS_MAINTAINING":     "ReadonlyMaintenance",
	"FailoverTesting":         "FailoverTesting",
	"ExitingFailoverTesting":  "FailoverTesting",
	"NetAddressCreating":      "NetChanging",
	"NetAddressDeleting":      "NetChanging",
	"SLBCreating":             "NetChanging",
	"SLBDeleting":             "NetChanging",
	"minor_version_upgrading": "Upgrading",
	"version_upgrading":       "Upgrading",
	"module_upgrading":        "Upgrading",
	"INITIALING":              "Initializing",
	"INSTALL_EXTENSIONS":      "InstallingExtensions",
}

// ── shared shapes stored in RawJSON ─────────────────────────────────────

type ckMetrics struct {
	DiskUsagePct  *float64 `json:"disk_usage_pct,omitempty"`
	DiskUsedBytes *int64   `json:"disk_used_bytes,omitempty"`
	CPUUsagePct   *float64 `json:"cpu_usage_pct,omitempty"`
	Connections   *float64 `json:"connections,omitempty"`
	Source        string   `json:"source,omitempty"` // cms | calculated
	SampledAt     int64    `json:"sampled_at,omitempty"`
}

type ckWhitelistGroup struct {
	Name    string `json:"name"`
	IPs     string `json:"ips"`
	NetType string `json:"net_type,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
}

type ckAccount struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

type ckBackupPolicy struct {
	Switch        bool   `json:"switch"`
	Period        string `json:"period,omitempty"`
	Time          string `json:"time,omitempty"`
	RetentionDays int    `json:"retention_days,omitempty"`
	Size          string `json:"size,omitempty"`
}

type ckEndpointLine struct {
	Name   string `json:"name,omitempty"`
	Conn   string `json:"connection_string"`
	IP     string `json:"ip_address,omitempty"`
	Net    string `json:"net_type,omitempty"`
	Ports  string `json:"ports,omitempty"`
	Status string `json:"status,omitempty"`
}

// ── fetch entry ────────────────────────────────────────────────────────

func fetchClickHouse(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var all []core.Resource
	now := time.Now()

	regions := p.ProfileRegions[p.ActiveProfile]
	if len(regions) == 0 && len(p.Regions) > 0 {
		regions = []string{p.Regions[0]}
	}

	for _, region := range regions {
		all = append(all, fetchCKEnterprise(ctx, p.ActiveProfile, region, now)...)
		all = append(all, fetchCKCommunity(ctx, p.ActiveProfile, region, now)...)
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].ResourceName != all[j].ResourceName {
			return all[i].ResourceName < all[j].ResourceName
		}
		return all[i].ResourceID < all[j].ResourceID
	})

	return all, nil
}

func ckCall(ctx context.Context, profile string, args []string, dst any) bool {
	out, err := runAliyun(ctx, args, profile)
	if err != nil {
		return false
	}
	return json.Unmarshal(out, dst) == nil
}

// ── enterprise edition (2023-05-22) ────────────────────────────────────

type ckEnterpriseInstance struct {
	AliUid             int64  `json:"AliUid"`
	Bid                string `json:"Bid"`
	ChargeType         string `json:"ChargeType"`
	CreateTime         string `json:"CreateTime"`
	DBInstanceID       string `json:"DBInstanceId"`
	DeletionProtection string `json:"DeletionProtection"`
	Description        string `json:"Description"`
	Engine             string `json:"Engine"`
	EngineVersion      string `json:"EngineVersion"`
	ExpireTime         string `json:"ExpireTime"`
	LockMode           string `json:"LockMode"`
	LockReason         string `json:"LockReason"`
	MaintainEndTime    string `json:"MaintainEndTime"`
	MaintainStartTime  string `json:"MaintainStartTime"`
	RegionID           string `json:"RegionId"`
	ResourceGroupID    string `json:"ResourceGroupId"`
	ScaleMax           int    `json:"ScaleMax"`
	ScaleMin           int    `json:"ScaleMin"`
	Status             string `json:"Status"`
	StorageType        string `json:"StorageType"`
	VSwitchID          string `json:"VSwitchId"`
	VpcID              string `json:"VpcId"`
	ZoneID             string `json:"ZoneId"`
}

type ckEnterpriseList struct {
	Data struct {
		TotalCount  int `json:"TotalCount"`
		DBInstances struct {
			DBInstance []ckEnterpriseInstance `json:"DBInstance"`
		} `json:"DBInstances"`
	} `json:"Data"`
}

type ckEnterpriseAttribute struct {
	AliUid                   int64  `json:"AliUid"`
	Category                 string `json:"Category"`
	ChargeType               string `json:"ChargeType"`
	CreateTime               string `json:"CreateTime"`
	DBInstanceID             string `json:"DBInstanceId"`
	DeletionProtection       bool   `json:"DeletionProtection"`
	DeploySchema             string `json:"DeploySchema"`
	Description              string `json:"Description"`
	DisabledPorts            string `json:"DisabledPorts"`
	Engine                   string `json:"Engine"`
	EngineMinorVersion       string `json:"EngineMinorVersion"`
	EngineVersion            string `json:"EngineVersion"`
	ExpireTime               string `json:"ExpireTime"`
	LatestEngineMinorVersion string `json:"LatestEngineMinorVersion"`
	LockMode                 string `json:"LockMode"`
	LockReason               string `json:"LockReason"`
	MaintainEndTime          string `json:"MaintainEndTime"`
	MaintainStartTime        string `json:"MaintainStartTime"`
	NodeCount                string `json:"NodeCount"`
	NodeScaleMax             string `json:"NodeScaleMax"`
	NodeScaleMin             string `json:"NodeScaleMin"`
	Nodes                    struct {
		Node []struct {
			NodeStatus string `json:"NodeStatus"`
			ZoneID     string `json:"ZoneId"`
		} `json:"Node"`
	} `json:"Nodes"`
	MultiZones struct {
		MultiZone []struct {
			VSwitchIDs struct {
				VSwitchID []string `json:"VSwitchId"`
			} `json:"VSwitchIds"`
			ZoneID string `json:"ZoneId"`
		} `json:"MultiZone"`
	} `json:"MultiZones"`
	ObjectStoreSize string `json:"ObjectStoreSize"`
	RegionID        string `json:"RegionId"`
	ResourceGroupID string `json:"ResourceGroupId"`
	ScaleMax        int    `json:"ScaleMax"`
	ScaleMin        int    `json:"ScaleMin"`
	Status          string `json:"Status"`
	StorageQuota    string `json:"StorageQuota"`
	StorageSize     int64  `json:"StorageSize"`
	StorageType     string `json:"StorageType"`
	VSwitchID       string `json:"VSwitchId"`
	VpcID           string `json:"VpcId"`
	ZoneID          string `json:"ZoneId"`
}

type ckEnterpriseEnriched struct {
	ckEnterpriseInstance
	Edition   string                 `json:"edition"`
	Attribute *ckEnterpriseAttribute `json:"attribute,omitempty"`
	Endpoints []ckEndpointLine       `json:"endpoints,omitempty"`
	Whitelist []ckWhitelistGroup     `json:"whitelist_groups,omitempty"`
	Accounts  []ckAccount            `json:"accounts,omitempty"`
	Backup    *ckBackupPolicy        `json:"backup_policy,omitempty"`
	Metrics   *ckMetrics             `json:"metrics,omitempty"`
}

func fetchCKEnterprise(ctx context.Context, profile, region string, now time.Time) []core.Resource {
	var resources []core.Resource
	collected := 0

	for page := 1; ; page++ {
		var resp ckEnterpriseList
		args := []string{
			"clickhouse", "DescribeDBInstances",
			"--RegionId", region,
			"--region", region,
			"--PageNumber", strconv.Itoa(page),
			"--PageSize", strconv.Itoa(ckPageSize),
		}
		if !ckCall(ctx, profile, args, &resp) {
			break
		}

		items := resp.Data.DBInstances.DBInstance
		for _, inst := range items {
			enriched := enrichCKEnterprise(ctx, profile, region, inst)
			rawJSON, _ := json.Marshal(enriched)

			status := ckEnterpriseStatus[inst.Status]
			if status == "" {
				status = inst.Status
			}
			name := inst.Description
			if name == "" {
				name = inst.DBInstanceID
			}
			resources = append(resources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ck",
				Region:       inst.RegionID,
				ResourceID:   inst.DBInstanceID,
				ResourceName: name,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		collected += len(items)
		if len(items) == 0 || len(items) < ckPageSize || collected >= resp.Data.TotalCount {
			break
		}
	}

	return resources
}

func enrichCKEnterprise(ctx context.Context, profile, region string, inst ckEnterpriseInstance) ckEnterpriseEnriched {
	e := ckEnterpriseEnriched{ckEnterpriseInstance: inst, Edition: "enterprise"}
	id := inst.DBInstanceID

	// Detail attributes (arch, nodes, storage, maintenance window)
	var attr struct {
		Data ckEnterpriseAttribute `json:"Data"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "DescribeDBInstanceAttribute",
		"--DBInstanceId", id,
		"--RegionId", region,
		"--region", region,
	}, &attr) {
		e.Attribute = &attr.Data
	}

	// Endpoints
	var eps struct {
		Data struct {
			Endpoints struct {
				Endpoint []struct {
					EndpointName     string `json:"EndpointName"`
					ConnectionString string `json:"ConnectionString"`
					IPAddress        string `json:"IPAddress"`
					NetType          string `json:"NetType"`
					Status           string `json:"Status"`
					Ports            struct {
						Port []struct {
							Port     int    `json:"Port"`
							Protocol string `json:"Protocol"`
						} `json:"Port"`
					} `json:"Ports"`
				} `json:"Endpoint"`
			} `json:"Endpoints"`
		} `json:"Data"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "DescribeEndpoints",
		"--DBInstanceId", id,
		"--RegionId", region,
		"--region", region,
	}, &eps) {
		for _, ep := range eps.Data.Endpoints.Endpoint {
			var ports []string
			for _, p := range ep.Ports.Port {
				ports = append(ports, fmt.Sprintf("%s:%d", p.Protocol, p.Port))
			}
			e.Endpoints = append(e.Endpoints, ckEndpointLine{
				Name:   ep.EndpointName,
				Conn:   ep.ConnectionString,
				IP:     ep.IPAddress,
				Net:    ep.NetType,
				Ports:  strings.Join(ports, " "),
				Status: ep.Status,
			})
		}
	}

	// Whitelist groups
	var wl struct {
		Data struct {
			GroupItems struct {
				GroupItems []struct {
					GroupName      string `json:"GroupName"`
					SecurityIPList string `json:"SecurityIPList"`
					NetType        string `json:"WhitelistNetType"`
				} `json:"GroupItems"`
			} `json:"GroupItems"`
		} `json:"Data"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "DescribeSecurityIPList",
		"--DBInstanceId", id,
		"--RegionId", region,
		"--region", region,
	}, &wl) {
		for _, g := range wl.Data.GroupItems.GroupItems {
			e.Whitelist = append(e.Whitelist, ckWhitelistGroup{
				Name: g.GroupName, IPs: g.SecurityIPList, NetType: g.NetType,
			})
		}
	}

	// Accounts
	var accts struct {
		Data struct {
			Accounts struct {
				Account []struct {
					Account     string `json:"Account"`
					AccountType string `json:"AccountType"`
					Description string `json:"Description"`
					Status      string `json:"Status"`
				} `json:"Account"`
			} `json:"Accounts"`
		} `json:"Data"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "DescribeAccounts",
		"--DBInstanceId", id,
		"--RegionId", region,
		"--region", region,
	}, &accts) {
		for _, a := range accts.Data.Accounts.Account {
			e.Accounts = append(e.Accounts, ckAccount{
				Name: a.Account, Type: a.AccountType, Status: a.Status, Description: a.Description,
			})
		}
	}

	// Backup policy
	var bp struct {
		BackupRetentionPeriod int    `json:"BackupRetentionPeriod"`
		BackupSize            string `json:"BackupSize"`
		PreferredBackupPeriod string `json:"PreferredBackupPeriod"`
		PreferredBackupTime   string `json:"PreferredBackupTime"`
		Switch                string `json:"Switch"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "DescribeBackupPolicy",
		"--DBInstanceId", id,
		"--RegionId", region,
		"--region", region,
	}, &bp) {
		e.Backup = &ckBackupPolicy{
			Switch:        bp.Switch == "true",
			Period:        bp.PreferredBackupPeriod,
			Time:          bp.PreferredBackupTime,
			RetentionDays: bp.BackupRetentionPeriod,
			Size:          bp.BackupSize,
		}
	}

	e.Metrics = ckEnterpriseMetrics(ctx, profile, region, id, e.Attribute)

	sort.SliceStable(e.Endpoints, func(i, j int) bool { return e.Endpoints[i].Conn < e.Endpoints[j].Conn })
	return e
}

// ckEnterpriseMetrics queries CPU from CMS (enterprise has no disk metric) and
// falls back to StorageSize/StorageQuota for the disk usage rate.
func ckEnterpriseMetrics(ctx context.Context, profile, region, id string, attr *ckEnterpriseAttribute) *ckMetrics {
	m := &ckMetrics{}

	if v, ts := ckCMSPoint(ctx, profile, region, id, "clickhouse_cpu_utilization_ee"); v != nil {
		m.CPUUsagePct = v
		m.SampledAt = ts
	}

	if attr != nil && attr.StorageQuota != "" {
		quota, errQ := strconv.ParseFloat(attr.StorageQuota, 64)
		if errQ == nil && quota > 0 {
			pct := float64(attr.StorageSize) / quota * 100
			m.DiskUsagePct = &pct
			used := attr.StorageSize * 1024 * 1024 * 1024
			m.DiskUsedBytes = &used
			m.Source = "calculated"
		}
	}

	if m.CPUUsagePct == nil && m.DiskUsagePct == nil {
		return nil
	}
	return m
}

// ── community-compatible edition (2019-11-11) ──────────────────────────

type ckCommunityCluster struct {
	AliUid               int64  `json:"AliUid"`
	Category             string `json:"Category"`
	CommodityCode        string `json:"CommodityCode"`
	ConnectionString     string `json:"ConnectionString"`
	ControlVersion       string `json:"ControlVersion"`
	CreateTime           string `json:"CreateTime"`
	DBClusterDescription string `json:"DBClusterDescription"`
	DBClusterID          string `json:"DBClusterId"`
	DBClusterNetworkType string `json:"DBClusterNetworkType"`
	DBClusterStatus      string `json:"DBClusterStatus"`
	DBNodeClass          string `json:"DBNodeClass"`
	DBNodeCount          int    `json:"DBNodeCount"`
	DBNodeStorage        int64  `json:"DBNodeStorage"`
	DbVersion            string `json:"DbVersion"`
	ExpireTime           string `json:"ExpireTime"`
	LockMode             string `json:"LockMode"`
	LockReason           string `json:"LockReason"`
	PayType              string `json:"PayType"`
	Port                 string `json:"Port"`
	RegionID             string `json:"RegionId"`
	ResourceGroupID      string `json:"ResourceGroupId"`
	StorageType          string `json:"StorageType"`
	VSwitchID            string `json:"VSwitchId"`
	VpcID                string `json:"VpcId"`
	ZoneID               string `json:"ZoneId"`
}

type ckCommunityList struct {
	TotalCount int
	PageNumber int
	PageSize   int
	DBClusters struct {
		DBCluster []ckCommunityCluster `json:"DBCluster"`
	} `json:"DBClusters"`
}

type ckCommunityAttribute struct {
	AliUid                   int64  `json:"AliUid"`
	AppointmentRestartTime   string `json:"AppointmentRestartTime"`
	Category                 string `json:"Category"`
	ConnectionString         string `json:"ConnectionString"`
	ControlVersion           string `json:"ControlVersion"`
	CreateTime               string `json:"CreateTime"`
	DBClusterDescription     string `json:"DBClusterDescription"`
	DBClusterID              string `json:"DBClusterId"`
	DBClusterNetworkType     string `json:"DBClusterNetworkType"`
	DBClusterStatus          string `json:"DBClusterStatus"`
	DBClusterType            string `json:"DBClusterType"`
	DBNodeClass              string `json:"DBNodeClass"`
	DBNodeCount              int    `json:"DBNodeCount"`
	DBNodeStorage            int64  `json:"DBNodeStorage"`
	EncryptionType           string `json:"EncryptionType"`
	Engine                   string `json:"Engine"`
	EngineLatestMinorVersion string `json:"EngineLatestMinorVersion"`
	EngineMinorVersion       string `json:"EngineMinorVersion"`
	EngineVersion            string `json:"EngineVersion"`
	ExpireTime               string `json:"ExpireTime"`
	LbKind                   string `json:"LbKind"`
	LockMode                 string `json:"LockMode"`
	LockReason               string `json:"LockReason"`
	MaintainAutoType         bool   `json:"MaintainAutoType"`
	MaintainTime             string `json:"MaintainTime"`
	PayType                  string `json:"PayType"`
	Port                     string `json:"Port"`
	PublicConnectionString   string `json:"PublicConnectionString"`
	PublicIPAddr             string `json:"PublicIpAddr"`
	PublicPort               string `json:"PublicPort"`
	RegionID                 string `json:"RegionId"`
	ResourceGroupID          string `json:"ResourceGroupId"`
	StorageType              string `json:"StorageType"`
	SupportBackup            int    `json:"SupportBackup"`
	SupportHttpsPort         bool   `json:"SupportHttpsPort"`
	SupportMysqlPort         bool   `json:"SupportMysqlPort"`
	VSwitchID                string `json:"VSwitchId"`
	VpcID                    string `json:"VpcId"`
	VpcIPAddr                string `json:"VpcIpAddr"`
	ZoneID                   string `json:"ZoneId"`
	ZookeeperClass           string `json:"ZookeeperClass"`
}

type ckCommunityNode struct {
	ShardID         string `json:"ShardId"`
	NodeIP          string `json:"NodeIp"`
	NodeName        string `json:"NodeName"`
	ReplicaID       string `json:"ReplicaId"`
	FailoverTesting bool   `json:"FailoverTesting"`
}

type ckCommunityZkNode struct {
	NodeName  string `json:"NodeName"`
	ReplicaID string `json:"ReplicaId"`
}

type ckCommunityNodes struct {
	TotalNodeCount int                 `json:"TotalNodeCount"`
	NodeInfos      []ckCommunityNode   `json:"NodeInfos"`
	ZkNodeInfos    []ckCommunityZkNode `json:"ZkNodeInfos"`
}

type ckCommunityEnriched struct {
	ckCommunityCluster
	Edition   string                `json:"edition"`
	Attribute *ckCommunityAttribute `json:"attribute,omitempty"`
	Nodes     *ckCommunityNodes     `json:"nodes,omitempty"`
	Whitelist []ckWhitelistGroup    `json:"whitelist_groups,omitempty"`
	Accounts  []ckAccount           `json:"accounts,omitempty"`
	Backup    *ckBackupPolicy       `json:"backup_policy,omitempty"`
	Metrics   *ckMetrics            `json:"metrics,omitempty"`
}

func fetchCKCommunity(ctx context.Context, profile, region string, now time.Time) []core.Resource {
	var resources []core.Resource
	collected := 0

	for page := 1; ; page++ {
		var resp ckCommunityList
		args := []string{
			"clickhouse", "describe-db-clusters",
			"--api-version", ckCommunityAPIVersion,
			"--biz-region-id", region,
			"--region", region,
			"--page-number", strconv.Itoa(page),
			"--page-size", strconv.Itoa(ckPageSize),
		}
		if !ckCall(ctx, profile, args, &resp) {
			break
		}

		items := resp.DBClusters.DBCluster
		for _, cluster := range items {
			enriched := enrichCKCommunity(ctx, profile, region, cluster)
			rawJSON, _ := json.Marshal(enriched)

			status := ckCommunityStatus[cluster.DBClusterStatus]
			if status == "" {
				status = cluster.DBClusterStatus
			}
			name := cluster.DBClusterDescription
			if name == "" {
				name = cluster.DBClusterID
			}
			regionID := cluster.RegionID
			if regionID == "" {
				regionID = region
			}
			resources = append(resources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ck",
				Region:       regionID,
				ResourceID:   cluster.DBClusterID,
				ResourceName: name,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		collected += len(items)
		if len(items) == 0 || len(items) < ckPageSize || collected >= resp.TotalCount {
			break
		}
	}

	return resources
}

func enrichCKCommunity(ctx context.Context, profile, region string, cluster ckCommunityCluster) ckCommunityEnriched {
	e := ckCommunityEnriched{ckCommunityCluster: cluster, Edition: "community"}
	id := cluster.DBClusterID

	// Cluster attribute (arch, node class, storage, maintenance window).
	// Unlike the list/node APIs this one rejects --biz-region-id.
	var attr struct {
		DBCluster ckCommunityAttribute `json:"DBCluster"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "describe-db-cluster-attribute",
		"--api-version", ckCommunityAPIVersion,
		"--db-cluster-id", id,
		"--region", region,
	}, &attr) {
		e.Attribute = &attr.DBCluster
	}

	// Node infos (ClickHouse replicas + ZooKeeper nodes)
	var nodes ckCommunityNodes
	if ckCall(ctx, profile, []string{
		"clickhouse", "describe-db-cluster-node-infos",
		"--api-version", ckCommunityAPIVersion,
		"--biz-region-id", region,
		"--db-cluster-id", id,
		"--region", region,
	}, &nodes) {
		e.Nodes = &nodes
	}

	// Whitelist
	var wl struct {
		DBClusterAccessWhiteList struct {
			IPArray []struct {
				DBClusterIPArrayName      string `json:"DBClusterIPArrayName"`
				SecurityIPList            string `json:"SecurityIPList"`
				DBClusterIPArrayAttribute string `json:"DBClusterIPArrayAttribute"`
			} `json:"IPArray"`
		} `json:"DBClusterAccessWhiteList"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "describe-db-cluster-access-white-list",
		"--api-version", ckCommunityAPIVersion,
		"--db-cluster-id", id,
		"--region", region,
	}, &wl) {
		for _, g := range wl.DBClusterAccessWhiteList.IPArray {
			e.Whitelist = append(e.Whitelist, ckWhitelistGroup{
				Name:   g.DBClusterIPArrayName,
				IPs:    g.SecurityIPList,
				Hidden: g.DBClusterIPArrayAttribute == "hidden",
			})
		}
	}

	// Accounts (paginated)
	for page := 1; ; page++ {
		var accts struct {
			TotalCount int `json:"TotalCount"`
			Accounts   struct {
				Account []struct {
					AccountName        string `json:"AccountName"`
					AccountDescription string `json:"AccountDescription"`
					AccountType        string `json:"AccountType"`
					AccountStatus      string `json:"AccountStatus"`
				} `json:"Account"`
			} `json:"Accounts"`
		}
		if !ckCall(ctx, profile, []string{
			"clickhouse", "describe-accounts",
			"--api-version", ckCommunityAPIVersion,
			"--db-cluster-id", id,
			"--page-number", strconv.Itoa(page),
			"--page-size", "100",
			"--region", region,
		}, &accts) {
			break
		}
		for _, a := range accts.Accounts.Account {
			e.Accounts = append(e.Accounts, ckAccount{
				Name: a.AccountName, Type: a.AccountType, Status: a.AccountStatus, Description: a.AccountDescription,
			})
		}
		if len(accts.Accounts.Account) < 100 || len(e.Accounts) >= accts.TotalCount {
			break
		}
	}

	// Backup policy
	var bp struct {
		BackupRetentionPeriod int    `json:"BackupRetentionPeriod"`
		BackupSize            string `json:"BackupSize"`
		PreferredBackupPeriod string `json:"PreferredBackupPeriod"`
		PreferredBackupTime   string `json:"PreferredBackupTime"`
		Switch                bool   `json:"Switch"`
	}
	if ckCall(ctx, profile, []string{
		"clickhouse", "describe-backup-policy",
		"--api-version", ckCommunityAPIVersion,
		"--db-cluster-id", id,
		"--region", region,
	}, &bp) {
		e.Backup = &ckBackupPolicy{
			Switch:        bp.Switch,
			Period:        bp.PreferredBackupPeriod,
			Time:          bp.PreferredBackupTime,
			RetentionDays: bp.BackupRetentionPeriod,
			Size:          bp.BackupSize,
		}
	}

	e.Metrics = ckCommunityMetrics(ctx, profile, region, id, e.Attribute, e.Nodes)

	sort.SliceStable(e.Accounts, func(i, j int) bool { return e.Accounts[i].Name < e.Accounts[j].Name })
	sort.SliceStable(e.Whitelist, func(i, j int) bool { return e.Whitelist[i].Name < e.Whitelist[j].Name })
	return e
}

// ckCommunityMetrics pulls disk/CPU/connection metrics from CloudMonitor. When
// the utilization metric is missing but used bytes and node capacity are known,
// the usage rate is derived from them.
func ckCommunityMetrics(ctx context.Context, profile, region, id string, attr *ckCommunityAttribute, nodes *ckCommunityNodes) *ckMetrics {
	m := &ckMetrics{}
	found := false

	if v, ts := ckCMSPoint(ctx, profile, region, id, "clickhouse_disk_utilization_cc"); v != nil {
		m.DiskUsagePct = v
		m.SampledAt = ts
		m.Source = "cms"
		found = true
	}
	if v, _ := ckCMSPoint(ctx, profile, region, id, "clickhouse_disk_used_cc"); v != nil {
		used := int64(*v)
		m.DiskUsedBytes = &used
		found = true
	}
	if v, _ := ckCMSPoint(ctx, profile, region, id, "clickhouse_cpu_utilization_cc"); v != nil {
		m.CPUUsagePct = v
		found = true
	}
	if v, _ := ckCMSPoint(ctx, profile, region, id, "clickhouse_conn_usage_count_cc"); v != nil {
		m.Connections = v
		found = true
	}

	// Fall back to used-bytes / provisioned-capacity when the utilization
	// metric is unavailable.
	if m.DiskUsagePct == nil && m.DiskUsedBytes != nil {
		capacityGB := ckCommunityCapacityGB(attr, nodes)
		if capacityGB > 0 {
			pct := float64(*m.DiskUsedBytes) / (float64(capacityGB) * 1024 * 1024 * 1024) * 100
			m.DiskUsagePct = &pct
			m.Source = "calculated"
		}
	}

	if !found && m.DiskUsagePct == nil {
		return nil
	}
	return m
}

func ckCommunityCapacityGB(attr *ckCommunityAttribute, nodes *ckCommunityNodes) int64 {
	if attr == nil || attr.DBNodeStorage <= 0 {
		return 0
	}
	replicas := int64(attr.DBNodeCount)
	if nodes != nil && len(nodes.NodeInfos) > 0 {
		replicas = int64(len(nodes.NodeInfos))
	}
	if replicas <= 0 {
		replicas = 1
	}
	return attr.DBNodeStorage * replicas
}

// ── CloudMonitor helper ────────────────────────────────────────────────

// ckCMSPoint returns the latest sample of a CloudMonitor metric, taking the
// maximum across pods at that timestamp. Returns (nil, 0) when the metric has
// no data.
func ckCMSPoint(ctx context.Context, profile, region, clusterID, metric string) (*float64, int64) {
	end := time.Now().UnixMilli()
	start := end - int64((15 * time.Minute).Milliseconds())

	args := []string{
		"cms", "DescribeMetricList",
		"--Namespace", "acs_clickhouse",
		"--MetricName", metric,
		"--Dimensions", `{"clusterId":"` + clusterID + `"}`,
		"--StartTime", strconv.FormatInt(start, 10),
		"--EndTime", strconv.FormatInt(end, 10),
		"--Period", "60",
		"--RegionId", region,
		"--region", region,
	}

	var resp struct {
		Datapoints string `json:"Datapoints"`
	}
	if !ckCall(ctx, profile, args, &resp) {
		return nil, 0
	}

	var pts []struct {
		Timestamp int64 `json:"timestamp"`
		Value     any   `json:"Value"`
	}
	if json.Unmarshal([]byte(resp.Datapoints), &pts) != nil || len(pts) == 0 {
		return nil, 0
	}

	var latest int64
	for _, p := range pts {
		if p.Timestamp > latest {
			latest = p.Timestamp
		}
	}

	var maxVal float64
	found := false
	for _, p := range pts {
		if p.Timestamp != latest {
			continue
		}
		v, ok := ckToFloat(p.Value)
		if !ok {
			continue
		}
		if !found || v > maxVal {
			maxVal = v
			found = true
		}
	}
	if !found {
		return nil, 0
	}
	return &maxVal, latest
}

func ckToFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	}
	return 0, false
}
