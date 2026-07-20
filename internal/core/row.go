package core

import (
	"encoding/json"
)

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
		Engine string `json:"Engine"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Engine}
}

func (r Resource) tairRow() []string {
	var d struct {
		InstanceType  string `json:"InstanceType"`
		InstanceClass string `json:"InstanceClass"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.InstanceType, d.InstanceClass}
}

func (r Resource) polarDBRow() []string {
	var d struct {
		Engine string `json:"Engine"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceID, r.ResourceName, r.Status, d.Engine}
}

func (r Resource) ossRow() []string {
	var d struct {
		StorageClass string `json:"StorageClass"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return []string{r.ResourceName, r.Region, d.StorageClass}
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
		InstanceType  string `json:"InstanceType"`
		EditionType   string `json:"EditionType"`
		EngineVersion string `json:"EngineVersion"`
		InstanceClass string `json:"InstanceClass"`
		ZoneId        string `json:"ZoneId"`
		VpcId         string `json:"VpcId"`
		VSwitchId     string `json:"VSwitchId"`
		ChargeType    string `json:"ChargeType"`
		CreateTime    string `json:"CreateTime"`
		EndTime       string `json:"EndTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Zone", d.ZoneId},
		{"Type", d.InstanceType},
		{"Edition", d.EditionType},
		{"Version", d.EngineVersion},
		{"Class", d.InstanceClass},
		{"ChargeType", d.ChargeType},
		{"VPC", d.VpcId},
		{"VSwitch", d.VSwitchId},
		{"Created", d.CreateTime},
		{"Expires", d.EndTime},
	}
}

func (r Resource) polarDBDetail() [][2]string {
	var d struct {
		Engine       string `json:"Engine"`
		DBVersion    string `json:"DBVersion"`
		DBNodeClass  string `json:"DBNodeClass"`
		DBNodeNumber string `json:"DBNodeNumber"`
		PayType      string `json:"PayType"`
		CreateTime   string `json:"CreateTime"`
		ExpireTime   string `json:"ExpireTime"`
	}
	_ = json.Unmarshal([]byte(r.RawJSON), &d)
	return [][2]string{
		{"ID", r.ResourceID},
		{"Name", r.ResourceName},
		{"Status", r.Status},
		{"Region", r.Region},
		{"Engine", d.Engine + " " + d.DBVersion},
		{"NodeClass", d.DBNodeClass},
		{"NodeCount", d.DBNodeNumber},
		{"PayType", d.PayType},
		{"Created", d.CreateTime},
		{"Expires", d.ExpireTime},
	}
}
