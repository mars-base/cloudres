package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
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
	InstanceID        string `json:"InstanceId"`
	InstanceName      string `json:"InstanceName"`
	Status            string `json:"Status"`
	RegionID          string `json:"RegionId"`
	ZoneID            string `json:"ZoneId"`
	InstanceType      string `json:"InstanceType"`
	CPU               int    `json:"Cpu"`
	Memory            int    `json:"Memory"`
	CreationTime      string `json:"CreationTime"`
	ExpiredTime       string `json:"ExpiredTime"`
	InstanceChargeType string `json:"InstanceChargeType"`
	PublicIPAddr      struct {
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

// ecsAutoRenewResponse mirrors the DescribeInstanceAutoRenewAttribute API response.
type ecsAutoRenewResponse struct {
	InstanceRenewAttributes struct {
		InstanceRenewAttribute []ecsAutoRenewAttr `json:"InstanceRenewAttribute"`
	} `json:"InstanceRenewAttributes"`
}

type ecsAutoRenewAttr struct {
	InstanceID       string `json:"InstanceId"`
	AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
	RenewalStatus    string `json:"RenewalStatus"`
	Duration         int    `json:"Duration"`
	PeriodUnit       string `json:"PeriodUnit"`
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

		// 收集本页所有包年包月实例的 ID，批量查询自动续费属性
		var prepaidIDs []string
		for _, inst := range resp.Instances.Instance {
			if inst.InstanceChargeType == "PrePaid" {
				prepaidIDs = append(prepaidIDs, inst.InstanceID)
			}
		}

		renewMap := make(map[string]ecsAutoRenewAttr)
		if len(prepaidIDs) > 0 {
			renewAttrs, err := fetchECSAutoRenewBatch(ctx, p, region, prepaidIDs)
			if err != nil {
				// best-effort: 自动续费查询失败不影响主流程
				renewAttrs = nil
			}
			for _, attr := range renewAttrs {
				renewMap[attr.InstanceID] = attr
			}
		}

		for _, inst := range resp.Instances.Instance {
			renewAttr := renewMap[inst.InstanceID]
			rawJSON, _ := json.Marshal(struct {
				ecsInstance
				AutoRenewEnabled bool   `json:"AutoRenewEnabled"`
				RenewalStatus    string `json:"RenewalStatus"`
				Duration         int    `json:"Duration"`
				PeriodUnit       string `json:"PeriodUnit"`
			}{
				ecsInstance:      inst,
				AutoRenewEnabled: renewAttr.AutoRenewEnabled,
				RenewalStatus:    renewAttr.RenewalStatus,
				Duration:         renewAttr.Duration,
				PeriodUnit:       renewAttr.PeriodUnit,
			})
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

// fetchECSAutoRenewBatch queries auto-renewal attributes for a batch of instance IDs.
// The API supports up to 100 instance IDs per request via comma-separated InstanceId parameter.
func fetchECSAutoRenewBatch(ctx context.Context, p *core.Provider, region string, instanceIDs []string) ([]ecsAutoRenewAttr, error) {
	args := []string{"ecs", "DescribeInstanceAutoRenewAttribute",
		"--InstanceId", strings.Join(instanceIDs, ","),
	}
	if region != "" {
		args = append(args, "--RegionId", region)
	}

	out, err := runAliyun(ctx, args, p.ActiveProfile)
	if err != nil {
		return nil, err
	}

	var resp ecsAutoRenewResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parse ecs auto-renew response: %w", err)
	}

	return resp.InstanceRenewAttributes.InstanceRenewAttribute, nil
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
