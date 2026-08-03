package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// CMS Alert Contacts (DescribeContactList)
// ---------------------------------------------------------------------------

type CMSContactFetcher struct{}

func (f *CMSContactFetcher) ResourceType() string { return "cms-ct" }

func (f *CMSContactFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchCMSContacts(ctx, p.ActiveProfile)
}

type cmsContactListResponse struct {
	Contacts struct {
		Contact []cmsContact `json:"Contact"`
	} `json:"Contacts"`
	Total int `json:"Total"`
}

type cmsContact struct {
	ContactID     int64  `json:"ContactId"`
	Name          string `json:"Name"`
	Desc          string `json:"Desc"`
	CreateTime    int64  `json:"CreateTime"`
	UpdateTime    int64  `json:"UpdateTime"`
	Channels      struct {
		Mail string `json:"Mail"`
		SMS  string `json:"SMS"`
	} `json:"Channels"`
	ChannelsState struct {
		Mail string `json:"Mail"`
		SMS  string `json:"SMS"`
	} `json:"ChannelsState"`
	ContactGroups struct {
		ContactGroup []string `json:"ContactGroup"`
	} `json:"ContactGroups"`
}

const cmsPageSize = 100

func fetchCMSContacts(ctx context.Context, profile string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"cms", "DescribeContactList",
			"--PageSize", fmt.Sprintf("%d", cmsPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp cmsContactListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse cms contact list: %w", err)
		}

		for _, c := range resp.Contacts.Contact {
			// Status summarizes channel verification state: OK only when
			// every configured channel is verified OK.
			states := make([]string, 0, 2)
			if c.ChannelsState.Mail != "" {
				states = append(states, c.ChannelsState.Mail)
			}
			if c.ChannelsState.SMS != "" {
				states = append(states, c.ChannelsState.SMS)
			}
			status := "-"
			if len(states) > 0 {
				status = "OK"
				for _, s := range states {
					if s != "OK" {
						status = s
						break
					}
				}
			}

			rawJSON, _ := json.Marshal(c)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "cms-ct",
				Region:       "",
				ResourceID:   fmt.Sprintf("%d", c.ContactID),
				ResourceName: c.Name,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.Total || len(resp.Contacts.Contact) < cmsPageSize {
			break
		}
	}

	return allResources, nil
}

// ---------------------------------------------------------------------------
// CMS Contact Groups (DescribeContactGroupList)
// ---------------------------------------------------------------------------

type CMSContactGroupFetcher struct{}

func (f *CMSContactGroupFetcher) ResourceType() string { return "cms-cg" }

func (f *CMSContactGroupFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchCMSContactGroups(ctx, p.ActiveProfile)
}

type cmsContactGroupListResponse struct {
	ContactGroupList struct {
		ContactGroup []cmsContactGroup `json:"ContactGroup"`
	} `json:"ContactGroupList"`
	Total int `json:"Total"`
}

type cmsContactGroup struct {
	Name              string `json:"Name"`
	Describe          string `json:"Describe"`
	CreateTime        int64  `json:"CreateTime"`
	UpdateTime        int64  `json:"UpdateTime"`
	EnableSubscribed  bool   `json:"EnableSubscribed"`
	EnabledWeeklyReport bool `json:"EnabledWeeklyReport"`
	Contacts          struct {
		Contact []string `json:"Contact"`
	} `json:"Contacts"`
}

func fetchCMSContactGroups(ctx context.Context, profile string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNumber := 1; ; pageNumber++ {
		args := []string{"cms", "DescribeContactGroupList",
			"--PageSize", fmt.Sprintf("%d", cmsPageSize),
			"--PageNumber", fmt.Sprintf("%d", pageNumber),
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp cmsContactGroupListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse cms contact group list: %w", err)
		}

		for _, g := range resp.ContactGroupList.ContactGroup {
			rawJSON, _ := json.Marshal(g)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "cms-cg",
				Region:       "",
				// Contact groups have no separate ID; the name is unique.
				ResourceID:   g.Name,
				ResourceName: g.Name,
				Status:       "Active",
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.Total || len(resp.ContactGroupList.ContactGroup) < cmsPageSize {
			break
		}
	}

	return allResources, nil
}

// ---------------------------------------------------------------------------
// CMS Alert Rules (DescribeMetricRuleList)
// ---------------------------------------------------------------------------

type CMSAlertRuleFetcher struct{}

func (f *CMSAlertRuleFetcher) ResourceType() string { return "cms-alert" }

func (f *CMSAlertRuleFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchCMSAlertRules(ctx, p.ActiveProfile)
}

type cmsAlertRuleListResponse struct {
	Alarms struct {
		Alarm []cmsAlertRule `json:"Alarm"`
	} `json:"Alarms"`
	Total int `json:"Total"`
}

type cmsAlertRule struct {
	RuleID            string `json:"RuleId"`
	RuleName          string `json:"RuleName"`
	AlertState        string `json:"AlertState"`
	EnableState       bool   `json:"EnableState"`
	Namespace         string `json:"Namespace"`
	MetricName        string `json:"MetricName"`
	Period            int    `json:"Period"`
	ContactGroups     string `json:"ContactGroups"`
	Resources         string `json:"Resources"`
	EffectiveInterval string `json:"EffectiveInterval"`
	SilenceTime       int    `json:"SilenceTime"`
	NoDataPolicy      string `json:"NoDataPolicy"`
	RuleType          string `json:"RuleType"`
	SourceType        string `json:"SourceType"`
	GmtCreate         int64  `json:"GmtCreate"`
	GmtUpdate         int64  `json:"GmtUpdate"`
}

// cmsRuleResource is one entry in the rule's Resources JSON array.
type cmsRuleResource struct {
	InstanceID string `json:"instanceId"`
}

func fetchCMSAlertRules(ctx context.Context, profile string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for page := 1; ; page++ {
		args := []string{"cms", "DescribeMetricRuleList",
			"--PageSize", fmt.Sprintf("%d", cmsPageSize),
			"--Page", fmt.Sprintf("%d", page),
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp cmsAlertRuleListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse cms alert rule list: %w", err)
		}

		for _, r := range resp.Alarms.Alarm {
			// Resolve instance IDs from the Resources JSON string so the
			// detail view can show which instances the rule covers.
			var resList []cmsRuleResource
			_ = json.Unmarshal([]byte(r.Resources), &resList)
			instanceIDs := make([]string, 0, len(resList))
			for _, res := range resList {
				if res.InstanceID != "" {
					instanceIDs = append(instanceIDs, res.InstanceID)
				}
			}

			rawJSON, _ := json.Marshal(struct {
				cmsAlertRule
				InstanceIDs []string `json:"InstanceIDs"`
			}{
				cmsAlertRule: r,
				InstanceIDs:  instanceIDs,
			})

			status := "Disabled"
			if r.EnableState {
				status = r.AlertState
			}
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "cms-alert",
				Region:       "",
				ResourceID:   r.RuleID,
				ResourceName: r.RuleName,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.Total || len(resp.Alarms.Alarm) < cmsPageSize {
			break
		}
	}

	return allResources, nil
}
