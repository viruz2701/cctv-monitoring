#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# P3-DR: DNS Failover Script
#
# Автоматическое переключение DNS записей при DR failover.
# Поддерживает Cloudflare DNS, AWS Route53, и generic DNS API.
#
# Usage:
#   ./failover.sh --from REGION --to REGION [--dry-run] [--provider cloudflare|route53]
#
# Examples:
#   ./failover.sh --from eu-central --to cis-east --dry-run
#   ./failover.sh --from eu-central --to cis-east --provider cloudflare
#
# Compliance:
#   - ISO 27001 A.17.1.2 (DR procedures — DNS failover)
#   - IEC 62443-3-3 SR 7.3 (Failover mechanisms)
#   - Приказ ОАЦ №66 п. 7.18.2 (Резервирование каналов связи)
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ── Конфигурация ──────────────────────────────────────────────────────────────

# DNS провайдер (cloudflare|route53|generic)
PROVIDER="${DNS_PROVIDER:-cloudflare}"

# Cloudflare
CLOUDFLARE_ZONE_ID="${CLOUDFLARE_ZONE_ID:-}"
CLOUDFLARE_API_TOKEN="${CLOUDFLARE_API_TOKEN:-}"

# AWS Route53
ROUTE53_ZONE_ID="${ROUTE53_ZONE_ID:-}"
AWS_PROFILE="${AWS_PROFILE:-default}"

# Generic DNS API
DNS_API_URL="${DNS_API_URL:-}"
DNS_API_KEY="${DNS_API_KEY:-}"

# DNS записи для failover (пространство имён CCTV)
# Формат: "record_name:primary_value:dr_value"
DNS_RECORDS=(
    "api.cctv.example.com:192.168.1.10:192.168.2.10"
    "nats.cctv.example.com:192.168.1.20:192.168.2.20"
    "db.cctv.example.com:192.168.1.30:192.168.2.30"
    "redis.cctv.example.com:192.168.1.40:192.168.2.40"
    "*.cctv.example.com:primary-cluster:dr-cluster"
)

# TTL для DNS записей при failover (30 секунд для быстрого переключения)
FAILOVER_TTL=30
NORMAL_TTL=300

# ── Helper Functions ─────────────────────────────────────────────────────────

log_info()  { echo "[INFO]  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_warn()  { echo "[WARN]  $(date '+%Y-%m-%d %H:%M:%S') $*"; }
log_error() { echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') $*" >&2; }

usage() {
    cat <<EOF
P3-DR: DNS Failover Script

Usage:
    $0 --from REGION --to REGION [options]

Options:
    --from REGION       Source region (e.g., eu-central, cis-east)
    --to REGION         Target region (e.g., cis-east, eu-central)
    --dry-run           Preview changes without applying
    --provider TYPE     DNS provider: cloudflare, route53, generic (default: \$DNS_PROVIDER)
    --ttl SECONDS       TTL for failover records (default: $FAILOVER_TTL)
    --verbose           Enable verbose output
    --help              Show this help message

Environment:
    CLOUDFLARE_ZONE_ID     Cloudflare Zone ID
    CLOUDFLARE_API_TOKEN   Cloudflare API Token
    ROUTE53_ZONE_ID        Route53 Hosted Zone ID
    DNS_API_URL            Generic DNS API URL
    DNS_API_KEY            Generic DNS API Key

Examples:
    $0 --from eu-central --to cis-east --dry-run
    $0 --from eu-central --to cis-east --provider cloudflare
EOF
    exit 1
}

# ── Parse Arguments ──────────────────────────────────────────────────────────

FROM_REGION=""
TO_REGION=""
DRY_RUN=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --from)     FROM_REGION="$2"; shift 2 ;;
        --to)       TO_REGION="$2"; shift 2 ;;
        --dry-run)  DRY_RUN=true; shift ;;
        --provider) PROVIDER="$2"; shift 2 ;;
        --ttl)      FAILOVER_TTL="$2"; shift 2 ;;
        --verbose)  VERBOSE=true; shift ;;
        --help)     usage ;;
        *)          log_error "Unknown option: $1"; usage ;;
    esac
done

# ── Validation ────────────────────────────────────────────────────────────────

if [[ -z "$FROM_REGION" || -z "$TO_REGION" ]]; then
    log_error "Both --from and --to are required"
    usage
fi

if [[ "$DRY_RUN" == "false" ]]; then
    case "$PROVIDER" in
        cloudflare)
            if [[ -z "$CLOUDFLARE_ZONE_ID" || -z "$CLOUDFLARE_API_TOKEN" ]]; then
                log_error "Cloudflare provider requires CLOUDFLARE_ZONE_ID and CLOUDFLARE_API_TOKEN"
                exit 1
            fi
            ;;
        route53)
            if [[ -z "$ROUTE53_ZONE_ID" ]]; then
                log_error "Route53 provider requires ROUTE53_ZONE_ID"
                exit 1
            fi
            ;;
        generic)
            if [[ -z "$DNS_API_URL" ]]; then
                log_error "Generic provider requires DNS_API_URL"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown provider: $PROVIDER (supported: cloudflare, route53, generic)"
            exit 1
            ;;
    esac
fi

# ── Failover Execution ────────────────────────────────────────────────────────

log_info "=== P3-DR: DNS Failover ==="
log_info "From:     $FROM_REGION"
log_info "To:       $TO_REGION"
log_info "Provider: $PROVIDER"
log_info "Dry-Run:  $DRY_RUN"
echo ""

FAILOVER_COUNT=0
FAILOVER_FAILED=0

for record in "${DNS_RECORDS[@]}"; do
    IFS=':' read -r name primary_value dr_value <<< "$record"

    log_info "Processing: $name"
    log_info "  Primary: $primary_value → DR: $dr_value"

    case "$PROVIDER" in
        cloudflare)
            # Cloudflare API v4 — обновление DNS записи
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "  [DRY-RUN] Would update Cloudflare DNS: $name → $dr_value (TTL: $FAILOVER_TTL)"
                FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
            else
                # Получаем ID записи
                RECORD_ID=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_ZONE_ID}/dns_records?name=${name}" \
                    -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
                    -H "Content-Type: application/json" | jq -r '.result[0].id')

                if [[ -z "$RECORD_ID" || "$RECORD_ID" == "null" ]]; then
                    log_error "  Failed to get DNS record ID for $name"
                    FAILOVER_FAILED=$((FAILOVER_FAILED + 1))
                    continue
                fi

                # Обновляем запись на DR значение
                RESPONSE=$(curl -s -X PUT "https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_ZONE_ID}/dns_records/${RECORD_ID}" \
                    -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
                    -H "Content-Type: application/json" \
                    -d "{\"type\":\"A\",\"name\":\"${name}\",\"content\":\"${dr_value}\",\"ttl\":${FAILOVER_TTL}}")

                if echo "$RESPONSE" | jq -e '.success' > /dev/null; then
                    log_info "  ✓ Updated $name → $dr_value"
                    FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
                else
                    ERROR_MSG=$(echo "$RESPONSE" | jq -r '.errors[0].message')
                    log_error "  ✗ Failed to update $name: $ERROR_MSG"
                    FAILOVER_FAILED=$((FAILOVER_FAILED + 1))
                fi
            fi
            ;;

        route53)
            # AWS Route53 — изменение resource record set
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "  [DRY-RUN] Would update Route53: $name → $dr_value (TTL: $FAILOVER_TTL)"
                FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
            else
                CHANGE_BATCH=$(cat <<EOF
{
    "Changes": [
        {
            "Action": "UPSERT",
            "ResourceRecordSet": {
                "Name": "${name}",
                "Type": "A",
                "TTL": ${FAILOVER_TTL},
                "ResourceRecords": [
                    {"Value": "${dr_value}"}
                ]
            }
        }
    ]
}
EOF
)
                if aws route53 change-resource-record-sets \
                    --hosted-zone-id "$ROUTE53_ZONE_ID" \
                    --change-batch "$CHANGE_BATCH" \
                    --profile "$AWS_PROFILE" > /dev/null 2>&1; then
                    log_info "  ✓ Updated $name → $dr_value"
                    FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
                else
                    log_error "  ✗ Failed to update $name via Route53"
                    FAILOVER_FAILED=$((FAILOVER_FAILED + 1))
                fi
            fi
            ;;

        generic)
            # Generic DNS API
            if [[ "$DRY_RUN" == "true" ]]; then
                log_info "  [DRY-RUN] Would call DNS API: $name → $dr_value"
                FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
            else
                RESPONSE=$(curl -s -X POST "${DNS_API_URL}/records" \
                    -H "Authorization: Bearer ${DNS_API_KEY}" \
                    -H "Content-Type: application/json" \
                    -d "{\"name\":\"${name}\",\"value\":\"${dr_value}\",\"ttl\":${FAILOVER_TTL}}")

                if [[ $? -eq 0 ]]; then
                    log_info "  ✓ Updated $name → $dr_value via generic API"
                    FAILOVER_COUNT=$((FAILOVER_COUNT + 1))
                else
                    log_error "  ✗ Failed to update $name via generic API"
                    FAILOVER_FAILED=$((FAILOVER_FAILED + 1))
                fi
            fi
            ;;

        *)
            log_error "Unsupported provider: $PROVIDER"
            exit 1
            ;;
    esac
done

# ── Summary ────────────────────────────────────────────────────────────────────

echo ""
log_info "=== Failover Summary ==="
log_info "Total records processed: $FAILOVER_COUNT"
log_info "Failed:                 $FAILOVER_FAILED"
log_info "From:                   $FROM_REGION"
log_info "To:                     $TO_REGION"

if [[ "$DRY_RUN" == "true" ]]; then
    log_warn "DRY-RUN completed — no actual changes were made"
    log_info "To apply changes, run without --dry-run"
fi

if [[ "$FAILOVER_FAILED" -gt 0 ]]; then
    log_error "Failover completed with $FAILOVER_FAILED failures"
    exit 1
fi

log_info "DNS failover completed successfully"
exit 0
