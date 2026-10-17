package events

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestPathEncodesSlash(t *testing.T) {
	got := Path("/x/gitgud", "feature/login")
	want := filepath.Join("/x/gitgud", "journal", "feature__login.jsonl")
	if got != want {
		t.Errorf("Path = %q; want %q", got, want)
	}
}

func TestAppendAssignsSeqAndKeepsLinesValid(t *testing.T) {
	path := Path(t.TempDir(), "feature/x")
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	typ := "feat"
	for i, rec := range []Record{
		{OpID: "op_1", Event: "branch.created", Branch: "feature/x", Source: "develop"},
		{OpID: "op_2", Event: "commit.recorded", Branch: "feature/x",
			Commit: strings.Repeat("a", 40), Subject: "feat: add", Type: &typ},
	} {
		if err := Append(path, &rec, now); err != nil {
			t.Fatal(err)
		}
		if rec.Seq != i+1 {
			t.Errorf("record %d seq = %d", i, rec.Seq)
		}
	}
	lines, err := ReadLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	for _, l := range lines {
		if !json.Valid([]byte(l)) {
			t.Errorf("invalid JSON line: %s", l)
		}
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first["type"] != nil {
		t.Errorf("event record type = %v; want null", first["type"])
	}
	if _, has := first["commit"]; has {
		t.Errorf("event record has commit field: %s", lines[0])
	}
}

func TestCommitsReadsRecordedSHAs(t *testing.T) {
	path := Path(t.TempDir(), "fix/y")
	sha := strings.Repeat("b", 40)
	if err := Append(path, &Record{OpID: "op", Event: "commit.recorded", Branch: "fix/y", Commit: sha}, time.Now()); err != nil {
		t.Fatal(err)
	}
	seen, err := Commits(path)
	if err != nil {
		t.Fatal(err)
	}
	if !seen[sha] || len(seen) != 1 {
		t.Errorf("Commits = %v", seen)
	}
	missing, err := Commits(Path(t.TempDir(), "none"))
	if err != nil || len(missing) != 0 {
		t.Errorf("missing journal: %v, %v", missing, err)
	}
}

func TestOpIDSortableAndUnique(t *testing.T) {
	base := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	var ids []string
	for i := 0; i < 10000; i++ {
		id, err := NewOpID(base.Add(time.Duration(i) * time.Microsecond))
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate op id %s", id)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if !sort.StringsAreSorted(ids) {
		t.Error("op IDs do not sort by creation time")
	}
	if !regexp.MustCompile(`^op_[0-9a-f]{16}[0-9a-f]{8}$`).MatchString(ids[0]) {
		t.Errorf("unexpected op id format: %s", ids[0])
	}
}
