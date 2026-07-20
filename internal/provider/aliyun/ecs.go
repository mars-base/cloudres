package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type ECSFetcher struct{}

func (f *ECSFetcher) ResourceType() string { return "ecs" }

func (f *ECSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource

	for _, region := range p.Regions {
		resources, err := fetchECSRegion(ctx, p, region)
		if err != nil {
			return nil, fmt.Errorf("ecs region %s: %w", region, err)
		}
		allResources = append(allResources, resources...)
	}

	// If no regions configured, try without region (uses default profile region)
	if len(p.Regions) == 0 {
		resources, err := fetchECSRegion(ctx, p, "")
		if err != nil {
			return nil, err
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

type ecsResponse struct {
	Instances struct {
		Instance []ecsInstance `json:"Instance"`
	} `json:"Instances"`
	TotalCount int `json:"TotalCount"`
	PageSize   int `json:"PageSize"`
	PageNumber int `json:"PageNumber"`
}

type ecsInstance struct {
	InstanceID   string `json:"InstanceId"`
	InstanceName string `json:"InstanceName"`
	Status       string `json:"Status"`
	RegionID     string `json:"RegionId"`
	ZoneID       string `json:"ZoneId"`
	InstanceType string `json:"InstanceType"`
	CPU          int    `json:"Cpu"`
	Memory       int    `json:"Memory"`
	CreationTime string `json:"CreationTime"`
	ExpiredTime  string `json:"ExpiredTime"`
	PublicIPAddr struct {
		IPAddress []string `json:"IpAddress"`
	} `json:"PublicIpAddress"`
	InnerIPAddr struct {
		IPAddress []string `json:"IpAddress"`
	} `json:"InnerIpAddress"`
	VPCAttributes struct {
		NatIPAddress  string `json:"NatIpAddress"`
		PrivateIPAddr struct {
			IPAddress []string `json:"IpAddress"`
		} `json:"PrivateIpAddress"`
		VSwitchID string `json:"VSwitchId"`
		VPCID     string `json:"VpcId"`
	} `json:"VpcAttributes"`
}

// ecsPageSize is the page size requested per DescribeInstances call.
// The aliyun API defaults to 10 per page, so without paging most accounts
// with more than 10 instances would silently lose results.
const ecsPageSize = 100

func fetchECSRegion(ctx context.Context, p *core.Provider, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"ecs", "DescribeInstances",
			"--PageSize", fmt.Sprintf("%d", ecsPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp ecsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse ecs response: %w", err)
		}

		for _, inst := range resp.Instances.Instance {
			rawJSON, _ := json.Marshal(inst)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ecs",
				Region:       inst.RegionID,
				ResourceID:   inst.InstanceID,
				ResourceName: inst.InstanceName,
				Status:       inst.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		// Stop once we've fetched all pages, or the API returned less than
		// a full page (defensive: avoids an infinite loop if TotalCount is
		// ever inconsistent with the actual instance count).
		if len(allResources) >= resp.TotalCount || len(resp.Instances.Instance) < ecsPageSize {
			break
		}
	}

	return allResources, nil
}

// runAliyun executes the aliyun CLI and returns stdout bytes.
func runAliyun(ctx context.Context, args []string, profile string) ([]byte, error) {
	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, args...)
	if profile != "" {
		fullArgs = append(fullArgs, "--profile", profile)
	}
	cmd := exec.CommandContext(ctx, "aliyun", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, &core.ProviderError{
				Provider:  "aliyun",
				Operation: args[0] + " " + args[1],
				Stderr:    string(exitErr.Stderr),
				ExitCode:  exitErr.ExitCode(),
			}
		}
		return nil, err
	}
	return out, nil
}
