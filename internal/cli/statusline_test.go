package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// statusOf runs `gitgud statusline` in dir with the given stdin payload.
func statusOf(t *testing.T, dir, stdin string) string {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(dir, []string{"statusline"}, strings.NewReader(stdin), &out, &errb)
	if code != 0 {
		t.Fatalf("statusline exit %d: %s", code, errb.String())
	}
	return strings.TrimRight(out.String(), "\n")
}

func TestStatusLineSilentOutsideRepo(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if got := statusOf(t, t.TempDir(), ""); got != "" {
		t.Errorf("outside a repo printed %q", got)
	}
}

func TestStatusLineStates(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	dir := newRepo(t)

	if got := statusOf(t, dir, ""); !strings.Contains(got, "not a gitgud branch") {
		t.Errorf("on main: %q", got)
	}

	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "login")
	if got := statusOf(t, dir, ""); !strings.Contains(got, "feature/login") ||
		!strings.Contains(got, "0 recorded") || !strings.Contains(got, "0 to record") {
		t.Errorf("fresh feature: %q", got)
	}

	writeCommit(t, dir, "a.txt", "a", "feat: login")
	if got := statusOf(t, dir, ""); !strings.Contains(got, "1 to record") {
		t.Errorf("after plain commit: %q", got)
	}

	gg(t, dir, "", "record")
	if got := statusOf(t, dir, ""); !strings.Contains(got, "1 recorded") || !strings.Contains(got, "0 to record") ||
		!strings.Contains(got, "last: commit.recorded") {
		t.Errorf("after record: %q", got)
	}

	gg(t, dir, "", "finish")
	runGit(t, dir, "checkout", "-q", "feature/login")
	if got := statusOf(t, dir, ""); !strings.Contains(got, "finished into develop") {
		t.Errorf("finished branch: %q", got)
	}
}

// An open stdin that never closes must not hang the status line; it falls back
// to the working directory.
func TestStatusLineDoesNotHangOnOpenStdin(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	repo := newRepo(t)
	gg(t, repo, "", "init")
	gg(t, repo, "", "new", "fix", "hang")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	defer r.Close()

	var out, errb bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- Run(repo, []string{"statusline"}, r, &out, &errb) }()
	select {
	case code := <-done:
		if code != 0 || !strings.Contains(out.String(), "fix/hang") {
			t.Errorf("exit %d, out %q", code, out.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("statusline hung on an open stdin")
	}
}

func TestStatusLineShowsConflict(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "clash")
	writeCommit(t, dir, "shared.txt", "feature", "feat: feature side")
	runGit(t, dir, "checkout", "-q", "develop")
	writeCommit(t, dir, "shared.txt", "develop", "fix: develop side")
	runGit(t, dir, "checkout", "-q", "feature/clash")
	if code, _, _ := gg(t, dir, "", "finish"); code != 2 {
		t.Fatalf("setup: finish code %d", code)
	}
	if got := statusOf(t, dir, ""); !strings.Contains(got, "merge in progress") || !strings.Contains(got, "feature/clash") {
		t.Errorf("during conflict: %q", got)
	}
}

// Claude Code sends the project directory on stdin; the status must follow it
// rather than the directory the command was started in.
func TestStatusLineFollowsStdinDirectory(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	repo := newRepo(t)
	gg(t, repo, "", "init")
	gg(t, repo, "", "new", "fix", "typo")

	payload := `{"cwd":"` + filepath.ToSlash(repo) + `","workspace":{"current_dir":"` + filepath.ToSlash(repo) + `"}}`
	got := statusOf(t, t.TempDir(), payload)
	if !strings.Contains(got, "fix/typo") {
		t.Errorf("stdin directory ignored: %q", got)
	}
}
