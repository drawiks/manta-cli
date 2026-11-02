#!/usr/bin/env bash
# A comment may not carry a marker and the Go code is written in English.
# Files are the arguments, or every tracked Go file when none are given.
set -euo pipefail

cd "$(dirname "$0")/.."

if [ "$#" -gt 0 ]; then
  files=("$@")
else
  mapfile -t files < <(git ls-files '*.go')
fi

markers="\b(TODO|FIXME|XXX|NOTE|HACK|BUG|OPTIMIZE)\b|\b(for now|remember to|don.t forget|should probably|needs refactoring|as discussed)\b"

comment_text() {
  awk '
    BEGIN { inblock = 0 }
    {
      s = $0
      text = ""
      if (inblock) {
        j = index(s, "*/")
        if (j > 0) { text = substr(s, 1, j - 1); inblock = 0 } else { text = s }
      } else {
        k = index(s, "/*")
        j = index(s, "//")
        if (k > 0 && (j == 0 || k < j)) {
          text = substr(s, k + 2)
          if (index(text, "*/") == 0) inblock = 1
        } else if (j > 0) {
          text = substr(s, j + 2)
        }
      }
      if (text != "") print FILENAME ":" NR ":" text
    }
  ' "$1"
}

status=0

for file in "${files[@]}"; do
  [ -f "$file" ] || continue

  found=$(comment_text "$file" | grep -E "$markers" || true)
  if [ -n "$found" ]; then
    echo "$file: a comment carries a marker:" >&2
    printf '%s\n' "$found" | sed 's/^/  /' >&2
    echo "  unfinished work belongs in a GitHub issue" >&2
    status=1
  fi

  cyrillic=$(grep -nP '[\p{Cyrillic}]' "$file" || true)
  if [ -n "$cyrillic" ]; then
    echo "$file: the code is in English:" >&2
    printf '%s\n' "$cyrillic" | sed "s|^|$file:|" >&2
    status=1
  fi
done

exit "$status"
