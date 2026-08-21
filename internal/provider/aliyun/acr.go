package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// ACR Instances (ListInstance + enrich via GetInstance / ListNamespace /
//                  ListRepository / GetInstanceEndpoint)
// ---------------------------------------------------------------------------

type ACRFetcher struct{}

func (f *ACRFetcher) ResourceType() string { return "acr" }

func (f *ACRFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchACRInstances(ctx, p.ActiveProfile)
}

// -- ListInstance response --

type acrInstanceListResponse struct {
	Instances  []acrInstance `json:"Instances"`
	TotalCount int           `json:"TotalCount"`
}

type acrInstance struct {
	InstanceID            string `json:"InstanceId"`
	InstanceName          string `json:"InstanceName"`
	InstanceSpecification string `json:"InstanceSpecification"`
	InstanceStatus        string `json:"InstanceStatus"`
	RegionID              string `json:"RegionId"`
	CreateTime            int64  `json:"CreateTime"`
}

// -- Enriched data stored alongside the base instance in RawJSON --

type acrEnriched struct {
	acrInstance
	ResourceGroupID string   `json:"ResourceGroupId"`
	ModifiedTime    int64    `json:"ModifiedTime"`
	Namespaces      []string `json:"namespaces"`
	RepoCount       int      `json:"repo_count"`
	Endpoint        string   `json:"endpoint"`
	ACLCount        int      `json:"acl_count"`
}

const acrPageSize = 100

func fetchACRInstances(ctx context.Context, profile string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	for pageNo := 1; ; pageNo++ {
		args := []string{"cr", "ListInstance",
			"--PageSize", fmt.Sprintf("%d", acrPageSize),
			"--PageNo", fmt.Sprintf("%d", pageNo),
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp acrInstanceListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse acr instance list: %w", err)
		}

		for _, inst := range resp.Instances {
			enriched := enrichACRInstance(ctx, inst, profile)

			rawJSON, _ := json.Marshal(enriched)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "acr",
				Region:       inst.RegionID,
				ResourceID:   inst.InstanceID,
				ResourceName: inst.InstanceName,
				Status:       inst.InstanceStatus,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(allResources) >= resp.TotalCount || len(resp.Instances) < acrPageSize {
			break
		}
	}

	return allResources, nil
}

// enrichACRInstance fetches additional details for an ACR instance.
func enrichACRInstance(ctx context.Context, inst acrInstance, profile string) acrEnriched {
	e := acrEnriched{
		acrInstance: inst,
	}

	// GetInstance — ResourceGroupId, ModifiedTime
	{
		args := []string{"cr", "GetInstance", "--InstanceId", inst.InstanceID}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var detail struct {
				ResourceGroupID string `json:"ResourceGroupId"`
				ModifiedTime    int64  `json:"ModifiedTime"`
			}
			if json.Unmarshal(out, &detail) == nil {
				e.ResourceGroupID = detail.ResourceGroupID
				e.ModifiedTime = detail.ModifiedTime
			}
		}
	}

	// ListNamespace — namespace names
	{
		args := []string{"cr", "ListNamespace", "--InstanceId", inst.InstanceID}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var ns struct {
				Namespaces []struct {
					NamespaceName string `json:"NamespaceName"`
				} `json:"Namespaces"`
			}
			if json.Unmarshal(out, &ns) == nil {
				for _, n := range ns.Namespaces {
					e.Namespaces = append(e.Namespaces, n.NamespaceName)
				}
			}
		}
	}

	// ListRepository — total count (across all namespaces)
	{
		totalRepos := 0
		for _, ns := range e.Namespaces {
			args := []string{"cr", "ListRepository",
				"--InstanceId", inst.InstanceID,
				"--RepoNamespaceName", ns,
			}
			if out, err := runAliyun(ctx, args, profile); err == nil {
				var repo struct {
					TotalCount int `json:"TotalCount"`
				}
				if json.Unmarshal(out, &repo) == nil {
					totalRepos += repo.TotalCount
				}
			}
		}
		e.RepoCount = totalRepos
	}

	// GetInstanceEndpoint (internet) — domain + ACL count
	{
		args := []string{"cr", "GetInstanceEndpoint",
			"--InstanceId", inst.InstanceID,
			"--EndpointType", "internet",
		}
		if out, err := runAliyun(ctx, args, profile); err == nil {
			var ep struct {
				Domains []struct {
					Domain string `json:"Domain"`
				} `json:"Domains"`
				ACLEntries []struct {
					Entry string `json:"Entry"`
				} `json:"AclEntries"`
			}
			if json.Unmarshal(out, &ep) == nil {
				if len(ep.Domains) > 0 {
					e.Endpoint = ep.Domains[0].Domain
				}
				e.ACLCount = len(ep.ACLEntries)
			}
		}
	}

	return e
}
