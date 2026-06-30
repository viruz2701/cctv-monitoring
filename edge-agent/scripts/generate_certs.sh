#!/bin/bash
# =============================================================================
# Certificate Generation Script for Edge Agent
# Generates mTLS certificates for secure Edge Agent ⇄ Backend / MQTT communication.
#
# Compliance:
#   - Приказ ОАЦ №66 п. 7.18.2 — mTLS 1.3 для всех соединений
#   - IEC 62443-3-3 SL-3 — шифрование каналов между зонами
#   - СТБ 34.101.30 — криптография (используем RSA/ECDSA для совместимости с TLS)
# =============================================================================

set -euo pipefail

: "${CA_DAYS:=3650}"
: "${CERT_DAYS:=730}"
: "${KEY_SIZE:=2048}"
: "${OUT_DIR:=./certs}"
: "${COUNTRY:=BY}"
: "${ORG:=CCTV-Health-Monitor}"
: "${CA_NAME:=Edge-Agent-CA}"

mkdir -p "${OUT_DIR}"

echo "=== Generating Certificate Authority ==="
# Generate CA private key
openssl genrsa -out "${OUT_DIR}/ca.key" "${KEY_SIZE}"

# Generate CA certificate
openssl req -x509 -new -nodes \
    -key "${OUT_DIR}/ca.key" \
    -sha256 -days "${CA_DAYS}" \
    -out "${OUT_DIR}/ca.crt" \
    -subj "/C=${COUNTRY}/O=${ORG}/CN=${CA_NAME}"

echo "=== Generating Edge Agent Certificate ==="
# Generate agent private key
openssl genrsa -out "${OUT_DIR}/agent.key" "${KEY_SIZE}"

# Generate CSR
openssl req -new \
    -key "${OUT_DIR}/agent.key" \
    -out "${OUT_DIR}/agent.csr" \
    -subj "/C=${COUNTRY}/O=${ORG}/CN=edge-agent-$(hostname)"

# Sign with CA
openssl x509 -req \
    -in "${OUT_DIR}/agent.csr" \
    -CA "${OUT_DIR}/ca.crt" \
    -CAkey "${OUT_DIR}/ca.key" \
    -CAcreateserial \
    -out "${OUT_DIR}/agent.crt" \
    -days "${CERT_DAYS}" \
    -sha256 \
    -extfile <(cat <<EOF
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth,serverAuth
subjectAltName=DNS:edge-agent,DNS:localhost,IP:127.0.0.1
EOF
)

echo "=== Generating MQTT Broker Certificate (for testing) ==="
openssl genrsa -out "${OUT_DIR}/mqtt.key" "${KEY_SIZE}"

openssl req -new \
    -key "${OUT_DIR}/mqtt.key" \
    -out "${OUT_DIR}/mqtt.csr" \
    -subj "/C=${COUNTRY}/O=${ORG}/CN=mqtt-broker"

openssl x509 -req \
    -in "${OUT_DIR}/mqtt.csr" \
    -CA "${OUT_DIR}/ca.crt" \
    -CAkey "${OUT_DIR}/ca.key" \
    -CAcreateserial \
    -out "${OUT_DIR}/mqtt.crt" \
    -days "${CERT_DAYS}" \
    -sha256 \
    -extfile <(cat <<EOF
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:mqtt-broker,DNS:localhost,IP:127.0.0.1
EOF
)

# Clean up CSRs
rm -f "${OUT_DIR}/agent.csr" "${OUT_DIR}/mqtt.csr"

echo ""
echo "=== Certificate Generation Complete ==="
echo "Output directory: ${OUT_DIR}"
echo ""
echo "Files generated:"
ls -la "${OUT_DIR}/"
echo ""
echo "=== Environment Variables to Set ==="
echo "EDGE_AGENT_MQTT_CERT=${OUT_DIR}/agent.crt"
echo "EDGE_AGENT_MQTT_KEY=${OUT_DIR}/agent.key"
echo "EDGE_AGENT_MQTT_CA=${OUT_DIR}/ca.crt"
echo ""
echo "NOTE: In production, use a proper PKI infrastructure!"
