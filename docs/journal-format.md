# Journal format

Each GitHero-flow branch has an append-only JSONL journal. One JSON object per line,
oldest first. GitHero-flow never rewrites or truncates a journal.

Location: `<git-common-dir>/gitgud/journal/<branch>.jsonl`, with `/` in the
branch name replaced by `__`.

## Record fields

| Field     | Type            | Meaning |
| --------- | --------------- | ------- |
| `seq`     | integer         | Position in this file, starting at 1. |
| `ts`      | string          | RFC 3339 UTC time the record was written. |
| `op_id`   | string          | Operation that wrote the record. Records from one command share it. Sorts by creation time. |
| `event`   | string          | What happened. See below. |
| `branch`  | string          | The branch this journal belongs to. |
| `commit`  | string, optional | Full 40-character commit SHA. Present on commit and merge events. |
| `parents` | array, optional | Parent SHAs of `commit`. |
| `subject` | string, optional | Subject line of `commit`. |
| `type`    | string or null  | Conventional Commit type of `subject` (`feat`, `fix`, ...). Null for events and for non-conventional subjects. Always present. |
| `source`  | string, optional | Branch the operation started from. |
| `target`  | string, optional | Branch the operation merged into. |
| `tag`     | string, optional | Tag name, on `tag.created`. |
| `files`   | array, optional | Conflicted paths, on `merge.conflict`. |

Fields that do not apply to an event are left out, except `type`, which is
always written as `null` when there is no value.

## Events

| Event             | Written when |
| ----------------- | ------------ |
| `branch.created`  | `gitgud new` creates the branch. `source` is the parent. |
| `commit.recorded` | A commit is journaled, by `new`, `record`, or `finish`. Carries `commit`, `parents`, `subject`, `type`. |
| `merged`          | A merge into `target` succeeded. `commit` is the merge commit. |
| `tag.created`     | A release tag was created on `main`. |
| `merge.conflict`  | A merge stopped with conflicts. `files` lists them. The merge is left in progress. |
| `branch.finished` | `gitgud finish` completed. `commit` is the last merge commit. |

## Example

A feature with two commits, then finished:

```json
{"seq":1,"ts":"2026-10-16T09:00:00Z","op_id":"op_0001...","event":"branch.created","branch":"feature/login","source":"develop","type":null}
{"seq":2,"ts":"2026-10-16T09:10:00Z","op_id":"op_0002...","event":"commit.recorded","branch":"feature/login","commit":"2f7e15b7...","parents":["c1d2e3f4..."],"subject":"feat(auth): add login form","type":"feat"}
{"seq":3,"ts":"2026-10-16T09:20:00Z","op_id":"op_0003...","event":"commit.recorded","branch":"feature/login","commit":"9a8b7c6d...","parents":["2f7e15b7..."],"subject":"fix: correct typo","type":"fix"}
{"seq":4,"ts":"2026-10-16T09:30:00Z","op_id":"op_0003...","event":"merged","branch":"feature/login","commit":"eedfbddd...","source":"feature/login","target":"develop"}
{"seq":5,"ts":"2026-10-16T09:30:00Z","op_id":"op_0003...","event":"branch.finished","branch":"feature/login","commit":"eedfbddd...","source":"feature/login","target":"develop"}
```

Shortened SHAs and op IDs are used here for readability. Real records hold full
values.

## jq recipes

Run these from the repository root. They read one branch journal.

List the commits on a branch, oldest first:

```sh
jq -r 'select(.event == "commit.recorded")
       | "\(.commit[0:7]) \(.type // "-") \(.subject)"' \
  .git/gitgud/journal/feature__login.jsonl
```

Show only `feat` commits:

```sh
jq -c 'select(.type == "feat")' .git/gitgud/journal/feature__login.jsonl
```

Show the merge commit for each finished branch in every journal:

```sh
jq -r 'select(.event == "branch.finished")
       | "\(.branch) -> \(.target) \(.commit)"' \
  .git/gitgud/journal/*.jsonl
```

Count records per event in one branch:

```sh
jq -s 'group_by(.event) | map({event: .[0].event, count: length})' \
  .git/gitgud/journal/feature__login.jsonl
```

Find every merge conflict across all branches:

```sh
jq -r 'select(.event == "merge.conflict")
       | "\(.branch) into \(.target): \(.files | join(", "))"' \
  .git/gitgud/journal/*.jsonl
```
