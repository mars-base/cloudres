package aliyun

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mars-base/cloudres/internal/core"
)

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

func (d *Detector) Name() string { return "aliyun" }

func (d *Detector) ResourceTypes() []string {
	return []string{"ecs", "vpc", "vsw", "rds", "tair", "pdb", "oss"}
}

func (d *Detector) Fetchers() []core.ResourceFetcher {
	return []core.ResourceFetcher{
		&ECSFetcher{},
		&VPCFetcher{},
		&VSwitchFetcher{},
		&RDSFetcher{},
		&TairFetcher{},
		&PolarDBFetcher{},
		&OSSFetcher{},
	}
}

func (d *Detector) Detect(ctx context.Context) (*core.Provider, error) {
	cliPath, err := exec.LookPath("aliyun")
	if err != nil {
		return nil, nil // not installed
	}

	configPath := aliyunConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // no config
	}

	profiles, profileRegions, err := parseAliyunConfig(configPath)
	if err != nil {
		return nil, nil // config parse error, skip silently
	}

	// Build overall regions list
	allRegions := make(map[string]bool)
	for _, regions := range profileRegions {
		for _, r := range regions {
			allRegions[r] = true
		}
	}

	var regions []string
	for r := range allRegions {
		regions = append(regions, r)
	}

	return &core.Provider{
		Name:           "aliyun",
		CLIPath:        cliPath,
		ConfigPath:     configPath,
		Profiles:       profiles,
		Regions:        regions,
		ProfileRegions: profileRegions,
	}, nil
}

// aliyunConfigPath returns ~/.aliyun/config.json
func aliyunConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".aliyun", "config.json")
}

type aliyunConfig struct {
	Current  string `json:"current"`
	Profiles []struct {
		Name     string `json:"name"`
		RegionID string `json:"region_id"`
	} `json:"profiles"`
}

func parseAliyunConfig(path string) (profiles []string, profileRegions map[string][]string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	var cfg aliyunConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}

	profileRegions = make(map[string][]string)
	for _, p := range cfg.Profiles {
		profiles = append(profiles, p.Name)
		var regions []string
		if p.RegionID != "" {
			regions = append(regions, p.RegionID)
		}
		profileRegions[p.Name] = regions
	}

	return profiles, profileRegions, nil
}
