// Package core — resource type display labels, kept provider-specific
// since the same short code can mean different things (or not exist)
// across cloud vendors.
package core

// resourceTypeLabels maps provider name -> resource type code -> a short
// human-readable label, used by the TUI to hint what each code means when
// prompting for a resource type (e.g. "vsw" -> "VSwitch").
var resourceTypeLabels = map[string]map[string]string{
	"aliyun": {
		"ecs":  "Elastic Compute Service",
		"vpc":  "Virtual Private Cloud",
		"vsw":  "VSwitch (Subnet)",
		"rds":  "Relational Database Service",
		"tair": "Tair (Redis-compatible)",
		"pdb":  "PolarDB",
		"oss":  "Object Storage Service",
		"slb":  "Server Load Balancer (CLB)",
		"alb":  "Application Load Balancer",
		"nlb":  "Network Load Balancer",
		"ess":  "Auto Scaling (ESS)",
	},
	"huawei": {
		"ecs":    "弹性云服务器 (ECS)",
		"vpc":    "虚拟私有云 (VPC)",
		"subnet": "子网 (Subnet)",
		"rds":    "云数据库 (RDS)",
		"dcs":    "分布式缓存服务 (DCS/Redis)",
		"evs":    "云硬盘 (EVS)",
		"eip":    "弹性公网IP (EIP)",
	},
}

// ResourceTypeLabel returns the human-readable label for a provider's
// resource type code, or "" if the provider/code isn't known — callers
// should fall back to showing the bare code in that case.
func ResourceTypeLabel(provider, resourceType string) string {
	return resourceTypeLabels[provider][resourceType]
}
