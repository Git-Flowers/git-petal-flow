# Git Petal Flow

A small workflow layer over `git`. It creates feature, fix and epic branches
with the right parent, merges them back with `--no-ff`, and keeps a per-branch
JSONL journal of the commits on each branch so the history can be read with
`jq` or any other line-oriented tool. Release branches are available too, for
teams that cut releases that way.

Git-petal-flow is not a replacement for git. Every operation is a plain `git` command
underneath, and the git history is the source of truth.

## Name

Git Petal flow  was formerly called **gitgud**. It was renamed to match what it
does: it is not GitFlow, and it does not follow GitFlow's conventions or its
feature set. This is why the cli is **gitgud** and **not git-petal-flow**. 

The workflow has three branch types:

- **feature** for new work, branched from `develop`.
- **fix** for bug fixes, also branched from `develop`. There is no hotfix
  branch; a fix goes through the same path as any other change.
- **epic** for a group of related features and fixes. Epics are long-lived
  branches that collect their children before they merge into `develop`.

`develop` and `main` are also long-lived. Release branches are optional, since
how you release depends on your project. The command and binary names, `gitgud`
and `gitgud-mcp`, have not changed yet, so the examples below still use them.

## Install

Requires Go 1.26.8 and git 2.31 or newer.

```sh
go build -o bin/gitgud ./cmd/gitgud
```

Put `bin/gitgud` on your `PATH`. Run `gitgud --version` to check it.

## Basic usage

Run these inside a git repository that has at least one commit.

### Set up once

```sh
gitgud init        # creates develop (from main) and main (from HEAD) if missing
```

### Start a feature or a fix

```sh
gitgud new feature "login form"          # feature/login-form from develop, checked out
gitgud new fix "readline from user"      # fix/readline-from-user from develop
gitgud new feature "epic-001/login validation" --parent epic/001-auth   # feature/epic-001/login-validation
```

Leave out the description and Git-petal-flow asks for one. The description becomes
the branch slug: `readline from user` gives `fix/readline-from-user`.

### Work and record

Commit with git as you normally would. Use Conventional Commit subjects
(`feat(auth): add login form`), because the journal reads the type from the
subject.

```sh
git add -A && git commit -m "feat(auth): add login form"
gitgud record      # journal any commits made with plain git since the last record
```

`record` is optional before `finish`, which journals the commits itself. Run it
when you want the journal up to date now.

### Finish a feature or fix

```sh
gitgud finish             # merge into the recorded parent with --no-ff; keep the branch
gitgud finish --delete    # the same, then delete the branch
```

`finish` refuses on a dirty tree (uncommitted or untracked files) and on a
branch that Git-petal-flow did not create.

### Release

```sh
gitgud new release 1.4.0  # release/1.4.0 from develop, checked out
# ...commit any release changes with git...
gitgud finish            # merge into main, tag v1.4.0, merge back into develop
```

The version must be `MAJOR.MINOR.PATCH`, with no `v` prefix and no suffix.

### Epics

An epic groups related features and fixes. Its children are the branches
Git-petal-flow created with `--parent epic/<name>`.

```sh
gitgud new epic "001 auth"                                  # epic/001-auth from develop
gitgud new feature "epic-001/login form" --parent epic/001-auth   # feature/epic-001/login-form
gitgud new fix "epic-001/session timeout" --parent epic/001-auth  # fix/epic-001/session-timeout
```

Each child merges back into the epic with `gitgud finish` as usual. Check out
the epic and finish it when the children are done:

```sh
git checkout epic/001-auth
gitgud finish             # merge epic/001-auth into develop with --no-ff
```

#### How the epic knows its children

The epic has no list of children in git. The link is stored in Git-petal-flow's
`state.json` (see "Beneath the ceramic hood"): every branch records its
`parent` when `gitgud new` creates it, and every branch records a `status` of
`open` or `finished`. When you run `finish` on an epic, Git-petal-flow looks through
`state.json` for branches whose parent is that epic and whose status is still
`open`. The git history of the children is not consulted.

Two consequences follow:

- **A child must be created with `gitgud new --parent`.** A branch made with
  plain git (`git checkout -b feature/x epic/001-auth`) is not recorded, so Git-petal-flow
  does not know it is a child and the epic will not wait for it.
- **Only `gitgud finish` closes a child.** The status changes to `finished`
  when the child is finished, not when git merges it. If you delete an open
  child branch with plain git, its entry stays `open` in `state.json` and the
  epic keeps refusing. Git-petal-flow has no command for abandoning a child yet.

When a child is still open, `finish` refuses and names it:

```text
gitgud: epic epic/001-auth still has open children: [feature/epic-001/login-form]; finish them first
```

Once the epic is finished, its status is `finished` and `new --parent
epic/001-auth` is refused, so no new children can join it.

#### Naming children so people can see them

`state.json` is the authority that `finish` uses, but a person reading `git
branch` should also be able to tell which epic a branch belongs to. The name
carries that information:

1. **Give each epic a number.** Epic names start with the number:
   `epic/001-some-epic`.
2. **Put the epic's identifier first in every child.** The child is
   `feature/epic-001/<name>` or `fix/epic-001/<name>`, with the same number.
3. **Create the child with the matching `--parent`.** The name is a hint for
   people. The `--parent` flag is what Git-petal-flow records:

   ```sh
   gitgud new feature "epic-001/some child feature" --parent epic/001-some-epic
   # created feature/epic-001/some-child-feature from epic/001-some-epic
   ```

4. **Never reuse a number.** Two epics with the same number make the names
   ambiguous.
5. **Check the children by name** when you want a quick view:

   ```sh
   git branch --list 'feature/epic-001/*' 'fix/epic-001/*'
   ```

Slashes are kept, so `feature/epic-001/some-child-feature` is created exactly
as written. Each part of a name is lowercased and cleaned on its own, so
`epic-001/Some Child!` becomes `epic-001/some-child`.

Git-petal-flow warns when a name does not follow the convention. If a child is created
with the wrong or missing number, `new` prints a hint with the suggested name
and still creates the branch:

```text
hint: fix/login is a child of epic/001-some-epic; name it fix/epic-001/login so people can see the epic from the branch name
```

The naming rule is a convention, not a check. Git-petal-flow does not reject a child
whose name does not match its parent, and `finish` ignores names entirely. The
`--parent` flag decides the link.

The status line shows a warning while an epic has open children, and `finish` on
the epic refuses until they are done. For the full picture of how epics,
children and releases fit together, see [WORKFLOW.md](WORKFLOW.md).

#### Typical epic session

```sh
gitgud new epic "001 auth"                                       # on epic/001-auth
git commit --allow-empty -m "feat(auth): scaffold"                # commit on the epic with git

gitgud new feature "epic-001/login form" --parent epic/001-auth  # on feature/epic-001/login-form
git commit --allow-empty -m "feat(auth): add login form"
gitgud finish                                                    # merges into epic/001-auth; you are now on epic/001-auth

gitgud new fix "epic-001/session timeout" --parent epic/001-auth # second child, on fix/epic-001/session-timeout
git commit --allow-empty -m "fix(auth): expire idle sessions"
gitgud finish                                                    # merges into epic/001-auth; back on epic/001-auth

gitgud finish                                                    # epic/001-auth merges into develop
```

`finish` checks out the branch it merges into, so after each child you are
back on the epic and can start the next child from there.

### Resolving a conflict

When `finish` cannot merge cleanly, it exits with code 2 and leaves the merge in
progress on the target branch:

```sh
gitgud finish                 # exit 2: merge stopped with conflicts in [shared.txt]
# edit shared.txt to the result you want
git add shared.txt
git commit -m "fix: resolve conflict in shared.txt"   # completes the merge on develop
git checkout feature/login-form
gitgud finish                 # merges again (already up to date) and marks it finished
```

While the merge is in progress, `gitgud new` and `gitgud finish` refuse to run.

### Reading the history

Each branch has a journal: one JSON object per line, oldest first.

```sh
gitgud log --branch feature/login-form              # the whole journal
gitgud log --limit 5                                 # the last five records of the current branch
                                                     # (run on a gitgud branch; develop has no journal)
gitgud log --branch feature/login-form | jq -c 'select(.type == "feat")'
```

The journal records what happened to the branch: `branch.created`,
`commit.recorded` (with commit, parents, subject and type), `merged`,
`merge.conflict`, `tag.created` and `branch.finished`. It does not record the
command line you typed. See [docs/journal-format.md](docs/journal-format.md)
for the fields and more `jq` recipes.

## Branch model

| Type      | Created from | Finished into                  |
| --------- | ------------ | ------------------------------ |
| `feature/*` | `develop` (or an open `epic/*`, with `--parent`) | its recorded parent |
| `fix/*`     | `develop` (or an open `epic/*`, with `--parent`) | its recorded parent |
| `epic/*`    | `develop`    | `develop`, once no child is open |
| `release/*` | `develop`    | `main` (tagged `vX.Y.Z`), then `develop` |

Every merge uses `--no-ff`. Git-petal-flow never rebases, never fast-forwards and
never pushes.

## Commands

```text
gitgud init                              create develop and main if missing
gitgud new feature|fix|epic [desc]       branch from develop (prompts if no desc)
      [--parent <branch>]                parent must be develop or an open epic
gitgud new release <version>             release/<version> from develop
gitgud record                            journal commits made with plain git
gitgud finish [--delete]                 merge the current branch into its parent
gitgud log [--branch <name>] [--limit N] print the journal as raw JSONL
gitgud status                            short git status (same as git status -s)
```

Example:

```sh
gitgud init
gitgud new fix                       # prompts: What does this fix do?
# ...commit your work with git...
gitgud finish --delete
gitgud log --branch fix/readline-from-user | jq -c .
```

`new` slugifies the description: `readline from user` becomes
`fix/readline-from-user`.

## Safety

- Commands refuse to run when the working tree has uncommitted or untracked
  files, or when a merge is in progress.
- A merge that conflicts is left in progress and recorded in the journal. Resolve
  the files and commit them on the target branch (`develop`, or `main` for a
  release). Then check out your branch and run `gitgud finish` again. Git-petal-flow
  never aborts or resets a merge. See "Resolving a conflict" below.
- Only branches created by `gitgud new` can be finished.

## Where Git-petal-flow keeps its data

Inside the git directory, so nothing extra is committed:

```text
<git-common-dir>/gitgud/state.json             branch type, parent, status
<git-common-dir>/gitgud/journal/<branch>.jsonl one file per branch
```

`/` in a branch name becomes `__` in the journal file name:
`feature/login` is `feature__login.jsonl`. See
[docs/journal-format.md](docs/journal-format.md) for the record fields, the
event names, and `jq` recipes.

## Use from Claude Code (MCP and skill)

`gitgud-mcp` serves the commands above as MCP tools (`gitgud_init`,
`gitgud_new`, `gitgud_record`, `gitgud_finish`, `gitgud_log`). The skill in
`skills/gitgud/` tells Claude when and how to use them.

```sh
scripts/install.sh all        # install both binaries, link the skill, register the server
scripts/install.sh uninstall  # undo all of it
```

`install` builds both binaries with `go install` into `GOBIN` (or `GOPATH/bin`)
and links `skills/gitgud` into `$CLAUDE_HOME/skills` (default `~/.claude`).
`register` adds `gitgud-mcp` to Claude Code's user config, and `all` runs it
too. Run `scripts/install.sh check` for the CI gate. The script needs only
bash, Go, and the `claude` CLI for `register`.

The server speaks MCP over stdio. It runs Git-petal-flow with argument lists, never
through a shell, and it never prompts: `gitgud_new` needs a description or a
version up front.

## Status line

`gitgud statusline` prints one line about the Git-petal-flow flow for Claude Code's
status bar: the current branch, its parent, how many commits are recorded and
how many are not yet, the last journal event, or the merge that is waiting on
you. It prints nothing outside a git repo.

```sh
scripts/install.sh statusline   # sets statusLine in ~/.claude/settings.json (backs it up first)
```

It only reads. It never changes the repository or the journal, and it always
exits 0, so a problem can't show as an error in the bar. Set `NO_COLOR=1` to
turn off the colours.

## Asking Claude about Git-petal-flow

Claude can't see the status bar, because it isn't part of the conversation. To
get the same information, ask Claude and it will run one of these:

- **Current state:** "what's the gitgud status?" Claude runs `gitgud statusline`
  in the repo and reports the line it prints.
- **Recent activity:** "what Git-petal-flow activity happened recently?" Claude runs
  `gitgud log --limit 10` and summarises the events.
- **Branch:** "what branch are we on?" Claude runs `git branch --show-current`.

With the skill installed, Claude knows to pass `</dev/null` to `gitgud
statusline` when it runs it through a shell, so the command never waits on an
open stdin. Claude can also call the `gitgud_log` tool over MCP instead of the
shell.

## Development

```sh
gofmt -l .        # must print nothing
go vet ./...
go test ./...
go build ./cmd/gitgud
```

The tests create temporary repositories and never touch the repository they
run from.

## Beneath the ceramic hood

The short version: Git-petal-flow never stores history itself. Git stores the commits,
and Git-petal-flow keeps a small side ledger of which commits it has seen, which
branch made them, and what happened to each branch. This section walks through
how that works, so nothing about it is a mystery.

### What lives where

| Thing | Where | Committed? | Purpose |
| ----- | ----- | ---------- | ------- |
| Commits, branches, tags | git's object store and refs | Yes, by git | The actual history |
| `state.json` | `<git-common-dir>/gitgud/state.json` | No | Branch type, parent, base SHA, status |
| Journal | `<git-common-dir>/gitgud/journal/<branch>.jsonl` | No | Append-only event log |

`<git-common-dir>` is `.git` in a normal clone. Git never tracks anything under
`.git/` (other than its own files), so the state and journal are invisible to
`git status`, never appear in a commit, and never reach a push. Linked
worktrees share the same common directory, so every worktree sees the same
ledger.

### How a branch is created

`gitgud new feature "login form"` does this, in order:

1. Refuses if the working tree is dirty or a merge is in progress.
2. Checks that the parent is allowed for the type (see the branch model) and
   that the branch name is free.
3. Records the parent's current SHA as `base_sha`. This is the anchor for every
   later "which commits belong to this branch?" question.
4. Runs `git checkout -b feature/login-form develop`.
5. Writes the branch's entry to `state.json`, using a temp file and a rename so
   a crash never leaves half a file.
6. Appends a `branch.created` line to the journal.

### How "logging into the branch" works

Nothing is written into the branch. The journal is simply a list of commit
SHAs and events, kept beside git. `gitgud record` (and `finish`, which calls
the same code) does this:

1. Reads the SHAs already present in the branch's journal into a set.
2. Asks git for the commits on the branch since `base_sha`, first-parent only:
   `git log --first-parent --reverse base_sha..HEAD`.
3. For each commit not in the set, appends a `commit.recorded` line holding the
   full 40-character SHA, the parents, the subject, and the Conventional Commit
   type parsed from the subject.

That is why nothing gets committed and nothing is rewritten. Running `record`
twice does nothing the second time, because the SHA is already in the set. Git
can rewrite its own history (for example with `commit --amend`), and the old
SHA stays in the journal as a record of what was seen.

### How a merge is recorded

`finish` runs `git merge --no-ff` into the target, then reads back `HEAD` for
the merge commit's SHA and journals it:

- `merged`: a clean merge, with the merge commit SHA.
- `merge.conflict`: git stopped, so the file list is journaled, the merge is
  left in progress, and the process exits 2. Git-petal-flow never runs `merge --abort`
  or `reset`. Resolving the conflict is git's job.
- `branch.finished`: the last record for the branch. `state.json` is updated to
  `status: "finished"` only after this line is written.

A release adds a `tag.created` record between its two merges, so the main
merge and the develop merge are both in the journal.

### How an epic finishes

An epic is finished only when no recorded child is still open. `finish` walks
`state.json` for every branch whose `parent` is the epic and whose `status` is
`open`. If it finds any, it refuses before touching git. Once they are all
finished, the epic merges into develop like any other branch. A finished epic
is closed to new children, because the `--parent` check reads the same
`status` field.

### How the status line stays read-only

`gitgud statusline` runs the same reads as `record` without the writes: it
loads `state.json`, counts journal SHAs, counts unrecorded commits, and reads
the last journal line. It writes nothing, always exits 0, and prints nothing
outside a repo, so a problem can never show up as an error in the bar.

### Why it is safe to be boring

- Every git call is a plain argument list. No shell string is built anywhere,
  and a test fails the build if the banned words (`rebase`, `reset`, `push`,
  `--force` and friends) appear in non-test Go code.
- Journals are append-only, and each line takes its `seq` from the line count.
- The only file ever replaced in place is `state.json`, and only through an
  atomic rename.

The record format is in [docs/journal-format.md](docs/journal-format.md).

Nothing under here is fine china. The journal is plain text, so the worst you
can break is a cup, never the whole set.

## License

MIT. See [LICENSE](LICENSE). 

## Additional notes

While the program is named **gitgud**, it is **NOT** in any way related to the [GitGud Project](https://github.com/GitGud-org/GitGud).

However, it is compatible with other "Git Ops" workflows (as described by Gitlab on this [page](https://about.gitlab.com/topics/gitops/gitops-workflow/)  in general. 

You may incorporate the journal method into your own workflow or modify the source code to better suit your specific requirements.


