# Contributing

## Commits

The format is Conventional Commits:

```
<type>(<scope>): <subject>
```

- `type`: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build`, `ci`, `chore`.
- `scope`: the package, file or function that changed, lower case, `[a-z0-9./-]`; optional.
- `subject`: imperative, lower case, no trailing dot, at least ten characters, first line under 50 characters.

The subject names what changed and where. The body explains why, not what, and may be several lines. Reference an issue on its own line: `Closes #123`.

Do not use past tense, lazy descriptions (`fix bugs`, `update code`, `clean up`, `misc`, `wip`), project phases (`phase 2`, `sprint 3`), ticket numbers in the subject or non-English text. One logical change per commit. Before committing, run `git diff --cached --stat` and stage only the files of that change.

The machine form of these rules is `scripts/check-commit-msg.sh`. The hooks run it on every commit and CI runs it over the pull request.

## Comments

The default is no comment. Write one only when the code cannot answer the question and a competent reader would otherwise get it wrong:

- the behaviour looks like a bug but is not, and changing it breaks the contract;
- a non-obvious constraint or a deliberate omission;
- a non-obvious numeric or protocol detail, such as a rounding rule or a wire format;
- a deliberately chosen ceiling, with the upgrade path named.

Do not restate the code, add section banners, name the type or receiver, narrate the change, or leave markers. `TODO`, `FIXME`, `XXX`, `NOTE`, `HACK`, `BUG`, `OPTIMIZE`, `for now`, `remember to`, `don't forget`, `should probably`, `needs refactoring` and `as discussed` are refused. Unfinished work belongs in a GitHub issue.

Comments are full English sentences ending with a period, one or two lines, explaining why rather than what. Match the density of the surrounding code. Delete a comment with the code it describes. The code is English.

The machine form is `scripts/check-comments.sh`. It runs on commit and in CI.

## Before a pull request

```bash
make tools
make hooks
make check
make parity
```

`make check` runs formatting, lint, tests, `go vet` and the comment rules. `make parity` needs the local replay corpus through `MANTA_REPLAY_DIR`.
