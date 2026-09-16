package service

import (
	"context"
	"testing"

	"github.com/piwriw/oas-go-template/internal/version"
)

// TestVersionInfoDegradesEmptyBuildMetadata verifies binaries built without ldflags report stable placeholders.
func TestVersionInfoDegradesEmptyBuildMetadata(t *testing.T) {
	restore := setBuildMetadata(t, "", "", "")
	defer restore()

	info := New(nil).VersionInfo(context.Background())
	if info.Version != degradedVersion {
		t.Errorf("VersionInfo() version = %q, want %q", info.Version, degradedVersion)
	}
	if info.GitCommit != degradedBuildMeta {
		t.Errorf("VersionInfo() git commit = %q, want %q", info.GitCommit, degradedBuildMeta)
	}
	if info.BuildTime != degradedBuildMeta {
		t.Errorf("VersionInfo() build time = %q, want %q", info.BuildTime, degradedBuildMeta)
	}
}

// TestVersionInfoKeepsPopulatedBuildMetadata verifies populated metadata passes through untouched.
func TestVersionInfoKeepsPopulatedBuildMetadata(t *testing.T) {
	restore := setBuildMetadata(t, "1.2.3", "abc123", "2026-01-01T00:00:00Z")
	defer restore()

	info := New(nil).VersionInfo(context.Background())
	if info.Version != "1.2.3" || info.GitCommit != "abc123" || info.BuildTime != "2026-01-01T00:00:00Z" {
		t.Errorf("VersionInfo() = %+v, want populated metadata passthrough", info)
	}
}

// setBuildMetadata overwrites the package build vars for one test and returns a restore func.
func setBuildMetadata(t *testing.T, v, commit, buildTime string) func() {
	t.Helper()
	original := version.Info()
	version.Version, version.GitCommit, version.BuildTime = v, commit, buildTime
	return func() {
		version.Version, version.GitCommit, version.BuildTime = original.Version, original.GitCommit, original.BuildTime
	}
}
