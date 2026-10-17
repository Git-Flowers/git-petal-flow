package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

// newRepo creates a temp repo with one commit on main and isolated git config.
// Nothing here touches the real repository.
func newRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	clearRepoEnv(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.invalid")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.invalid")
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "commit", "-q", "--allow-empty", "-m", "chore: init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitOK(dir string, args ...string) bool {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run() == nil
}

func gg(t *testing.T, dir, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(dir, args, strings.NewReader(stdin), &out, &errb)
	return code, out.String(), errb.String()
}

func writeCommit(t *testing.T, dir, file, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", file)
	runGit(t, dir, "commit", "-q", "-m", msg)
}

func journalEvents(t *testing.T, dir, branch string) []map[string]any {
	t.Helper()
	name := strings.ReplaceAll(branch, "/", "__") + ".jsonl"
	data, err := os.ReadFile(filepath.Join(dir, ".git", "gitgud", "journal", name))
	if err != nil {
		t.Fatalf("read journal for %s: %v", branch, err)
	}
	var recs []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var r map[string]any
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("journal line is not JSON: %s", line)
		}
		recs = append(recs, r)
	}
	return recs
}

func TestFeatureAndReleaseFlow(t *testing.T) {
	dir := newRepo(t)

	if code, out, errs := gg(t, dir, "", "init"); code != 0 {
		t.Fatalf("init: %d %s %s", code, out, errs)
	}
	if !gitOK(dir, "show-ref", "--verify", "--quiet", "refs/heads/develop") {
		t.Fatal("init did not create develop")
	}

	// Prompted description, slugged to feature/login-form.
	if code, _, errs := gg(t, dir, "login form\n", "new", "feature"); code != 0 {
		t.Fatalf("new feature: %d %s", code, errs)
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/login-form" {
		t.Fatalf("current branch = %s", got)
	}
	writeCommit(t, dir, "login.txt", "form", "feat(auth): add login form")
	writeCommit(t, dir, "typo.txt", "fixed", "fix: correct typo")

	code, out, _ := gg(t, dir, "", "record")
	if code != 0 || !strings.Contains(out, "recorded 2 commit(s)") {
		t.Fatalf("record: %d %q", code, out)
	}

	// A dirty tree must refuse finish before touching any branch.
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := gg(t, dir, "", "finish"); code != 1 || !strings.Contains(errs, "uncommitted") {
		t.Fatalf("dirty finish: %d %q; want refusal", code, errs)
	}
	if err := os.Remove(filepath.Join(dir, "scratch.txt")); err != nil {
		t.Fatal(err)
	}

	if code, _, errs := gg(t, dir, "", "finish", "--delete"); code != 0 {
		t.Fatalf("finish feature: %d %s", code, errs)
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "develop" {
		t.Errorf("after finish on %s; want develop", got)
	}
	if parents := strings.Fields(runGit(t, dir, "rev-list", "--parents", "-n1", "develop")); len(parents) != 3 {
		t.Errorf("develop tip is not a two-parent merge: %v", parents)
	}
	if gitOK(dir, "show-ref", "--verify", "--quiet", "refs/heads/feature/login-form") {
		t.Error("feature branch was not deleted with --delete")
	}

	feat := journalEvents(t, dir, "feature/login-form")
	var names []string
	for _, r := range feat {
		names = append(names, r["event"].(string))
	}
	want := "branch.created commit.recorded commit.recorded merged branch.finished"
	if strings.Join(names, " ") != want {
		t.Errorf("feature journal events = %v; want %s", names, want)
	}
	if feat[1]["type"] != "feat" || feat[2]["type"] != "fix" {
		t.Errorf("commit types = %v, %v", feat[1]["type"], feat[2]["type"])
	}

	code, out, errs := gg(t, dir, "", "log", "--branch", "feature/login-form", "--limit", "1")
	if code != 0 || !strings.Contains(out, `"event":"branch.finished"`) || errs != "" {
		t.Errorf("log --limit 1: %d %q %q", code, out, errs)
	}

	// Release from develop, tagged on main, merged back to develop.
	if code, _, errs := gg(t, dir, "", "new", "release", "1.0.0"); code != 0 {
		t.Fatalf("new release: %d %s", code, errs)
	}
	writeCommit(t, dir, "CHANGELOG.md", "1.0.0", "chore: release notes")
	if code, _, errs := gg(t, dir, "", "finish"); code != 0 {
		t.Fatalf("finish release: %d %s", code, errs)
	}
	if !gitOK(dir, "show-ref", "--verify", "--quiet", "refs/tags/v1.0.0") {
		t.Error("tag v1.0.0 missing")
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "develop" {
		t.Errorf("after release finish on %s; want develop", got)
	}
	if !strings.Contains(runGit(t, dir, "branch", "--merged", "develop"), "release/1.0.0") {
		t.Error("release not merged into develop")
	}
	rel := journalEvents(t, dir, "release/1.0.0")
	var relNames []string
	for _, r := range rel {
		relNames = append(relNames, r["event"].(string))
	}
	if strings.Join(relNames, " ") != "branch.created commit.recorded merged tag.created merged branch.finished" {
		t.Errorf("release journal events = %v", relNames)
	}
}

func TestFinishConflictLeavesMergeInProgress(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	if code, _, errs := gg(t, dir, "", "new", "feature", "clash"); code != 0 {
		t.Fatalf("new: %d %s", code, errs)
	}
	writeCommit(t, dir, "shared.txt", "feature", "feat: feature side")
	runGit(t, dir, "checkout", "-q", "develop")
	writeCommit(t, dir, "shared.txt", "develop", "fix: develop side")
	runGit(t, dir, "checkout", "-q", "feature/clash")

	code, _, errs := gg(t, dir, "", "finish")
	if code != 2 || !strings.Contains(errs, "conflicts") || !strings.Contains(errs, "finish again") {
		t.Fatalf("finish with conflict: code %d, stderr %q; want 2 and the recovery hint", code, errs)
	}
	if !gitOK(dir, "rev-parse", "--quiet", "--verify", "MERGE_HEAD") {
		t.Error("merge was aborted; MERGE_HEAD should still exist")
	}
	var found bool
	for _, r := range journalEvents(t, dir, "feature/clash") {
		if r["event"] == "merge.conflict" {
			found = true
			files := r["files"].([]any)
			if len(files) != 1 || files[0] != "shared.txt" {
				t.Errorf("conflict files = %v", files)
			}
		}
	}
	if !found {
		t.Error("merge.conflict not journaled")
	}
}

func TestNewRefusesDirtyTreeAndBadParent(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	if code, _, errs := gg(t, dir, "", "new", "feature", "--parent", "main", "x"); code != 1 || !strings.Contains(errs, "not allowed") {
		t.Errorf("feature from main: %d %q", code, errs)
	}
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := gg(t, dir, "", "new", "fix", "readline from user"); code != 1 || !strings.Contains(errs, "uncommitted") {
		t.Errorf("untracked tree: %d %q", code, errs)
	}
	if gitOK(dir, "show-ref", "--verify", "--quiet", "refs/heads/fix/readline-from-user") {
		t.Error("branch created despite refusal")
	}
}
