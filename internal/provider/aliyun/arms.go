package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

const armsPageSize = 100

// ---------------------------------------------------------------------------
// ARMS Alert Contacts (DescribeContacts)
// ---------------------------------------------------------------------------

type ARMSContactFetcher struct{}

func (f *ARMSContactFetcher) ResourceType() string { return "arms-ct" }

func (f *ARMSContactFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for page := 1; ; page++ {
		args := []string{"arms", "DescribeContacts",
			"--Page", fmt.Sprintf("%d", page),
			"--Size", fmt.Sprintf("%d", armsPageSize),
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp struct {
			PageBean struct {
				AlertContacts []struct {
					ContactID     int    `json:"ContactId"`
					ContactName   string `json:"ContactName"`
					Email         string `json:"Email"`
					Phone         string `json:"Phone"`
					IsVerify      bool   `json:"IsVerify"`
					IsEmailVerify bool   `json:"IsEmailVerify"`
					ArmsContactID int    `json:"ArmsContactId"`
				} `json:"AlertContacts"`
				Page  int `json:"Page"`
				Size  int `json:"Size"`
				Total int `json:"Total"`
			} `json:"PageBean"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse arms contacts: %w", err)
		}

		for _, c := range resp.PageBean.AlertContacts {
			status := "unverified"
			if c.IsVerify && c.IsEmailVerify {
				status = "verified"
			} else if c.IsVerify {
				status = "phone_verified"
			} else if c.IsEmailVerify {
				status = "email_verified"
			}

			rawJSON, _ := json.Marshal(c)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "arms-ct",
				Region:       "",
				ResourceID:   fmt.Sprintf("%d", c.ContactID),
				ResourceName: c.ContactName,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.PageBean.AlertContacts) < armsPageSize {
			break
		}
		if resp.PageBean.Total <= page*armsPageSize {
			break
		}
	}

	return allResources, nil
}

// ---------------------------------------------------------------------------
// ARMS Alert Contact Groups (DescribeContactGroups)
// ---------------------------------------------------------------------------

type ARMSContactGroupFetcher struct{}

func (f *ARMSContactGroupFetcher) ResourceType() string { return "arms-cg" }

func (f *ARMSContactGroupFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for page := 1; ; page++ {
		args := []string{"arms", "DescribeContactGroups",
			"--Page", fmt.Sprintf("%d", page),
			"--Size", fmt.Sprintf("%d", armsPageSize),
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp struct {
			PageBean struct {
				AlertContactGroups []struct {
					ContactGroupID   int    `json:"ContactGroupId"`
					ContactGroupName string `json:"ContactGroupName"`
					ArmsContactGroupID int  `json:"ArmsContactGroupId"`
				} `json:"AlertContactGroups"`
				Page  int `json:"Page"`
				Size  int `json:"Size"`
				Total int `json:"Total"`
			} `json:"PageBean"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse arms contact groups: %w", err)
		}

		for _, g := range resp.PageBean.AlertContactGroups {
			rawJSON, _ := json.Marshal(g)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "arms-cg",
				Region:       "",
				ResourceID:   fmt.Sprintf("%d", g.ContactGroupID),
				ResourceName: g.ContactGroupName,
				Status:       "Active",
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.PageBean.AlertContactGroups) < armsPageSize {
			break
		}
		if resp.PageBean.Total <= page*armsPageSize {
			break
		}
	}

	return allResources, nil
}

// ---------------------------------------------------------------------------
// ARMS Alert Rules (SearchAlertRules)
// ---------------------------------------------------------------------------

type ARMSAlertRuleFetcher struct{}

func (f *ARMSAlertRuleFetcher) ResourceType() string { return "arms-alert" }

func (f *ARMSAlertRuleFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	// SearchAlertRules requires RegionId — use provider regions if configured,
	// otherwise fall back to default profile region.
	regions := p.Regions
	if len(regions) == 0 {
		regions = []string{""}
	}

	for _, region := range regions {
		resources, err := fetchARMSAlertRules(ctx, p, region, now)
		if err != nil {
			// best-effort: skip regions that fail rather than blocking all
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

func fetchARMSAlertRules(ctx context.Context, p *core.Provider, region string, now time.Time) ([]core.Resource, error) {
	var resources []core.Resource

	for page := 1; ; page++ {
		args := []string{"arms", "SearchAlertRules",
			"--CurrentPage", fmt.Sprintf("%d", page),
			"--PageSize", fmt.Sprintf("%d", armsPageSize),
		}
		if region != "" {
			args = append(args, "--RegionId", region)
		}

		out, err := runAliyun(ctx, args, p.ActiveProfile)
		if err != nil {
			return nil, err
		}

		var resp struct {
			PageBean struct {
				AlertRules []armsAlertRule `json:"AlertRules"`
				Total      int             `json:"Total"`
			} `json:"PageBean"`
		}

		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse arms alert rules: %w", err)
		}

		for _, r := range resp.PageBean.AlertRules {
			// Build status: combine running/stopped + alert level
			status := r.Status
			if r.AlertLevel != "" {
				status = r.Status + "/" + r.AlertLevel
			}

			rawJSON, _ := json.Marshal(r)
			resources = append(resources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "arms-alert",
				Region:       r.RegionID,
				ResourceID:   fmt.Sprintf("%d", r.ID),
				ResourceName: r.AlertTitle,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.PageBean.AlertRules) < armsPageSize {
			break
		}
		if resp.PageBean.Total > 0 && resp.PageBean.Total <= page*armsPageSize {
			break
		}
	}

	return resources, nil
}

// armsAlertRule mirrors the SearchAlertRules response for one alert rule.
type armsAlertRule struct {
	ID                 int      `json:"Id"`
	AlertTitle         string   `json:"AlertTitle"`
	AlertLevel         string   `json:"AlertLevel"`
	Status             string   `json:"Status"`
	AlertType          int      `json:"AlertType"`
	AlertWays          []string `json:"AlertWays"`
	RegionID           string   `json:"RegionId"`
	ContactGroupIDList string   `json:"ContactGroupIdList"`
	CreateTime         int64    `json:"CreateTime"`
	UpdateTime         int64    `json:"UpdateTime"`
	HostByAlertManager bool     `json:"HostByAlertManager"`
	AlertRule          struct {
		Operator string `json:"Operator"`
		Rules    []struct {
			Measure  string  `json:"Measure"`
			NValue   int     `json:"NValue"`
			Operator string  `json:"Operator"`
			Value    float64 `json:"Value"`
		} `json:"Rules"`
	} `json:"AlertRule"`
}
