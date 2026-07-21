package core

import (
	"encoding/json"
	"fmt"
	"strconv"
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
		InstanceType  string `json:"InstanceType"`
		VpcAttributes struct {
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
	return []string{r.ResourceID, r.ResourceName, r.Status, d.InstanceType, ip}
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
		InstanceType  string `json:"InstanceType"`
		Cpu           int    `json:"Cpu"`
		Memory        int    `json:"Memory"`
		ZoneId        string `json:"ZoneId"`
		OSName        string `json:"OSName"`
		CreationTime  string `json:"CreationTime"`
		ExpiredTime   string `json:"ExpiredTime"`
		VpcAttributes struct {
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
	return [][2]string{
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
		{"Created", d.CreationTime},
		{"Expires", d.ExpiredTime},
	}
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
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
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
