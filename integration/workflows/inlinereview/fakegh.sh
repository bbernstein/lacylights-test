#!/usr/bin/env bash
# Fake `gh` for the claude-inline-review setup-step tests. It supports only the calls that step makes:
#
#   gh api [--paginate] ENDPOINT [-H HEADER] [--jq FILTER]
#
# Responses come from fixture files in $FAKE_GH_DIR:
#   pull.json        GET repos/O/R/pulls/N (its .head.sha is replaced, see FAKE_GH_HEADS)
#   pull_files.json  GET repos/O/R/pulls/N/files
#   comments.json    GET repos/O/R/pulls/N/comments
#   compare.json     GET repos/O/R/compare/B...H
#   compare.diff     GET repos/O/R/compare/B...H with "Accept: application/vnd.github.diff"
#
# Environment:
#   FAKE_GH_HEADS  comma-separated head SHAs returned by successive pulls/N calls; the last one repeats.
#   FAKE_GH_FAIL   comma-separated keys that fail: pull, pull_files, comments, compare, compare_diff.
#                  Like the real gh, a failure prints error JSON to stdout and exits 1.
# Every call is appended to $FAKE_GH_DIR/calls.log.
set -euo pipefail

dir=${FAKE_GH_DIR:?FAKE_GH_DIR must be set}
echo "$*" >> "$dir/calls.log"

[ "${1:-}" = api ] || { echo "fake gh: only 'gh api' is supported" >&2; exit 2; }
shift

endpoint="" filter="" accept=""
while [ $# -gt 0 ]; do
  case "$1" in
    --paginate) shift ;;
    --jq) filter=$2; shift 2 ;;
    -H) accept=$2; shift 2 ;;
    -*) echo "fake gh: unsupported flag $1" >&2; exit 2 ;;
    *) endpoint=$1; shift ;;
  esac
done

case "$endpoint" in
  repos/*/*/pulls/*/files) key=pull_files ;;
  repos/*/*/pulls/*/comments) key=comments ;;
  repos/*/*/pulls/*) key=pull ;;
  repos/*/*/compare/*)
    if [ "$accept" = "Accept: application/vnd.github.diff" ]; then key=compare_diff; else key=compare; fi ;;
  *) echo '{"message":"Not Found","status":"404"}'; exit 1 ;;
esac

case ",${FAKE_GH_FAIL:-}," in
  *",$key,"*) echo '{"message":"Server Error","status":"500"}'; exit 1 ;;
esac

if [ "$key" = compare_diff ]; then
  cat "$dir/compare.diff"
  exit 0
fi

body=$(cat "$dir/$key.json")
if [ "$key" = pull ]; then
  # Successive calls return successive entries of FAKE_GH_HEADS.
  n=$(cat "$dir/pull.count" 2>/dev/null || echo 0)
  echo $((n + 1)) > "$dir/pull.count"
  IFS=, read -r -a heads <<< "${FAKE_GH_HEADS:-}"
  if [ "${#heads[@]}" -gt 0 ]; then
    i=$n; [ "$i" -lt "${#heads[@]}" ] || i=$((${#heads[@]} - 1))
    body=$(jq --arg h "${heads[$i]}" '.head.sha = $h' <<< "$body")
  fi
fi

if [ -n "$filter" ]; then
  # gh --jq prints strings raw and other values as compact JSON.
  jq -r -c "$filter" <<< "$body"
else
  printf '%s\n' "$body"
fi
