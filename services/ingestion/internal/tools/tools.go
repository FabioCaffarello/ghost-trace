//go:build tools
// +build tools

// Package tools pins the build-tools the ingestion service depends on at
// generation time. The build tag prevents inclusion in normal builds.
// `make tools` runs `go install` against each blank-imported package per
// the conventional Go tools-pinning pattern.
package tools

import (
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
