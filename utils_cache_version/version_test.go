package utils_cache_version

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestFirstVersionInitialization(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})

	got, err := m.SelectVersion(SelectOptions{Version: "v1", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.StableVersion != "v1" || got.InRelease {
		t.Fatalf("first version result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{Version: "v1", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.StableVersion != "v1" || got.InRelease {
		t.Fatalf("stable version without business id result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: "biz-a", Version: "v1", Now: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.StableVersion != "v1" || got.InRelease {
		t.Fatalf("stable version result = %+v", got)
	}
}

func TestGradualReleaseProgress(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	mustSelect(t, m, SelectOptions{Version: "v1", Now: start})

	got, err := m.SelectVersion(SelectOptions{BusinessID: "biz-start", Version: "v2", Now: start})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.TargetVersion != "v2" || got.Progress != 0 || got.Released {
		t.Fatalf("release start result = %+v", got)
	}

	releasedID := findBusinessID(t, "v2", 0.5, true)
	unreleasedID := findBusinessID(t, "v2", 0.5, false)
	half := start.Add(90 * time.Minute)

	got, err = m.SelectVersion(SelectOptions{BusinessID: releasedID, Version: "v2", Now: half})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v2" || !got.Released || got.Progress != 0.5 {
		t.Fatalf("released half result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: unreleasedID, Version: "v2", Now: half})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1" || got.Released || got.Progress != 0.5 {
		t.Fatalf("unreleased half result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: "biz-complete", Version: "v2", Now: start.Add(3 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v2" || got.StableVersion != "v2" || got.InRelease {
		t.Fatalf("complete result = %+v", got)
	}
}

func TestSameIdentifierIsDeterministic(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	mustSelect(t, m, SelectOptions{Version: "v1", Now: start})
	mustSelect(t, m, SelectOptions{BusinessID: "biz-start", Version: "v2", Now: start})

	opt := SelectOptions{
		BusinessID: "biz-deterministic",
		Version:    "v2",
		Now:        start.Add(90 * time.Minute),
	}
	first, err := m.SelectVersion(opt)
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.SelectVersion(opt)
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != second.Version || first.Released != second.Released {
		t.Fatalf("results differ: first=%+v second=%+v", first, second)
	}
}

func TestStickyIdentifierDoesNotFallBack(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	mustSelect(t, m, SelectOptions{Version: "v1", Now: start})
	mustSelect(t, m, SelectOptions{BusinessID: "biz-start", Version: "v2", Now: start})

	id := findBusinessID(t, "v2", 0.5, true)
	half := start.Add(90 * time.Minute)
	got, err := m.SelectVersion(SelectOptions{BusinessID: id, Version: "v2", Now: half})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v2" || !got.Released {
		t.Fatalf("expected released to v2: %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: id, Version: "v3", Now: half})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v3" || got.TargetVersion != "v3" || !got.Released {
		t.Fatalf("expected sticky id to follow latest target: %+v", got)
	}
}

func TestLatestVersionUpdatesActiveTarget(t *testing.T) {
	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{ReleaseDuration: 3 * time.Hour})
	mustSelect(t, m, SelectOptions{Version: "v1.1.0", Now: start})

	got, err := m.SelectVersion(SelectOptions{BusinessID: "biz-a", Version: "v1.1.1", Now: start})
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetVersion != "v1.1.1" || got.Progress != 0 {
		t.Fatalf("initial target result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: "biz-b", Version: "v1.1.4", Now: start.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetVersion != "v1.1.4" || got.Progress != float64(1)/float64(3) {
		t.Fatalf("updated target result = %+v", got)
	}

	got, err = m.SelectVersion(SelectOptions{BusinessID: "biz-c", Version: "v1.1.4", Now: start.Add(3 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v1.1.4" || got.StableVersion != "v1.1.4" || got.InRelease {
		t.Fatalf("latest stable result = %+v", got)
	}
}

func TestValidationErrors(t *testing.T) {
	_, err := New(Config{}).SelectVersion(SelectOptions{Version: "v1"})
	if !errors.Is(err, ErrInvalidReleaseDuration) {
		t.Fatalf("invalid duration err = %v", err)
	}

	m := New(Config{ReleaseDuration: time.Hour})
	_, err = m.SelectVersion(SelectOptions{})
	if !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("empty version err = %v", err)
	}

	start := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	mustSelect(t, m, SelectOptions{Version: "v1", Now: start})
	_, err = m.SelectVersion(SelectOptions{Version: "v2", Now: start})
	if !errors.Is(err, ErrEmptyBusinessID) {
		t.Fatalf("empty business id err = %v", err)
	}
}

func TestConfigNowControlsProgress(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	m := New(Config{
		ReleaseDuration: 3 * time.Hour,
		Now: func() time.Time {
			return now
		},
	})

	mustSelect(t, m, SelectOptions{Version: "v1"})
	mustSelect(t, m, SelectOptions{BusinessID: "biz-start", Version: "v2"})

	now = now.Add(90 * time.Minute)
	id := findBusinessID(t, "v2", 0.5, true)
	got, err := m.SelectVersion(SelectOptions{BusinessID: id, Version: "v2"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "v2" || got.Progress != 0.5 {
		t.Fatalf("config now result = %+v", got)
	}
}

func mustSelect(t *testing.T, m *Manager, opt SelectOptions) Result {
	t.Helper()
	got, err := m.SelectVersion(opt)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func findBusinessID(t *testing.T, version string, progress float64, released bool) string {
	t.Helper()
	threshold := progressThreshold(progress)
	for i := 0; i < 100000; i++ {
		id := fmt.Sprintf("biz-%d", i)
		isReleased := bucket(id, version) < threshold
		if isReleased == released {
			return id
		}
	}
	t.Fatalf("no business id found for version %q progress %v released %v", version, progress, released)
	return ""
}
