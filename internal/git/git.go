// Package git runs the git binary with argv slices only. It never builds a
// shell string, so arguments are never interpreted by a shell.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrNotRepo      = errors.New("not inside a git repository")
	ErrDetachedHead = errors.New("HEAD is detached; check out a branch first")
)

// Error reports a git command that exited non-zero.
type Error struct {
	Argv   []string
	Code   int
	Stderr string
}

func (e *Error) Error() string {
	return fmt.Sprintf("git %s failed (exit %d): %s",
		strings.Join(e.Argv, " "), e.Code, strings.TrimSpace(e.Stderr))
}

// Commit is one commit on a branch's first-parent line.
type Commit struct {
	SHA     string
	Parents []string
	Subject string
}

// Repo runs git inside one directory.
type Repo struct {
	Dir string
}

// Run executes git with args and returns trimmed stdout.
func (r Repo) Run(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = r.Dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		code := -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return "", &Error{Argv: args, Code: code, Stderr: msg}
	}
	return strings.TrimRight(stdout.String(), "\r\n"), nil
}

// Open verifies that dir is inside a work tree.
func Open(dir string) (Repo, error) {
	r := Repo{Dir: dir}
	out, err := r.Run("rev-parse", "--is-inside-work-tree")
	if err != nil || out != "true" {
		return Repo{}, ErrNotRepo
	}
	return r, nil
}

// CommonDir returns the absolute common .git directory, shared by worktrees.
func (r Repo) CommonDir() (string, error) {
	return r.Run("rev-parse", "--path-format=absolute", "--git-common-dir")
}

// CurrentBranch returns the checked-out branch name.
func (r Repo) CurrentBranch() (string, error) {
	out, err := r.Run("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || out == "" {
		return "", ErrDetachedHead
	}
	return out, nil
}

// RevParse resolves rev to a full commit SHA.
func (r Repo) RevParse(rev string) (string, error) {
	return r.Run("rev-parse", "--verify", "--quiet", rev+"^{commit}")
}

// Head returns the full SHA of HEAD.
func (r Repo) Head() (string, error) {
	return r.RevParse("HEAD")
}

// RefExists reports whether a full ref name such as refs/heads/main exists.
func (r Repo) RefExists(ref string) bool {
	_, err := r.Run("show-ref", "--verify", "--quiet", ref)
	return err == nil
}

// Dirty reports uncommitted changes, including untracked files.
func (r Repo) Dirty() (bool, error) {
	out, err := r.Run("status", "--porcelain", "--untracked-files=all")
	return out != "", err
}

// MergeInProgress reports whether a merge is waiting for resolution.
func (r Repo) MergeInProgress() bool {
	_, err := r.Run("rev-parse", "--quiet", "--verify", "MERGE_HEAD")
	return err == nil
}

// Checkout switches to an existing branch.
func (r Repo) Checkout(branch string) error {
	_, err := r.Run("checkout", branch)
	return err
}

// CheckoutNew creates name from start and switches to it.
func (r Repo) CheckoutNew(name, start string) error {
	_, err := r.Run("checkout", "-b", name, start)
	return err
}

// CreateBranch creates name at start without switching to it.
func (r Repo) CreateBranch(name, start string) error {
	_, err := r.Run("branch", name, start)
	return err
}

// MergeNoFF merges branch into the current branch with a merge commit.
func (r Repo) MergeNoFF(branch, msg string) error {
	_, err := r.Run("merge", "--no-ff", "-m", msg, branch)
	return err
}

// Unmerged lists paths with unresolved conflicts.
func (r Repo) Unmerged() ([]string, error) {
	out, err := r.Run("diff", "--name-only", "--diff-filter=U")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// Tag creates an annotated tag at HEAD.
func (r Repo) Tag(name, msg string) error {
	_, err := r.Run("tag", "-a", name, "-m", msg)
	return err
}

// DeleteBranch deletes a fully merged branch.
func (r Repo) DeleteBranch(name string) error {
	_, err := r.Run("branch", "-d", name)
	return err
}

// Commits lists the first-parent commits in base..head, oldest first. An
// empty base lists everything reachable from head.
func (r Repo) Commits(base, head string) ([]Commit, error) {
	rng := head
	if base != "" {
		rng = base + ".." + head
	}
	out, err := r.Run("log", "--first-parent", "--reverse",
		"--format=%H%x1f%P%x1f%s", rng, "--")
	if err != nil || out == "" {
		return nil, err
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\x1f", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("unexpected git log line: %q", line)
		}
		commits = append(commits, Commit{
			SHA:     parts[0],
			Parents: strings.Fields(parts[1]),
			Subject: parts[2],
		})
	}
	return commits, nil
}
