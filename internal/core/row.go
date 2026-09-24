package core

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// formatBytes renders a byte count as a human-readable size (GB, with one
// decimal place, since DB/cache usage figures are reported in bytes but
// meaningfully compared in gigabytes). Returns "-" for non-positive/unknown
// values, e.g. when the underlying API omitted the field.
func formatBytes(b int64) string {
	if b <= 0 {
		return "-"
	}
	const gb = 1024 * 1024 * 1024
	return fmt.Sprintf("%.1fGB", float64(b)/gb)
}

// formatMillis renders a millisecond timestamp as "2006-01-02 15:04".
// CMS APIs (contact/rule metadata) return times as epoch millis rather
// than ISO strings like other Aliyun products do.
func formatMillis(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04")
}

// formatPercent renders a used/quota ratio as a percentage with one decimal
// place. Returns "-" if quota is non-positive (unknown/no data).
func formatPercent(used, quota int64) string {
	if quota <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", float64(used)/float64(quota)*100)
}

func (r Resource) ecsRow() []string {
	var d struct {
		InstanceType     string `json:"InstanceType"`
		AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
		RenewalStatus    string `json:"RenewalStatus"`
		VpcAttributes    struct {
			PrivateIpAddress struct {
				IpAddress []string `json:"IpAddress"`
			} `json:"PrivateIpAddress"`
		} `json:"VpcAttributes"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	ip := ""
	if len(d.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		ip = d.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}
	autoRenew := "-"
	if d.RenewalStatus != "" {
		if d.AutoRenewEnabled {
			autoRenew = "On"
		} else {
			autoRenew = "Off"
		}
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, d.InstanceType, ip, autoRenew}
}

func (r Resource) vpcRow() []string {
	var d struct {
		CidrBlock string `json:"CidrBlock"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, d.CidrBlock, r.Status}
}

func (r Resource) vswRow() []string {
	var d struct {
		CidrBlock string `json:"CidrBlock"`
		ZoneId    string `json:"ZoneId"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, d.CidrBlock, d.ZoneId, r.Status}
}

func (r Resource) rdsRow() []string {
	var d struct {
		Engine   string `json:"Engine"`
		DiskUsed int64  `json:"DiskUsed"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Engine, formatBytes(d.DiskUsed)}
}

func (r Resource) tairRow() []string {
	var d struct {
		InstanceType  string `json:"InstanceType"`
		EditionType   string `json:"EditionType"`
		EngineVersion string `json:"EngineVersion"`
		UsedMemory    int64  `json:"UsedMemory"`
		QuotaMemory   int64  `json:"QuotaMemory"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.InstanceType, d.EditionType, d.EngineVersion, formatPercent(d.UsedMemory, d.QuotaMemory)}
}

func (r Resource) polarDBRow() []string {
	var d struct {
		Engine         string `json:"Engine"`
		StorageUsed    int64  `json:"StorageUsed"`
		StorageSpace   int64  `json:"StorageSpace"`
		StoragePayType string `json:"StoragePayType"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	// StorageSpace is only a meaningful cap for Prepaid (包年包月) storage —
	// for Postpaid (按量付费/弹性存储) it's not a real limit and StorageUsed
	// can legitimately exceed it, so showing "used/space" (and a % of it)
	// there is misleading.
	var usage string
	if d.StoragePayType == "Prepaid" {
		usage = formatBytes(d.StorageUsed) + "/" + formatBytes(d.StorageSpace) +
			" (" + formatPercent(d.StorageUsed, d.StorageSpace) + ")"
	} else {
		usage = formatBytes(d.StorageUsed)
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Engine, usage}
}

func (r Resource) ossRow() []string {
	var d struct {
		StorageClass string `json:"StorageClass"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceName, r.Region, d.StorageClass}
}

func (r Resource) slbRow() []string {
	var d struct {
		AddressType string `json:"AddressType"`
		Address     string `json:"Address"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.AddressType, d.Address}
}

func (r Resource) albRow() []string {
	var d struct {
		LoadBalancerEdition string `json:"LoadBalancerEdition"`
		DNSName             string `json:"DNSName"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.LoadBalancerEdition, d.DNSName}
}

func (r Resource) nlbRow() []string {
	var d struct {
		LoadBalancerType string `json:"LoadBalancerType"`
		DNSName          string `json:"DNSName"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.LoadBalancerType, d.DNSName}
}

func (r Resource) essRow() []string {
	var d struct {
		ActiveCapacity int    `json:"ActiveCapacity"`
		MaxSize        int    `json:"MaxSize"`
		GroupType      string `json:"GroupType"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	capacity := fmt.Sprintf("%d/%d", d.ActiveCapacity, d.MaxSize)
	return []string{r.ResourceID, r.ResourceName, r.Status, capacity, d.GroupType}
}

func (r Resource) ecsDetail() [][2]string {
	var d struct {
		InstanceType     string `json:"InstanceType"`
		Cpu              int    `json:"Cpu"`
		Memory           int    `json:"Memory"`
		ZoneId           string `json:"ZoneId"`
		OSName           string `json:"OSName"`
		CreationTime     string `json:"CreationTime"`
		ExpiredTime      string `json:"ExpiredTime"`
		AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
		RenewalStatus    string `json:"RenewalStatus"`
		Duration         int    `json:"Duration"`
		PeriodUnit       string `json:"PeriodUnit"`
		VpcAttributes    struct {
			PrivateIpAddress struct {
				IpAddress []string `json:"IpAddress"`
			} `json:"PrivateIpAddress"`
			VpcId     string `json:"VpcId"`
			VSwitchId string `json:"VSwitchId"`
		} `json:"VpcAttributes"`
		PublicIpAddress struct {
			IpAddress []string `json:"IpAddress"`
		} `json:"PublicIpAddress"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	privIP := ""
	if len(d.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		privIP = d.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}
	pubIP := ""
	if len(d.PublicIpAddress.IpAddress) > 0 {
		pubIP = d.PublicIpAddress.IpAddress[0]
	}
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.ZoneId},
		{"Type", d.InstanceType},
		{"CPU", itoa(d.Cpu)},
		{"Memory(MB)", itoa(d.Memory)},
		{"OS", d.OSName},
		{"PrivateIP", privIP},
		{"PublicIP", pubIP},
		{"VPC", d.VpcAttributes.VpcId},
		{"VSwitch", d.VpcAttributes.VSwitchId},
	}
	if d.RenewalStatus != "" {
		autoRenewStatus := "Off"
		if d.AutoRenewEnabled {
			autoRenewStatus = "On"
		}
		pairs = append(pairs, [2]string{"AutoRenew", autoRenewStatus})
		pairs = append(pairs, [2]string{"RenewalStatus", d.RenewalStatus})
		if d.Duration > 0 {
			pairs = append(pairs, [2]string{"RenewDuration", fmt.Sprintf("%d %s", d.Duration, d.PeriodUnit)})
		}
	}
	pairs = append(pairs, [2]string{"Created", d.CreationTime})
	pairs = append(pairs, [2]string{"Expires", d.ExpiredTime})
	return pairs
}

func (r Resource) vpcDetail() [][2]string {
	var d struct {
		CidrBlock     string `json:"CidrBlock"`
		CreationTime  string `json:"CreationTime"`
		Ipv6CidrBlock string `json:"Ipv6CidrBlock"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"CIDR", d.CidrBlock},
		{"IPv6 CIDR", d.Ipv6CidrBlock},
		{"Created", d.CreationTime},
	}
}

func (r Resource) vswDetail() [][2]string {
	var d struct {
		CidrBlock    string `json:"CidrBlock"`
		ZoneId       string `json:"ZoneId"`
		VpcId        string `json:"VpcId"`
		CreationTime string `json:"CreationTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.ZoneId},
		{"CIDR", d.CidrBlock},
		{"VPC", d.VpcId},
		{"Created", d.CreationTime},
	}
}

func (r Resource) rdsDetail() [][2]string {
	var d struct {
		Engine                string `json:"Engine"`
		EngineVersion         string `json:"EngineVersion"`
		DBInstanceClass       string `json:"DBInstanceClass"`
		DBInstanceStorageType string `json:"DBInstanceStorageType"`
		DBInstanceMemory      int    `json:"DBInstanceMemory"`
		PayType               string `json:"PayType"`
		ConnectionString      string `json:"ConnectionString"`
		ZoneId                string `json:"ZoneId"`
		VpcId                 string `json:"VpcId"`
		CreateTime            string `json:"CreateTime"`
		ExpireTime            string `json:"ExpireTime"`
		DataSize              int64  `json:"DataSize"`
		DiskUsed              int64  `json:"DiskUsed"`
		BackupSize            int64  `json:"BackupSize"`
		IPArrayList           []struct {
			Name  string `json:"DBInstanceIPArrayName"`
			IPs   string `json:"SecurityIPList"`
			Attrs string `json:"DBInstanceIPArrayAttribute"`
		} `json:"ip_arrays"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.ZoneId},
		{"Engine", d.Engine + " " + d.EngineVersion},
		{"Class", d.DBInstanceClass},
		{"Memory(MB)", itoa(d.DBInstanceMemory)},
		{"Storage", d.DBInstanceStorageType},
		{"DiskUsed", formatBytes(d.DiskUsed)},
		{"DataSize", formatBytes(d.DataSize)},
		{"BackupSize", formatBytes(d.BackupSize)},
		{"PayType", d.PayType},
		{"Endpoint", d.ConnectionString},
		{"VPC", d.VpcId},
		{"Created", d.CreateTime},
		{"Expires", d.ExpireTime},
	}

	// Whitelist (ACL) groups: one line per group, name as label and its
	// comma-separated IP/CIDR list as the value (wrapped on commas).
	for _, g := range d.IPArrayList {
		label := "ACL:" + g.Name
		if g.Attrs == "hidden" {
			label += " (hidden)"
		}
		pairs = append(pairs, [2]string{label, g.IPs})
	}

	return pairs
}

func (r Resource) ossDetail() [][2]string {
	var d struct {
		StorageClass string `json:"StorageClass"`
		CreationTime string `json:"CreationTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"Bucket", r.ResourceName},
		{"Region", r.Region},
		{"StorageClass", d.StorageClass},
		{"Created", d.CreationTime},
	}
}

func (r Resource) tairDetail() [][2]string {
	var d struct {
		InstanceType      string `json:"InstanceType"`
		EditionType       string `json:"EditionType"`
		EngineVersion     string `json:"EngineVersion"`
		InstanceClass     string `json:"InstanceClass"`
		ShardCount        int    `json:"ShardCount"`
		RealInstanceClass string `json:"RealInstanceClass"`
		Capacity          int64  `json:"Capacity"`
		Bandwidth         int    `json:"Bandwidth"`
		Connections       int64  `json:"Connections"`
		QPS               int64  `json:"QPS"`
		ZoneId            string `json:"ZoneId"`
		VpcId             string `json:"VpcId"`
		VSwitchId         string `json:"VSwitchId"`
		ChargeType        string `json:"ChargeType"`
		CreateTime        string `json:"CreateTime"`
		EndTime           string `json:"EndTime"`
		UsedMemory        int64  `json:"UsedMemory"`
		QuotaMemory       int64  `json:"QuotaMemory"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.ZoneId},
		{"Type", d.InstanceType},
		{"Edition", d.EditionType},
		{"Version", d.EngineVersion},
		{"Class", d.InstanceClass},
	}
	if d.RealInstanceClass != "" {
		pairs = append(pairs, [2]string{"RealClass", d.RealInstanceClass})
	}
	// Capacity is total memory (MB); show total + per-shard when shard count > 1.
	if d.Capacity > 0 {
		if d.ShardCount > 1 {
			perShard := formatBytes(d.Capacity * 1024 * 1024 / int64(d.ShardCount))
			pairs = append(pairs, [2]string{"Capacity", fmt.Sprintf("%s (%d shards × %s)",
				formatBytes(d.Capacity*1024*1024), d.ShardCount, perShard)})
		} else {
			pairs = append(pairs, [2]string{"Capacity", formatBytes(d.Capacity * 1024 * 1024)})
		}
	} else if d.ShardCount > 0 {
		pairs = append(pairs, [2]string{"Shards", itoa(d.ShardCount)})
	}
	if d.Bandwidth > 0 {
		pairs = append(pairs, [2]string{"Bandwidth", fmt.Sprintf("%d MB/s", d.Bandwidth)})
	}
	if d.Connections > 0 {
		pairs = append(pairs, [2]string{"MaxConnections", fmt.Sprintf("%d", d.Connections)})
	}
	if d.QPS > 0 {
		pairs = append(pairs, [2]string{"QPS", fmt.Sprintf("%d", d.QPS)})
	}
	pairs = append(pairs, [][2]string{
		{"MemoryUsed", formatBytes(d.UsedMemory)},
		{"MemoryQuota", formatBytes(d.QuotaMemory)},
		{"MemoryUsage", formatPercent(d.UsedMemory, d.QuotaMemory)},
		{"ChargeType", d.ChargeType},
		{"VPC", d.VpcId},
		{"VSwitch", d.VSwitchId},
		{"Created", d.CreateTime},
		{"Expires", d.EndTime},
	}...)
	return pairs
}

func (r Resource) polarDBDetail() [][2]string {
	var d struct {
		Engine                string `json:"Engine"`
		DBVersion             string `json:"DBVersion"`
		DBNodeClass           string `json:"DBNodeClass"`
		DBNodeNumber          string `json:"DBNodeNumber"`
		PayType               string `json:"PayType"`
		CreateTime            string `json:"CreateTime"`
		ExpireTime            string `json:"ExpireTime"`
		StorageUsed           int64  `json:"StorageUsed"`
		StorageSpace          int64  `json:"StorageSpace"`
		StorageType           string `json:"StorageType"`
		StoragePayType        string `json:"StoragePayType"`
		PrimaryEndpoint       string `json:"PrimaryEndpoint"`
		PrimaryEndpointPublic string `json:"PrimaryEndpointPublic"`
		ClusterEndpoint       string `json:"ClusterEndpoint"`
		DBNodes               []struct {
			DBNodeRole  string `json:"DBNodeRole"`
			CpuCores    string `json:"CpuCores"`
			MemorySize  string `json:"MemorySize"`
			DBNodeClass string `json:"DBNodeClass"`
			ZoneId      string `json:"ZoneId"`
		} `json:"DBNodes"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	storageSpace := "-"
	if d.StoragePayType == "Prepaid" {
		storageSpace = formatBytes(d.StorageSpace) + " (prepaid)"
	} else if d.StoragePayType != "" {
		storageSpace = "- (postpaid, no fixed cap)"
	}
	usagePct := "-"
	if d.StoragePayType == "Prepaid" {
		usagePct = formatPercent(d.StorageUsed, d.StorageSpace)
	}

	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Engine", d.Engine + " " + d.DBVersion},
		{"NodeClass", d.DBNodeClass},
		{"NodeCount", d.DBNodeNumber},
		{"StorageType", d.StorageType},
		{"StorageUsed", formatBytes(d.StorageUsed)},
		{"StorageSpace", storageSpace},
		{"StorageUsagePct", usagePct},
		{"PayType", d.PayType},
		{"PrimaryEndpoint", d.PrimaryEndpoint},
		{"PrimaryEndpointPublic", d.PrimaryEndpointPublic},
		{"ClusterEndpoint", d.ClusterEndpoint},
	}

	for i, n := range d.DBNodes {
		label := fmt.Sprintf("Node-%d", i+1)
		// MemorySize is in MB; convert to GB for readability.
		memGB := "-"
		if memMB, err := strconv.ParseInt(n.MemorySize, 10, 64); err == nil && memMB > 0 {
			memGB = fmt.Sprintf("%dGB", memMB/1024)
		}
		value := fmt.Sprintf("%s %sC/%s %s %s",
			n.DBNodeRole, n.CpuCores, memGB, n.DBNodeClass, n.ZoneId)
		pairs = append(pairs, [2]string{label, value})
	}

	pairs = append(pairs, [][2]string{
		{"Created", d.CreateTime},
		{"Expires", d.ExpireTime},
	}...)

	return pairs
}

func (r Resource) slbDetail() [][2]string {
	var d struct {
		AddressType      string `json:"AddressType"`
		Address          string `json:"Address"`
		NetworkType      string `json:"NetworkType"`
		LoadBalancerSpec string `json:"LoadBalancerSpec"`
		Bandwidth        int    `json:"Bandwidth"`
		VpcId            string `json:"VpcId"`
		VSwitchId        string `json:"VSwitchId"`
		MasterZoneId     string `json:"MasterZoneId"`
		SlaveZoneId      string `json:"SlaveZoneId"`
		CreateTime       string `json:"CreateTime"`
		PayType          string `json:"PayType"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"AddressType", d.AddressType},
		{"Address", d.Address},
		{"NetworkType", d.NetworkType},
		{"Spec", d.LoadBalancerSpec},
		{"Bandwidth(Mbps)", fmt.Sprintf("%d", d.Bandwidth)},
		{"VPC", d.VpcId},
		{"VSwitch", d.VSwitchId},
		{"MasterZone", d.MasterZoneId},
		{"SlaveZone", d.SlaveZoneId},
		{"PayType", d.PayType},
		{"Created", d.CreateTime},
	}
}

func (r Resource) albDetail() [][2]string {
	var d struct {
		DNSName             string `json:"DNSName"`
		AddressType         string `json:"AddressType"`
		LoadBalancerEdition string `json:"LoadBalancerEdition"`
		VpcId               string `json:"VpcId"`
		CreateTime          string `json:"CreateTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"AddressType", d.AddressType},
		{"Edition", d.LoadBalancerEdition},
		{"DNS", d.DNSName},
		{"VPC", d.VpcId},
		{"Created", d.CreateTime},
	}
}

func (r Resource) nlbDetail() [][2]string {
	var d struct {
		DNSName          string `json:"DNSName"`
		AddressType      string `json:"AddressType"`
		LoadBalancerType string `json:"LoadBalancerType"`
		VpcId            string `json:"VpcId"`
		RegionId         string `json:"RegionId"`
		CreateTime       string `json:"CreateTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", d.RegionId},
		{"AddressType", d.AddressType},
		{"Type", d.LoadBalancerType},
		{"DNS", d.DNSName},
		{"VPC", d.VpcId},
		{"Created", d.CreateTime},
	}
}

func (r Resource) essDetail() [][2]string {
	var d struct {
		ActiveCapacity int    `json:"ActiveCapacity"`
		MaxSize        int    `json:"MaxSize"`
		MinSize        int    `json:"MinSize"`
		GroupType      string `json:"GroupType"`
		CreationTime   string `json:"CreationTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"GroupType", d.GroupType},
		{"ActiveCapacity", fmt.Sprintf("%d", d.ActiveCapacity)},
		{"MinSize", fmt.Sprintf("%d", d.MinSize)},
		{"MaxSize", fmt.Sprintf("%d", d.MaxSize)},
		{"Created", d.CreationTime},
	}
}

func (r Resource) ramRow() []string {
	var d struct {
		DisplayName string `json:"DisplayName"`
		CreateDate  string `json:"CreateDate"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	display := d.DisplayName
	if display == "" {
		display = "-"
	}
	return []string{r.ResourceID, r.ResourceName, display, d.CreateDate}
}

func (r Resource) ramDetail() [][2]string {
	var d struct {
		UserName    string `json:"UserName"`
		DisplayName string `json:"DisplayName"`
		Comments    string `json:"Comments"`
		CreateDate  string `json:"CreateDate"`
		UpdateDate  string `json:"UpdateDate"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"UserID", r.ResourceID},
		{"UserName", d.UserName},
		{"DisplayName", d.DisplayName},
		{"Comments", d.Comments},
		{"Created", d.CreateDate},
		{"Updated", d.UpdateDate},
	}
}

func (r Resource) cmsContactRow() []string {
	var d struct {
		Channels struct {
			Mail string `json:"Mail"`
			SMS  string `json:"SMS"`
		} `json:"Channels"`
		ContactGroups struct {
			ContactGroup []string `json:"ContactGroup"`
		} `json:"ContactGroups"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	mail := d.Channels.Mail
	if mail == "" {
		mail = "-"
	}
	sms := d.Channels.SMS
	if sms == "" {
		sms = "-"
	}
	groups := strings.Join(d.ContactGroups.ContactGroup, ",")
	if groups == "" {
		groups = "-"
	}
	return []string{r.ResourceID, r.ResourceName, mail, sms, groups}
}

func (r Resource) cmsContactDetail() [][2]string {
	var d struct {
		Desc       string `json:"Desc"`
		CreateTime int64  `json:"CreateTime"`
		UpdateTime int64  `json:"UpdateTime"`
		Channels   struct {
			Mail string `json:"Mail"`
			SMS  string `json:"SMS"`
		} `json:"Channels"`
		ChannelsState struct {
			Mail string `json:"Mail"`
			SMS  string `json:"SMS"`
		} `json:"ChannelsState"`
		ContactGroups struct {
			ContactGroup []string `json:"ContactGroup"`
		} `json:"ContactGroups"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
	}
	if d.Desc != "" {
		pairs = append(pairs, [2]string{"Desc", d.Desc})
	}
	if d.Channels.Mail != "" {
		state := d.ChannelsState.Mail
		if state == "" {
			state = "Unknown"
		}
		pairs = append(pairs, [2]string{"Mail", d.Channels.Mail + " (" + state + ")"})
	}
	if d.Channels.SMS != "" {
		state := d.ChannelsState.SMS
		if state == "" {
			state = "Unknown"
		}
		pairs = append(pairs, [2]string{"SMS", d.Channels.SMS + " (" + state + ")"})
	}
	if len(d.ContactGroups.ContactGroup) > 0 {
		pairs = append(pairs, [2]string{"Groups", strings.Join(d.ContactGroups.ContactGroup, ",")})
	}
	pairs = append(pairs, [2]string{"Created", formatMillis(d.CreateTime)})
	pairs = append(pairs, [2]string{"Updated", formatMillis(d.UpdateTime)})
	return pairs
}

func (r Resource) cmsCGRow() []string {
	var d struct {
		CreateTime int64 `json:"CreateTime"`
		Contacts   struct {
			Contact []string `json:"Contact"`
		} `json:"Contacts"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	contacts := strings.Join(d.Contacts.Contact, ",")
	if contacts == "" {
		contacts = "-"
	}
	return []string{r.ResourceName, contacts, formatMillis(d.CreateTime)}
}

func (r Resource) cmsCGDetail() [][2]string {
	var d struct {
		Describe            string `json:"Describe"`
		CreateTime          int64  `json:"CreateTime"`
		UpdateTime          int64  `json:"UpdateTime"`
		EnableSubscribed    bool   `json:"EnableSubscribed"`
		EnabledWeeklyReport bool   `json:"EnabledWeeklyReport"`
		Contacts            struct {
			Contact []string `json:"Contact"`
		} `json:"Contacts"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"Name", r.ResourceName},
	}
	for i, c := range d.Contacts.Contact {
		pairs = append(pairs, [2]string{fmt.Sprintf("Contact-%d", i+1), c})
	}
	if d.Describe != "" {
		pairs = append(pairs, [2]string{"Describe", d.Describe})
	}
	pairs = append(pairs, [2]string{"Subscribed", fmt.Sprintf("%v", d.EnableSubscribed)})
	pairs = append(pairs, [2]string{"WeeklyReport", fmt.Sprintf("%v", d.EnabledWeeklyReport)})
	pairs = append(pairs, [2]string{"Created", formatMillis(d.CreateTime)})
	pairs = append(pairs, [2]string{"Updated", formatMillis(d.UpdateTime)})
	return pairs
}

func (r Resource) cmsAlertRow() []string {
	var d struct {
		Namespace     string `json:"Namespace"`
		MetricName    string `json:"MetricName"`
		EnableState   bool   `json:"EnableState"`
		ContactGroups string `json:"ContactGroups"`
		Escalations   struct {
			Critical cmsEscalationView `json:"Critical"`
			Warn     cmsEscalationView `json:"Warn"`
			Info     cmsEscalationView `json:"Info"`
		} `json:"Escalations"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	enabled := "Off"
	if d.EnableState {
		enabled = "On"
	}
	groups := d.ContactGroups
	if groups == "" {
		groups = "-"
	}
	threshold := cmsEscalationSummary(d.Escalations.Critical)
	if threshold == "-" {
		threshold = cmsEscalationSummary(d.Escalations.Warn)
	}
	if threshold == "-" {
		threshold = cmsEscalationSummary(d.Escalations.Info)
	}
	return []string{r.ResourceName, r.Status, enabled, d.Namespace, d.MetricName, threshold, groups}
}

// cmsEscalationView mirrors aliyun.cmsEscalation for display purposes.
type cmsEscalationView struct {
	ComparisonOperator string `json:"ComparisonOperator"`
	Statistics         string `json:"Statistics"`
	Threshold          string `json:"Threshold"`
	Times              int    `json:"Times"`
}

// cmsEscalationSummary renders one escalation level compactly, e.g.
// "Average >= 90 ×3". Returns "-" for an empty level.
func cmsEscalationSummary(e cmsEscalationView) string {
	if e.Threshold == "" && e.Statistics == "" {
		return "-"
	}
	s := e.Statistics + " " + cmsOperatorSymbol(e.ComparisonOperator) + " " + e.Threshold
	if e.Times > 0 {
		s += " ×" + itoa(e.Times)
	}
	return s
}

// cmsOperatorSymbol maps Aliyun comparison operators to compact symbols.
func cmsOperatorSymbol(op string) string {
	switch op {
	case "GreaterThanOrEqualToThreshold":
		return ">="
	case "GreaterThanThreshold":
		return ">"
	case "LessThanOrEqualToThreshold":
		return "<="
	case "LessThanThreshold":
		return "<"
	case "NotEqualToThreshold":
		return "!="
	case "EqualToThreshold":
		return "="
	case "GreaterThanYesterday":
		return ">yesterday"
	case "LessThanYesterday":
		return "<yesterday"
	case "GreaterThanLastWeek":
		return ">lastweek"
	case "LessThanLastWeek":
		return "<lastweek"
	case "":
		return "?"
	default:
		return op
	}
}

func (r Resource) cmsAlertDetail() [][2]string {
	var d struct {
		Namespace         string   `json:"Namespace"`
		MetricName        string   `json:"MetricName"`
		Period            int      `json:"Period"`
		ContactGroups     string   `json:"ContactGroups"`
		EffectiveInterval string   `json:"EffectiveInterval"`
		SilenceTime       int      `json:"SilenceTime"`
		NoDataPolicy      string   `json:"NoDataPolicy"`
		RuleType          string   `json:"RuleType"`
		SourceType        string   `json:"SourceType"`
		GmtCreate         int64    `json:"GmtCreate"`
		GmtUpdate         int64    `json:"GmtUpdate"`
		InstanceIDs       []string `json:"InstanceIDs"`
		Escalations       struct {
			Critical cmsEscalationView `json:"Critical"`
			Warn     cmsEscalationView `json:"Warn"`
			Info     cmsEscalationView `json:"Info"`
		} `json:"Escalations"`
		CompositeExpression struct {
			ExpressionList struct {
				ExpressionList []struct {
					MetricName         string `json:"MetricName"`
					ComparisonOperator string `json:"ComparisonOperator"`
					Statistics         string `json:"Statistics"`
					Threshold          string `json:"Threshold"`
					Period             int    `json:"Period"`
				} `json:"ExpressionList"`
			} `json:"ExpressionList"`
			ExpressionListJoin string `json:"ExpressionListJoin"`
			Level              string `json:"Level"`
			Times              int    `json:"Times"`
		} `json:"CompositeExpression"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Namespace", d.Namespace},
		{"Metric", d.MetricName},
	}
	if d.Period > 0 {
		pairs = append(pairs, [2]string{"Period", fmt.Sprintf("%ds", d.Period)})
	}
	for _, lv := range []struct {
		label string
		esc   cmsEscalationView
	}{
		{"Critical", d.Escalations.Critical},
		{"Warn", d.Escalations.Warn},
		{"Info", d.Escalations.Info},
	} {
		if s := cmsEscalationSummary(lv.esc); s != "-" {
			pairs = append(pairs, [2]string{lv.label, s})
		}
	}
	if len(d.CompositeExpression.ExpressionList.ExpressionList) > 0 {
		for i, e := range d.CompositeExpression.ExpressionList.ExpressionList {
			cond := fmt.Sprintf("%s %s %s %s", e.MetricName, e.Statistics, e.ComparisonOperator, e.Threshold)
			if i > 0 && d.CompositeExpression.ExpressionListJoin != "" {
				cond = d.CompositeExpression.ExpressionListJoin + " " + cond
			}
			pairs = append(pairs, [2]string{fmt.Sprintf("Condition-%d", i+1), cond})
		}
		pairs = append(pairs, [2]string{"Level", d.CompositeExpression.Level})
	}
	if len(d.InstanceIDs) > 0 {
		for i, id := range d.InstanceIDs {
			pairs = append(pairs, [2]string{fmt.Sprintf("Instance-%d", i+1), id})
		}
	}
	if d.ContactGroups != "" {
		pairs = append(pairs, [2]string{"ContactGroups", d.ContactGroups})
	}
	if d.EffectiveInterval != "" {
		pairs = append(pairs, [2]string{"Effective", d.EffectiveInterval})
	}
	if d.SilenceTime > 0 {
		pairs = append(pairs, [2]string{"Silence", fmt.Sprintf("%ds", d.SilenceTime)})
	}
	if d.NoDataPolicy != "" {
		pairs = append(pairs, [2]string{"NoDataPolicy", d.NoDataPolicy})
	}
	if d.RuleType != "" {
		pairs = append(pairs, [2]string{"RuleType", d.RuleType})
	}
	pairs = append(pairs, [2]string{"Created", formatMillis(d.GmtCreate)})
	pairs = append(pairs, [2]string{"Updated", formatMillis(d.GmtUpdate)})
	return pairs
}

// ARMS Alert Contact
func (r Resource) armsContactRow() []string {
	var d struct {
		Email         string `json:"Email"`
		Phone         string `json:"Phone"`
		IsVerify      bool   `json:"IsVerify"`
		IsEmailVerify bool   `json:"IsEmailVerify"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	email := d.Email
	if email == "" {
		email = "-"
	}
	phone := d.Phone
	if phone == "" {
		phone = "-"
	}
	emailStatus := "unverified"
	if d.IsEmailVerify {
		emailStatus = "verified"
	}
	phoneStatus := "unverified"
	if d.IsVerify {
		phoneStatus = "verified"
	}
	return []string{r.ResourceID, r.ResourceName, email, phone, emailStatus, phoneStatus}
}

func (r Resource) armsContactDetail() [][2]string {
	var d struct {
		ContactName   string `json:"ContactName"`
		Email         string `json:"Email"`
		Phone         string `json:"Phone"`
		IsVerify      bool   `json:"IsVerify"`
		IsEmailVerify bool   `json:"IsEmailVerify"`
		ArmsContactID int    `json:"ArmsContactId"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	phoneStatus := "unverified"
	if d.IsVerify {
		phoneStatus = "verified"
	}
	emailStatus := "unverified"
	if d.IsEmailVerify {
		emailStatus = "verified"
	}
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"ArmsContactID", itoa(d.ArmsContactID)},
	}
	if d.Email != "" {
		pairs = append(pairs, [2]string{"Email", d.Email})
		pairs = append(pairs, [2]string{"EmailStatus", emailStatus})
	}
	if d.Phone != "" {
		pairs = append(pairs, [2]string{"Phone", d.Phone})
		pairs = append(pairs, [2]string{"PhoneStatus", phoneStatus})
	}
	return pairs
}

// ARMS Alert Contact Group
func (r Resource) armsContactGroupRow() []string {
	return []string{r.ResourceID, r.ResourceName, r.Status}
}

func (r Resource) armsContactGroupDetail() [][2]string {
	var d struct {
		ContactGroupName   string `json:"ContactGroupName"`
		ArmsContactGroupID int    `json:"ArmsContactGroupId"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"ArmsContactGroupID", itoa(d.ArmsContactGroupID)},
		{"Status", r.Status},
	}
}

// ARMS Alert Rules
func (r Resource) armsAlertRow() []string {
	var d struct {
		AlertLevel string   `json:"AlertLevel"`
		AlertType  int      `json:"AlertType"`
		AlertWays  []string `json:"AlertWays"`
		CreateTime int64    `json:"CreateTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	ways := "-"
	if len(d.AlertWays) > 0 {
		ways = strings.Join(d.AlertWays, ",")
	}
	alertType := armsAlertTypeName(d.AlertType)
	return []string{r.ResourceName, r.Status, r.Region, d.AlertLevel, alertType, ways, formatMillis(d.CreateTime)}
}

func (r Resource) armsAlertDetail() [][2]string {
	var d struct {
		AlertLevel         string   `json:"AlertLevel"`
		AlertType          int      `json:"AlertType"`
		AlertWays          []string `json:"AlertWays"`
		RegionID           string   `json:"RegionId"`
		CreateTime         int64    `json:"CreateTime"`
		UpdateTime         int64    `json:"UpdateTime"`
		HostByAlertManager bool     `json:"HostByAlertManager"`
		AlertRule          struct {
			Operator string `json:"Operator"`
			Rules    []struct {
				Measure  string  `json:"Measure"`
				NValue   int     `json:"NValue"`
				Operator string  `json:"Operator"`
				Value    float64 `json:"Value"`
			} `json:"Rules"`
		} `json:"AlertRule"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", d.RegionID},
		{"Level", d.AlertLevel},
		{"Type", armsAlertTypeName(d.AlertType)},
	}
	if len(d.AlertWays) > 0 {
		pairs = append(pairs, [2]string{"AlertWays", strings.Join(d.AlertWays, ",")})
	}
	if d.HostByAlertManager {
		pairs = append(pairs, [2]string{"AlertManager", "true"})
	}
	for i, rule := range d.AlertRule.Rules {
		label := fmt.Sprintf("Rule-%d", i+1)
		desc := rule.Measure
		if desc == "" {
			desc = fmt.Sprintf("%s %.2f (N=%d)", rule.Operator, rule.Value, rule.NValue)
		}
		pairs = append(pairs, [2]string{label, desc})
	}
	pairs = append(pairs, [2]string{"Created", formatMillis(d.CreateTime)})
	pairs = append(pairs, [2]string{"Updated", formatMillis(d.UpdateTime)})
	return pairs
}

// armsAlertTypeName maps ARMS AlertType numeric codes to short labels.
func armsAlertTypeName(t int) string {
	switch t {
	case 5:
		return "app"
	case 7:
		return "prometheus"
	case 101:
		return "prometheus-hosted"
	default:
		return fmt.Sprintf("type-%d", t)
	}
}

// ── CAS (Certificate) ───────────────────────────────────────────────

func (r Resource) casRow() []string {
	var d struct {
		Domain            string `json:"Domain"`
		CertificateStatus string `json:"CertificateStatus"`
		Issuer            string `json:"Issuer"`
		NotAfter          int64  `json:"NotAfter"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	expire := formatMillis(d.NotAfter)
	return []string{r.ResourceID, r.ResourceName, d.Domain, d.CertificateStatus, d.Issuer, expire}
}

func (r Resource) casDetail() [][2]string {
	var d struct {
		CertificateId     string   `json:"CertificateId"`
		CertificateName   string   `json:"CertificateName"`
		Domain            string   `json:"Domain"`
		CommonName        string   `json:"CommonName"`
		Issuer            string   `json:"Issuer"`
		CertificateStatus string   `json:"CertificateStatus"`
		CertificateSource string   `json:"CertificateSource"`
		NotBefore         int64    `json:"NotBefore"`
		NotAfter          int64    `json:"NotAfter"`
		Algorithm         string   `json:"Algorithm"`
		KeySize           int      `json:"KeySize"`
		UsingProductList  []string `json:"UsingProductList"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", d.CertificateName},
		{"Region", r.Region},
		{"Domain", d.Domain},
		{"CommonName", d.CommonName},
		{"Status", d.CertificateStatus},
		{"Source", d.CertificateSource},
		{"Issuer", d.Issuer},
		{"Algorithm", fmt.Sprintf("%s %d", d.Algorithm, d.KeySize)},
		{"NotBefore", formatMillis(d.NotBefore)},
		{"NotAfter", formatMillis(d.NotAfter)},
	}
	if len(d.UsingProductList) > 0 {
		pairs = append(pairs, [2]string{"UsingProducts", strings.Join(d.UsingProductList, ", ")})
	}
	return pairs
}

// ── ACK (Kubernetes Cluster) ────────────────────────────────────────────

func (r Resource) ackRow() []string {
	var d struct {
		ClusterType    string `json:"cluster_type"`
		CurrentVersion string `json:"current_version"`
		Size           int    `json:"size"`
		Created        string `json:"created"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	created := d.Created
	if len(created) > 10 {
		created = created[:10] // keep date only
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, d.CurrentVersion, d.ClusterType, r.Region, fmt.Sprintf("%d", d.Size), created}
}

func (r Resource) ackDetail() [][2]string {
	var d struct {
		ClusterID      string `json:"cluster_id"`
		Name           string `json:"name"`
		ClusterType    string `json:"cluster_type"`
		ClusterSpec    string `json:"cluster_spec"`
		State          string `json:"state"`
		CurrentVersion string `json:"current_version"`
		RegionID       string `json:"region_id"`
		Size           int    `json:"size"`
		Created        string `json:"created"`
		NetworkMode    string `json:"network_mode"`
		ProxyMode      string `json:"proxy_mode"`
		ServiceCIDR    string `json:"service_cidr"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	return [][2]string{
		{"ID", d.ClusterID},
		{"Name", d.Name},
		{"Region", d.RegionID},
		{"Status", d.State},
		{"Type", d.ClusterType},
		{"Spec", d.ClusterSpec},
		{"Version", d.CurrentVersion},
		{"Nodes", fmt.Sprintf("%d", d.Size)},
		{"NetworkMode", d.NetworkMode},
		{"ProxyMode", d.ProxyMode},
		{"ServiceCIDR", d.ServiceCIDR},
		{"Created", d.Created},
	}
}

// ── ACR (Container Registry) ────────────────────────────────────────────

func (r Resource) acrRow() []string {
	var d struct {
		InstanceSpecification string   `json:"InstanceSpecification"`
		RegionID              string   `json:"RegionId"`
		CreateTime            int64    `json:"CreateTime"`
		Namespaces            []string `json:"namespaces"`
		RepoCount             int      `json:"repo_count"`
		ACLCount              int      `json:"acl_count"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{
		r.ResourceID, r.ResourceName, r.Status, d.InstanceSpecification,
		d.RegionID,
		itoa(len(d.Namespaces)),
		itoa(d.RepoCount),
		itoa(d.ACLCount),
		formatMillis(d.CreateTime),
	}
}

func (r Resource) acrDetail() [][2]string {
	var d struct {
		InstanceID            string   `json:"InstanceId"`
		InstanceName          string   `json:"InstanceName"`
		InstanceSpecification string   `json:"InstanceSpecification"`
		InstanceStatus        string   `json:"InstanceStatus"`
		RegionID              string   `json:"RegionId"`
		CreateTime            int64    `json:"CreateTime"`
		ModifiedTime          int64    `json:"ModifiedTime"`
		ResourceGroupID       string   `json:"ResourceGroupId"`
		Namespaces            []string `json:"namespaces"`
		RepoCount             int      `json:"repo_count"`
		Endpoint              string   `json:"endpoint"`
		ACLCount              int      `json:"acl_count"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	pairs := [][2]string{
		{"ID", d.InstanceID},
		{"Name", d.InstanceName},
		{"Region", d.RegionID},
		{"Status", d.InstanceStatus},
		{"Spec", d.InstanceSpecification},
		{"ResourceGroup", d.ResourceGroupID},
		{"Namespaces", strings.Join(d.Namespaces, ", ")},
		{"Repositories", itoa(d.RepoCount)},
		{"Endpoint", d.Endpoint},
		{"ACL Entries", itoa(d.ACLCount)},
		{"Created", formatMillis(d.CreateTime)},
		{"Modified", formatMillis(d.ModifiedTime)},
	}
	return pairs
}

// ── ACL (Access Control List) ───────────────────────────────────────────

func (r Resource) aclRow() []string {
	var d struct {
		CreateTime string   `json:"CreateTime"`
		IPCount    int      `json:"ip_count"`
		ALBs       []string `json:"albs"`
		Listeners  int      `json:"listeners"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	created := d.CreateTime
	if len(created) > 10 {
		created = created[:10]
	}
	return []string{
		r.ResourceID, r.ResourceName, r.Status,
		itoa(d.IPCount), itoa(len(d.ALBs)), itoa(d.Listeners),
		created,
	}
}

func (r Resource) aclDetail() [][2]string {
	var d struct {
		AclID            string   `json:"AclId"`
		AclName          string   `json:"AclName"`
		AclStatus        string   `json:"AclStatus"`
		AddressIPVersion string   `json:"AddressIPVersion"`
		CreateTime       string   `json:"CreateTime"`
		ResourceGroupID  string   `json:"ResourceGroupId"`
		IPCount          int      `json:"ip_count"`
		IPs              []string `json:"ips"`
		ALBs             []string `json:"albs"`
		Listeners        int      `json:"listeners"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	pairs := [][2]string{
		{"ID", d.AclID},
		{"Name", d.AclName},
		{"Region", r.Region},
		{"Status", d.AclStatus},
		{"IPVersion", d.AddressIPVersion},
		{"ResourceGroup", d.ResourceGroupID},
		{"IP Entries", itoa(d.IPCount)},
		{"Bound ALBs", strings.Join(d.ALBs, ", ")},
		{"Listeners", itoa(d.Listeners)},
		{"Created", d.CreateTime},
	}
	// Append each IP as a separate row in the detail view
	for _, ip := range d.IPs {
		pairs = append(pairs, [2]string{"IP", ip})
	}
	return pairs
}

// ── Kafka (Message Queue) ───────────────────────────────────────────────

func (r Resource) kafkaRow() []string {
	var d struct {
		Series        string `json:"Series"`
		CreateTime    int64  `json:"CreateTime"`
		Topics        []any  `json:"topics"`
		ConsumerGroup []any  `json:"consumer_groups"`
		SASLUsers     []any  `json:"sasl_users"`
		ACLs          []any  `json:"acls"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{
		r.ResourceID, r.ResourceName, r.Status, d.Series,
		itoa(len(d.Topics)), itoa(len(d.ConsumerGroup)),
		itoa(len(d.SASLUsers)), itoa(len(d.ACLs)),
		r.Region, formatMillis(d.CreateTime),
	}
}

func (r Resource) kafkaDetail() [][2]string {
	d := r.parseKafka()

	pairs := [][2]string{
		{"ID", d.InstanceID},
		{"Name", d.Name},
		{"Region", d.RegionID},
		{"Status", r.Status},
		{"Series", d.Series},
		{"Spec", d.SpecType},
		{"VPC", d.VpcID},
		{"SecurityGroup", d.SecurityGroup},
		{"Zone", d.StandardZoneID},
		{"MsgRetain(h)", itoa(d.MsgRetain)},
		{"Endpoint", d.DomainEndpoint},
		{"SASLEndpoint", d.SaslEndPoint},
		{"Created", formatMillis(d.CreateTime)},
	}

	// Topics
	for _, t := range d.Topics {
		auto := "no"
		if t.AutoCreate {
			auto = "yes"
		}
		pairs = append(pairs, [2]string{"Topic", fmt.Sprintf("%s  partitions=%d  autocreate=%s", t.Topic, t.PartitionNum, auto)})
	}

	// SASL users
	for _, u := range d.SASLUsers {
		pairs = append(pairs, [2]string{"SASL", fmt.Sprintf("%s  (%s/%s)", u.Username, u.Type, u.Mechanism)})
	}

	// Consumer groups
	for _, c := range d.ConsumerGroup {
		pairs = append(pairs, [2]string{"Group", c.ConsumerGroup})
	}

	// ACLs
	for _, a := range d.ACLs {
		pairs = append(pairs, [2]string{"ACL", fmt.Sprintf("%s  %s %s  %s/%s", a.Username, a.AclPermissionType, a.AclOperationType, a.AclResourceType, a.AclResourceName)})
	}

	return pairs
}

// kafkaDetailFixed pins the Kafka configuration parameters so they stay
// visible while the (potentially long) topic/user/ACL lists scroll. Config
// keys are long (e.g. auto.create.topics.enable), so the renderer gives each
// a full-width line rather than squeezing it into the 24-col label gutter.
func (r Resource) kafkaDetailFixed() [][2]string {
	d := r.parseKafka()

	var pairs [][2]string
	for _, k := range d.AllConfigKeys {
		pairs = append(pairs, [2]string{"Config:" + k, d.AllConfig[k]})
	}

	return pairs
}

// kafkaView is the decoded shape of a Kafka instance's RawJSON. AllConfigKeys
// is derived (sorted) after unmarshal, so it carries no json tag.
type kafkaView struct {
	InstanceID     string            `json:"InstanceId"`
	Name           string            `json:"Name"`
	RegionID       string            `json:"RegionId"`
	SpecType       string            `json:"SpecType"`
	Series         string            `json:"Series"`
	VpcID          string            `json:"VpcId"`
	SecurityGroup  string            `json:"SecurityGroup"`
	StandardZoneID string            `json:"StandardZoneId"`
	CreateTime     int64             `json:"CreateTime"`
	ExpiredTime    int64             `json:"ExpiredTime"`
	DomainEndpoint string            `json:"DomainEndpoint"`
	SaslEndPoint   string            `json:"SaslEndPoint"`
	MsgRetain      int               `json:"MsgRetain"`
	AllConfig      map[string]string `json:"all_config"`
	AllConfigKeys  []string
	Topics         []struct {
		Topic        string `json:"Topic"`
		PartitionNum int    `json:"PartitionNum"`
		AutoCreate   bool   `json:"AutoCreate"`
	} `json:"topics"`
	SASLUsers []struct {
		Username  string `json:"Username"`
		Type      string `json:"Type"`
		Mechanism string `json:"Mechanism"`
	} `json:"sasl_users"`
	ConsumerGroup []struct {
		ConsumerGroup string `json:"ConsumerGroup"`
		Remark        string `json:"Remark"`
	} `json:"consumer_groups"`
	ACLs []struct {
		Username          string `json:"Username"`
		AclResourceType   string `json:"AclResourceType"`
		AclResourceName   string `json:"AclResourceName"`
		AclOperationType  string `json:"AclOperationType"`
		AclPermissionType string `json:"AclPermissionType"`
	} `json:"acls"`
}

// parseKafka decodes the RawJSON of a Kafka instance resource.
func (r Resource) parseKafka() kafkaView {
	var d kafkaView
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	// Sort config keys for stable display.
	keys := make([]string, 0, len(d.AllConfig))
	for k := range d.AllConfig {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	d.AllConfigKeys = keys

	return d
}
