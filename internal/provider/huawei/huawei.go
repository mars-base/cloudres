package huawei

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

func (d *Detector) Name() string { return "huawei" }

func (d *Detector) ResourceTypes() []string {
	return []string{"ecs", "vpc", "subnet", "rds", "dcs", "evs", "eip"}
}

func (d *Detector) Fetchers() []core.ResourceFetcher {
	return []core.ResourceFetcher{
		&ECSFetcher{},
		&VPCFetcher{},
		&SubnetFetcher{},
		&RDSFetcher{},
		&DCSFetcher{},
		&EVSFetcher{},
		&EIPFetcher{},
	}
}

func (d *Detector) Detect(ctx context.Context) (*core.Provider, error) {
	cliPath, err := exec.LookPath("hcloud")
	if err != nil {
		return nil, nil // not installed
	}

	configPath := huaweiConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil // no config
	}

	profiles, profileRegions, currentProfile, err := parseHuaweiConfig(configPath)
	if err != nil {
		return nil, nil
	}

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

	active := currentProfile
	if active == "" && len(profiles) > 0 {
		active = profiles[0]
	}

	return &core.Provider{
		Name:           "huawei",
		CLIPath:        cliPath,
		ConfigPath:     configPath,
		Profiles:       profiles,
		Regions:        regions,
		ProfileRegions: profileRegions,
		ActiveProfile:  active,
	}, nil
}

func huaweiConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hcloud", "config.json")
}

type huaweiConfig struct {
	Current  string `json:"current"`
	Language string `json:"language"`
	Profiles []struct {
		Name      string `json:"name"`
		Mode      string `json:"mode"`
		Region    string `json:"region"`
		ProjectID string `json:"projectId"`
	} `json:"profiles"`
}

func parseHuaweiConfig(path string) (profiles []string, profileRegions map[string][]string, current string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, "", err
	}

	var cfg huaweiConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, "", err
	}

	profileRegions = make(map[string][]string)
	for _, p := range cfg.Profiles {
		profiles = append(profiles, p.Name)
		var regions []string
		if p.Region != "" {
			regions = append(regions, p.Region)
		}
		profileRegions[p.Name] = regions
	}

	return profiles, profileRegions, cfg.Current, nil
}
