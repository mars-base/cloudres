package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type EVSFetcher struct{}

func (f *EVSFetcher) ResourceType() string { return "evs" }

func (f *EVSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchByProfiles(ctx, p, fetchEVSProfile)
}

type evsResponse struct {
	Volumes []evsVolume `json:"volumes"`
	Count   int         `json:"count"`
}

type evsVolume struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	Size             int    `json:"size"`
	Bootable         string `json:"bootable"`
	Encrypted        bool   `json:"encrypted"`
	VolumeType       string `json:"volume_type"`
	Shareable        string `json:"shareable"`
	Multiattach      bool   `json:"multiattach"`
	AvailabilityZone string `json:"availability_zone"`
	CreatedAt        string `json:"created_at"`
	Attachments      []struct {
		ServerID string `json:"server_id"`
		Device   string `json:"device"`
	} `json:"attachments"`
}

const evsPageSize = 200

func fetchEVSProfile(ctx context.Context, profile, region string) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()
	marker := ""

	for {
		args := []string{"EVS", "ListVolumes", "--cli-output=json",
			"--limit=" + fmt.Sprintf("%d", evsPageSize),
		}
		if region != "" {
			args = append(args, "--cli-region="+region)
		}
		if marker != "" {
			args = append(args, "--marker="+marker)
		}

		out, err := runHuawei(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp evsResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse evs response: %w", err)
		}

		for _, v := range resp.Volumes {
			rawJSON, _ := json.Marshal(v)
			allResources = append(allResources, core.Resource{
				Provider:     "huawei",
				ResourceType: "evs",
				Region:       region,
				ResourceID:   v.ID,
				ResourceName: v.Name,
				Status:       v.Status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.Volumes) < evsPageSize {
			break
		}
		if len(resp.Volumes) > 0 {
			marker = resp.Volumes[len(resp.Volumes)-1].ID
		} else {
			break
		}
	}

	return allResources, nil
}
