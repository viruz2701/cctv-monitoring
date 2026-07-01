#!/bin/sh
# OTA Update Script for Edge Agent — Dual-boot A/B
#
# Uses A/B partition approach with Ed25519 signature verification.
# No single backup file — rollback switches between slots.
#
# Architecture:
#   /usr/local/bin/edge-agent.a   ← Slot A
#   /usr/local/bin/edge-agent.b   ← Slot B
#   /usr/local/bin/edge-agent     ← Symlink → active slot
#
# Commands:
#   ota_update.sh update          — Download, verify, switch, restart
#   ota_update.sh rollback        — Switch to inactive slot
#   ota_update.sh status          — Show slot status
#   ota_update.sh switch          — Manually switch active slot
#   ota_update.sh verify <file> <sigfile> <pubkey> — Verify Ed25519 signature
#
# Compliance:
#   - IEC 62443-3-3 SL-3         (Zone 5 — Edge)
#   - Приказ ОАЦ №66 п. 7.18.3   — Контроль целостности (Ed25519)
#   - Приказ ОАЦ №66 п. 7.18.5   — Управление обновлениями (rollback)
#   - OWASP ASVS V12             — File integrity verification

set -euo pipefail

# ═══ Configuration ═══
SLOT_A="/usr/local/bin/edge-agent.a"
SLOT_B="/usr/local/bin/edge-agent.b"
SYMLINK="/usr/local/bin/edge-agent"
SERVICE_NAME="edge-agent"
ROLLBACK_TIMEOUT="${ROLLBACK_TIMEOUT:-30}"
OTA_DIR="${OTA_DIR:-/usb/ota}"
LOG_TAG="ota-update"

# ═══ Helpers ═══

log() {
    logger -t "${LOG_TAG}" "$@"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    log "ERROR: $*"
    exit 1
}

# current_slot resolves the currently active A/B slot via symlink.
current_slot() {
    if [ ! -L "${SYMLINK}" ]; then
        echo "none"
        return
    fi

    target=$(readlink -f "${SYMLINK}" 2>/dev/null || readlink "${SYMLINK}" 2>/dev/null)
    case "${target}" in
        "${SLOT_A}") echo "A" ;;
        "${SLOT_B}") echo "B" ;;
        *)           echo "unknown (${target})" ;;
    esac
}

# inactive_slot returns the slot NOT currently active.
inactive_slot() {
    case "$(current_slot)" in
        "A") echo "B" ;;
        "B") echo "A" ;;
        *)   echo "A" ;;  # default target if no active slot
    esac
}

# slot_path returns the binary path for a given slot label (A or B).
slot_path() {
    case "$1" in
        "A") echo "${SLOT_A}" ;;
        "B") echo "${SLOT_B}" ;;
        *)   echo "" ;;
    esac
}

# ═══ Core Operations ═══

# verify_ed25519 verifies an Ed25519 signature on a binary file.
#
# Usage: verify_ed25519 <binary> <signature_file> <public_key_pem>
#
# Compliance:
#   - Приказ ОАЦ №66 п. 7.18.3: Контроль целостности
#   - IEC 62443-3-3 SL-3: Signed firmware
verify_ed25519() {
    binary="$1"
    sig_file="$2"
    pubkey="$3"

    if [ ! -f "${binary}" ]; then
        error "Binary not found: ${binary}"
    fi
    if [ ! -f "${sig_file}" ]; then
        error "Signature file not found: ${sig_file}"
    fi
    if [ ! -f "${pubkey}" ]; then
        error "Public key not found: ${pubkey}"
    fi

    log "Verifying Ed25519 signature: ${binary}"

    # Ed25519 verification via openssl
    if ! openssl dgst -sha512 -verify "${pubkey}" -signature "${sig_file}" "${binary}" 2>/dev/null; then
        error "Ed25519 signature verification FAILED — possible tampering detected"
    fi

    log "Ed25519 signature verified: ${binary}"
}

# switch_slot atomically updates the symlink to point to the given slot.
#
# Uses a temporary symlink + rename for atomicity.
# Compliance: Приказ ОАЦ №66 п. 7.18.5 — атомарное обновление
switch_slot() {
    target_slot="$1"
    target_path=$(slot_path "${target_slot}")

    if [ -z "${target_path}" ]; then
        error "Invalid slot: ${target_slot}"
    fi

    if [ ! -f "${target_path}" ]; then
        error "Slot ${target_slot} binary not found: ${target_path}"
    fi

    current=$(current_slot)
    log "Switching symlink: ${current} → ${target_slot}"

    # Atomic: create temp symlink, then rename
    tmp_symlink="${SYMLINK}.tmp"
    rm -f "${tmp_symlink}"
    ln -s "${target_path}" "${tmp_symlink}"
    mv -f "${tmp_symlink}" "${SYMLINK}"
    sync

    log "Symlink switched to slot ${target_slot}: ${SYMLINK} → ${target_path}"
}

# restart_service calls systemctl restart.
restart_service() {
    log "Restarting service ${SERVICE_NAME}"
    systemctl restart "${SERVICE_NAME}"
}

# health_check polls systemctl is-active with a timeout.
health_check() {
    timeout="${1:-${ROLLBACK_TIMEOUT}}"
    elapsed=0
    interval=2

    while [ "${elapsed}" -lt "${timeout}" ]; do
        if systemctl is-active --quiet "${SERVICE_NAME}"; then
            log "Health check passed — service ${SERVICE_NAME} is active"
            return 0
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done

    error "Health check failed — service ${SERVICE_NAME} did not start within ${timeout}s"
    return 1
}

# ═══ Command: update ═══
#
# Downloads a new binary to the inactive slot, verifies the Ed25519 signature,
# switches the symlink, restarts the service, and performs a health check.
# On failure, automatically rolls back to the previous slot.
#
# Usage: ota_update.sh update <binary_path> <sig_path> <pubkey_path>
#
# Parameters:
#   binary_path  — path to the downloaded binary
#   sig_path     — path to the .sig Ed25519 signature file
#   pubkey_path  — path to the Ed25519 public key PEM file
cmd_update() {
    if [ $# -lt 3 ]; then
        error "Usage: $0 update <binary_path> <sig_path> <pubkey_path>"
    fi

    binary_path="$1"
    sig_path="$2"
    pubkey_path="$3"

    target_slot=$(inactive_slot)
    target_path=$(slot_path "${target_slot}")

    log "Starting OTA update to slot ${target_slot}"

    # Verify source exists
    if [ ! -f "${binary_path}" ]; then
        error "Binary not found: ${binary_path}"
    fi
    if [ ! -f "${sig_path}" ]; then
        error "Signature not found: ${sig_path}"
    fi
    if [ ! -f "${pubkey_path}" ]; then
        error "Public key not found: ${pubkey_path}"
    fi

    # Verify Ed25519 signature before installation
    verify_ed25519 "${binary_path}" "${sig_path}" "${pubkey_path}"

    # Install to inactive slot
    log "Installing binary to slot ${target_slot}: ${target_path}"
    cp -f "${binary_path}" "${target_path}"
    chmod +x "${target_path}"
    sync

    # Save previous slot for rollback
    previous_slot=$(current_slot)

    # Switch symlink
    switch_slot "${target_slot}"

    # Restart service
    restart_service

    # Health check with automatic rollback on failure
    if ! health_check "${ROLLBACK_TIMEOUT}"; then
        log "Health check failed after update, rolling back to slot ${previous_slot}"

        switch_slot "${previous_slot}"
        restart_service

        if health_check "${ROLLBACK_TIMEOUT}"; then
            log "Rollback completed successfully"
        else
            error "Rollback failed — service did not start after rollback"
        fi

        error "Update failed, rolled back to slot ${previous_slot}"
    fi

    log "OTA update completed successfully: slot ${target_slot} active"
}

# ═══ Command: rollback ═══
#
# Switches to the inactive slot (previous version) and restarts.
# No backup file needed — both slots preserve their binaries.
#
# Compliance: Приказ ОАЦ №66 п. 7.18.5 — rollback без backup-файла
cmd_rollback() {
    current=$(current_slot)
    target=$(inactive_slot)

    log "Initiating rollback: ${current} → ${target}"

    target_path=$(slot_path "${target}")
    if [ ! -f "${target_path}" ]; then
        error "Rollback target slot ${target} has no binary at ${target_path}"
    fi

    # Switch symlink
    switch_slot "${target}"

    # Restart service
    restart_service

    # Health check
    if health_check "${ROLLBACK_TIMEOUT}"; then
        log "Rollback completed successfully: slot ${target} active"
    else
        # Critical: try to switch back
        log "Rollback health check failed, attempting recovery to slot ${current}"
        switch_slot "${current}"
        restart_service
        error "Rollback failed, recovered to slot ${current}"
    fi
}

# ═══ Command: status ═══
#
# Shows the status of both A/B slots.
cmd_status() {
    echo "=== Edge Agent OTA Slot Status ==="
    echo ""

    active=$(current_slot)
    echo "Active slot: ${active}"
    echo "Symlink: ${SYMLINK} → $(readlink -f "${SYMLINK}" 2>/dev/null || echo 'not found')"
    echo ""

    for slot_label in "A" "B"; do
        slot_path=$(slot_path "${slot_label}")
        label="inactive"
        [ "${slot_label}" = "${active}" ] && label="ACTIVE"

        if [ -f "${slot_path}" ]; then
            size=$(stat -c%s "${slot_path}" 2>/dev/null || stat -f%z "${slot_path}" 2>/dev/null || echo "?")
            mod_time=$(stat -c%y "${slot_path}" 2>/dev/null || stat -f%Sm "${slot_path}" 2>/dev/null || echo "?")
            echo "  Slot ${slot_label} (${label}):"
            echo "    Path: ${slot_path}"
            echo "    Size: ${size} bytes"
            echo "    Modified: ${mod_time}"
        else
            echo "  Slot ${slot_label} (${label}): ABSENT"
        fi
        echo ""
    done
}

# ═══ Command: switch ═══
#
# Manually switch the active slot (for recovery).
cmd_switch() {
    target_slot="$1"

    if [ -z "${target_slot}" ]; then
        echo "Available slots:"
        for slot_label in "A" "B"; do
            slot_path=$(slot_path "${slot_label}")
            active_flag=""
            [ "${slot_label}" = "$(current_slot)" ] && active_flag=" (active)"
            if [ -f "${slot_path}" ]; then
                echo "  ${slot_label}${active_flag}"
            else
                echo "  ${slot_label}${active_flag} — ABSENT"
            fi
        done
        error "Usage: $0 switch <A|B>"
    fi

    case "${target_slot}" in
        A|B) ;;
        *) error "Invalid slot: ${target_slot}. Use A or B." ;;
    esac

    current=$(current_slot)
    if [ "${target_slot}" = "${current}" ]; then
        log "Slot ${target_slot} is already active"
        exit 0
    fi

    switch_slot "${target_slot}"
    restart_service

    if health_check "${ROLLBACK_TIMEOUT}"; then
        log "Switched to slot ${target_slot} successfully"
    else
        # Revert
        switch_slot "${current}"
        restart_service
        error "Switch to slot ${target_slot} failed, reverted to ${current}"
    fi
}

# ═══ Command: verify ═══
#
# Verify Ed25519 signature on a binary file.
cmd_verify() {
    if [ $# -lt 3 ]; then
        error "Usage: $0 verify <binary> <sig_file> <pubkey_pem>"
    fi

    verify_ed25519 "$1" "$2" "$3"
    log "Signature verification passed"
}

# ═══ Main ═══

usage() {
    echo "Edge Agent OTA Update — Dual-boot A/B"
    echo ""
    echo "Usage: $0 {update|rollback|status|switch|verify}"
    echo ""
    echo "Commands:"
    echo "  update   <binary> <sig> <pubkey>   Verify, install, restart"
    echo "  rollback                            Switch to inactive slot"
    echo "  status                              Show A/B slot status"
    echo "  switch    <A|B>                     Manually switch active slot"
    echo "  verify    <binary> <sig> <pubkey>   Verify Ed25519 signature"
    echo ""
    echo "Environment:"
    echo "  OTA_DIR           OTA working directory (default: /usb/ota)"
    echo "  ROLLBACK_TIMEOUT  Health check timeout in seconds (default: 30)"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

COMMAND="$1"
shift

case "${COMMAND}" in
    update)
        cmd_update "$@"
        ;;
    rollback)
        cmd_rollback
        ;;
    status)
        cmd_status
        ;;
    switch)
        cmd_switch "$@"
        ;;
    verify)
        cmd_verify "$@"
        ;;
    *)
        usage
        ;;
esac
