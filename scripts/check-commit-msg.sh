#!/usr/bin/env bash
# Checks the commit format, types, scope, lengths, lazy subjects, phase words,
# ticket references and that the message is English.
set -euo pipefail

message_file="${1:-}"
if [ -z "$message_file" ] || [ ! -f "$message_file" ]; then
  echo "usage: check-commit-msg.sh <message-file>" >&2
  exit 1
fi

subject=$(head -1 "$message_file")

types="feat fix refactor perf test docs build ci chore"
lazy="fix bugs|minor bugs|update code|clean up|various changes|minor fixes|refactor|stuff|things|misc|wip|some changes|small fix|tiny fix"
phase="phase|part|stage|sprint|epic|milestone"

prefix=${subject%%:*}
if [ "$prefix" != "$subject" ]; then
  if [[ "$prefix" == *"("*")" ]]; then
    type=${prefix%%(*}
    scope=${prefix#*(}
    scope=${scope%)}
  else
    type=$prefix
    scope=""
  fi
  type=${type%!}
  text=${subject#"$prefix: "}
else
  type=""
  scope=""
  text=""
fi

fail() {
  echo "commit message rejected: $1" >&2
  echo "  got:  $subject" >&2
  echo "  want: $2" >&2
  exit 1
}

[ -n "$subject" ] || fail "the message is empty" "type(scope): what changed"

if ! printf '%s' "$subject" | grep -Eq '^[a-z]+(\([a-z0-9./-]+\))?!?: .+$'; then
  fail "the format is wrong" "type(scope): what changed"
fi

if ! printf '%s' "$types" | grep -qw "$type"; then
  fail "\"$type\" is not a type of commit" "one of: $types"
fi

if [ "${#subject}" -ge 50 ]; then
  fail "the first line is ${#subject} characters" "forty-nine or fewer"
fi

if [ "${#text}" -lt 10 ]; then
  fail "the subject is ${#text} characters" "ten or more"
fi

if printf '%s' "$text" | grep -q '\.$'; then
  fail "the subject ends with a dot" "no trailing dot"
fi

if printf '%s' "$subject" | grep -q '#[0-9]'; then
  fail "the subject names a ticket" "Closes #123 goes in the body, on its own line"
fi

if printf '%s' "$text" | grep -Eqi "\b($phase) +[0-9]"; then
  fail "the subject names a phase of the project" "a change is explained by the diff alone"
fi

if printf '%s' "$text" | grep -Eqi "($lazy)"; then
  fail "the subject describes nothing" "name the package, the file or the function"
fi

if printf '%s' "$subject" | grep -qP '[\p{Cyrillic}]'; then
  fail "the message is not English" "the commit messages are in English"
fi

exit 0
