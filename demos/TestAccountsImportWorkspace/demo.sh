#!/usr/bin/env bash
# Generated from TestAccountsImportWorkspace by demogen. Do not edit — re-record instead.
# See README.md for how to run this, and manifest.json for where it came from.
set -uo pipefail
cd "$(dirname "$0")" || exit 1
: "${DEMO_MAGIC:=$HOME/workspace/demo-magic/demo-magic.sh}"
. "$DEMO_MAGIC"
TYPE_SPEED=90
NO_WAIT=true
DEMO_PROMPT="${GREEN}➜ ${CYAN}TestAccountsImportWorkspace ${COLOR_RESET}\$ "
: "${RUDDERSTACK_API_URL:=http://localhost:15580}"
: "${RUDDERSTACK_CLI_EXPERIMENTAL:=true}"
: "${RUDDERSTACK_X_UNVERIFIED_DESTINATIONS:=true}"
[ -f ./profile.env ] && . ./profile.env

if [ -z "${RUDDERSTACK_API_URL:-}" ]; then
  echo "refusing: RUDDERSTACK_API_URL is unset, so this script would target api.rudderstack.com." >&2
  echo "  The first command below is 'destroy --confirm=false' — it wipes the target workspace." >&2
  echo "  Source a profile first, e.g.: . ./profile.env" >&2
  exit 1
fi
case "$RUDDERSTACK_API_URL" in
*api.rudderstack.com*)
  echo "refusing: RUDDERSTACK_API_URL points at production, and this script wipes its workspace." >&2
  echo "  The first command below is 'destroy --confirm=false'." >&2
  exit 1
  ;;
esac

clear

p "# Accounts import workspace"
pe "rudder-cli destroy --confirm=false"
pe "rudder-cli import workspace -l /var/folders/sn/1hbr6wkd37v51zh7zcv920vr0000gn/T/TestAccountsImportWorkspace1735749527/001"
pe "rudder-cli apply -l /var/folders/sn/1hbr6wkd37v51zh7zcv920vr0000gn/T/TestAccountsImportWorkspace1735749527/001 --var-file /var/folders/sn/1hbr6wkd37v51zh7zcv920vr0000gn/T/TestAccountsImportWorkspace1735749527/001/imported/secrets.vars.yaml --confirm=false"
pe "rudder-cli apply -l /var/folders/sn/1hbr6wkd37v51zh7zcv920vr0000gn/T/TestAccountsImportWorkspace1735749527/001 --var-file /var/folders/sn/1hbr6wkd37v51zh7zcv920vr0000gn/T/TestAccountsImportWorkspace1735749527/001/imported/secrets.vars.yaml --confirm=false"
p "# (verified in Go, not visible here — see TestAccountsImportWorkspace)"


if [ -z "${RUDDER_DEMO_RECORDING:-}" ] && [ -f ANNOTATE.md ]; then
  if [ -t 1 ] && [ -z "${RUDDER_DEMO_AGENT:-}" ]; then
    printf '\n\033[33m%s\033[0m\n' "Some steps in this demo have no narration. See ANNOTATE.md."
  else
    printf '\n%s\n' "Unannotated demo. ANNOTATE.md lists the steps needing demo.Say and where to add them."
    exit 3
  fi
fi
