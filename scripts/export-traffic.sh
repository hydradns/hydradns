#!/usr/bin/env bash
# export-traffic.sh — export GitHub repository traffic (views, clones, popular referrers,
# popular paths) into dated CSV files.
#
# Why this exists: GitHub's Traffic tab / REST API only retains a rolling 14-day window
# (https://docs.github.com/en/repositories/viewing-activity-and-data-for-your-repository/viewing-traffic-to-a-repository).
# Without an export, everything older than 14 days is gone for good. This script is meant to
# run on a schedule (see .github/workflows/traffic-export.yml) and append that day's numbers to
# CSV files that accumulate history indefinitely.
#
# Auth requirement (important, and not the default GITHUB_TOKEN):
#   The four traffic endpoints this script calls
#   (/traffic/views, /traffic/clones, /traffic/popular/referrers, /traffic/popular/paths)
#   all require a token with "Administration" repository permission (read), per GitHub's own
#   REST API docs. The default GITHUB_TOKEN available in a GitHub Actions workflow CANNOT be
#   granted this permission under any `permissions:` configuration -- "administration" is not
#   one of the scopes assignable to GITHUB_TOKEN. A fine-grained personal access token (or a
#   GitHub App token) with Administration: Read-only on this repo is required instead. See
#   00-roshan-only-checklist.md ("Create the TRAFFIC_TOKEN secret") for the exact click path to
#   create one.
#
# Usage:
#   REPO=hydradns/hydradns OUT_DIR=traffic-data GH_TOKEN=<token with Administration:read> \
#     ./scripts/export-traffic.sh
#
# Idempotent: re-running on the same UTC day replaces that day's rows in each CSV rather than
# duplicating them, so this is safe to run more than once on the same day (e.g. a manual
# workflow_dispatch on top of the weekly schedule).
#
# No secrets are printed. Requires: gh (authenticated via GH_TOKEN/GITHUB_TOKEN env var), jq.

set -euo pipefail

REPO="${REPO:-hydradns/hydradns}"
OUT_DIR="${OUT_DIR:-traffic-data}"

if ! command -v gh >/dev/null 2>&1; then
  echo "error: gh CLI not found in PATH" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq not found in PATH" >&2
  exit 1
fi

if [ -z "${GH_TOKEN:-}" ] && [ -z "${GITHUB_TOKEN:-}" ]; then
  cat >&2 <<'EOF'
error: no GH_TOKEN or GITHUB_TOKEN set.

The GitHub traffic API (views/clones/popular referrers/popular paths) requires a token with
repository "Administration" permission (read). The default Actions GITHUB_TOKEN cannot be
granted this permission under any configuration -- a fine-grained personal access token (or
GitHub App token) with Administration: Read-only on this repo is required instead.

In CI, this script expects a repository secret named TRAFFIC_TOKEN (see
.github/workflows/traffic-export.yml and 00-roshan-only-checklist.md for how to create one).

Locally, export GH_TOKEN=<your fine-grained PAT> before running this script.
EOF
  exit 1
fi

mkdir -p "$OUT_DIR"

VIEWS_CSV="$OUT_DIR/views.csv"
CLONES_CSV="$OUT_DIR/clones.csv"
REFERRERS_CSV="$OUT_DIR/referrers.csv"
PATHS_CSV="$OUT_DIR/paths.csv"

today="$(date -u +%F)"

# merge_by_date <csv_file> <header> <new_rows_file>
#
# Rows in every CSV here start with a plain ISO date (e.g. 2026-09-19,...) as the first field.
# Dates never contain a comma, so matching on "does this line start with <date>," is safe even
# though later fields (referrer names, path titles) are jq-@csv-quoted and may themselves
# contain commas. This keeps the merge idempotent per day: any existing row(s) for a date that
# appears in the new data are dropped and replaced by the fresh row(s) for that date.
merge_by_date() {
  local csv="$1" header="$2" newfile="$3"
  local tmp dates_tmp
  tmp="$(mktemp)"
  dates_tmp="$(mktemp)"

  # Distinct dates present in the new data (first CSV field, unquoted plain date string).
  cut -d',' -f1 "$newfile" | sort -u > "$dates_tmp"

  {
    echo "$header"
    if [ -f "$csv" ]; then
      tail -n +2 "$csv" 2>/dev/null | while IFS= read -r line; do
        d="${line%%,*}"
        if ! grep -qxF "$d" "$dates_tmp"; then
          printf '%s\n' "$line"
        fi
      done
    fi
    cat "$newfile"
  } > "$tmp"

  { head -n1 "$tmp"; tail -n +2 "$tmp" | sort -t',' -k1,1; } > "$csv"
  rm -f "$tmp" "$dates_tmp"
}

echo "Exporting traffic for $REPO ..." >&2

# --- Views (daily, up to 14 days of history per call) ---
views_new="$(mktemp)"
gh api "repos/$REPO/traffic/views" --jq \
  '.views[]? | [(.timestamp | split("T")[0]), .count, .uniques] | @csv' \
  > "$views_new" || {
    echo "error: failed to fetch traffic/views -- check that your token has Administration:read on $REPO" >&2
    rm -f "$views_new"
    exit 1
  }
merge_by_date "$VIEWS_CSV" "date,count,uniques" "$views_new"
rm -f "$views_new"

# --- Clones (daily, up to 14 days of history per call) ---
clones_new="$(mktemp)"
gh api "repos/$REPO/traffic/clones" --jq \
  '.clones[]? | [(.timestamp | split("T")[0]), .count, .uniques] | @csv' \
  > "$clones_new" || {
    echo "error: failed to fetch traffic/clones -- check that your token has Administration:read on $REPO" >&2
    rm -f "$clones_new"
    exit 1
  }
merge_by_date "$CLONES_CSV" "date,count,uniques" "$clones_new"
rm -f "$clones_new"

# --- Popular referrers (snapshot only, not dated by the API -- stamp with today) ---
# Note: `gh api --jq` only accepts a filter string, not extra jq flags like --arg (that's a
# plain-jq flag, not a `gh api` one) -- so the raw JSON is piped to a separate `jq` invocation
# instead of trying to pass --arg through `gh api` directly.
referrers_new="$(mktemp)"
gh api "repos/$REPO/traffic/popular/referrers" > "$referrers_new.json" || {
  echo "error: failed to fetch traffic/popular/referrers -- check that your token has Administration:read on $REPO" >&2
  rm -f "$referrers_new" "$referrers_new.json"
  exit 1
}
jq -r --arg d "$today" '.[]? | [$d, .referrer, .count, .uniques] | @csv' "$referrers_new.json" > "$referrers_new"
merge_by_date "$REFERRERS_CSV" "date,referrer,count,uniques" "$referrers_new"
rm -f "$referrers_new" "$referrers_new.json"

# --- Popular paths (snapshot only, not dated by the API -- stamp with today) ---
paths_new="$(mktemp)"
gh api "repos/$REPO/traffic/popular/paths" > "$paths_new.json" || {
  echo "error: failed to fetch traffic/popular/paths -- check that your token has Administration:read on $REPO" >&2
  rm -f "$paths_new" "$paths_new.json"
  exit 1
}
jq -r --arg d "$today" '.[]? | [$d, .path, .title, .count, .uniques] | @csv' "$paths_new.json" > "$paths_new"
merge_by_date "$PATHS_CSV" "date,path,title,count,uniques" "$paths_new"
rm -f "$paths_new" "$paths_new.json"

echo "Done. Updated: $VIEWS_CSV, $CLONES_CSV, $REFERRERS_CSV, $PATHS_CSV" >&2
