# AGENTS.md — GitHero-flow

Operating notes for AI coding agents working in this repo. This file covers what
is not obvious from the code. Go and git basics are assumed.

## What this is

`gitgud` is a small Go CLI that adds a GitFlow-style workflow on top of plain
`git`. It creates feature, fix, epic and release branches from the right parent,
merges them with `--no-ff`, and writes a per-branch append-only JSONL journal of
commits, so the history can be read with `jq`.

It is public (MIT). Assume every file and every commit message will be read by
strangers.

## Toolchain

- Go **1.26.8**, installed on the dev machine (WSL Ubuntu 26.04, x86_64).
  `go.mod` pins the same version; CI pins it too. Keep all three in step.
- git **2.31+**. The code uses `rev-parse --path-format=absolute`.
- One external Go dependency: `github.com/mark3labs/mcp-go` **v1.1.1**, used only by
  `internal/mcpserver` and `cmd/gitgud-mcp`. It was chosen as the Go FastMCP port
  and pinned to the newest release at least two weeks old (released 2026-09-23;
  v1.3.0 was two days old when chosen). Do not bump it casually. Keep `internal/cli`
  and the core packages free of it.
- `jq` is **not** installed on the dev machine. Install it before testing the
  recipes in `docs/journal-format.md`.

## Commands

```sh
gofmt -l .            # must print nothing
go vet ./...
go test -count=1 ./...
go build ./cmd/gitgud
```

These four are exactly what CI runs. Run them before you call a change done.

## Layout

```
cmd/gitgud/main.go         entry point; passes os.Getwd() to cli.Run
cmd/gitgud-mcp/main.go     MCP server over stdio; stdout is the protocol, log to stderr only
internal/mcpserver/       MCP tools; each runs gitgud through a Runner (argv only)
internal/statusline/      status bar line; read-only, reads the repo and journal
skills/gitgud/SKILL.md     Claude skill that describes when to use the MCP tools
scripts/install.sh         install, register, uninstall, check; needs bash (no make on the dev box)
internal/cli/              one file per command: root, init, new, finish, record, log
internal/git/              runs git with argv slices; repo guards; commit listing
internal/workflow/         slug, branch policy, semver check, Conventional Commit type
internal/state/            state.json: branch type, parent, status
internal/events/           journal records, op IDs, journal read/append
docs/journal-format.md     record fields, event names, jq recipes
.claude/                   Claude Code tooling (commands and agents); not product code
```

## Rules the code depends on

- **Git runs as argv slices only.** Never build a shell string. `internal/cli/safety_test.go`
  fails the build if `"rebase"`, `"--ff-only"`, `"reset"`, `"--hard"`, `"push"` or
  `"--force"` appear in non-test Go source. If that test fails, the fix is almost
  always to remove the string, not to edit the test.
- **No rebase, no fast-forward merges, no push from GitHero-flow.** Integrations use
  `--no-ff`. Do not rewrite published commits.
- **Dirty means dirty.** `Dirty()` uses `status --porcelain --untracked-files=all`,
  so untracked files block `new` and `finish`. Do not weaken this without a
  decision recorded in the CSV.
- **A merge that conflicts is left in progress.** Never `merge --abort`, never
  `reset`. The conflict is journaled as `merge.conflict` and the process exits 2.
- **State and journal live in the common git dir**, `<git-common-dir>/gitgud/`,
  so linked worktrees share them. Do not resolve these paths from `.git` directly.
- **The journal is the history record; `state.json` only answers "what is the
  parent" and "is it finished".** Put new history facts in the journal.
- **Branch `/` becomes `__` in journal file names.** `feature/login` →
  `feature__login.jsonl`. `events.Path` is the only place that maps names.
- **Journal records are flat.** `type` is always present and is `null` for
  events. Other optional fields are omitted when empty. Do not add nested
  objects; `jq` users rely on the flat shape.
- **Commit records take the full 40-char SHA** from `git log %H`. Never shorten.
- **Exit codes:** 0 ok, 1 error or refused, 2 merge conflict. Scripts depend on 2.
- **`gitgud statusline` is read-only and always exits 0.** It must never write, and
  must never print an error to stdout, because that text lands in the status bar.
  It prints nothing outside a repo. It reads stdin only when stdin is not a terminal.
- **Only `new`-created branches can be finished.** An unrecorded branch is refused.

## Tests

- Every test that touches git makes its own temp repo with `t.TempDir()`. Never
  point a test at the real repo.
- Helpers in `internal/cli` (`newRepo`, `runGit`, `gg`, `writeCommit`,
  `journalEvents`) isolate git config with `GIT_CONFIG_GLOBAL=/dev/null` and
  set a test identity. Keep that isolation in any new git-touching test; a
  developer's global signing setting must not break the suite.
- Do not name a test helper `git`. It shadows the `internal/git` package import
  in the same package and fails vet. Use `runGit`.
- The integration test drives the real CLI through `cli.Run`, not the internals.
  Prefer that for behaviour tests.

## Scope

The MVP is `init`, `new` (feature, fix, epic, release), `record`, `finish`
(feature, fix, epic, release), and `log`. An epic finishes into develop only
when no recorded child is open (GG-015).

Deferred, with CSV rows in `tasks.csv`: GG-013 `sync`,
GG-018 `status`, GG-020 JSON envelope, GG-027 `compare`, GG-028 `changelog`.

Anything marked "deferred" in the CSV is not a bug if it is missing.

## Open items (from validate)

Known gaps, not blockers for the MVP:

- GG-014 and GG-017 still mention JSON output and `next_actions`. The JSON
  envelope is deferred, so those clauses are unmet. Decide whether to defer or
  rewrite them before you claim either task done.
- GG-005: release targets (main, then develop) live in `finish.go`, not in the
  policy table.
- jq recipes in `docs/journal-format.md` are unverified.
- The full-history secret scan has not been run (`gitleaks` is not installed).

## Public-repo hygiene

- No secrets, tokens, or personal paths in any committed file. `.claude/settings.local.json`
  is gitignored; keep it that way.
- Commits use Conventional Commits (`feat(scope): …`, `fix: …`, `docs: …`). The
  journal reads the type from the subject, so a non-conventional subject loses
  its type.
- Commit trailers: when an AI agent authored a commit, end the message with the
  `Co-Authored-By:` line the session specifies.

## Where to look

- `docs/journal-format.md` — the journal contract for readers and jq users.
- `README.md` — user-facing commands and branch model.
- `WORKFLOW.md` — epics, child naming, the open-child warnings and releases.
- `CONTRIBUTING.md` — contributor rules.
- `tasks.csv` — the task list and acceptance criteria. Status is `MVP` or `Deferred`.
- `new.md` — the original design notes this project started from.
