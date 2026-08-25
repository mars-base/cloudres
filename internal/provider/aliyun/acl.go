package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// ACL — Access Control Lists for ALB
//   ListAcls → ListAclEntries (IP entries) → ListAclRelations (listener bindings)
// ---------------------------------------------------------------------------

type ACLFetcher struct{}

func (f *ACLFetcher) ResourceType() string { return "acl" }

func (f *ACLFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchACLs(ctx, p)
}

// -- ListAcls response --

type aclListResponse struct {
	ACLs       []aclBase `json:"Acls"`
	MaxResults int       `json:"MaxResults"`
	NextToken  string    `json:"NextToken"`
	TotalCount int       `json:"TotalCount"`
}

type aclBase struct {
	AclID            string `json:"AclId"`
	AclName          string `json:"AclName"`
	AclStatus        string `json:"AclStatus"`
	AddressIPVersion string `json:"AddressIPVersion"`
	CreateTime       string `json:"CreateTime"`
	ResourceGroupID  string `json:"ResourceGroupId"`
}

// -- ListAclEntries response --

type aclEntriesResponse struct {
	ACLEntries []aclEntry `json:"AclEntries"`
	MaxResults int        `json:"MaxResults"`
	NextToken  string     `json:"NextToken"`
	TotalCount int        `json:"TotalCount"`
}

type aclEntry struct {
	Entry  string `json:"Entry"`
	Status string `json:"Status"`
}

// -- ListAclRelations response --

type aclRelationsResponse struct {
	ACLRelations []aclRelation `json:"AclRelations"`
}

type aclRelation struct {
	AclID            string           `json:"AclId"`
	RelatedListeners []aclListenerRel `json:"RelatedListeners"`
}

type aclListenerRel struct {
	ListenerID     string `json:"ListenerId"`
	ListenerPort   int    `json:"ListenerPort"`
	ListenerProto  string `json:"ListenerProtocol"`
	LoadBalancerID string `json:"LoadBalancerId"`
	Status         string `json:"Status"`
}

// -- Enriched data stored in RawJSON --

type aclEnriched struct {
	aclBase
	IPCount   int      `json:"ip_count"`
	IPs       []string `json:"ips"`
	ALBs      []string `json:"albs"`
	Listeners int      `json:"listeners"`
}

func fetchACLs(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	regions := p.ProfileRegions[p.ActiveProfile]
	if len(regions) == 0 {
		regions = []string{p.Regions[0]}
	}

	for _, region := range regions {
		acls, err := fetchACLList(ctx, region, p.ActiveProfile)
		if err != nil {
			continue
		}

		for _, acl := range acls {
			enriched := enrichACL(ctx, acl, region, p.ActiveProfile)

			rawJSON, _ := json.Marshal(enriched)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "acl",
				Region:       region,
				Profile:      p.ActiveProfile,
				ResourceID:   acl.AclID,
				ResourceName: acl.AclName,
				Status:       acl.AclStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}
	}

	return allResources, nil
}

func fetchACLList(ctx context.Context, region, profile string) ([]aclBase, error) {
	var all []aclBase
	nextToken := ""

	for {
		args := []string{"alb", "ListAcls", "--region", region, "--force"}
		if nextToken != "" {
			args = append(args, "--NextToken", nextToken)
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp aclListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse acl list: %w", err)
		}

		all = append(all, resp.ACLs...)

		if resp.NextToken == "" || len(all) >= resp.TotalCount {
			break
		}
		nextToken = resp.NextToken
	}

	return all, nil
}

func enrichACL(ctx context.Context, acl aclBase, region, profile string) aclEnriched {
	e := aclEnriched{aclBase: acl}

	// ListAclEntries — fetch all IPs with pagination
	{
		var allIPs []string
		nextToken := ""
		for {
			args := []string{"alb", "ListAclEntries", "--region", region, "--AclId", acl.AclID}
			if nextToken != "" {
				args = append(args, "--NextToken", nextToken)
			}
			out, err := runAliyun(ctx, args, profile)
			if err != nil {
				break
			}
			var resp aclEntriesResponse
			if json.Unmarshal(out, &resp) != nil {
				break
			}
			for _, entry := range resp.ACLEntries {
				allIPs = append(allIPs, entry.Entry)
			}
			if resp.NextToken == "" || len(allIPs) >= resp.TotalCount {
				e.IPCount = resp.TotalCount
				break
			}
			nextToken = resp.NextToken
		}
		e.IPs = allIPs
	}

	// ListAclRelations — bound listeners / ALBs
	{
		args := []string{"alb", "ListAclRelations", "--region", region,
			"--AclIds.1", acl.AclID, "--force"}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var resp aclRelationsResponse
			if json.Unmarshal(out, &resp) == nil {
				albSet := make(map[string]bool)
				for _, rel := range resp.ACLRelations {
					for _, lsn := range rel.RelatedListeners {
						albSet[lsn.LoadBalancerID] = true
						e.Listeners++
					}
				}
				for alb := range albSet {
					e.ALBs = append(e.ALBs, alb)
				}
			}
		}
	}

	return e
}
