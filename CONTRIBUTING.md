# Contributing

## Ground rules

- **Conventional Commits.** Commit subjects look like `feat(scope): summary`,
  `fix: summary`, `docs: summary`. The journal reads the type from the subject.
- **No rebase, no fast-forward merges, no pushes from GitHero-flow.** Merges into
  integration branches use `--no-ff`. Do not rewrite published commits.
- **Git runs through argument lists only.** Do not build shell strings. The
  safety test in `internal/cli` fails on `rebase`, `--ff-only`, `reset`,
  `--hard`, `push` and `--force` in non-test source.
- **Tests never touch a real repository.** Use `t.TempDir()` and isolate git
  configuration as the existing tests do.

## Before you open a pull request

```sh
gofmt -l .        # must print nothing
go vet ./...
go test ./...
go build ./cmd/gitgud
```

The same four steps run in CI.

## Git hooks

An optional `pre-commit` hook in `.githooks/` runs `gofmt`, `go vet` and
`go test` before each commit. Enable it once after cloning:

```sh
git config core.hooksPath .githooks
```

It needs nothing beyond Go and git. CI runs the same checks either way.

## Scope

The MVP covers `init`, `new` (feature, fix, release), `record`, `finish`
(feature, fix, release) and `log`. Epic finish, `sync`, `status`, the JSON
envelope and the comparator are deferred. Check the open tasks before proposing
a change in those areas.
