#!/bin/sh
# OTA Update Script for Edge Agent
#
# Used by OTAUpdater (ota.go) for binary update and rollback operations.
# Can also be invoked directly for manual recovery.
#
# Usage:
#   ota_update.sh update <sha256_hash>  — verify + install + restart
#   ota_update.sh rollback              — restore previous version
#   ota_update.sh verify <sha256_hash>  — integrity check only
#
# Compliance:
#   - IEC 62443-3-3 SL-3   (Zone 5 — Edge)
#   - Приказ ОАЦ №66 п. 7.18.3 — Контроль целостности
#   - Приказ ОАЦ №66 п. 7.18.5 — Управление обновлениями (rollback)
#   - OWASP ASVS V12        — File integrity verification

set -euo pipefail

# ═══ Configuration ═══
AGENT_BINARY="/usr/local/bin/edge-agent"
OTA_DIR="${OTA_DIR:-/usb/ota}"
BACKUP_PATH="${OTA_DIR}/edge-agent.bak"
NEW_BINARY="${OTA_DIR}/edge-agent.new"
SERVICE_NAME="edge-agent"
ROLLBACK_TIMEOUT="${ROLLBACK_TIMEOUT:-30}"
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

# verify_integrity checks SHA256 hash of a file.
# Compliance: Приказ ОАЦ №66 п. 7.18.3, OWASP ASVS V12.3
verify_integrity() {
    file="$1"
    expected_hash="$2"

    if [ ! -f "${file}" ]; then
        error "File not found: ${file}"
    fi

    actual_hash=$(sha256sum "${file}" | cut -d' ' -f1)

    if [ "${actual_hash}" != "${expected_hash}" ]; then
        error "SHA256 mismatch: got ${actual_hash}, expected ${expected_hash}"
    fi

    log "SHA256 integrity verified: ${file}"
}

# backup_current copies the running binary to the backup path.
backup_current() {
    if [ -f "${AGENT_BINARY}" ]; then
        log "Backing up current binary to ${BACKUP_PATH}"
        cp -f "${AGENT_BINARY}" "${BACKUP_PATH}"
        chmod --reference="${AGENT_BINARY}" "${BACKUP_PATH}" 2>/dev/null || true
    else
        log "No current binary at ${AGENT_BINARY}, skipping backup"
    fi
}

# install_binary copies src to dst and sets executable permissions.
install_binary() {
    src="$1"
    dst="$2"

    log "Installing binary from ${src} to ${dst}"
    cp -f "${src}" "${dst}"
    chmod +x "${dst}"
    sync
}

# restart_service calls systemctl restart.
restart_service() {
    log "Restarting service ${SERVICE_NAME}"
    systemctl restart "${SERVICE_NAME}"
}

# check_service_health polls systemctl is-active with a timeout.
check_service_health() {
    timeout="$1"
    elapsed=0
    interval=2

    while [ "${elapsed}" -lt "${timeout}" ]; do
        if systemctl is-active --quiet "${SERVICE_NAME}"; then
            log "Service ${SERVICE_NAME} is active"
            return 0
        fi
        sleep "${interval}"
        elapsed=$((elapsed + interval))
    done

    error "Service ${SERVICE_NAME} did not start within ${timeout}s"
    return 1
}

# rollback restores the backup binary and restarts the service.
# Compliance: Приказ ОАЦ №66 п. 7.18.5 — атомарное обновление с rollback
rollback() {
    log "Initiating rollback"

    if [ ! -f "${BACKUP_PATH}" ]; then
        error "Backup not found at ${BACKUP_PATH}, cannot rollback"
    fi

    log "Restoring backup from ${BACKUP_PATH}"
    cp -f "${BACKUP_PATH}" "${AGENT_BINARY}"
    chmod +x "${AGENT_BINARY}"
    sync

    restart_service

    if check_service_health "${ROLLBACK_TIMEOUT}"; then
        log "Rollback completed successfully"
    else
        error "Rollback failed: service did not start"
    fi
}

# cleanup removes temporary files.
cleanup() {
    log "Cleaning up temporary files"
    rm -f "${NEW_BINARY}"
    rm -f "${BACKUP_PATH}"
}

# ═══ Main ═══

usage() {
    echo "Usage: $0 {update|rollback|verify}"
    echo ""
    echo "Commands:"
    echo "  update   <sha256_hash>  Verify, install, and restart"
    echo "  rollback                Restore previous version"
    echo "  verify   <sha256_hash>  Verify binary integrity"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

COMMAND="$1"
shift

case "${COMMAND}" in
    update)
        if [ $# -lt 1 ]; then
            error "Usage: $0 update <sha256_hash>"
        fi
        EXPECTED_HASH="$1"

        if [ ! -f "${NEW_BINARY}" ]; then
            error "New binary not found at ${NEW_BINARY}"
        fi

        log "Starting OTA update"

        # Verify integrity of downloaded binary
        verify_integrity "${NEW_BINARY}" "${EXPECTED_HASH}"

        # Backup current binary
        backup_current

        # Install new binary
        install_binary "${NEW_BINARY}" "${AGENT_BINARY}"

        # Restart service
        restart_service

        # Check health with rollback on failure
        if ! check_service_health "${ROLLBACK_TIMEOUT}"; then
            log "Health check failed, initiating rollback"
            rollback
            error "Update failed, rolled back to previous version"
        fi

        log "OTA update completed successfully"
        cleanup
        ;;

    rollback)
        rollback
        ;;

    verify)
        if [ $# -lt 1 ]; then
            error "Usage: $0 verify <sha256_hash>"
        fi
        verify_integrity "${AGENT_BINARY}" "$1"
        log "Binary integrity verified"
        ;;

    *)
        usage
        ;;
esac
