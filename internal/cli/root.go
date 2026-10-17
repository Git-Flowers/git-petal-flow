// Package cli implements the gitgud command line.
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/josepalala-dev/gitgud/internal/events"
	"github.com/josepalala-dev/gitgud/internal/git"
	"github.com/josepalala-dev/gitgud/internal/state"
	"github.com/josepalala-dev/gitgud/internal/workflow"
)

// Version is the semantic version printed by gitgud --version. Keep it equal
// to the release tag without the leading "v". TestVersionIsSemver guards the
// format.
const Version = "1.1.1"

// Exit codes. Conflicts get their own code so scripts can tell them apart.
const (
	exitOK       = 0
	exitErr      = 1
	exitConflict = 2
)

const usageText = `gitgud - GitFlow-style branches with a per-branch JSONL journal

Usage:
  gitgud init                              create develop and main if missing
  gitgud new feature|fix|epic [desc]       branch from develop (prompts if no desc)
        [--parent <branch>]                parent must be develop or an open epic
  gitgud new release <version>             release/<version> from develop
  gitgud record                            journal commits made with plain git
  gitgud finish [--delete]                 merge the current branch into its parent
  gitgud log [--branch <name>] [--limit N] print the journal as raw JSONL (pipe to jq)
  gitgud status                            short git status (same as git status -s)
  gitgud statusline                        one-line gitgud status for Claude Code's status bar
  gitgud --version

Exit codes: 0 ok, 1 error or refused, 2 merge conflict (merge left in progress).
`

type app struct {
	repo      git.Repo
	stateDir  string
	statePath string
	in        *bufio.Reader
	out       io.Writer
	errw      io.Writer
	now       func() time.Time
}

// conflictError reports a merge that stopped with conflicts. The merge is left
// in progress so the developer can resolve it.
type conflictError struct {
	branch string
	target string
	files  []string
}

func (e *conflictError) Error() string {
	return fmt.Sprintf("merge of %s into %s stopped with conflicts in %v; resolve them and commit on %s, then check out %s and run gitgud finish again",
		e.branch, e.target, e.files, e.target, e.branch)
}

// Run executes one gitgud command in dir and returns the process exit code.
func Run(dir string, args []string, in io.Reader, out, errw io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(out, usageText)
		return exitOK
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintln(out, "gitgud", Version)
		return exitOK
	}
	if args[0] == "statusline" {
		return statusLine(dir, in, out)
	}
	repo, err := git.Open(dir)
	if err != nil {
		return fail(errw, err)
	}
	common, err := repo.CommonDir()
	if err != nil {
		return fail(errw, err)
	}
	stateDir := filepath.Join(common, "gitgud")
	a := &app{
		repo:      repo,
		stateDir:  stateDir,
		statePath: filepath.Join(stateDir, "state.json"),
		in:        bufio.NewReader(in),
		out:       out,
		errw:      errw,
		now:       time.Now,
	}
	rest := args[1:]
	switch args[0] {
	case "init":
		err = a.initRepo()
	case "new":
		err = a.newBranch(rest)
	case "finish":
		err = a.finish(rest)
	case "record":
		err = a.record()
	case "log":
		err = a.log(rest)
	case "status":
		err = a.status(rest)
	default:
		return fail(errw, fmt.Errorf("unknown command %q (run gitgud help)", args[0]))
	}
	if err == nil {
		return exitOK
	}
	var conflict *conflictError
	if errors.As(err, &conflict) {
		fmt.Fprintln(errw, "gitgud:", err)
		return exitConflict
	}
	return fail(errw, err)
}

func fail(w io.Writer, err error) int {
	fmt.Fprintln(w, "gitgud:", err)
	return exitErr
}

func (a *app) say(format string, args ...any) {
	fmt.Fprintf(a.out, format+"\n", args...)
}

// guardClean refuses to touch branches while the tree is dirty or a merge is
// pending. Untracked files count as dirty.
func (a *app) guardClean() error {
	if a.repo.MergeInProgress() {
		return errors.New("a merge is in progress; resolve it and commit first")
	}
	dirty, err := a.repo.Dirty()
	if err != nil {
		return err
	}
	if dirty {
		return errors.New("working tree has uncommitted or untracked changes; commit or stash them first")
	}
	return nil
}

// prompt asks one question on stdin and returns the trimmed answer.
func (a *app) prompt(question string) (string, error) {
	fmt.Fprint(a.out, question)
	line, err := a.in.ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("no input: %w", err)
	}
	s := strings.TrimSpace(line)
	if s == "" {
		return "", errors.New("a value is required")
	}
	return s, nil
}

// capture journals every first-parent commit on branch that is not yet
// recorded. The branch must be checked out.
func (a *app) capture(branch, opID string) (int, error) {
	st, err := state.Load(a.statePath)
	if err != nil {
		return 0, err
	}
	b := st.Branches[branch]
	path := events.Path(a.stateDir, branch)
	seen, err := events.Commits(path)
	if err != nil {
		return 0, err
	}
	commits, err := a.repo.Commits(b.BaseSHA, "HEAD")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range commits {
		if seen[c.SHA] {
			continue
		}
		rec := events.Record{
			OpID:    opID,
			Event:   "commit.recorded",
			Branch:  branch,
			Commit:  c.SHA,
			Parents: c.Parents,
			Subject: c.Subject,
			Type:    workflow.ConventionalType(c.Subject),
		}
		if err := events.Append(path, &rec, a.now()); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
