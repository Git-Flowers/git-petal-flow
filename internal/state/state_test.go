package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gitgud", "state.json")
	in := Empty()
	in.Branches["feature/login"] = Branch{
		Type: "feature", Parent: "develop", Slug: "login",
		CreatedAt: "2026-10-16T00:00:00Z", BaseSHA: "abc", Status: "open",
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip changed state:\n in=%+v\nout=%+v", in, out)
	}
	if out.SchemaVersion != "1.0" {
		t.Errorf("schema_version = %q", out.SchemaVersion)
	}
}

func TestMissingFileIsEmptyState(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Branches) != 0 || s.SchemaVersion != "1.0" {
		t.Errorf("missing file gave %+v", s)
	}
}

// Save writes through a temp file, so no temp files remain after a save.
func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	for i := 0; i < 3; i++ {
		if err := Save(path, Empty()); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Errorf("unexpected files after save: %v", entries)
	}
}

func TestOpenChildren(t *testing.T) {
	s := Empty()
	s.Branches["epic/001-auth"] = Branch{Type: "epic", Parent: "develop", Status: "open"}
	s.Branches["feature/epic-001-b"] = Branch{Type: "feature", Parent: "epic/001-auth", Status: "open"}
	s.Branches["feature/epic-001-a"] = Branch{Type: "feature", Parent: "epic/001-auth", Status: "open"}
	s.Branches["fix/epic-001-c"] = Branch{Type: "fix", Parent: "epic/001-auth", Status: "finished"}
	s.Branches["feature/other"] = Branch{Type: "feature", Parent: "develop", Status: "open"}
	want := []string{"feature/epic-001-a", "feature/epic-001-b"}
	if got := OpenChildren(s, "epic/001-auth"); !reflect.DeepEqual(got, want) {
		t.Errorf("OpenChildren = %v; want %v", got, want)
	}
	if got := OpenChildren(s, "epic/none"); len(got) != 0 {
		t.Errorf("OpenChildren for unknown epic = %v; want none", got)
	}
}
