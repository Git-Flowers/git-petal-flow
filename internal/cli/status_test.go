package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStatusMatchesGitShort checks that gitgud status prints git status -s.
func TestStatusMatchesGitShort(t *testing.T) {
	dir := newRepo(t)
	if code, out, _ := gg(t, dir, "", "status"); code != 0 || out != "" {
		t.Fatalf("clean tree: code %d, stdout %q; want 0 and nothing", code, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ := gg(t, dir, "", "status")
	want := runGit(t, dir, "status", "-s")
	if code != 0 || strings.TrimSpace(out) != want {
		t.Errorf("gitgud status = %q (code %d); want git status -s %q", out, code, want)
	}
	if code, _, _ := gg(t, dir, "", "status", "extra"); code != 1 {
		t.Errorf("status with args: code %d; want 1", code)
	}
}
