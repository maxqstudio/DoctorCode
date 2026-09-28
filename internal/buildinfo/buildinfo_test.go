package buildinfo

import "testing"

func TestCurrentNormalizesEmptyInjectedValues(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, BuildDate
	t.Cleanup(func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldDate
	})

	Version, Commit, BuildDate = " ", "", "\t"
	got := Current()
	if got.Version != "0.1.0-dev" || got.Commit != "unknown" || got.BuildDate != "unknown" {
		t.Fatalf("unexpected normalized build info: %#v", got)
	}
}

func TestCurrentPreservesInjectedReleaseMetadata(t *testing.T) {
	oldVersion, oldCommit, oldDate := Version, Commit, BuildDate
	t.Cleanup(func() {
		Version, Commit, BuildDate = oldVersion, oldCommit, oldDate
	})

	Version = "0.1.0"
	Commit = "0123456789abcdef0123456789abcdef01234567"
	BuildDate = "2026-09-28T04:00:00+00:00"
	got := Current()
	if got.Version != Version || got.Commit != Commit || got.BuildDate != BuildDate {
		t.Fatalf("release metadata changed: %#v", got)
	}
}
