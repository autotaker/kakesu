package upstreamtransport

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRoundTripRejectsBeforeResolver(t *testing.T) {
	var resolves, dials int
	transport := newWithDependencies(func(context.Context, string) ([]netip.Addr, error) {
		resolves++
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(context.Context, string, string) (net.Conn, error) {
		dials++
		return nil, errors.New("unexpected dial")
	})

	requests := []*http.Request{
		nil,
		{URL: mustURL(t, "http://api.github.com/x")},
		{URL: mustURL(t, "https://example.invalid/x")},
		{URL: mustURL(t, "https://api.github.com:444/x")},
		{URL: mustURL(t, "https://api.openai.com:444/x")},
		{URL: mustURL(t, "https://github.com:444/x")},
		{URL: mustURL(t, "https://user:pass@api.github.com/x")},
		{URL: &url.URL{Scheme: "https", Opaque: "api.github.com/x"}},
		{URL: mustURL(t, "https://api.github.com/x#fragment")},
		{URL: mustURL(t, "https://api.github.com/x"), Host: "api.openai.com"},
	}
	for _, request := range requests {
		response, err := transport.RoundTrip(request)
		if response != nil || !errors.Is(err, ErrTransport) {
			t.Fatalf("invalid request result: response=%v err=%v", response, err)
		}
	}
	if resolves != 0 || dials != 0 {
		t.Fatalf("invalid request reached network: resolves=%d dials=%d", resolves, dials)
	}
}

func TestValidateRequestAcceptsCanonicalDefaultHTTPSPortForEveryPolicyHost(t *testing.T) {
	for _, host := range []string{githubHost, githubGitHost, openAIHost} {
		for _, authority := range []string{host, host + ":443"} {
			request, err := http.NewRequest(http.MethodGet, "https://"+authority+"/allowed", nil)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := validateRequest(request)
			if !ok || got != host {
				t.Fatalf("authority=%q host=(%q,%v)", authority, got, ok)
			}
		}

		mismatch, err := http.NewRequest(http.MethodGet, "https://"+host+":443/allowed", nil)
		if err != nil {
			t.Fatal(err)
		}
		mismatch.Host = host
		if got, ok := validateRequest(mismatch); ok || got != "" {
			t.Fatalf("Host mismatch accepted for %q: (%q,%v)", host, got, ok)
		}
	}
}

func TestNilZeroAndFormatAreFailClosed(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	var nilTransport *Transport
	if response, roundTripErr := nilTransport.RoundTrip(request); response != nil || !errors.Is(roundTripErr, ErrTransport) {
		t.Fatalf("nil receiver: response=%v err=%v", response, roundTripErr)
	}
	var zeroTransport Transport
	if response, roundTripErr := zeroTransport.RoundTrip(request); response != nil || !errors.Is(roundTripErr, ErrTransport) {
		t.Fatalf("zero receiver: response=%v err=%v", response, roundTripErr)
	}
	nilTransport.CloseIdleConnections()
	zeroTransport.CloseIdleConnections()
	if formatted := fmt.Sprintf("%+v %#v", &zeroTransport, zeroTransport); formatted != "upstreamtransport.Transport upstreamtransport.Transport" {
		t.Fatalf("format leaked state: %q", formatted)
	}
}

func TestInnerTransportSecuritySettings(t *testing.T) {
	roots := x509.NewCertPool()
	transport := newWithRootPool(nil, nil, roots)
	inner := transport.newHTTPTransport("api.github.com", []netip.Addr{netip.MustParseAddr("93.184.216.34")})
	defer inner.CloseIdleConnections()
	if inner.Proxy != nil || inner.DialContext == nil || inner.DialTLSContext != nil ||
		!inner.DisableKeepAlives || !inner.DisableCompression || inner.ForceAttemptHTTP2 ||
		inner.TLSHandshakeTimeout != tlsHandshakeTimeout || inner.ResponseHeaderTimeout != responseHeaderTimeout {
		t.Fatalf("unsafe HTTP transport settings: %#v", inner)
	}
	if inner.Protocols == nil || !inner.Protocols.HTTP1() || inner.Protocols.HTTP2() || inner.Protocols.UnencryptedHTTP2() {
		t.Fatalf("protocols are not HTTP/1-only: %#v", inner.Protocols)
	}
	config := inner.TLSClientConfig
	if config == nil || config.MinVersion != tls.VersionTLS12 || config.ServerName != "api.github.com" ||
		config.RootCAs != roots || config.InsecureSkipVerify || len(config.NextProtos) != 1 || config.NextProtos[0] != "http/1.1" {
		t.Fatalf("unsafe TLS settings: %#v", config)
	}
}

func TestNormalizeResponseClosesResponseWithError(t *testing.T) {
	body := &closeRecorder{Reader: bytes.NewBufferString("secret")}
	response, roundTripErr := normalizeResponse(&http.Response{Body: body}, errors.New("dns=secret ip=10.0.0.1"))
	if response != nil || !errors.Is(roundTripErr, ErrTransport) || !body.closed {
		t.Fatalf("normalization: response=%v err=%v closed=%v", response, roundTripErr, body.closed)
	}
	if strings.Contains(roundTripErr.Error(), "secret") || strings.Contains(roundTripErr.Error(), "10.0.0.1") {
		t.Fatalf("error leaked detail: %q", roundTripErr)
	}
}

func TestNormalizeResponseRejectsHTTP10AndClosesBody(t *testing.T) {
	body := &closeRecorder{Reader: bytes.NewBufferString("legacy")}
	response, roundTripErr := normalizeResponse(&http.Response{
		ProtoMajor: 1,
		ProtoMinor: 0,
		Body:       body,
	}, nil)
	if response != nil || roundTripErr != ErrTransport || !body.closed {
		t.Fatalf("HTTP/1.0 response was not rejected and closed: response=%v err=%v closed=%v", response, roundTripErr, body.closed)
	}
}

func TestUnsafeAnswerBoundaries(t *testing.T) {
	for _, text := range []string{"", "fc00::1", "::ffff:10.0.0.1", "fe80::1%en0"} {
		answer, err := netip.ParseAddr(text)
		if err != nil && text != "" {
			t.Fatalf("parse %q: %v", text, err)
		}
		if _, ok := safeCandidates([]netip.Addr{answer}); ok {
			t.Fatalf("unsafe answer accepted: %q", text)
		}
	}
	answers := make([]netip.Addr, maxDNSAnswers+1)
	for i := range answers {
		answers[i] = netip.MustParseAddr("93.184.216." + strconv.Itoa(34+i%2))
	}
	if _, ok := safeCandidates(answers); ok {
		t.Fatal("answer limit exceeded")
	}
}

func TestSafeCandidatesRejectAllUnsafeAnswersAndDedupe(t *testing.T) {
	unsafe := []netip.Addr{
		netip.MustParseAddr("0.0.0.0"),
		netip.MustParseAddr("127.0.0.1"),
		netip.MustParseAddr("10.0.0.1"),
		netip.MustParseAddr("169.254.1.1"),
		netip.MustParseAddr("224.0.0.1"),
		netip.MustParseAddr("::1"),
		netip.MustParseAddr("fe80::1"),
	}
	for _, answer := range unsafe {
		if candidates, ok := safeCandidates([]netip.Addr{answer}); ok || candidates != nil {
			t.Fatalf("unsafe answer accepted: %v", answer)
		}
	}
	if _, ok := safeCandidates([]netip.Addr{
		netip.MustParseAddr("93.184.216.34"),
		netip.MustParseAddr("10.0.0.1"),
	}); ok {
		t.Fatal("mixed answer accepted")
	}
	if _, ok := safeCandidates([]netip.Addr{{}}); ok {
		t.Fatal("invalid answer accepted")
	}
	candidates, ok := safeCandidates([]netip.Addr{
		netip.MustParseAddr("93.184.216.34"),
		netip.MustParseAddr("2001:db8::1"),
		netip.MustParseAddr("93.184.216.34"),
	})
	if !ok || len(candidates) != 2 || candidates[0].String() != "93.184.216.34" || candidates[1].String() != "2001:db8::1" {
		t.Fatalf("unexpected candidates: %#v", candidates)
	}
}

func TestRoundTripTLSHostnameAndHTTP11(t *testing.T) {
	certificate, roots := testCertificate(t, "api.github.com")
	server, client := net.Pipe()
	defer server.Close()
	var serverName string
	var serverErr error
	go func() {
		defer server.Close()
		config := &tls.Config{
			Certificates: []tls.Certificate{certificate},
			GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				serverName = hello.ServerName
				return nil, nil
			},
		}
		connection := tls.Server(server, config)
		if err := connection.Handshake(); err != nil {
			serverErr = err
			return
		}
		request, err := http.ReadRequest(bufio.NewReader(connection))
		if err != nil {
			serverErr = err
			return
		}
		_ = request.Body.Close()
		response := &http.Response{
			StatusCode: http.StatusOK,
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
		}
		response.Header.Set("Content-Length", "2")
		serverErr = response.Write(connection)
		_ = connection.Close()
	}()

	var addresses []string
	transport := newWithRootPool(func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(_ context.Context, network, address string) (net.Conn, error) {
		addresses = append(addresses, network+" "+address)
		return client, nil
	}, roots)
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, roundTripErr := transport.RoundTrip(request)
	if roundTripErr != nil {
		t.Fatalf("RoundTrip: %v", roundTripErr)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || string(data) != "ok" {
		t.Fatalf("response body: data=%q read=%v close=%v", data, readErr, closeErr)
	}
	if serverErr != nil || serverName != "api.github.com" {
		t.Fatalf("TLS server: err=%v sni=%q", serverErr, serverName)
	}
	if len(addresses) != 1 || addresses[0] != "tcp 93.184.216.34:443" {
		t.Fatalf("dial addresses: %v", addresses)
	}
	if response.ProtoMajor != 1 || response.ProtoMinor != 1 {
		t.Fatalf("unexpected protocol: %s", response.Proto)
	}
}

func TestRoundTripGitHubGitHostUsesPinnedTLSAndOneDial(t *testing.T) {
	certificate, roots := testCertificate(t, githubGitHost)
	server, client := net.Pipe()
	defer server.Close()
	var serverName string
	var serverErr error
	go func() {
		defer server.Close()
		connection := tls.Server(server, &tls.Config{
			Certificates: []tls.Certificate{certificate},
			GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
				serverName = hello.ServerName
				return nil, nil
			},
		})
		if err := connection.Handshake(); err != nil {
			serverErr = err
			return
		}
		request, err := http.ReadRequest(bufio.NewReader(connection))
		if err != nil {
			serverErr = err
			return
		}
		if request.Host != githubGitHost+":443" || request.URL.Path != "/acme/widget.git/git-upload-pack" || request.ProtoMajor != 1 || request.ProtoMinor != 1 {
			serverErr = errors.New("unexpected request")
			return
		}
		_ = request.Body.Close()
		response := &http.Response{StatusCode: http.StatusOK, ProtoMajor: 1, ProtoMinor: 1, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("git"))}
		response.Header.Set("Content-Length", "3")
		serverErr = response.Write(connection)
		_ = connection.Close()
	}()

	var resolved []string
	var dialed []string
	transport := newWithRootPool(func(_ context.Context, host string) ([]netip.Addr, error) {
		resolved = append(resolved, host)
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(_ context.Context, network, address string) (net.Conn, error) {
		dialed = append(dialed, network+" "+address)
		return client, nil
	}, roots)
	request, err := http.NewRequest(http.MethodPost, "https://github.com:443/acme/widget.git/git-upload-pack", strings.NewReader("0000"))
	if err != nil {
		t.Fatal(err)
	}
	response, roundTripErr := transport.RoundTrip(request)
	if roundTripErr != nil {
		t.Fatalf("RoundTrip: %v", roundTripErr)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || string(data) != "git" || response.ProtoMajor != 1 || response.ProtoMinor != 1 {
		t.Fatalf("response data=%q read=%v close=%v proto=%s", data, readErr, closeErr, response.Proto)
	}
	if serverErr != nil || serverName != githubGitHost || fmt.Sprint(resolved) != "[github.com]" || fmt.Sprint(dialed) != "[tcp 93.184.216.34:443]" {
		t.Fatalf("server=%v sni=%q resolved=%v dialed=%v", serverErr, serverName, resolved, dialed)
	}
}

func TestGitHubGitHostNearMatchesRejectBeforeResolver(t *testing.T) {
	resolves := 0
	transport := newWithDependencies(func(context.Context, string) ([]netip.Addr, error) {
		resolves++
		return nil, errors.New("must not resolve")
	}, func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("must not dial") })
	for _, rawURL := range []string{
		"https://www.github.com/acme/widget.git/git-upload-pack",
		"https://GITHUB.COM/acme/widget.git/git-upload-pack",
		"https://github.com./acme/widget.git/git-upload-pack",
		"https://github.com:444/acme/widget.git/git-upload-pack",
		"https://user@github.com/acme/widget.git/git-upload-pack",
		"http://github.com/acme/widget.git/git-upload-pack",
	} {
		request, err := http.NewRequest(http.MethodPost, rawURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		if response, err := transport.RoundTrip(request); response != nil || !errors.Is(err, ErrTransport) {
			t.Fatalf("url=%q response=%v err=%v", rawURL, response, err)
		}
	}
	if resolves != 0 {
		t.Fatalf("near matches resolved=%d", resolves)
	}
}

func TestRoundTripFallsBackOnlyOnDialFailure(t *testing.T) {
	certificate, roots := testCertificate(t, "api.openai.com")
	server, client := net.Pipe()
	defer server.Close()
	go serveOne(t, server, certificate)

	var addresses []string
	transport := newWithRootPool(func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("93.184.216.35"),
		}, nil
	}, func(_ context.Context, _ string, address string) (net.Conn, error) {
		addresses = append(addresses, address)
		if len(addresses) == 1 {
			return nil, errors.New("connect failed")
		}
		return client, nil
	}, roots)
	request, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, roundTripErr := transport.RoundTrip(request)
	if roundTripErr != nil {
		t.Fatalf("RoundTrip: %v", roundTripErr)
	}
	_ = response.Body.Close()
	if len(addresses) != 2 || addresses[0] != "93.184.216.34:443" || addresses[1] != "93.184.216.35:443" {
		t.Fatalf("fallback addresses: %v", addresses)
	}
}

func TestTLSFailureDoesNotFallback(t *testing.T) {
	cases := []struct {
		name       string
		certHost   string
		minVersion uint16
	}{
		{name: "wrong-host", certHost: "wrong.example", minVersion: tls.VersionTLS12},
		{name: "tls11", certHost: "api.github.com", minVersion: tls.VersionTLS11},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			certificate, roots := testCertificate(t, testCase.certHost)
			server, client := net.Pipe()
			defer server.Close()
			go serveTLS(server, &tls.Config{
				Certificates: []tls.Certificate{certificate},
				MinVersion:   testCase.minVersion,
				MaxVersion:   testCase.minVersion,
			}, false)
			var addresses []string
			transport := newWithRootPool(func(context.Context, string) ([]netip.Addr, error) {
				return []netip.Addr{
					netip.MustParseAddr("93.184.216.34"),
					netip.MustParseAddr("93.184.216.35"),
				}, nil
			}, func(_ context.Context, _ string, address string) (net.Conn, error) {
				addresses = append(addresses, address)
				return client, nil
			}, roots)
			request, err := http.NewRequest(http.MethodGet, "https://api.github.com/v1", nil)
			if err != nil {
				t.Fatal(err)
			}
			response, roundTripErr := transport.RoundTrip(request)
			if response != nil || !errors.Is(roundTripErr, ErrTransport) {
				t.Fatalf("TLS failure result: response=%v err=%v", response, roundTripErr)
			}
			if len(addresses) != 1 {
				t.Fatalf("TLS failure retried dial: %v", addresses)
			}
		})
	}
}

func TestHTTPFailureDoesNotRedialAfterRequestWrite(t *testing.T) {
	certificate, roots := testCertificate(t, "api.github.com")
	server, client := net.Pipe()
	defer server.Close()
	go serveTLS(server, &tls.Config{Certificates: []tls.Certificate{certificate}}, false)
	var dials int
	transport := newWithRootPool(func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(context.Context, string, string) (net.Conn, error) {
		dials++
		return client, nil
	}, roots)
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, roundTripErr := transport.RoundTrip(request)
	if response != nil || !errors.Is(roundTripErr, ErrTransport) || dials != 1 {
		t.Fatalf("HTTP failure result: response=%v err=%v dials=%d", response, roundTripErr, dials)
	}
}

func TestRoundTripCancellationReturnsFixedError(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	transport := newWithDependencies(func(context.Context, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(context.Context, string, string) (net.Conn, error) {
		return client, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, roundTripErr := transport.RoundTrip(request)
	if response != nil || !errors.Is(roundTripErr, ErrTransport) {
		t.Fatalf("cancel result: response=%v err=%v", response, roundTripErr)
	}
}

func TestRoundTripDoesNotConsultProxyEnvironment(t *testing.T) {
	oldHTTP, oldHTTPS, oldNoProxy := os.Getenv("HTTP_PROXY"), os.Getenv("HTTPS_PROXY"), os.Getenv("NO_PROXY")
	defer func() {
		_ = os.Setenv("HTTP_PROXY", oldHTTP)
		_ = os.Setenv("HTTPS_PROXY", oldHTTPS)
		_ = os.Setenv("NO_PROXY", oldNoProxy)
	}()
	_ = os.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	_ = os.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	_ = os.Setenv("NO_PROXY", "")
	var dialed bool
	var resolves int
	transport := newWithDependencies(func(context.Context, string) ([]netip.Addr, error) {
		resolves++
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	}, func(context.Context, string, string) (net.Conn, error) {
		dialed = true
		return nil, errors.New("expected test failure")
	})
	request, err := http.NewRequest(http.MethodGet, "https://api.github.com/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = transport.RoundTrip(request)
	if !dialed || resolves != 1 {
		t.Fatalf("proxy environment or resolver count: dialed=%v resolves=%d", dialed, resolves)
	}
}

func TestFixedErrorDoesNotLeakRequestOrDependencyDetails(t *testing.T) {
	transport := newWithDependencies(func(context.Context, string) ([]netip.Addr, error) {
		return nil, errors.New("resolver secret api.github.com 10.0.0.1")
	}, func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("dial secret")
	})
	request, err := http.NewRequest(http.MethodPost, "https://api.github.com/v1", strings.NewReader("body-secret"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer authorization-secret")
	_, roundTripErr := transport.RoundTrip(request)
	if roundTripErr != ErrTransport {
		t.Fatalf("unexpected fixed error: %v", roundTripErr)
	}
	formatted := fmt.Sprintf("%v %+v %#v", roundTripErr, roundTripErr, roundTripErr)
	for _, secret := range []string{"body-secret", "authorization-secret", "resolver secret", "api.github.com", "10.0.0.1"} {
		if strings.Contains(formatted, secret) {
			t.Fatalf("error leaked %q: %q", secret, formatted)
		}
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func testCertificate(t *testing.T, hostname string) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: hostname},
		DNSNames:              []string{hostname},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := tls.X509KeyPair(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	return pair, pool
}

func serveOne(t *testing.T, server net.Conn, certificate tls.Certificate) {
	t.Helper()
	serveTLS(server, &tls.Config{Certificates: []tls.Certificate{certificate}}, true)
}

func serveTLS(server net.Conn, config *tls.Config, writeResponse bool) {
	connection := tls.Server(server, config)
	defer connection.Close()
	if err := connection.Handshake(); err != nil {
		return
	}
	request, err := http.ReadRequest(bufio.NewReader(connection))
	if err != nil {
		return
	}
	_ = request.Body.Close()
	if !writeResponse {
		return
	}
	response := &http.Response{
		StatusCode: http.StatusOK,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
	}
	response.Header.Set("Content-Length", "2")
	_ = response.Write(connection)
}

type closeRecorder struct {
	io.Reader
	closed bool
}

func (c *closeRecorder) Close() error {
	c.closed = true
	return nil
}
