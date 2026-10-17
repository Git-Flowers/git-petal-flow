package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestInitIsIdempotent(t *testing.T) {
	dir := newRepo(t)
	if code, _, errs := gg(t, dir, "", "init"); code != 0 {
		t.Fatalf("first init: %d %s", code, errs)
	}
	code, out, errs := gg(t, dir, "", "init")
	if code != 0 || !strings.Contains(out, "already initialized") {
		t.Errorf("second init: %d %q %q; want no-op", code, out, errs)
	}
}

func TestNewRefusesExistingBranch(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	if code, _, errs := gg(t, dir, "", "new", "fix", "dup"); code != 0 {
		t.Fatalf("first new: %d %s", code, errs)
	}
	runGit(t, dir, "checkout", "-q", "develop")
	code, _, errs := gg(t, dir, "", "new", "fix", "dup")
	if code != 1 || !strings.Contains(errs, "already exists") {
		t.Errorf("duplicate new: %d %q; want refusal", code, errs)
	}
}

func TestFinishRefusesUnrecordedBranch(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	runGit(t, dir, "checkout", "-q", "-b", "feature/manual", "develop")
	writeCommit(t, dir, "m.txt", "m", "feat: manual")
	code, _, errs := gg(t, dir, "", "finish")
	if code != 1 || !strings.Contains(errs, "not recorded") {
		t.Errorf("finish unrecorded: %d %q; want refusal", code, errs)
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/manual" {
		t.Errorf("refused finish still moved HEAD to %s", got)
	}
}

func TestRecordIsIdempotentAndUsesFullSHA(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "idem")
	writeCommit(t, dir, "i.txt", "i", "feat: idempotent")

	code, out, _ := gg(t, dir, "", "record")
	if code != 0 || !strings.Contains(out, "recorded 1 commit(s)") {
		t.Fatalf("first record: %d %q", code, out)
	}
	if _, out, _ := gg(t, dir, "", "record"); !strings.Contains(out, "recorded 0 commit(s)") {
		t.Errorf("second record journaled again: %q", out)
	}
	fullSHA := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, r := range journalEvents(t, dir, "feature/idem") {
		if c, ok := r["commit"].(string); ok && !fullSHA.MatchString(c) {
			t.Errorf("commit %q is not a full 40-char SHA", c)
		}
	}
}

func TestMergeInProgressBlocksNewBranch(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "clash")
	writeCommit(t, dir, "shared.txt", "feature", "feat: feature side")
	runGit(t, dir, "checkout", "-q", "develop")
	writeCommit(t, dir, "shared.txt", "develop", "fix: develop side")
	runGit(t, dir, "checkout", "-q", "feature/clash")
	if code, _, _ := gg(t, dir, "", "finish"); code != 2 {
		t.Fatalf("setup: finish code %d; want 2", code)
	}
	code, _, errs := gg(t, dir, "", "new", "fix", "other")
	if code != 1 || !strings.Contains(errs, "merge is in progress") {
		t.Errorf("new during merge: %d %q; want refusal", code, errs)
	}
}

func TestLogPassesMalformedLineThrough(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "bad")
	path := filepath.Join(dir, ".git", "gitgud", "journal", "feature__bad.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	code, out, errs := gg(t, dir, "", "log", "--branch", "feature/bad")
	if code != 0 || !strings.Contains(out, "{not json") || !strings.Contains(errs, "1 malformed") {
		t.Errorf("log with corrupt line: %d out=%q err=%q", code, out, errs)
	}
}

// A linked worktree must journal into the main repository's .git directory.
func TestWorktreeWritesToCommonDir(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	wt := filepath.Join(t.TempDir(), "wt")
	runGit(t, dir, "worktree", "add", "-q", "--detach", wt, "develop")

	if code, _, errs := gg(t, wt, "", "new", "fix", "wt"); code != 0 {
		t.Fatalf("new in worktree: %d %s", code, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git", "gitgud", "journal", "fix__wt.jsonl")); err != nil {
		t.Errorf("journal not in common .git dir: %v", err)
	}
	state, err := os.ReadFile(filepath.Join(dir, ".git", "gitgud", "state.json"))
	if err != nil || !strings.Contains(string(state), "fix/wt") {
		t.Errorf("state not in common .git dir: %v", err)
	}
}

func TestReleaseMergesMainWithTwoParents(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	if code, _, errs := gg(t, dir, "", "new", "release", "2.0.0"); code != 0 {
		t.Fatalf("new release: %d %s", code, errs)
	}
	writeCommit(t, dir, "NOTES.md", "2.0.0", "chore: notes")
	if code, _, errs := gg(t, dir, "", "finish"); code != 0 {
		t.Fatalf("finish: %d %s", code, errs)
	}
	parents := strings.Fields(runGit(t, dir, "rev-list", "--parents", "-n1", "main"))
	if len(parents) != 3 {
		t.Errorf("main tip is not a merge of the release: %v", parents)
	}
	if runGit(t, dir, "tag", "--list", "v2.0.0") != "v2.0.0" {
		t.Error("tag v2.0.0 missing")
	}
}

// The recovery path from a conflict: resolve and commit on the target, check
// out the branch, and run finish again. The branch must end up finished.
func TestFinishRecoversAfterConflictResolved(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "feature", "recover")
	writeCommit(t, dir, "shared.txt", "feature", "feat: feature side")
	runGit(t, dir, "checkout", "-q", "develop")
	writeCommit(t, dir, "shared.txt", "develop", "fix: develop side")
	runGit(t, dir, "checkout", "-q", "feature/recover")
	if code, _, _ := gg(t, dir, "", "finish"); code != 2 {
		t.Fatalf("setup: finish code %d; want 2", code)
	}

	// The merge is still in progress on develop; committing completes it.
	writeCommit(t, dir, "shared.txt", "resolved", "fix: resolve recover")
	runGit(t, dir, "checkout", "-q", "feature/recover")
	if code, _, errs := gg(t, dir, "", "finish"); code != 0 {
		t.Fatalf("finish after resolve: %d %s", code, errs)
	}
	st, err := os.ReadFile(filepath.Join(dir, ".git", "gitgud", "state.json"))
	if err != nil || !strings.Contains(string(st), `"status": "finished"`) {
		t.Errorf("branch not marked finished after recovery: %v", err)
	}
}
