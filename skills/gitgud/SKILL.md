---
name: gitgud
description: Create and finish GitFlow-style feature, fix and release branches with GitHero-flow, and read each branch's JSONL journal. Use when the user asks to start a feature or fix branch, start a release, finish a branch or release, record commits, or review a branch's history or changelog. Requires the GitHero-flow MCP server (gitgud-mcp).
---

# GitHero-flow

GitHero-flow runs GitFlow-style branching over plain git and keeps an append-only
JSONL journal of each branch's commits. Use the `gitgud_*` MCP tools rather
than raw `git checkout -b` or `git merge`, so the parent branch is recorded and
merges use `--no-ff`.

## Tools

| Tool | Does |
| --- | --- |
| `gitgud_init` | Creates `develop` and `main` if missing. Safe to repeat. |
| `gitgud_new` | Starts a branch: `feature`, `fix` or `epic` from `develop` (or `--parent`), or `release` with a version. Checks it out. |
| `gitgud_record` | Journals commits made with git on the current branch. |
| `gitgud_finish` | Merges the current branch into its parent. Releases are tagged on `main` and merged back to `develop`. |
| `gitgud_log` | Returns a branch's journal, one JSON object per line. |

All tools take an optional `repo` (absolute path). Without it they act in the
server's working directory.

## Starting work

- Ask for a short description if the user did not give one. It becomes the
  branch slug: "readline from user" gives `fix/readline-from-user`.
- Call `gitgud_new` with `type`, `description` and optionally `parent`.
  `parent` must be `develop`, or an open `epic/<name>` for feature and fix.
- A release needs `version` as `MAJOR.MINOR.PATCH`, with no `v` prefix.
- If the tool reports a dirty working tree, do not stash or discard changes on
  your own. Ask the user what to do with them.

## Committing and recording

Make commits with git as normal, then call `gitgud_record`. It is safe to run
again: it records only commits it has not seen.

## Finishing

- Call `gitgud_finish`. Pass `delete: true` only when the user wants the branch
  removed.
- Exit code 2 means a merge conflict. GitHero-flow leaves the merge in progress. Show
  the user the conflicting files, let them or you resolve them and commit, then
  resolve the conflicting files and commit them on the target branch
  (`develop`, or `main` for a release). Then check out the branch and call
  `gitgud_finish` again. Do not abort the merge, and do not run `git reset`.
- Epics cannot be finished yet. Say so instead of trying a workaround.

## Reading history

`gitgud_log` returns the journal. Useful fields:

- `event`: `branch.created`, `commit.recorded`, `merged`, `merge.conflict`,
  `tag.created`, `branch.finished`.
- `commit`, `subject`, and `type` (the Conventional Commit type, or `null`) on
  `commit.recorded`.
- Filter by `type` to build a changelog. Group `feat` under Features and `fix`
  under Fixes.

## Answering status questions

When the user asks where things stand, run these from the repo and report the
output. Do not guess from memory.

- "What's the GitHero-flow status?" or "what's happening in the flow?" Run
  `gitgud statusline </dev/null` and quote its line. Always pass `</dev/null`:
  an open stdin would otherwise make the command wait.
- "What GitHero-flow activity happened recently?" Run `gitgud log --limit 10` and
  summarise the events in order.
- "What branch are we on?" Run `git branch --show-current`.

The journal records events, not the command line. If the user asks which
commands were run, say that the journal shows the events they caused (such as
`branch.created` or `merged`), and list those.

## Never

- Rebase, fast-forward merge, force-push, or reset a branch to get past a
  GitHero-flow refusal. Report the refusal and ask.
- Finish a branch that GitHero-flow did not create. It refuses, and the fix is to
  start the work with `gitgud_new`, not to bypass the check.
