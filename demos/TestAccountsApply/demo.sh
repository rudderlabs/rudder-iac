#!/usr/bin/env bash
# Generated from TestAccountsApply by demogen. Do not edit — re-record instead.
# See README.md for how to run this, and manifest.json for where it came from.
set -uo pipefail
cd "$(dirname "$0")" || exit 1
: "${DEMO_MAGIC:=$HOME/workspace/demo-magic/demo-magic.sh}"
. "$DEMO_MAGIC"
TYPE_SPEED=90
NO_WAIT=true
DEMO_PROMPT="${GREEN}➜ ${CYAN}TestAccountsApply ${COLOR_RESET}\$ "
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

p "# Accounts apply"
p "# Start from a clean workspace: destroy removes anything a previous run left behind, so the create below always starts from nothing."
pe "rudder-cli destroy --confirm=false"
p "# (verified in Go, not visible here — see TestAccountsApply)"

p "# Apply create"
p "# Accounts are created from spec with no ids anywhere — everything upstream is resolved by external ID reference, not by a value the user had to look up first."
pe "rudder-cli apply -l accounts/create --var-file accounts/credentials.vars.yaml --confirm=false"
p "# (verified in Go, not visible here — see TestAccountsApply/apply_create)"

p "# Apply update"
p "# Re-applying the same accounts with changed values updates them in place upstream, rather than deleting and recreating."
pe "rudder-cli apply -l accounts/update --var-file accounts/credentials.vars.yaml --confirm=false"
p "# (verified in Go, not visible here — see TestAccountsApply/apply_update)"

p "# Re-apply leaves non-secret upstream state unchanged"
p "# Applying this same spec again with nothing changed should be a no-op — except the API never returns write-only secrets, so the CLI cannot diff them and must re-send the secret on every apply. Watch the non-secret fields stay identical even though the request goes out again."
pe "rudder-cli apply -l accounts/update --var-file accounts/credentials.vars.yaml --confirm=false"
p "# (verified in Go, not visible here — see TestAccountsApply/re-apply_leaves_non-secret_upstream_state_unchanged)"


if [ -z "${RUDDER_DEMO_RECORDING:-}" ] && [ -f ANNOTATE.md ]; then
  if [ -t 1 ] && [ -z "${RUDDER_DEMO_AGENT:-}" ]; then
    printf '\n\033[33m%s\033[0m\n' "Some steps in this demo have no narration. See ANNOTATE.md."
  else
    printf '\n%s\n' "Unannotated demo. ANNOTATE.md lists the steps needing demo.Say and where to add them."
    exit 3
  fi
fi
