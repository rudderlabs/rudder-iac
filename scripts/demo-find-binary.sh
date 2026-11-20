#!/usr/bin/env bash
# Print the absolute rudder-cli binary path recorded in a demo journal.
#
# The binary lives only in the journal's own recorded argv[0] — TestMain
# built it into a t.TempDir() that no longer exists once the run ends — so
# this is how both `make demo-generate` and `scripts/demo-check.sh`
# rediscover it. There is nothing to build here to get a fresh one from.
#
# JSON, not grep: the journal is a JSONL file Go wrote with encoding/json,
# which HTML-escapes characters like < and > in string values. A text
# pipeline grepping the raw bytes can therefore see something other than
# what json.Unmarshal would — the exact class of bug this branch already hit
# once with the secrets check (see the design doc). Parse it properly.
#
# Usage: scripts/demo-find-binary.sh path/to/journal.jsonl
set -euo pipefail

JOURNAL="${1:?usage: demo-find-binary.sh <journal.jsonl>}"

python3 - "$JOURNAL" <<'PY'
import json
import re
import sys

path = sys.argv[1]
pattern = re.compile(r"/rudder-cli(\.exe)?$")

with open(path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        record = json.loads(line)
        argv = record.get("argv") or []
        if argv and pattern.search(argv[0]):
            print(argv[0])
            sys.exit(0)

sys.exit(1)
PY
