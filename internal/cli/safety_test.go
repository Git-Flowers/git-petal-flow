package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbidden are argv strings gitgud must never pass to git: no history
// rewrites, no fast-forward-only merges, no pushes, no force flags.
var forbidden = []string{`"rebase"`, `"--ff-only"`, `"reset"`, `"--hard"`, `"push"`, `"--force"`}

// scanForbidden returns the forbidden strings found in non-test Go files under
// root, as "path: needle" entries.
func scanForbidden(t *testing.T, root string) []string {
	t.Helper()
	var hits []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, needle := range forbidden {
			if strings.Contains(string(data), needle) {
				hits = append(hits, path+": "+needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hits
}

func TestNoForbiddenGitInvocationsInSource(t *testing.T) {
	if hits := scanForbidden(t, "../.."); len(hits) > 0 {
		t.Errorf("forbidden git usage in source: %v", hits)
	}
}

func TestSafetyScannerCatchesInjectedUsage(t *testing.T) {
	dir := t.TempDir()
	src := `package x
var _ = []string{"git", "rebase", "main"}
`
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if hits := scanForbidden(t, dir); len(hits) != 1 {
		t.Errorf("injected rebase not detected: %v", hits)
	}
}
