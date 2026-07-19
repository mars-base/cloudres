package main

import (
	"os"

	"github.com/mars-base/cloudres/internal/cli"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	cli.Version = version
	cli.BuildTime = buildTime
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
