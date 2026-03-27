#!/usr/bin/env bash
# prune-orphaned-pvcs.sh
#
# Identifies and optionally deletes orphaned PersistentVolumeClaims (PVCs) in
# EKS (Kubernetes) clusters.
#
# A PVC is considered "orphaned" when it meets ONE OR MORE of the following:
#   1. Its status is "Lost"     – the backing PV is gone / no longer valid.
#   2. Its status is "Released" – the PV it was bound to has been freed but
#                                  not yet reclaimed (can happen with Retain
#                                  reclaim policy after pod/pvc deletion).
#   3. Its status is "Pending"  for longer than --pending-age-minutes (default
#                                  60) – stuck waiting for a PV that will never
#                                  arrive (e.g. wrong storage class, capacity).
#   4. It is "Bound" but NOT referenced by any running or scheduled Pod across
#      the same namespace – the workload that owned it has been removed.
#
# Usage:
#   prune-orphaned-pvcs.sh [OPTIONS]
#
# Options:
#   -n, --namespace <ns>         Only inspect this namespace (default: all namespaces)
#   -c, --context <ctx>          kubectl context to use (default: current context)
#   -k, --kubeconfig <path>      Path to kubeconfig file (default: ~/.kube/config)
#   --dry-run                    Print what would be deleted; do NOT delete (default)
#   --delete                     Actually delete orphaned PVCs (requires confirmation
#                                unless --yes is also supplied)
#   --yes                        Skip interactive confirmation when --delete is used
#   --pending-age-minutes <n>    Minutes a Pending PVC must be old to be considered
#                                orphaned (default: 60)
#   --include-lost               Include PVCs with status=Lost  (default: true)
#   --exclude-lost               Exclude PVCs with status=Lost
#   --include-released           Include PVCs with status=Released (default: true)
#   --exclude-released           Exclude PVCs with status=Released
#   --include-pending            Include Pending PVCs older than threshold (default: true)
#   --exclude-pending            Exclude Pending PVCs from orphan detection
#   --include-unbound            Include Bound PVCs not referenced by any Pod (default: true)
#   --exclude-unbound            Exclude Bound-but-unreferenced PVCs from orphan detection
#   --label-selector <sel>       kubectl label selector to filter PVCs (e.g. app=myapp)
#   --output <fmt>               Output format for the orphan list: table (default) | json | csv
#   -h, --help                   Show this help text and exit
#
# Requirements:
#   - kubectl  (configured to reach the target cluster)
#   - jq       (>= 1.6)
#   - bash     (>= 4.0)
#
# Examples:
#   # Dry-run across all namespaces using current kubeconfig context
#   ./prune-orphaned-pvcs.sh --dry-run
#
#   # Delete with confirmation in a specific namespace / context
#   ./prune-orphaned-pvcs.sh --delete --namespace production --context eks-prod
#
#   # Delete without prompt (CI/CD usage) – be careful!
#   ./prune-orphaned-pvcs.sh --delete --yes --namespace staging
#
#   # Only flag Lost / Released PVCs, ignore Pending and unbound Bound PVCs
#   ./prune-orphaned-pvcs.sh --exclude-pending --exclude-unbound

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Defaults
# ──────────────────────────────────────────────────────────────────────────────
NAMESPACE=""          # empty = all namespaces
CONTEXT=""
KUBECONFIG_PATH=""
DRY_RUN=true
SKIP_CONFIRM=false
PENDING_AGE_MINUTES=60
INCLUDE_LOST=true
INCLUDE_RELEASED=true
INCLUDE_PENDING=true
INCLUDE_UNBOUND=true
LABEL_SELECTOR=""
OUTPUT_FORMAT="table"

# ──────────────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────────────
log()  { echo "[INFO]  $*" >&2; }
warn() { echo "[WARN]  $*" >&2; }
err()  { echo "[ERROR] $*" >&2; }
die()  { err "$*"; exit 1; }

require_tool() {
    command -v "$1" &>/dev/null || die "'$1' not found in PATH – please install it."
}

usage() {
    sed -n '/^# Usage:/,/^[^#]/{ /^#/{ s/^# \{0,1\}//; p }; /^[^#]/q }' "$0"
    exit 0
}

# ──────────────────────────────────────────────────────────────────────────────
# Argument parsing
# ──────────────────────────────────────────────────────────────────────────────
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -n|--namespace)          NAMESPACE="$2";           shift 2 ;;
            -c|--context)            CONTEXT="$2";             shift 2 ;;
            -k|--kubeconfig)         KUBECONFIG_PATH="$2";     shift 2 ;;
            --dry-run)               DRY_RUN=true;             shift   ;;
            --delete)                DRY_RUN=false;            shift   ;;
            --yes)                   SKIP_CONFIRM=true;        shift   ;;
            --pending-age-minutes)   PENDING_AGE_MINUTES="$2"; shift 2 ;;
            --include-lost)          INCLUDE_LOST=true;        shift   ;;
            --exclude-lost)          INCLUDE_LOST=false;       shift   ;;
            --include-released)      INCLUDE_RELEASED=true;    shift   ;;
            --exclude-released)      INCLUDE_RELEASED=false;   shift   ;;
            --include-pending)       INCLUDE_PENDING=true;     shift   ;;
            --exclude-pending)       INCLUDE_PENDING=false;    shift   ;;
            --include-unbound)       INCLUDE_UNBOUND=true;     shift   ;;
            --exclude-unbound)       INCLUDE_UNBOUND=false;    shift   ;;
            --label-selector)        LABEL_SELECTOR="$2";      shift 2 ;;
            --output)                OUTPUT_FORMAT="$2";       shift 2 ;;
            -h|--help)               usage ;;
            *) die "Unknown option: $1  (run with --help for usage)" ;;
        esac
    done
}

# ──────────────────────────────────────────────────────────────────────────────
# kubectl wrapper – injects --context / --kubeconfig / --namespace when set
# ──────────────────────────────────────────────────────────────────────────────
KUBECTL_BASE_ARGS=()

build_kubectl_base() {
    [[ -n "$KUBECONFIG_PATH" ]] && KUBECTL_BASE_ARGS+=(--kubeconfig "$KUBECONFIG_PATH")
    [[ -n "$CONTEXT"         ]] && KUBECTL_BASE_ARGS+=(--context   "$CONTEXT")
}

kctl() {
    kubectl "${KUBECTL_BASE_ARGS[@]}" "$@"
}

# ──────────────────────────────────────────────────────────────────────────────
# ISO-8601 creation timestamp → unix epoch (seconds)
# kubectl returns creationTimestamp like "2024-01-15T10:30:00Z"
# ──────────────────────────────────────────────────────────────────────────────
iso_to_epoch() {
    # Try GNU date first, then BSD/macOS date
    date -d "$1" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$1" +%s 2>/dev/null || echo 0
}

# ──────────────────────────────────────────────────────────────────────────────
# Collect all PVCs (JSON) for the target namespace(s)
# ──────────────────────────────────────────────────────────────────────────────
fetch_pvcs() {
    local ns_args=()
    if [[ -n "$NAMESPACE" ]]; then
        ns_args=(--namespace "$NAMESPACE")
    else
        ns_args=(--all-namespaces)
    fi

    local label_args=()
    [[ -n "$LABEL_SELECTOR" ]] && label_args=(--selector "$LABEL_SELECTOR")

    kctl get pvc "${ns_args[@]}" "${label_args[@]}" \
        -o json 2>/dev/null
}

# ──────────────────────────────────────────────────────────────────────────────
# Collect all Pods and build a set of "namespace/pvc-name" strings for every
# PVC that is currently mounted by a Pod (Running, Pending, or Succeeded).
# We also include pods in ContainerCreating / Terminating because the PVC is
# still referenced until the pod is fully gone.
# ──────────────────────────────────────────────────────────────────────────────
fetch_pod_pvc_refs() {
    local ns_args=()
    if [[ -n "$NAMESPACE" ]]; then
        ns_args=(--namespace "$NAMESPACE")
    else
        ns_args=(--all-namespaces)
    fi

    # Extract "namespace pvcName" pairs from every pod volume referencing a PVC
    kctl get pods "${ns_args[@]}" -o json 2>/dev/null \
    | jq -r '
        .items[]
        | . as $pod
        | .spec.volumes[]?
        | select(.persistentVolumeClaim != null)
        | [$pod.metadata.namespace, .persistentVolumeClaim.claimName]
        | join("/")
      '
}

# ──────────────────────────────────────────────────────────────────────────────
# Main logic
# ──────────────────────────────────────────────────────────────────────────────
main() {
    parse_args "$@"

    require_tool kubectl
    require_tool jq

    build_kubectl_base

    # Verify cluster connectivity early
    if ! kctl cluster-info &>/dev/null; then
        die "Cannot connect to the Kubernetes cluster. Check your kubeconfig / context."
    fi

    local current_context
    current_context=$(kctl config current-context 2>/dev/null || echo "unknown")
    log "Using context: ${CONTEXT:-$current_context}"
    [[ -n "$NAMESPACE" ]] && log "Namespace filter: $NAMESPACE" || log "Inspecting all namespaces"

    local now_epoch
    now_epoch=$(date +%s)
    local pending_age_seconds=$(( PENDING_AGE_MINUTES * 60 ))

    log "Fetching PVCs..."
    local pvc_json
    pvc_json=$(fetch_pvcs)

    log "Fetching Pod→PVC references..."
    local pod_pvc_set
    # Build a newline-separated list then convert to a lookup-friendly format
    pod_pvc_set=$(fetch_pod_pvc_refs | sort -u)

    # Temporary files for the orphan list (namespace, name, status, reason)
    local tmpfile
    tmpfile=$(mktemp /tmp/orphaned-pvcs.XXXXXX)
    trap 'rm -f "$tmpfile"' EXIT

    # Parse PVCs and identify orphans
    while IFS=$'\t' read -r ns name status created_at; do
        local reason=""
        local is_orphan=false

        case "$status" in
            Lost)
                if $INCLUDE_LOST; then
                    is_orphan=true
                    reason="status=Lost (backing PV gone)"
                fi
                ;;
            Released)
                if $INCLUDE_RELEASED; then
                    is_orphan=true
                    reason="status=Released (PV released but not reclaimed)"
                fi
                ;;
            Pending)
                if $INCLUDE_PENDING; then
                    local created_epoch
                    created_epoch=$(iso_to_epoch "$created_at")
                    local age_seconds=$(( now_epoch - created_epoch ))
                    if (( age_seconds >= pending_age_seconds )); then
                        is_orphan=true
                        local age_min=$(( age_seconds / 60 ))
                        reason="status=Pending for ${age_min}m (threshold: ${PENDING_AGE_MINUTES}m)"
                    fi
                fi
                ;;
            Bound)
                if $INCLUDE_UNBOUND; then
                    local ref_key="${ns}/${name}"
                    if ! echo "$pod_pvc_set" | grep -qxF "$ref_key"; then
                        is_orphan=true
                        reason="Bound but not referenced by any Pod"
                    fi
                fi
                ;;
        esac

        if $is_orphan; then
            printf '%s\t%s\t%s\t%s\n' "$ns" "$name" "$status" "$reason" >> "$tmpfile"
        fi
    done < <(
        echo "$pvc_json" \
        | jq -r '
            .items[]
            | [
                .metadata.namespace,
                .metadata.name,
                .status.phase,
                .metadata.creationTimestamp
              ]
            | @tsv
          '
    )

    local orphan_count
    orphan_count=$(wc -l < "$tmpfile" | tr -d ' ')

    if (( orphan_count == 0 )); then
        log "No orphaned PVCs found."
        exit 0
    fi

    log "Found ${orphan_count} orphaned PVC(s)."

    # ── Output ────────────────────────────────────────────────────────────────
    case "$OUTPUT_FORMAT" in
        json)
            echo "["
            local first=true
            while IFS=$'\t' read -r ns name status reason; do
                $first || echo ","
                first=false
                printf '  {"namespace":"%s","name":"%s","status":"%s","reason":"%s"}' \
                    "$ns" "$name" "$status" "$reason"
            done < "$tmpfile"
            echo ""
            echo "]"
            ;;
        csv)
            echo "namespace,name,status,reason"
            while IFS=$'\t' read -r ns name status reason; do
                printf '"%s","%s","%s","%s"\n' "$ns" "$name" "$status" "$reason"
            done < "$tmpfile"
            ;;
        table|*)
            printf '\n%-30s %-50s %-12s %s\n' "NAMESPACE" "NAME" "STATUS" "REASON"
            printf '%s\n' "$(printf '─%.0s' {1..110})"
            while IFS=$'\t' read -r ns name status reason; do
                printf '%-30s %-50s %-12s %s\n' "$ns" "$name" "$status" "$reason"
            done < "$tmpfile"
            echo ""
            ;;
    esac

    # ── Dry-run or delete ─────────────────────────────────────────────────────
    if $DRY_RUN; then
        warn "DRY-RUN mode: no PVCs were deleted. Use --delete to remove them."
        exit 0
    fi

    # Confirm before deleting
    if ! $SKIP_CONFIRM; then
        echo ""
        read -rp "Delete the ${orphan_count} orphaned PVC(s) listed above? [y/N] " answer
        case "$answer" in
            [yY][eE][sS]|[yY]) ;;
            *) log "Aborted – no PVCs deleted."; exit 0 ;;
        esac
    fi

    log "Deleting orphaned PVCs..."
    local deleted=0
    local failed=0

    while IFS=$'\t' read -r ns name _status _reason; do
        if kctl delete pvc "$name" --namespace "$ns" --wait=false 2>/dev/null; then
            log "Deleted: $ns/$name"
            (( deleted++ )) || true
        else
            warn "Failed to delete: $ns/$name"
            (( failed++ )) || true
        fi
    done < "$tmpfile"

    log "Done. Deleted: ${deleted}, Failed: ${failed}."
    (( failed > 0 )) && exit 1 || exit 0
}

main "$@"
