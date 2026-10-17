package git

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// clearRepoEnv drops the variables git exports to hooks, so tests that run
// under a pre-commit hook still talk to their own temp repos.
func clearRepoEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR",
		"GIT_PREFIX", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
	} {
		if v, ok := os.LookupEnv(k); ok {
			os.Unsetenv(k)
			t.Cleanup(func() { os.Setenv(k, v) })
		}
	}
}

func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	clearRepoEnv(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
}

func TestRunErrorCarriesArgvAndStderr(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	_, err := Repo{Dir: dir}.Run("rev-parse", "--verify", "no-such-ref")
	var gerr *Error
	if !errors.As(err, &gerr) {
		t.Fatalf("err = %v; want *Error", err)
	}
	if !strings.Contains(err.Error(), "rev-parse") || gerr.Stderr == "" {
		t.Errorf("error lacks argv or stderr: %v", err)
	}
}

func TestOpenRejectsNonRepo(t *testing.T) {
	isolateGit(t)
	if _, err := Open(t.TempDir()); !errors.Is(err, ErrNotRepo) {
		t.Errorf("Open outside repo err = %v; want ErrNotRepo", err)
	}
}

func TestDetachedHeadAndDirty(t *testing.T) {
	isolateGit(t)
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "chore: init")
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := r.CurrentBranch(); err != nil || b != "main" {
		t.Fatalf("CurrentBranch = %q, %v", b, err)
	}
	if err := os.WriteFile(dir+"/untracked.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirty, err := r.Dirty(); err != nil || !dirty {
		t.Errorf("untracked file not reported dirty: %v, %v", dirty, err)
	}
	run("checkout", "-q", "--detach")
	if _, err := r.CurrentBranch(); !errors.Is(err, ErrDetachedHead) {
		t.Errorf("detached err = %v; want ErrDetachedHead", err)
	}
}
