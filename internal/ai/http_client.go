package ai

import (
	"context"
	"net"
	"net/http"
	"time"
)

var (
	dialTimeout              = 10 * time.Second
	tlsHandshakeTimeout      = 10 * time.Second
	responseHeaderTimeout    = 30 * time.Second
	idleConnectionTimeout    = 90 * time.Second
	automationRequestTimeout = 45 * time.Second
	gmRequestTimeout         = 10 * time.Minute
)

// NewHTTPClient returns a client suitable for both ordinary provider requests
// and streaming responses. Transport phases are bounded, but Client.Timeout is
// intentionally unset because it would terminate healthy long-running streams.
func NewHTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       idleConnectionTimeout,
		MaxIdleConns:          32,
	}}
}

func withAutomationDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, automationRequestTimeout)
}

func withGMDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, gmRequestTimeout)
}
