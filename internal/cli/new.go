package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/josepalala-dev/gitgud/internal/events"
	"github.com/josepalala-dev/gitgud/internal/state"
	"github.com/josepalala-dev/gitgud/internal/workflow"
)

const newUsage = "usage: gitgud new feature|fix|epic [description] [--parent <branch>] | gitgud new release <version>"

// newBranch creates a typed branch from its parent and records it.
func (a *app) newBranch(args []string) error {
	parent := ""
	var words []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--parent" {
			if i+1 >= len(args) {
				return errors.New("--parent needs a branch name")
			}
			parent = args[i+1]
			i++
			continue
		}
		words = append(words, args[i])
	}
	if len(words) == 0 {
		return errors.New(newUsage)
	}
	t, err := workflow.ParseType(words[0])
	if err != nil {
		return err
	}
	if parent == "" {
		parent = workflow.Develop
	}

	var slug string
	if t == workflow.Release {
		version := ""
		if len(words) > 1 {
			version = words[1]
		}
		if version == "" {
			version, err = a.prompt("Which version (MAJOR.MINOR.PATCH)? ")
			if err != nil {
				return err
			}
		}
		if err := workflow.ValidateVersion(version); err != nil {
			return err
		}
		slug = version
	} else {
		desc := strings.Join(words[1:], " ")
		if desc == "" {
			desc, err = a.prompt(fmt.Sprintf("What does this %s do? ", t))
			if err != nil {
				return err
			}
		}
		if slug, err = workflow.Slugify(desc); err != nil {
			return err
		}
	}
	name := t.Prefix() + slug

	if err := a.guardClean(); err != nil {
		return err
	}
	st, err := state.Load(a.statePath)
	if err != nil {
		return err
	}
	isOpenEpic := func(b string) bool {
		rec, ok := st.Branches[b]
		return ok && rec.Type == string(workflow.Epic) && rec.Status == "open"
	}
	if err := workflow.ValidateParent(t, parent, isOpenEpic); err != nil {
		return fmt.Errorf("%s from %s: %w", t, parent, err)
	}
	if a.repo.RefExists("refs/heads/" + name) {
		return fmt.Errorf("branch %s already exists", name)
	}
	base, err := a.repo.RevParse(parent)
	if err != nil {
		return fmt.Errorf("parent branch %s not found", parent)
	}
	if err := a.repo.CheckoutNew(name, parent); err != nil {
		return err
	}

	now := a.now()
	st.Branches[name] = state.Branch{
		Type:      string(t),
		Parent:    parent,
		Slug:      slug,
		CreatedAt: now.UTC().Format(time.RFC3339),
		BaseSHA:   base,
		Status:    "open",
	}
	if err := state.Save(a.statePath, st); err != nil {
		return err
	}
	opID, err := events.NewOpID(now)
	if err != nil {
		return err
	}
	rec := events.Record{OpID: opID, Event: "branch.created", Branch: name, Source: parent}
	if err := events.Append(events.Path(a.stateDir, name), &rec, now); err != nil {
		return err
	}
	a.say("created %s from %s", name, parent)
	if hint := workflow.NamingHint(t, parent, name); hint != "" {
		a.say("%s", hint)
	}
	return nil
}
