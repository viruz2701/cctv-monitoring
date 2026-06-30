#!/bin/bash
# =============================================================================
# OpenWrt Cross-Compilation Script for Edge Agent
# Builds the edge-agent binary for OpenWrt (MIPS, ARM, x86_64 targets).
#
# Prerequisites:
#   - OpenWrt SDK installed (or use Docker)
#   - Go 1.25+ cross-compiler toolchain
#
# Usage:
#   ./scripts/build_openwrt.sh [target]
#
# Targets:
#   mips      - MIPS 24Kc (MT7620/MT7628, common OpenWrt routers)
#   arm       - ARMv7 (ipq40xx, mt7623)
#   arm64     - ARMv8/AArch64 (ipq8074, mt7986)
#   amd64     - x86_64 (x86/64)
#   all       - Build all targets
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_DIR="${PROJECT_DIR}/build"

# Application metadata
APP_NAME="edge-agent"
APP_VERSION="${VERSION:-1.0.0}"
LD_FLAGS="-s -w -X main.Version=${APP_VERSION}"

# OpenWrt target configurations
declare -A TARGETS
TARGETS[mips]="GOOS=linux GOARCH=mipsle GOMIPS=softfloat"
TARGETS[arm]="GOOS=linux GOARCH=arm GOARM=7"
TARGETS[arm64]="GOOS=linux GOARCH=arm64"
TARGETS[amd64]="GOOS=linux GOARCH=amd64"

build_target() {
    local target_name="$1"
    local target_config="${TARGETS[$target_name]:-}"

    if [ -z "${target_config}" ]; then
        echo "ERROR: Unknown target '${target_name}'"
        echo "Available targets: ${!TARGETS[*]}"
        return 1
    fi

    echo "=== Building for ${target_name} ==="
    local output_name="${APP_NAME}-${target_name}"
    local output_path="${OUTPUT_DIR}/${output_name}"

    # Set cross-compilation environment
    eval "export ${target_config}"
    export CGO_ENABLED=0

    cd "${PROJECT_DIR}"

    # Build
    go build \
        -ldflags="${LD_FLAGS}" \
        -o "${output_path}" \
        -trimpath \
        ./cmd/agent

    # Strip and compress
    if command -v upx &> /dev/null; then
        echo "Compressing with UPX..."
        upx --lzma "${output_path}" 2>/dev/null || true
    fi

    # Calculate size and hash
    local size
    size=$(stat -c%s "${output_path}" 2>/dev/null || stat -f%z "${output_path}" 2>/dev/null)
    local hash
    hash=$(sha256sum "${output_path}" | cut -d' ' -f1)

    echo "  Output: ${output_path}"
    echo "  Size: $((size / 1024)) KB"
    echo "  SHA256: ${hash}"

    # Create versioned copy
    cp "${output_path}" "${OUTPUT_DIR}/${output_name}-v${APP_VERSION}"

    echo "=== Build complete for ${target_name} ==="
    echo ""
}

# Main
mkdir -p "${OUTPUT_DIR}"

TARGET="${1:-all}"

if [ "${TARGET}" = "all" ]; then
    for t in "${!TARGETS[@]}"; do
        build_target "${t}"
    done
else
    build_target "${TARGET}"
fi

echo ""
echo "=== All builds complete ==="
echo "Output directory: ${OUTPUT_DIR}"
ls -lh "${OUTPUT_DIR}/"
