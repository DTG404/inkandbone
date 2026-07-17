package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateListenSecurity(t *testing.T) {
	certFile, keyFile := writeTestKeyPair(t, "primary")
	_, otherKeyFile := writeTestKeyPair(t, "other")
	secret := strings.Repeat("x", 32)
	publicConfig := ListenSecurityConfig{
		AuthSecret:     secret,
		TLSCertFile:    certFile,
		TLSKeyFile:     keyFile,
		AllowedOrigins: []string{"https://table.example"},
	}

	tests := []struct {
		name    string
		addr    string
		cfg     ListenSecurityConfig
		wantErr string
	}{
		{name: "IPv4 loopback needs no auth", addr: "127.0.0.1:7432"},
		{name: "IPv6 loopback needs no auth", addr: "[::1]:7432"},
		{name: "localhost needs no auth", addr: "localhost:7432"},
		{
			name:    "loopback rejects a blank allowed origin",
			addr:    "127.0.0.1:7432",
			cfg:     ListenSecurityConfig{AllowedOrigins: []string{" "}},
			wantErr: "allowed origin",
		},
		{
			name:    "loopback rejects a malformed allowed origin",
			addr:    "127.0.0.1:7432",
			cfg:     ListenSecurityConfig{AllowedOrigins: []string{"https://table.example/path"}},
			wantErr: "valid HTTP or HTTPS origin",
		},
		{
			name:    "loopback rejects a non-HTTP allowed origin",
			addr:    "127.0.0.1:7432",
			cfg:     ListenSecurityConfig{AllowedOrigins: []string{"file://table.example"}},
			wantErr: "valid HTTP or HTTPS origin",
		},
		{name: "public IPv4 needs secret", addr: "0.0.0.0:7432", wantErr: "TTRPG_AUTH_SECRET"},
		{name: "public IPv6 needs secret", addr: "[::]:7432", wantErr: "TTRPG_AUTH_SECRET"},
		{name: "hostname needs secret", addr: "table.example:7432", wantErr: "TTRPG_AUTH_SECRET"},
		{name: "unspecified host needs secret", addr: ":7432", wantErr: "TTRPG_AUTH_SECRET"},
		{name: "malformed address is rejected", addr: "127.0.0.1", wantErr: "listen address"},
		{
			name:    "public needs a long enough secret",
			addr:    "0.0.0.0:7432",
			cfg:     ListenSecurityConfig{AuthSecret: strings.Repeat("x", 31)},
			wantErr: "at least 32 bytes",
		},
		{
			name:    "public needs TLS paths",
			addr:    "0.0.0.0:7432",
			cfg:     ListenSecurityConfig{AuthSecret: secret},
			wantErr: "TLS certificate and key",
		},
		{
			name: "public needs an allowed origin",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:  secret,
				TLSCertFile: certFile,
				TLSKeyFile:  keyFile,
			},
			wantErr: "allowed origin",
		},
		{
			name: "public rejects a missing TLS file",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    filepath.Join(t.TempDir(), "missing-cert.pem"),
				TLSKeyFile:     keyFile,
				AllowedOrigins: []string{"https://table.example"},
			},
			wantErr: "valid TLS certificate and key",
		},
		{
			name: "public rejects an unreadable TLS file",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    t.TempDir(),
				TLSKeyFile:     keyFile,
				AllowedOrigins: []string{"https://table.example"},
			},
			wantErr: "valid TLS certificate and key",
		},
		{
			name: "public rejects a mismatched TLS pair",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    certFile,
				TLSKeyFile:     otherKeyFile,
				AllowedOrigins: []string{"https://table.example"},
			},
			wantErr: "valid TLS certificate and key",
		},
		{
			name: "public rejects an empty allowed origin",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    certFile,
				TLSKeyFile:     keyFile,
				AllowedOrigins: []string{""},
			},
			wantErr: "allowed origin",
		},
		{
			name: "public rejects a whitespace allowed origin",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    certFile,
				TLSKeyFile:     keyFile,
				AllowedOrigins: []string{"  \t"},
			},
			wantErr: "allowed origin",
		},
		{
			name: "public rejects a blank among allowed origins",
			addr: "0.0.0.0:7432",
			cfg: ListenSecurityConfig{
				AuthSecret:     secret,
				TLSCertFile:    certFile,
				TLSKeyFile:     keyFile,
				AllowedOrigins: []string{"https://table.example", " "},
			},
			wantErr: "allowed origin",
		},
		{name: "public complete", addr: "0.0.0.0:7432", cfg: publicConfig},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListenSecurity(tt.addr, tt.cfg)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func writeTestKeyPair(t *testing.T, commonName string) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	require.NoError(t, os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER}), 0o600))
	require.NoError(t, os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}), 0o600))
	return certFile, keyFile
}
