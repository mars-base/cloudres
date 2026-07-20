// Package core — resource type display labels, kept provider-specific
// since the same short code can mean different things (or not exist)
// across cloud vendors.
package core

// resourceTypeLabels maps provider name -> resource type code -> a short
// human-readable label, used by the TUI to hint what each code means when
// prompting for a resource type (e.g. "vsw" -> "VSwitch").
var resourceTypeLabels = map[string]map[string]string{
	"aliyun": {
		"ecs": "Elastic Compute Service",
		"vpc": "Virtual Private Cloud",
		"vsw": "VSwitch (Subnet)",
		"rds": "Relational Database Service",
		"oss": "Object Storage Service",
	},
}

// ResourceTypeLabel returns the human-readable label for a provider's
// resource type code, or "" if the provider/code isn't known — callers
// should fall back to showing the bare code in that case.
func ResourceTypeLabel(provider, resourceType string) string {
	return resourceTypeLabels[provider][resourceType]
}
