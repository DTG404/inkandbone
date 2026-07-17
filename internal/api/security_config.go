package api

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ListenSecurityConfig contains the security settings required to serve on a
// non-loopback address.
type ListenSecurityConfig struct {
	AuthSecret     string
	TLSCertFile    string
	TLSKeyFile     string
	AllowedOrigins []string
}

// ValidateListenSecurity rejects incomplete non-loopback configurations.
func ValidateListenSecurity(addr string, cfg ListenSecurityConfig) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen address: %w", err)
	}

	ip := net.ParseIP(host)
	if strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback()) {
		return nil
	}

	if len(cfg.AuthSecret) < 32 {
		return errors.New("non-loopback listen requires TTRPG_AUTH_SECRET of at least 32 bytes")
	}
	if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
		return errors.New("non-loopback listen requires TLS certificate and key")
	}
	if len(cfg.AllowedOrigins) == 0 {
		return errors.New("non-loopback listen requires at least one allowed origin")
	}
	if _, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil {
		return fmt.Errorf("non-loopback listen requires valid TLS certificate and key: %w", err)
	}

	return nil
}
