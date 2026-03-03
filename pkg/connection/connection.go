// Package connection provides credential resolution for kubectl-debug-queries.
//
// Credentials flow through three tiers (highest priority first):
//  1. SSE HTTP headers (per-session)
//  2. CLI defaults (kubeconfig via client-go)
//  3. Auto-discovered from kubeconfig
package connection

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type contextKey string

const (
	restConfigKey contextKey = "rest_config"
)

// --- Transport helpers ---

type bearerTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(r)
}

// InsecureTransport returns a transport that skips TLS verification.
func InsecureTransport() http.RoundTripper {
	return &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

// NewBearerTokenTransport creates a transport that adds a bearer token
// to every request, with InsecureSkipVerify for self-signed certs.
func NewBearerTokenTransport(token string) http.RoundTripper {
	return &bearerTokenTransport{
		token: token,
		base:  InsecureTransport(),
	}
}

// --- Context accessors ---

// WithRESTConfig adds a rest.Config to the context.
func WithRESTConfig(ctx context.Context, cfg *rest.Config) context.Context {
	return context.WithValue(ctx, restConfigKey, cfg)
}

// GetRESTConfig retrieves the rest.Config from the context.
func GetRESTConfig(ctx context.Context) (*rest.Config, bool) {
	if ctx == nil {
		return nil, false
	}
	cfg, ok := ctx.Value(restConfigKey).(*rest.Config)
	return cfg, ok
}

// WithCredsFromHeaders extracts credentials from HTTP headers and builds
// a rest.Config, adding it to the context.
//
// Supported headers:
//   - Authorization: Bearer <token>
//   - X-Kubernetes-Server: <url>
func WithCredsFromHeaders(ctx context.Context, headers http.Header) context.Context {
	if headers == nil {
		return ctx
	}

	var token, server string

	if auth := headers.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if s := headers.Get("X-Kubernetes-Server"); s != "" {
		server = s
	}

	if token == "" && server == "" {
		return ctx
	}

	cfg := &rest.Config{
		BearerToken: token,
		Host:        server,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
	klog.V(2).Infof("[auth] Built rest.Config from SSE headers (server=%s, token-len=%d)", server, len(token))
	return WithRESTConfig(ctx, cfg)
}

// --- Package-level defaults (set from CLI flags at startup) ---

var defaultRESTConfig *rest.Config

// SetDefaultRESTConfig sets the default rest.Config resolved from kubeconfig.
func SetDefaultRESTConfig(cfg *rest.Config) { defaultRESTConfig = cfg }

// ResolveRESTConfig returns a rest.Config using the 3-tier precedence:
//
//	context (SSE headers) > CLI defaults > nil
func ResolveRESTConfig(ctx context.Context) *rest.Config {
	if cfg, ok := GetRESTConfig(ctx); ok && cfg != nil {
		return cfg
	}
	return defaultRESTConfig
}
