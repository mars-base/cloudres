package core

import (
	"encoding/json"
	"fmt"
)

// ── Huawei Cloud Row (compact table columns) ──────────────────────────

func (r Resource) huaweiRow() []string {
	switch r.ResourceType {
	case "ecs":
		return r.huaweiECSRow()
	case "vpc":
		return r.huaweiVPCRow()
	case "subnet":
		return r.huaweiSubnetRow()
	case "rds":
		return r.huaweiRDSRow()
	case "dcs":
		return r.huaweiDCSRow()
	case "evs":
		return r.huaweiEVSRow()
	case "eip":
		return r.huaweiEIPRow()
	default:
		return []string{r.ResourceID, r.ResourceName, r.Status, r.Region}
	}
}

// ── Huawei Cloud Detail (key-value pairs) ─────────────────────────────

func (r Resource) huaweiDetail() [][2]string {
	switch r.ResourceType {
	case "ecs":
		return r.huaweiECSDetail()
	case "vpc":
		return r.huaweiVPCDetail()
	case "subnet":
		return r.huaweiSubnetDetail()
	case "rds":
		return r.huaweiRDSDetail()
	case "dcs":
		return r.huaweiDCSDetail()
	case "evs":
		return r.huaweiEVSDetail()
	case "eip":
		return r.huaweiEIPDetail()
	default:
		return [][2]string{
			{"ID", r.ResourceID},
			{"Name", r.ResourceName},
			{"Status", r.Status},
			{"Region", r.Region},
		}
	}
}

// ── ECS ────────────────────────────────────────────────────────────────

func (r Resource) huaweiECSRow() []string {
	var d struct {
		Flavor struct {
			Name string `json:"name"`
		} `json:"flavor"`
		Addresses map[string][]struct {
			Addr string `json:"addr"`
			Type string `json:"OS-EXT-IPS:type"`
		} `json:"addresses"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	ip := ""
	for _, addrs := range d.Addresses {
		for _, a := range addrs {
			if a.Type == "fixed" && ip == "" {
				ip = a.Addr
			}
		}
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Flavor.Name, ip}
}

func (r Resource) huaweiECSDetail() [][2]string {
	var d struct {
		Flavor struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Vcpus string `json:"vcpus"`
			RAM   string `json:"ram"`
			Disk  string `json:"disk"`
		} `json:"flavor"`
		Addresses map[string][]struct {
			Addr string `json:"addr"`
			Type string `json:"OS-EXT-IPS:type"`
		} `json:"addresses"`
		Metadata struct {
			OsType string `json:"os_type"`
			VpcID  string `json:"vpc_id"`
		} `json:"metadata"`
		AvailabilityZone string   `json:"OS-EXT-AZ:availability_zone"`
		KeyName          string   `json:"key_name"`
		Created          string   `json:"created"`
		Description      string   `json:"description"`
		Tags             []string `json:"tags"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	privIP := ""
	pubIP := ""
	vpcID := d.Metadata.VpcID
	for _, addrs := range d.Addresses {
		for _, a := range addrs {
			if a.Type == "fixed" && privIP == "" {
				privIP = a.Addr
			}
			if a.Type == "floating" && pubIP == "" {
				pubIP = a.Addr
			}
		}
	}

	memMB, _ := parseInt(d.Flavor.RAM)
	memGB := "-"
	if memMB > 0 {
		memGB = fmt.Sprintf("%dGB", memMB/1024)
	}

	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.AvailabilityZone},
		{"Flavor", fmt.Sprintf("%s (%sC/%s)", d.Flavor.Name, d.Flavor.Vcpus, memGB)},
		{"OS", d.Metadata.OsType},
		{"PrivateIP", privIP},
		{"PublicIP", pubIP},
		{"VPC", vpcID},
		{"KeyName", d.KeyName},
		{"Created", d.Created},
		{"Description", d.Description},
	}
}

// ── VPC ────────────────────────────────────────────────────────────────

func (r Resource) huaweiVPCRow() []string {
	var d struct {
		Cidr string `json:"cidr"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, d.Cidr, r.Status}
}

func (r Resource) huaweiVPCDetail() [][2]string {
	var d struct {
		Cidr        string `json:"cidr"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"CIDR", d.Cidr},
		{"Description", d.Description},
		{"Created", d.CreatedAt},
		{"Updated", d.UpdatedAt},
	}
}

// ── Subnet ────────────────────────────────────────────────────────────

func (r Resource) huaweiSubnetRow() []string {
	var d struct {
		Cidr  string `json:"cidr"`
		VpcID string `json:"vpc_id"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, d.Cidr, d.VpcID, r.Status}
}

func (r Resource) huaweiSubnetDetail() [][2]string {
	var d struct {
		Cidr             string `json:"cidr"`
		VpcID            string `json:"vpc_id"`
		GatewayIP        string `json:"gateway_ip"`
		DhcpEnable       bool   `json:"dhcp_enable"`
		AvailabilityZone string `json:"availability_zone"`
		CreatedAt        string `json:"created_at"`
		UpdatedAt        string `json:"updated_at"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	dhcp := "false"
	if d.DhcpEnable {
		dhcp = "true"
	}
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"CIDR", d.Cidr},
		{"VPC", d.VpcID},
		{"Gateway", d.GatewayIP},
		{"Zone", d.AvailabilityZone},
		{"DHCP", dhcp},
		{"Created", d.CreatedAt},
		{"Updated", d.UpdatedAt},
	}
}

// ── RDS ────────────────────────────────────────────────────────────────

func (r Resource) huaweiRDSRow() []string {
	var d struct {
		Datastore struct {
			Type    string `json:"type"`
			Version string `json:"version"`
		} `json:"datastore"`
		Type      string `json:"type"`
		FlavorRef string `json:"flavor_ref"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	engine := d.Datastore.Type + " " + d.Datastore.Version
	return []string{r.ResourceID, r.ResourceName, r.Status, engine, d.Type, d.FlavorRef}
}

func (r Resource) huaweiRDSDetail() [][2]string {
	var d struct {
		Datastore struct {
			Type    string `json:"type"`
			Version string `json:"version"`
		} `json:"datastore"`
		Type       string   `json:"type"`
		FlavorRef  string   `json:"flavor_ref"`
		PrivateIPs []string `json:"private_ips"`
		PublicIPs  []string `json:"public_ips"`
		Port       int      `json:"port"`
		VpcID      string   `json:"vpc_id"`
		SubnetID   string   `json:"subnet_id"`
		CPU        string   `json:"cpu"`
		Mem        string   `json:"mem"`
		Volume     struct {
			Type string `json:"type"`
			Size int    `json:"size"`
		} `json:"volume"`
		Nodes   []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
			Zone string `json:"availability_zone"`
		} `json:"nodes"`
		Created string `json:"created"`
		Updated string `json:"updated"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Engine", d.Datastore.Type + " " + d.Datastore.Version},
		{"Type", d.Type},
		{"Flavor", d.FlavorRef},
		{"CPU", d.CPU},
		{"Memory(MB)", d.Mem},
		{"Volume", fmt.Sprintf("%s %dGB", d.Volume.Type, d.Volume.Size)},
	}
	if len(d.PrivateIPs) > 0 {
		pairs = append(pairs, [2]string{"PrivateIP", d.PrivateIPs[0]})
	}
	if len(d.PublicIPs) > 0 {
		pairs = append(pairs, [2]string{"PublicIP", d.PublicIPs[0]})
	}
	pairs = append(pairs, [][2]string{
		{"Port", fmt.Sprintf("%d", d.Port)},
		{"VPC", d.VpcID},
		{"Subnet", d.SubnetID},
	}...)
	for i, n := range d.Nodes {
		pairs = append(pairs, [2]string{
			fmt.Sprintf("Node-%d", i+1),
			fmt.Sprintf("%s %s %s %s", n.Role, n.Name, n.ID, n.Zone),
		})
	}
	pairs = append(pairs, [][2]string{
		{"Created", d.Created},
		{"Updated", d.Updated},
	}...)
	return pairs
}

// ── DCS ────────────────────────────────────────────────────────────────

func (r Resource) huaweiDCSRow() []string {
	var d struct {
		Engine   string `json:"engine"`
		Capacity int    `json:"capacity"`
		IP       string `json:"ip"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	capStr := "-"
	if d.Capacity > 0 {
		capStr = fmt.Sprintf("%dGB", d.Capacity)
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Engine, capStr, d.IP}
}

func (r Resource) huaweiDCSDetail() [][2]string {
	var d struct {
		Engine     string `json:"engine"`
		EngineVer  string `json:"engine_version"`
		Capacity   int    `json:"capacity"`
		MaxMemory  int    `json:"max_memory"`
		UsedMemory int    `json:"used_memory"`
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		VpcID      string `json:"vpc_id"`
		SubnetID   string `json:"subnet_id"`
		Zone       string `json:"zone"`
		ChargeMode string `json:"charging_mode"`
		CreatedAt  string `json:"created_at"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.Zone},
		{"Engine", d.Engine + " " + d.EngineVer},
		{"Capacity", fmt.Sprintf("%dGB", d.Capacity)},
		{"MaxMemory", fmt.Sprintf("%dMB", d.MaxMemory)},
		{"UsedMemory", fmt.Sprintf("%dMB", d.UsedMemory)},
		{"IP", d.IP},
		{"Port", fmt.Sprintf("%d", d.Port)},
		{"VPC", d.VpcID},
		{"Subnet", d.SubnetID},
		{"ChargeMode", d.ChargeMode},
		{"Created", d.CreatedAt},
	}
}

// ── EVS ────────────────────────────────────────────────────────────────

func (r Resource) huaweiEVSRow() []string {
	var d struct {
		Size        int    `json:"size"`
		VolumeType  string `json:"volume_type"`
		Bootable    string `json:"bootable"`
		Attachments []struct {
			ServerID string `json:"server_id"`
		} `json:"attachments"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	attached := "-"
	if d.Bootable == "true" && len(d.Attachments) > 0 {
		attached = d.Attachments[0].ServerID
	} else if len(d.Attachments) > 0 {
		attached = d.Attachments[0].ServerID
	}
	return []string{r.ResourceID, r.ResourceName, r.Status, fmt.Sprintf("%d", d.Size), d.VolumeType, attached}
}

func (r Resource) huaweiEVSDetail() [][2]string {
	var d struct {
		Size             int    `json:"size"`
		VolumeType       string `json:"volume_type"`
		Bootable         string `json:"bootable"`
		Encrypted        bool   `json:"encrypted"`
		Shareable        string `json:"shareable"`
		Multiattach      bool   `json:"multiattach"`
		AvailabilityZone string `json:"availability_zone"`
		CreatedAt        string `json:"created_at"`
		Attachments      []struct {
			ServerID string `json:"server_id"`
			Device   string `json:"device"`
		} `json:"attachments"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	pairs := [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.AvailabilityZone},
		{"Size", fmt.Sprintf("%dGB", d.Size)},
		{"Type", d.VolumeType},
		{"Bootable", d.Bootable},
		{"Encrypted", fmt.Sprintf("%t", d.Encrypted)},
		{"Shareable", d.Shareable},
	}
	for i, a := range d.Attachments {
		pairs = append(pairs, [2]string{
			fmt.Sprintf("Attach-%d", i+1),
			fmt.Sprintf("%s → %s", a.ServerID, a.Device),
		})
	}
	pairs = append(pairs, [2]string{"Created", d.CreatedAt})
	return pairs
}

// ── EIP ────────────────────────────────────────────────────────────────

func (r Resource) huaweiEIPRow() []string {
	var d struct {
		Type     string `json:"type"`
		Bandwidth struct {
			Size       int    `json:"size"`
			ChargeMode string `json:"charge_mode"`
		} `json:"bandwidth"`
		Vnic struct {
			DeviceID string `json:"device_id"`
		} `json:"vnic"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	bw := fmt.Sprintf("%dM %s", d.Bandwidth.Size, d.Bandwidth.ChargeMode)
	inst := d.Vnic.DeviceID
	if inst == "" {
		inst = "-"
	}
	return []string{r.ResourceName, r.Status, d.Type, bw, inst}
}

func (r Resource) huaweiEIPDetail() [][2]string {
	var d struct {
		PublicIPAddress   string `json:"public_ip_address"`
		PublicIPv6Address string `json:"public_ipv6_address"`
		Type              string `json:"type"`
		IPVersion         int    `json:"ip_version"`
		Description       string `json:"description"`
		CreatedAt         string `json:"created_at"`
		Bandwidth         struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Size       int    `json:"size"`
			ShareType  string `json:"share_type"`
			ChargeMode string `json:"charge_mode"`
		} `json:"bandwidth"`
		Vnic struct {
			PrivateIPAddress string `json:"private_ip_address"`
			DeviceID         string `json:"device_id"`
			VpcID            string `json:"vpc_id"`
			PortID           string `json:"port_id"`
			InstanceID       string `json:"instance_id"`
			InstanceType     string `json:"instance_type"`
		} `json:"vnic"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)

	return [][2]string{
		{"ID", r.ResourceID},
		{"PublicIP", d.PublicIPAddress},
		{"IPv6", d.PublicIPv6Address},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Type", d.Type},
		{"IPVersion", fmt.Sprintf("IPv%d", d.IPVersion)},
		{"Bandwidth", fmt.Sprintf("%dM %s/%s", d.Bandwidth.Size, d.Bandwidth.ChargeMode, d.Bandwidth.ShareType)},
		{"PrivateIP", d.Vnic.PrivateIPAddress},
		{"VPC", d.Vnic.VpcID},
		{"PortID", d.Vnic.PortID},
		{"InstanceID", d.Vnic.InstanceID},
		{"InstanceType", d.Vnic.InstanceType},
		{"DeviceID", d.Vnic.DeviceID},
		{"Description", d.Description},
		{"Created", d.CreatedAt},
	}
}

// parseInt parses a decimal string to int.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
