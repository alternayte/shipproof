#!/usr/bin/env bash
# Build the three reports for the comprehension review of Section 14.6.
#
# Usage:
#   scripts/u1-packet.sh [output-directory]
#
# It writes report-a.html, report-b.html, and report-c.html. One shows each
# verdict. The reader sees only the pages, never this script.
set -euo pipefail

OUT="${1:-$PWD/u1-packet}"
BIN="$(cd "$(dirname "$0")/.." && pwd)/bin/shipproof"

if [ ! -x "$BIN" ]; then
  echo "Build the binary first: just verify" >&2
  exit 1
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$OUT"

# scenario <name> <proof-command> <letter>
scenario() {
  local name="$1" proof="$2" letter="$3"
  local dir="$WORK/$name"
  mkdir -p "$dir"
  cd "$dir"
  git init -q .
  git config user.email "review@example.com"
  git config user.name "Review"

  cat > checkout.md <<'INTENT'
# Checkout retry

### SP-1-R1 — Retry a failed charge

The gateway retries a charge when the processor returns a transient error.

### SP-1-R2 — Cap the retry count

The gateway stops after five attempts.
INTENT

  "$BIN" init . >/dev/null
  # The default gate runs `just verify`, which no demo repository holds. The
  # review is about the verdict, not about a missing build tool.
  "$BIN" config set verification.command true --local >/dev/null 2>&1 || \
    sed -i.bak 's|^  command: .*|  command: "true"|' .shipproof/config.yaml
  "$BIN" start SP-1 --intent checkout.md --ceremony 0 >/dev/null

  cat > .shipproof/changes/SP-1/verification.json <<PLAN
{
  "schema_version": "0.1",
  "change_id": "SP-1",
  "requirements": [
    {
      "id": "SP-1-R1",
      "statement": "Retry a failed charge",
      "proof": [{"type": "command", "target": "$proof", "command": "$proof"}]
    },
    {
      "id": "SP-1-R2",
      "statement": "Cap the retry count",
      "proof": [{"type": "command", "target": "$proof", "command": "$proof"}]
    }
  ],
  "invariants": []
}
PLAN

  mkdir -p src
  echo "package src" > src/gateway.go
  git add -A >/dev/null
  git commit -qm "add the retry path"
  # Scenario A must reach PROVEN, which needs a measured unexplained count of
  # zero. A range whose base equals its head holds no changed line, so the
  # count is a true zero rather than an unknown.
  if [ "$letter" = "a" ]; then
    "$BIN" pack SP-1 --base HEAD --head HEAD >/dev/null 2>&1 || true
  else
    "$BIN" pack SP-1 >/dev/null 2>&1 || true
  fi
  cp .shipproof/changes/SP-1/report.html "$OUT/report-$letter.html"
  cd - >/dev/null
}

# A. Every proof passes.
scenario proven true a
# B. A proof exists and never ran at this revision.
scenario notproven true b
# C. A proof ran and failed.
scenario failed false c

# B must read NOT PROVEN. Remove the recorded result so nothing proves it.
cd "$WORK/notproven"
rm -f .shipproof/runs/SP-1/proofs.json
"$BIN" pack SP-1 --no-prove >/dev/null 2>&1 || true
cp .shipproof/changes/SP-1/report.html "$OUT/report-b.html"
cd - >/dev/null

echo "Wrote three reports to $OUT"
for letter in a b c; do
  printf '  report-%s.html  %s\n' "$letter" \
    "$(grep -o 'VERDICT: [A-Z ]*' "$OUT/report-$letter.html" | head -1)"
done
