package cli

import (
	"strings"
	"testing"

	"github.com/josepalala-dev/gitgud/internal/statusline"
)

// TestEpicFinish drives an epic through the real CLI: an open child blocks
// finish, the epic merges into develop with --no-ff once the child is done,
// and a finished epic accepts no new children.
func TestEpicFinish(t *testing.T) {
	dir := newRepo(t)
	if code, _, errs := gg(t, dir, "", "init"); code != 0 {
		t.Fatalf("init: %d %s", code, errs)
	}
	if code, _, errs := gg(t, dir, "", "new", "epic", "auth"); code != 0 {
		t.Fatalf("new epic: %d %s", code, errs)
	}
	writeCommit(t, dir, "auth.txt", "scaffold\n", "feat(auth): scaffold epic")

	if code, _, errs := gg(t, dir, "", "new", "feature", "login", "--parent", "epic/auth"); code != 0 {
		t.Fatalf("new feature: %d %s", code, errs)
	}
	writeCommit(t, dir, "login.txt", "form\n", "feat(auth): add login form")

	// The child is still open, so finishing the epic must be refused.
	runGit(t, dir, "checkout", "-q", "epic/auth")
	code, _, errs := gg(t, dir, "", "finish")
	if code != 1 || !strings.Contains(errs, "open children") || !strings.Contains(errs, "feature/login") {
		t.Fatalf("finish with open child: code %d, stderr %q; want 1 naming feature/login", code, errs)
	}

	// Finish the child into the epic, then the epic into develop.
	runGit(t, dir, "checkout", "-q", "feature/login")
	if code, _, errs := gg(t, dir, "", "finish"); code != 0 {
		t.Fatalf("finish child: %d %s", code, errs)
	}
	runGit(t, dir, "checkout", "-q", "epic/auth")
	epicTip := runGit(t, dir, "rev-parse", "HEAD")
	if code, out, errs := gg(t, dir, "", "finish"); code != 0 {
		t.Fatalf("finish epic: %d %s %s", code, out, errs)
	}

	if got := runGit(t, dir, "rev-parse", "develop^2"); got != epicTip {
		t.Fatalf("develop merge second parent = %s; want epic tip %s", got, epicTip)
	}
	if got := runGit(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); got != "develop" {
		t.Fatalf("after epic finish HEAD = %s; want develop", got)
	}

	recs := journalEvents(t, dir, "epic/auth")
	last := recs[len(recs)-1]
	if last["event"] != "branch.finished" || last["target"] != "develop" {
		t.Fatalf("last epic journal record = %v; want branch.finished into develop", last)
	}

	// Finished epics cannot be finished again, and take no new children.
	runGit(t, dir, "checkout", "-q", "epic/auth")
	if code, _, _ := gg(t, dir, "", "finish"); code != 1 {
		t.Fatalf("second finish of epic: code %d; want 1", code)
	}
	runGit(t, dir, "checkout", "-q", "develop")
	if code, _, _ := gg(t, dir, "", "new", "feature", "late", "--parent", "epic/auth"); code != 1 {
		t.Fatalf("new child of finished epic: code %d; want 1", code)
	}
}

// TestEpicNamingAndWarnings checks the naming hint printed by new, and the
// open-child warning on the status line, before any finish is attempted.
func TestEpicNamingAndWarnings(t *testing.T) {
	dir := newRepo(t)
	gg(t, dir, "", "init")
	gg(t, dir, "", "new", "epic", "001-auth")
	writeCommit(t, dir, "auth.txt", "scaffold\n", "feat(auth): scaffold")

	_, out, _ := gg(t, dir, "", "new", "feature", "login", "--parent", "epic/001-auth")
	if !strings.Contains(out, "hint: ") || !strings.Contains(out, "feature/epic-001/login") {
		t.Errorf("misnamed child: stdout %q; want a hint suggesting feature/epic-001/login", out)
	}
	writeCommit(t, dir, "login.txt", "form\n", "feat(auth): login")

	runGit(t, dir, "checkout", "-q", "epic/001-auth")
	line := statusline.Render(dir)
	if !strings.Contains(line, "1 open child") {
		t.Errorf("epic status line %q; want an open-child warning", line)
	}
	code, out, _ := gg(t, dir, "", "new", "fix", "epic-001/session", "--parent", "epic/001-auth")
	if code != 0 || strings.Contains(out, "hint:") {
		t.Errorf("fix named epic-001-session should not hint: code %d, stdout %q", code, out)
	}
}
