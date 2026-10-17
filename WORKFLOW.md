# GitHero-flow workflow: epics, children and releases

This document explains how the pieces of GitHero-flow fit together: what an epic is,
what makes a branch a child of one, what the warnings mean, and how a release
differs from an epic. The README covers the commands. This covers the concepts.

## The four branch kinds

| Kind | Branch name | Created from | Finished into |
| ---- | ----------- | ------------ | ------------- |
| Feature | `feature/<name>` or `feature/epic-NNN/<name>` | `develop`, or an open epic | its parent |
| Fix | `fix/<name>` or `fix/epic-NNN/<name>` | `develop`, or an open epic | its parent |
| Epic | `epic/NNN-<name>` | `develop` | `develop`, once no child is open |
| Release | `release/X.Y.Z` | `develop` | `main` (tagged `vX.Y.Z`), then `develop` |

- A **feature** or **fix** is one unit of work. It is finished into its parent,
  which is `develop` or an epic.
- An **epic** groups the features and fixes that belong together. It is a
  branch of its own, and it finishes into `develop`.
- A **release** is a version. Finishing it merges into `main` and creates a
  tag. The tag is the release. The release branch exists only to make that
  merge and tag.

## Epics and their children

A child is a feature or fix that was created with `--parent epic/NNN-<name>`.
Its work lands in the epic first, not in `develop`.

```text
develop ──────────────────────────────────────────────────────────►  develop
   │                                                                  ▲
   └─► epic/001-auth  (epic: groups the work)                         │
          │                                                           │
          ├─► feature/epic-001/login-form ──finish──► epic/001-auth ──┤ finish
          │        (child)                                            │ (--no-ff)
          └─► fix/epic-001/session-timeout ──finish──► epic/001-auth ─┘
                   (child)
```

Read it top to bottom:

1. `gitgud new epic "001 auth"` creates `epic/001-auth` from `develop`.
2. `gitgud new feature "epic-001/login form" --parent epic/001-auth` creates a
   child. It starts from the epic, not from `develop`.
3. `gitgud finish` on the child merges it into the epic. The child is now
   finished.
4. Once every child is finished, `gitgud finish` on the epic merges it into
   `develop`.

A child can also be a fix. A fix under an epic is related to that epic's work,
the same way a feature is.

## How GitHero-flow knows which branches are children

The link is recorded in GitHero-flow's `state.json`, not in git. When `gitgud new`
creates a branch, it records the branch's `parent` and a `status` of `open`.
`gitgud finish` marks the status `finished`. To list an epic's open children,
GitHero-flow looks at `state.json` for branches whose parent is that epic and whose
status is still `open`.

That is why the link has to be made with `gitgud new --parent`. A branch made
with plain `git checkout -b` is invisible to GitHero-flow, so the epic will not wait
for it.

## Naming children for people

`state.json` is for GitHero-flow. A person reading `git branch` should be able to see
the epic too, so the name carries the number:

- The epic is `epic/NNN-<name>`, for example `epic/001-auth`.
- Each child is `feature/epic-NNN/<name>` or `fix/epic-NNN/<name>`, with the
  same number, for example `feature/epic-001/login-form`.
- To list the children of an epic:

  ```sh
  git branch --list 'feature/epic-001/*' 'fix/epic-001/*'
  ```

The name is a convention, not a check. GitHero-flow does not reject a child whose
number does not match its parent. But `gitgud new` prints a hint when it sees a
mismatch, with the name it suggests:

```text
hint: fix/login is a child of epic/001-auth; name it fix/epic-001/login so people can see the epic from the branch name
```

Keep the numbers unique. Two epics with the same number make the names
ambiguous.

## The warnings

GitHero-flow gives warnings at three points, all based on `state.json`:

- **When you create a child with a name that does not follow the convention.**
  `gitgud new` prints the `hint:` line above and still creates the branch.
- **On the status line, while the epic has open children.** For example:

  ```text
  gitgud ● epic/001-auth · from develop · ... · ⚠ 1 open child, finish them before the epic
  ```

- **When you try to finish an epic with open children.** The finish is refused
  and the children are named:

  ```text
  gitgud: epic epic/001-auth still has open children: [feature/epic-001/login-form]; finish them first
  ```

The status line and the refusal both read `state.json`, so they agree. If a
child was deleted with plain git but is still `open` in `state.json`, the epic
keeps refusing. Only `gitgud finish` changes a child's status.

## Releases

A release is a tag on `main`. It is not a folder of work.

- An epic is usually released together with other epics and fixes. They are
  finished into `develop` first. The release then picks up everything in
  `develop`.
- `gitgud new release 1.4.0` creates `release/1.4.0` from `develop`.
  `gitgud finish` then merges it into `main`, creates the tag `v1.4.0`, and
  merges it back into `develop`.
- The version must be `MAJOR.MINOR.PATCH`, for example `1.4.0`. The tag is
  `v1.4.0`. GitHero-flow does not accept a form like `v001`.

```text
develop ──► release/1.4.0 ──► main  (tag v1.4.0) ──► develop
```

## Checklist for a human

Before you create a child:

- [ ] The epic has a number, for example `epic/001-auth`.
- [ ] The child name starts with `epic-001/`, using the same number.
- [ ] The child is created with `gitgud new ... --parent epic/001-auth`.

Before you finish an epic:

- [ ] `git branch --list 'feature/epic-001/*' 'fix/epic-001/*'` shows the
      children you expect.
- [ ] The status line shows no open children, or you have finished them.
- [ ] Every child is finished with `gitgud finish`, not with plain git.

Before you release:

- [ ] The epics and fixes you want are finished into `develop`.
- [ ] The version is `MAJOR.MINOR.PATCH`.

## See also

- [README.md](README.md) for the commands, the naming hint and the journal.
- [docs/journal-format.md](docs/journal-format.md) for the record fields.
