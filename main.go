// Package main is the entry point for the pulumi-exporter binary.
package main

import (
	"os"

	"github.com/pulumi-labs/pulumi-exporter/cmd/pulumiexporter"
)

func main() {
	os.Exit(pulumiexporter.Main())
}
