// Package provider implements the provider registry and cloud provider detectors.
package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectConfigVendor inspects a vendor CLI config file and reports which
// cloud provider it belongs to. Vendors are told apart by their profile key
// naming: Aliyun uses snake_case (region_id, access_key_id) while Huawei
// Cloud's KooCLI uses camelCase (projectId, accessKeyId, ...).
func DetectConfigVendor(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read config file: %w", err)
	}

	var probe struct {
		Profiles []map[string]json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return "", fmt.Errorf("parse config file: %w", err)
	}

	aliyun, huawei := false, false
	for _, profile := range probe.Profiles {
		for key := range profile {
			switch key {
			case "region_id", "access_key_id", "access_key_secret":
				aliyun = true
			case "projectId", "accessKeyId", "secretAccessKey", "domainId":
				huawei = true
			}
		}
	}

	switch {
	case aliyun && huawei:
		return "", fmt.Errorf("cannot detect vendor: profiles contain both Aliyun (region_id/access_key_id) and Huawei (projectId/accessKeyId) keys")
	case aliyun:
		return "aliyun", nil
	case huawei:
		return "huawei", nil
	}

	// Fallback: the file's parent directory may hint the vendor even when the
	// profile keys look unfamiliar (e.g. an empty or future-format config).
	dir := strings.ToLower(filepath.Base(filepath.Dir(path)))
	switch {
	case strings.Contains(dir, ".aliyun"):
		return "aliyun", nil
	case strings.Contains(dir, ".hcloud"):
		return "huawei", nil
	}
	return "", fmt.Errorf("cannot detect vendor: no recognizable Aliyun or Huawei profile keys and no .aliyun/.hcloud directory hint")
}
