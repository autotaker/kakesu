// Package upstreamtransport provides the broker's deliberately small HTTPS
// boundary for the fixed provider hosts.  It resolves an allowlisted hostname
// once, validates every answer, and then connects to an address literal while
// retaining the original hostname for TLS verification.
package upstreamtransport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"
)

const (
	githubHost    = "api.github.com"
	githubGitHost = "github.com"
	openAIHost    = "api.openai.com"

	// These limits are fixed production values.  They bound work performed by
	// a single request without making them part of the public API.
	maxDNSAnswers         = 32
	connectTimeout        = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 10 * time.Second
)

// Error is a fixed, comparable error.  No URL, address, request secret, or
// operating-system detail is included in an error returned by this package.
type Error string

func (e Error) Error() string { return string(e) }

// ErrTransport is returned for every invalid request and every failed
// resolution, connection, TLS, or HTTP exchange.
const ErrTransport Error = "upstream-transport-failed"

// resolverFunc and dialerFunc are package-private seams used by hermetic
// tests.  Production construction below supplies the system resolver and
// net.Dialer; callers cannot replace either through the exported API.
type resolverFunc func(context.Context, string) ([]netip.Addr, error)
type dialerFunc func(context.Context, string, string) (net.Conn, error)

// Transport implements http.RoundTripper.  A fresh net/http.Transport is
// created for each request, so no connection pool, redirect policy, or
// caller-controlled transport state is retained between requests.
type Transport struct {
	resolve resolverFunc
	dial    dialerFunc
	// roots is nil for production, which instructs crypto/tls to use the
	// operating system roots.  A package-private pool is used only by runtime
	// certificate fixtures in hermetic tests.
	roots *x509.CertPool
}

var _ http.RoundTripper = (*Transport)(nil)

// New returns the fixed production transport.  It has no error path: all
// malformed or unsupported requests fail closed from RoundTrip.
func New() *Transport {
	resolver := func(ctx context.Context, host string) ([]netip.Addr, error) {
		return (&net.Resolver{}).LookupNetIP(ctx, "ip", host)
	}
	dialer := (&net.Dialer{Timeout: connectTimeout}).DialContext
	return newWithDependencies(resolver, dialer)
}

// newWithDependencies constructs a transport with package-private seams for
// deterministic tests.  A nil seam is deliberately retained as nil; the
// request path treats it as a failure rather than panicking.
func newWithDependencies(resolve resolverFunc, dial dialerFunc) *Transport {
	return newWithRootPool(resolve, dial, nil)
}

// newWithRootPool is a package-private test seam.  Production New always
// passes nil, preserving the system trust store boundary.
func newWithRootPool(resolve resolverFunc, dial dialerFunc, roots *x509.CertPool) *Transport {
	return &Transport{resolve: resolve, dial: dial, roots: roots}
}

// RoundTrip performs one allowlisted HTTPS exchange.  It returns ownership of
// a response body only on complete success; every response/error combination
// on failure is closed before the fixed error is returned.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil || t.resolve == nil || t.dial == nil || req == nil {
		return nil, ErrTransport
	}
	host, ok := validateRequest(req)
	if !ok {
		return nil, ErrTransport
	}
	if req.Context().Err() != nil {
		return nil, ErrTransport
	}

	answers, resolveErr := t.resolve(req.Context(), host)
	if resolveErr != nil {
		return nil, ErrTransport
	}
	candidates, ok := safeCandidates(answers)
	if !ok {
		return nil, ErrTransport
	}
	if req.Context().Err() != nil {
		return nil, ErrTransport
	}

	inner := t.newHTTPTransport(host, candidates)

	response, roundTripErr := inner.RoundTrip(req)
	return normalizeResponse(response, roundTripErr)
}

// Format deliberately avoids exposing the resolver, dialer, root pool, or
// any other request-bound state through ordinary formatting verbs.
func (t Transport) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "upstreamtransport.Transport")
}

func normalizeResponse(response *http.Response, roundTripErr error) (*http.Response, error) {
	if roundTripErr != nil || response == nil || response.Body == nil || response.ProtoMajor != 1 || response.ProtoMinor != 1 {
		closeResponse(response)
		return nil, ErrTransport
	}
	return response, nil
}

// CloseIdleConnections is safe on nil and zero receivers.  Every request owns
// a short-lived inner transport with keep-alive disabled, so there are no idle
// connections retained by this wrapper.
func (t *Transport) CloseIdleConnections() {
}

func (t *Transport) newHTTPTransport(hostname string, candidates []netip.Addr) *http.Transport {
	// A custom TLSClientConfig disables net/http's implicit HTTP/2 setup when
	// ForceAttemptHTTP2 is false.  NextProtos makes the HTTP/1.1-only boundary
	// explicit to the TLS peer as well.
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         hostname,
		RootCAs:            t.roots, // nil uses the host's system root pool
		InsecureSkipVerify: false,
		NextProtos:         []string{"http/1.1"},
	}

	var dialMu sync.Mutex
	attempted := false
	dialContext := func(ctx context.Context, network, _ string) (net.Conn, error) {
		// net/http may otherwise retry an idempotent request after an early
		// connection error.  Allow exactly one invocation of this closure; the
		// invocation itself performs the bounded, dial-only candidate fallback.
		dialMu.Lock()
		if attempted {
			dialMu.Unlock()
			return nil, errDialAlreadyAttempted
		}
		attempted = true
		dialMu.Unlock()
		for _, candidate := range candidates {
			if ctx.Err() != nil {
				return nil, errDialFailed
			}
			address := net.JoinHostPort(candidate.String(), "443")
			conn, err := t.dial(ctx, network, address)
			if err == nil && conn != nil {
				return conn, nil
			}
			if conn != nil {
				_ = conn.Close()
			}
		}
		return nil, errDialFailed
	}

	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	return &http.Transport{
		Proxy:                 nil,
		DialContext:           dialContext,
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		DisableCompression:    true,
		DisableKeepAlives:     true,
		ForceAttemptHTTP2:     false,
		Protocols:             protocols,
		MaxIdleConns:          0,
		IdleConnTimeout:       0,
		ExpectContinueTimeout: 0,
	}
}

var (
	errDialFailed           = errors.New("dial-failed")
	errDialAlreadyAttempted = errors.New("dial-already-attempted")
)

func validateRequest(req *http.Request) (string, bool) {
	if req == nil || req.URL == nil {
		return "", false
	}
	u := req.URL
	if u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Fragment != "" || u.RawFragment != "" {
		return "", false
	}
	hostname, ok := allowedHostname(u.Host)
	if !ok {
		return "", false
	}
	if req.Host != "" && req.Host != u.Host {
		return "", false
	}
	return hostname, true
}

// allowedHostname accepts only the canonical authority or its explicit
// default HTTPS port. The returned value is always port-free so DNS and TLS
// SNI/certificate verification use the original allowlisted hostname.
func allowedHostname(authority string) (string, bool) {
	switch authority {
	case githubHost, githubHost + ":443":
		return githubHost, true
	case githubGitHost, githubGitHost + ":443":
		return githubGitHost, true
	case openAIHost, openAIHost + ":443":
		return openAIHost, true
	default:
		return "", false
	}
}

func safeCandidates(answers []netip.Addr) ([]netip.Addr, bool) {
	if len(answers) == 0 || len(answers) > maxDNSAnswers {
		return nil, false
	}
	seen := make(map[netip.Addr]struct{}, len(answers))
	candidates := make([]netip.Addr, 0, len(answers))
	for _, answer := range answers {
		if !answer.IsValid() || answer.Zone() != "" {
			return nil, false
		}
		// Unmap IPv4-in-IPv6 answers before policy checks and dialing.  This
		// gives one canonical literal and makes duplicate answers harmless.
		answer = answer.Unmap()
		if answer.IsUnspecified() || answer.IsLoopback() || answer.IsPrivate() ||
			answer.IsLinkLocalUnicast() || answer.IsLinkLocalMulticast() ||
			answer.IsMulticast() || !answer.IsGlobalUnicast() {
			return nil, false
		}
		if _, duplicate := seen[answer]; duplicate {
			continue
		}
		seen[answer] = struct{}{}
		candidates = append(candidates, answer)
	}
	return candidates, len(candidates) > 0
}

func closeResponse(response *http.Response) {
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
}
