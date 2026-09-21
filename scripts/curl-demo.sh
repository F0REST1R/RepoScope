#!/usr/bin/env bash
set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
OWNER="${OWNER:-golang}"
REPO="${REPO:-go}"
OUT_FILE="${OUT_FILE:-curl-search-response.json}"

headline() { printf '\n=== %s ===\n' "$1"; }
check_code() {
  local expected="$1" actual="$2"
  if [[ "$actual" == "$expected" ]]; then printf 'OK: HTTP %s\n' "$actual"; else printf 'ERROR: expected %s, got %s\n' "$expected" "$actual" >&2; return 1; fi
}

headline "GET health: headers and body"
curl --fail-with-body --silent --show-error --include \
  -H 'Accept: application/json' "$BASE_URL/health"

headline "GET search: query parameters, save body"
code=$(curl --silent --show-error --output "$OUT_FILE" --write-out '%{http_code}' --get \
  -H 'Accept: application/json' -H 'X-Request-ID: curl-unix-search' \
  --data-urlencode 'q=http server' --data-urlencode 'language=Go' \
  --data 'sort=stars' --data 'order=desc' --data 'page=1' --data 'per_page=5' \
  "$BASE_URL/repositories/search")
check_code 200 "$code"
printf 'Body saved to %s\n' "$OUT_FILE"

headline "Response headers only (ordinary GET, body discarded)"
curl --silent --show-error --dump-header - --output /dev/null "$BASE_URL/health"

headline "Verbose repository request (protocol details go to stderr)"
curl --verbose --output /dev/null -H 'Accept: application/json' \
  "$BASE_URL/repositories/$OWNER/$REPO"

headline "POST favorite: JSON body and status check"
curl --silent --show-error --output /dev/null --request DELETE "$BASE_URL/favorites/$OWNER/$REPO"
code=$(curl --silent --show-error --output /tmp/reposcope-favorite.json --write-out '%{http_code}' \
  --request POST -H 'Accept: application/json' -H 'Content-Type: application/json' \
  --data "{\"owner\":\"$OWNER\",\"repo\":\"$REPO\"}" "$BASE_URL/favorites")
check_code 201 "$code"

headline "POST duplicate: expected conflict"
code=$(curl --silent --show-error --output /tmp/reposcope-error.json --write-out '%{http_code}' \
  --request POST -H 'Content-Type: application/json' \
  --data "{\"owner\":\"$OWNER\",\"repo\":\"$REPO\"}" "$BASE_URL/favorites")
check_code 409 "$code"

headline "DELETE favorite"
code=$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' \
  --request DELETE "$BASE_URL/favorites/$OWNER/$REPO")
check_code 204 "$code"

headline "Negative validation scenario"
code=$(curl --silent --show-error --output /tmp/reposcope-validation.json --write-out '%{http_code}' \
  "$BASE_URL/repositories/search?q=x&page=0")
check_code 422 "$code"

printf '\nAll cURL scenarios completed.\n'
