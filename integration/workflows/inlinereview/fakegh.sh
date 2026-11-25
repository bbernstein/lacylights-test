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
#                  An entry of "fail" makes that call fail; "null" returns a null head.sha (exit 0).
#   FAKE_GH_FAIL   comma-separated keys that fail: pull, pull_files, comments, compare (the commit
#                  list), compare_files (the compare file list), compare_total (the ?per_page=1
#                  total_commits call), compare_diff.
#                  Like the real gh, a failure prints error JSON to stdout and exits 1.
# Every call is appended to $FAKE_GH_DIR/calls.log. Every unknown endpoint or unsupported flag or header is
# also appended to $FAKE_GH_DIR/refused.log, which the tests require to be empty: the step guards its gh
# calls, so a 404 alone would only become a warning.
set -euo pipefail

dir=${FAKE_GH_DIR:?FAKE_GH_DIR must be set}
echo "$*" >> "$dir/calls.log"

refuse() { echo "$*" >> "$dir/refused.log"; echo "fake gh: $*" >&2; exit 2; }
[ "${1:-}" = api ] || refuse "only 'gh api' is supported: $*"
shift

endpoint="" filter="" accept="" paginate=""
while [ $# -gt 0 ]; do
  case "$1" in
    --paginate) paginate=1; shift ;;
    --jq) [ $# -ge 2 ] || refuse "--jq without a filter"; filter=$2; shift 2 ;;
    -H) [ $# -ge 2 ] || refuse "-H without a header"
        [ -z "$accept" ] || refuse "more than one -H header: $accept / $2"; accept=$2; shift 2 ;;
    -*) refuse "unsupported flag $1" ;;
    *) [ -z "$endpoint" ] || refuse "more than one endpoint: $endpoint / $1"; endpoint=$1; shift ;;
  esac
done

# Match endpoints exactly (in a case pattern, * also matches "/", so use regexes). Anything else is a
# 404, so a workflow change that calls a new endpoint fails the tests instead of getting wrong data.
# A query string (such as ?per_page=100) is allowed and ignored. Only the compare endpoint accepts a
# header, and only the diff Accept header.
# Anchored to the step's own $REPO and $PR, and compare to exactly <encoded BASE_REF>...$HEAD_SHA (three
# dots), so a call about another repo, PR or range is refused. The variable parts are quoted, so they
# match literally rather than as regex.
q='(\?[^/]*)?'
pr_path="repos/${REPO:?}/pulls/${PR:?}"
range="$(jq -rn --arg r "${BASE_REF:?}" '$r|@uri')...${HEAD_SHA:?}"
if [[ "$endpoint" =~ ^"$pr_path"/files$q$ ]]; then key=pull_files
elif [[ "$endpoint" =~ ^"$pr_path"/comments$q$ ]]; then key=comments
elif [[ "$endpoint" =~ ^"$pr_path"$q$ ]]; then key=pull
elif [[ "$endpoint" =~ ^"repos/$REPO/compare/$range"$q$ ]]; then
  case "$accept" in
    "") case "$endpoint" in
          *"?per_page=1") key=compare_total ;;   # total_commits
          *"?per_page="*) key=compare ;;         # the paginated commit list
          *) key=compare_files ;;                # the compare file list (fallback source)
        esac ;;
    "Accept: application/vnd.github.diff") key=compare_diff ;;
    *) refuse "unsupported header $accept" ;;
  esac
else
  echo "unknown endpoint $endpoint" >> "$dir/refused.log"
  echo '{"message":"Not Found","status":"404"}'; exit 1
fi
# Real gh follows Link headers with --paginate and prints one result per page, so it only belongs on the
# list calls; elsewhere (for example the ?per_page=1 total) it would change the output.
case "$key" in
  pull_files|comments|compare) ;;
  *) [ -z "$paginate" ] || refuse "--paginate on $key ($endpoint)" ;;
esac
if [ -n "$accept" ] && [ "$key" != compare_diff ]; then
  refuse "unexpected header $accept for $endpoint"
fi

case ",${FAKE_GH_FAIL:-}," in
  *",$key,"*) echo '{"message":"Server Error","status":"500"}'; exit 1 ;;
esac

case "$key" in compare_total|compare_files) body_key=compare ;; *) body_key=$key ;; esac
if [ "$key" = compare_diff ]; then
  # Real gh would run --jq on the diff text, which is not JSON, and fail.
  [ -z "$filter" ] || refuse "--jq on the diff call"
  cat "$dir/compare.diff"
  exit 0
fi

body=$(cat "$dir/$body_key.json")
if [ "$key" = pull ]; then
  # Successive calls return successive entries of FAKE_GH_HEADS.
  n=$(cat "$dir/pull.count" 2>/dev/null || echo 0)
  echo $((n + 1)) > "$dir/pull.count"
  IFS=, read -r -a heads <<< "${FAKE_GH_HEADS:-}"
  if [ "${#heads[@]}" -gt 0 ]; then
    i=$n; [ "$i" -lt "${#heads[@]}" ] || i=$((${#heads[@]} - 1))
    if [ "${heads[$i]}" = fail ]; then
      echo '{"message":"Server Error","status":"500"}'; exit 1
    fi
    if [ "${heads[$i]}" = null ]; then
      body=$(jq '.head.sha = null' <<< "$body")   # a successful call with no usable SHA
    else
      body=$(jq --arg h "${heads[$i]}" '.head.sha = $h' <<< "$body")
    fi
  fi
fi

if [ -n "$filter" ]; then
  # gh --jq prints strings raw and other values as compact JSON. Real gh evaluates --jq with its built-in
  # gojq, so use gojq when it is installed. With plain jq, a jq-only builtin could pass here but fail on
  # the runner.
  if command -v gojq >/dev/null 2>&1; then jqbin=gojq; else jqbin=jq; fi
  # gh prints a null result as an empty line, not the text "null". The closing paren goes on its own
  # line, so a jq "#" comment on the filter's last line can't swallow it.
  "$jqbin" -r -c "($filter
) | if . == null then \"\" else . end" <<< "$body"
else
  printf '%s\n' "$body"
fi
