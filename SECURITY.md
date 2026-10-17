# Security policy

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub's "Report a vulnerability" button on the
repository's Security tab. Include the GitHero-flow version (`gitgud --version`), the
git version, and steps to reproduce.

You can expect an acknowledgement within a week. Fixes are released as a new
version, and the report is credited unless you ask otherwise.

## Scope

GitHero-flow runs the local `git` binary with argument lists, never through a shell.
It does not contact the network and does not push. A report is in scope if it
can cause GitHero-flow to destroy work, rewrite history, or leak data from a
repository it was not run in.
