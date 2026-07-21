package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type RAMFetcher struct{}

func (f *RAMFetcher) ResourceType() string { return "ram" }

func (f *RAMFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchRAM(ctx, p.ActiveProfile)
}

type ramResponse struct {
	Users struct {
		User []ramUser `json:"User"`
	} `json:"Users"`
	IsTruncated bool   `json:"IsTruncated"`
	Marker      string `json:"Marker"`
}

type ramUser struct {
	UserId      string `json:"UserId"`
	UserName    string `json:"UserName"`
	DisplayName string `json:"DisplayName"`
	Comments    string `json:"Comments"`
	CreateDate  string `json:"CreateDate"`
	UpdateDate  string `json:"UpdateDate"`
}

const ramPageSize = 100

func fetchRAM(ctx context.Context, profile string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	marker := ""

	for {
		args := []string{"ram", "ListUsers",
			"--MaxItems", fmt.Sprintf("%d", ramPageSize),
		}
		if marker != "" {
			args = append(args, "--Marker", marker)
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp ramResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse ram response: %w", err)
		}

		for _, u := range resp.Users.User {
			rawJSON, _ := json.Marshal(u)
			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "ram",
				Region:       "",
				ResourceID:   u.UserId,
				ResourceName: u.UserName,
				Status:       "Active",
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if !resp.IsTruncated || len(resp.Users.User) == 0 {
			break
		}
		marker = resp.Marker
	}

	return allResources, nil
}
