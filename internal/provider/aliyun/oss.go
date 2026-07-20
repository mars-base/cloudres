package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	"github.com/mars-base/cloudres/internal/core"
)

type OSSFetcher struct{}

func (f *OSSFetcher) ResourceType() string { return "oss" }

func (f *OSSFetcher) Fetch(ctx context.Context, p *core.Provider) ([]core.Resource, error) {
	// ossutil ls outputs text, not JSON.
	// NOTE: ossutil is a separate embedded binary invoked via the aliyun CLI
	// wrapper; it silently ignores --profile when placed after the "ossutil"
	// subcommand, so --profile must come before "ossutil" as a global flag.
	args := []string{"ossutil", "ls"}
	if p.ActiveProfile != "" {
		args = append([]string{"--profile", p.ActiveProfile}, args...)
	}
	cmd := exec.CommandContext(ctx, "aliyun", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, &core.ProviderError{
				Provider:  "aliyun",
				Operation: "ossutil ls",
				Stderr:    string(exitErr.Stderr),
				ExitCode:  exitErr.ExitCode(),
			}
		}
		return nil, err
	}

	return parseOSSOutput(string(out))
}

// parseOSSOutput parses the text output from `aliyun ossutil ls`.
// Format:
//
//	CreationTime   Region   StorageClass   BucketName
//	2025-06-06 11:17:34 +0800 CST   oss-cn-hangzhou   Standard   oss://bucket-name
var ossLineRe = regexp.MustCompile(
	`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+[+\-]\d{4}\s+\w+)\s+` + // creation time
		`(oss-[\w-]+)\s+` + // region
		`(\w+)\s+` + // storage class
		`(oss://[\w.-]+)`, // bucket name
)

func parseOSSOutput(output string) ([]core.Resource, error) {
	matches := ossLineRe.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	now := time.Now()
	resources := make([]core.Resource, 0, len(matches))
	for _, m := range matches {
		bucket := m[4]
		region := m[2]
		storageClass := m[3]
		created := m[1]

		rawData := map[string]string{
			"CreationTime": created,
			"Region":       region,
			"StorageClass": storageClass,
		}
		rawJSON, _ := json.Marshal(rawData)

		resources = append(resources, core.Resource{
			Provider:     "aliyun",
			ResourceType: "oss",
			Region:       region,
			ResourceID:   bucket,
			ResourceName: bucket,
			Status:       "Available",
			RawJSON:      string(rawJSON),
			SyncedAt:     now,
		})
	}

	return resources, nil
}

func init() {
	// Verify the regex compiles (compile-time safety)
	if ossLineRe.NumSubexp() != 4 {
		panic(fmt.Sprintf("oss regex should have 4 groups, got %d", ossLineRe.NumSubexp()))
	}
}
