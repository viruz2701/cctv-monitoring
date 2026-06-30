package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// LoadClientTLS creates a mTLS configuration for client connections.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.2 — mTLS 1.3 для всех соединений
//   - IEC 62443-3-3 SL-3 — шифрование каналов между зонами
//   - СТБ 34.101.27 — защита информации при передаче
//
// Parameters:
//   - certFile: path to client certificate (PEM)
//   - keyFile: path to client private key (PEM)
//   - caFile: path to CA certificate (PEM)
//   - serverName: expected server name for verification
//
// Returns tls.Config ready for use with mTLS connections.
func LoadClientTLS(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("client cert and key are required for mTLS")
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		// Client certificate for mTLS authentication
		Certificates: []tls.Certificate{cert},

		// CA certificate for server verification
		RootCAs: caCertPool,

		// Server name for SNI and certificate verification
		ServerName: serverName,

		// TLS 1.3 minimum (Приказ ОАЦ №66 п. 7.18.2)
		MinVersion: tls.VersionTLS13,

		// Strong cipher suites (СТБ 34.101.30 compatible where available)
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},

		// Additional security settings
		InsecureSkipVerify:     false,
		SessionTicketsDisabled: true,
		Renegotiation:          tls.RenegotiateNever,
	}

	return tlsConfig, nil
}

// LoadServerTLS creates a mTLS configuration for server-side connections.
// Used if the edge agent needs to run its own TLS server.
func LoadServerTLS(certFile, keyFile, caFile string, requireClientCert bool) (*tls.Config, error) {
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("server cert and key are required")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key: %w", err)
	}

	// Build TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
		},
		SessionTicketsDisabled: true,
		Renegotiation:          tls.RenegotiateNever,
	}

	// If client certificate verification is required
	if requireClientCert && caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// LoadCertPool creates a certificate pool from a PEM file.
// Useful for loading trusted CA certificates.
func LoadCertPool(caFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
	}

	return pool, nil
}
