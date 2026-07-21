package huawei

import (
	"bytes"
	"context"
	"os/exec"

	"github.com/mars-base/cloudres/internal/core"
)

func runHuawei(ctx context.Context, args []string, profile string) ([]byte, error) {
	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, args...)
	if profile != "" {
		fullArgs = append(fullArgs, "--cli-profile="+profile)
	}

	cmd := exec.CommandContext(ctx, "hcloud", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, &core.ProviderError{
				Provider:  "huawei",
				Operation: args[0] + " " + args[1],
				Stderr:    string(exitErr.Stderr),
				ExitCode:  exitErr.ExitCode(),
			}
		}
		return nil, err
	}

	// hcloud prepends multi-version API warnings before valid JSON.
	// Strip everything before the first '{'.
	if idx := bytes.IndexByte(out, '{'); idx > 0 {
		out = out[idx:]
	}

	return out, nil
}
