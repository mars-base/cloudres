package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

// ---------------------------------------------------------------------------
// CAS Certificates (ListCertificates)
// ---------------------------------------------------------------------------

type CASFetcher struct{}

func (f *CASFetcher) ResourceType() string { return "cas" }

func (f *CASFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	return fetchCASCertificates(ctx, p)
}

type casListResponse struct {
	CertificateList []casCertificate `json:"CertificateList"`
	TotalCount      int              `json:"TotalCount"`
}

type casCertificate struct {
	CertificateId      string   `json:"CertificateId"`
	CertificateName    string   `json:"CertificateName"`
	Domain             string   `json:"Domain"`
	CommonName         string   `json:"CommonName"`
	Issuer             string   `json:"Issuer"`
	CertificateStatus  string   `json:"CertificateStatus"`
	CertificateSource  string   `json:"CertificateSource"`
	NotBefore          int64    `json:"NotBefore"`
	NotAfter           int64    `json:"NotAfter"`
	Algorithm          string   `json:"Algorithm"`
	KeySize            int      `json:"KeySize"`
	UsingProductList   []string `json:"UsingProductList"`
}

const casPageSize = 100

func fetchCASCertificates(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	var allResources []core.Resource
	now := time.Now()

	regions := p.Regions
	if len(regions) == 0 {
		// Default to common regions if none configured
		regions = []string{"cn-hangzhou", "cn-beijing", "cn-shanghai"}
	}

	for _, region := range regions {
		resources, err := fetchCASCertificatesForRegion(ctx, p.ActiveProfile, region, now)
		if err != nil {
			// best-effort: skip regions that fail
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

func fetchCASCertificatesForRegion(ctx context.Context, profile, region string, now time.Time) ([]core.Resource, error) {
	var allResources []core.Resource

	for currentPage := 1; ; currentPage++ {
		args := []string{"cas", "ListCertificates",
			"--region", region,
			"--CurrentPage", fmt.Sprintf("%d", currentPage),
			"--ShowSize", fmt.Sprintf("%d", casPageSize),
		}

		out, err := runAliyun(ctx, args, profile)
		if err != nil {
			return nil, err
		}

		var resp casListResponse
		if err := json.Unmarshal(out, &resp); err != nil {
			return nil, fmt.Errorf("parse cas certificates: %w", err)
		}

		for _, cert := range resp.CertificateList {
			rawJSON, _ := json.Marshal(cert)

			status := cert.CertificateStatus
			if status == "" {
				status = "unknown"
			}

			allResources = append(allResources, core.Resource{
				Provider:     "aliyun",
				ResourceType: "cas",
				Region:       region,
				ResourceID:   cert.CertificateId,
				ResourceName: cert.CertificateName,
				Status:       status,
				RawJSON:      string(rawJSON),
				SyncedAt:     now,
			})
		}

		if len(resp.CertificateList) < casPageSize {
			break
		}
		if resp.TotalCount > 0 && resp.TotalCount <= currentPage*casPageSize {
			break
		}
	}

	return allResources, nil
}
