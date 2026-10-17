package cli

import (
	"fmt"

	"github.com/josepalala-dev/gitgud/internal/events"
	"github.com/josepalala-dev/gitgud/internal/state"
	"github.com/josepalala-dev/gitgud/internal/workflow"
)

// finish merges the current feature, fix, epic or release into its recorded
// targets with --no-ff. An epic is refused while any recorded child is open.
func (a *app) finish(args []string) error {
	deleteBranch := false
	for _, arg := range args {
		if arg != "--delete" {
			return fmt.Errorf("unknown argument %q", arg)
		}
		deleteBranch = true
	}

	branch, err := a.repo.CurrentBranch()
	if err != nil {
		return err
	}
	st, err := state.Load(a.statePath)
	if err != nil {
		return err
	}
	b, ok := st.Branches[branch]
	if !ok {
		return fmt.Errorf("branch %s is not recorded; only branches made with gitgud new can be finished", branch)
	}
	if b.Status == "finished" {
		return fmt.Errorf("%s is already finished", branch)
	}
	t := workflow.Type(b.Type)
	if t == workflow.Epic {
		if open := state.OpenChildren(st, branch); len(open) > 0 {
			return fmt.Errorf("epic %s still has open children: %v; finish them first", branch, open)
		}
	}
	tag := ""
	if t == workflow.Release {
		tag = "v" + b.Slug
		if a.repo.RefExists("refs/tags/" + tag) {
			return fmt.Errorf("tag %s already exists", tag)
		}
	}
	if err := a.guardClean(); err != nil {
		return err
	}

	now := a.now()
	opID, err := events.NewOpID(now)
	if err != nil {
		return err
	}
	if _, err := a.capture(branch, opID); err != nil {
		return err
	}
	journal := events.Path(a.stateDir, branch)

	var final string
	if t == workflow.Release {
		mainSHA, err := a.merge(branch, workflow.Main, opID, journal)
		if err != nil {
			return err
		}
		if err := a.repo.Tag(tag, "Release "+b.Slug); err != nil {
			return err
		}
		rec := events.Record{OpID: opID, Event: "tag.created", Branch: branch, Commit: mainSHA, Tag: tag, Target: workflow.Main}
		if err := events.Append(journal, &rec, now); err != nil {
			return err
		}
		final, err = a.merge(branch, workflow.Develop, opID, journal)
		if err != nil {
			return err
		}
	} else {
		final, err = a.merge(branch, b.Parent, opID, journal)
		if err != nil {
			return err
		}
	}

	rec := events.Record{OpID: opID, Event: "branch.finished", Branch: branch, Commit: final, Source: branch, Target: b.Parent}
	if t == workflow.Release {
		rec.Target = workflow.Develop
	}
	if err := events.Append(journal, &rec, now); err != nil {
		return err
	}
	b.Status = "finished"
	st.Branches[branch] = b
	if err := state.Save(a.statePath, st); err != nil {
		return err
	}
	if deleteBranch {
		if err := a.repo.DeleteBranch(branch); err != nil {
			return err
		}
	}
	a.say("finished %s", branch)
	return nil
}

// merge checks out target, merges branch into it with --no-ff, and journals
// the result. On conflict it records merge.conflict, leaves the merge in
// progress, and returns a conflictError.
func (a *app) merge(branch, target, opID, journal string) (string, error) {
	if err := a.repo.Checkout(target); err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Merge branch '%s' into %s", branch, target)
	if err := a.repo.MergeNoFF(branch, msg); err != nil {
		files, uerr := a.repo.Unmerged()
		if uerr != nil || len(files) == 0 {
			return "", err
		}
		rec := events.Record{OpID: opID, Event: "merge.conflict", Branch: branch, Target: target, Files: files}
		if aerr := events.Append(journal, &rec, a.now()); aerr != nil {
			return "", aerr
		}
		return "", &conflictError{branch: branch, target: target, files: files}
	}
	sha, err := a.repo.Head()
	if err != nil {
		return "", err
	}
	rec := events.Record{OpID: opID, Event: "merged", Branch: branch, Commit: sha, Source: branch, Target: target}
	if err := events.Append(journal, &rec, a.now()); err != nil {
		return "", err
	}
	return sha, nil
}
