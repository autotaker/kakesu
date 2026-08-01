package connectsession

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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerexchange"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerhttp"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/proxyca"
)

type countingAuthority struct {
	mu    sync.Mutex
	cert  *proxyca.Authority
	calls int
	err   error
}

func (a *countingAuthority) Issue(host string) (tls.Certificate, error) {
	a.mu.Lock()
	a.calls++
	err := a.err
	ca := a.cert
	a.mu.Unlock()
	if err != nil {
		return tls.Certificate{}, err
	}
	return ca.Issue(host)
}
func (a *countingAuthority) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

type testResolver struct {
	mu      sync.Mutex
	calls   int
	ctx     context.Context
	subject egresstransaction.Subject
}

func (r *testResolver) Resolve(ctx context.Context) (egresstransaction.Subject, error) {
	r.mu.Lock()
	r.calls++
	r.ctx = ctx
	subject := r.subject
	r.mu.Unlock()
	return subject, nil
}

type testExchange struct {
	mu    sync.Mutex
	calls int
}

func (e *testExchange) Do(egresstransaction.Subject, egresstransaction.Request) (brokerexchange.Response, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	return brokerexchange.Response{StatusCode: http.StatusOK, ContentType: "application/json", Body: []byte(`{"ok":true}`)}, nil
}
func newTestAuthority(t *testing.T) (*countingAuthority, *x509.Certificate) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	must(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	must(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	ca, err := proxyca.New(proxyca.Rules{CACertificatePEM: certPEM, CAPrivateKeyPEM: keyPEM, Clock: proxyca.ClockFunc(func() time.Time { return now })})
	must(t, err)
	cert, err := x509.ParseCertificate(der)
	must(t, err)
	return &countingAuthority{cert: ca}, cert
}
func TestSessionConnectTLSAndRealHandler(t *testing.T) {
	authority, caCert := newTestAuthority(t)
	resolver := &testResolver{subject: egresstransaction.Subject{AgentInstanceID: "agent"}}
	exchange := &testExchange{}
	handler, err := brokerhttp.New(brokerhttp.Rules{Exchange: exchange, Resolver: resolver, MaxBodyBytes: 4096})
	must(t, err)
	session, err := New(Rules{Authority: authority, Handler: handler})
	must(t, err)
	for _, host := range []string{githubHost, openAIHost} {
		t.Run(host, func(t *testing.T) {
			server, client := net.Pipe()
			ctx := context.WithValue(context.Background(), "caller", "trusted")
			serveDone := make(chan error, 1)
			go func() { serveDone <- session.Serve(ctx, server) }()
			if _, err := io.WriteString(client, "CONNECT "+host+":443 HTTP/1.1\r\nHost: "+host+":443\r\nUser-Agent: test\r\nProxy-Connection: Keep-Alive\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			connectResponse := readHeader(t, client)
			if connectResponse != connectEstablished {
				t.Fatalf("CONNECT response=%q", connectResponse)
			}
			roots := x509.NewCertPool()
			roots.AddCert(caCert)
			tlsClient := tls.Client(client, &tls.Config{RootCAs: roots, ServerName: host, NextProtos: []string{"http/1.1"}, MinVersion: tls.VersionTLS12})
			if err := tlsClient.Handshake(); err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(tlsClient, "GET /v1/responses HTTP/1.1\r\nHost: "+host+"\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			response, err := http.ReadResponse(bufio.NewReader(tlsClient), nil)
			must(t, err)
			body, err := io.ReadAll(response.Body)
			must(t, err)
			response.Body.Close()
			if response.StatusCode != http.StatusOK || string(body) != `{"ok":true}` {
				t.Fatalf("response status=%d body=%q", response.StatusCode, body)
			}
			if err := <-serveDone; err != nil {
				t.Fatal(err)
			}
		})
	}
	if authority.count() != 2 || exchange.calls != 2 || resolver.calls != 2 {
		t.Fatalf("calls authority=%d exchange=%d resolver=%d", authority.count(), exchange.calls, resolver.calls)
	}
	resolver.mu.Lock()
	if resolver.ctx == nil || resolver.ctx.Value("caller") != "trusted" {
		t.Fatalf("caller context was not inherited: %v", resolver.ctx)
	}
	resolver.mu.Unlock()
}
func TestConcurrentSessionsKeepContextAndHostIsolated(t *testing.T) {
	authority, caCert := newTestAuthority(t)
	var mu sync.Mutex
	seen := map[string]string{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		marker, _ := r.Context().Value("marker").(string)
		mu.Lock()
		seen[r.Host] = marker
		mu.Unlock()
		_, _ = io.WriteString(w, marker)
	})
	session, err := New(Rules{Authority: authority, Handler: handler})
	must(t, err)
	results := make(chan error, 2)
	for _, tc := range []struct{ host, marker string }{{githubHost, "gh"}, {openAIHost, "oa"}} {
		tc := tc
		go func() {
			results <- runSession(session, caCert, tc.host, tc.marker, context.WithValue(context.Background(), "marker", tc.marker))
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if seen[githubHost] != "gh" || seen[openAIHost] != "oa" || authority.count() != 2 {
		t.Fatalf("isolated state=%v authority calls=%d", seen, authority.count())
	}
}
func TestHTTPPhaseDeadlinePropagation(t *testing.T) {
	for _, tc := range []struct {
		name string
		dur  time.Duration
	}{{"phase cap", 0}, {"caller earlier", 2 * time.Second}} {
		t.Run(tc.name, func(t *testing.T) {
			authority, caCert := newTestAuthority(t)
			seen := make(chan [2]time.Time, 1)
			session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				deadline, _ := r.Context().Deadline()
				seen <- [2]time.Time{time.Now(), deadline}
				_, _ = io.WriteString(w, "ok")
			})})
			must(t, err)
			ctx, caller := context.Background(), time.Time{}
			if tc.dur > 0 {
				caller = time.Now().Add(tc.dur)
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, caller)
				defer cancel()
			}
			if err := runSession(session, caCert, githubHost, "ok", ctx); err != nil {
				t.Fatal(err)
			}
			observed := <-seen
			if observed[1].IsZero() {
				t.Fatal("handler context has no deadline")
			}
			if tc.dur > 0 {
				if delta := observed[1].Sub(caller); delta < -500*time.Millisecond || delta > 500*time.Millisecond {
					t.Fatalf("caller deadline=%v observed=%v", caller, observed[1])
				}
			} else if observed[1].After(observed[0].Add(phaseTimeout + time.Second)) {
				t.Fatalf("phase deadline=%v observed at=%v", observed[1], observed[0])
			}
		})
	}
}
func TestResponseFailuresCloseWithoutInnerResponse(t *testing.T) {
	authority, caCert := newTestAuthority(t)
	for name, handler := range map[string]http.Handler{
		"overflow": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), maxResponseBody+1))
		}),
		"header name": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header()["Bad\r\nX"] = []string{"value"}
			_, _ = w.Write([]byte("body"))
		}),
		"header case": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header()["content-length"] = []string{"0"} }),
		"header aggregate": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for i := 0; i < 2048; i++ {
				w.Header()[fmt.Sprintf("X-%04d", i)] = nil
			}
		}),
	} {
		t.Run(name, func(t *testing.T) {
			session, err := New(Rules{Authority: authority, Handler: handler})
			must(t, err)
			server, client := net.Pipe()
			done := make(chan error, 1)
			go func() { done <- session.Serve(context.Background(), server) }()
			go io.WriteString(client, "CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\n")
			if readHeader(t, client) != connectEstablished {
				t.Fatal("CONNECT denied")
			}
			tlsClient := tls.Client(client, &tls.Config{RootCAs: trust(caCert), ServerName: githubHost, NextProtos: []string{"http/1.1"}})
			if err := tlsClient.Handshake(); err != nil {
				t.Fatal(err)
			}
			_, _ = io.WriteString(tlsClient, "GET / HTTP/1.1\r\nHost: api.github.com\r\n\r\n")
			_ = client.SetReadDeadline(time.Now().Add(time.Second))
			var one [1]byte
			if n, _ := tlsClient.Read(one[:]); n != 0 {
				t.Fatal("inner response was written")
			}
			if err := <-done; !errors.Is(err, ErrSession) {
				t.Fatalf("serve error=%v", err)
			}
		})
	}
}
func runSession(session *Session, caCert *x509.Certificate, host, marker string, ctx context.Context) error {
	server, client := net.Pipe()
	defer client.Close()
	done := make(chan error, 1)
	go func() { done <- session.Serve(ctx, server) }()
	go io.WriteString(client, "CONNECT "+host+":443 HTTP/1.1\r\nHost: "+host+":443\r\n\r\n")
	if got, err := readHeaderNoFatal(client); err != nil {
		return err
	} else if got != connectEstablished {
		return errors.New("CONNECT denied")
	}
	tlsClient := tls.Client(client, &tls.Config{RootCAs: trust(caCert), ServerName: host, NextProtos: []string{"http/1.1"}})
	if err := tlsClient.Handshake(); err != nil {
		return err
	}
	_, _ = io.WriteString(tlsClient, "GET / HTTP/1.1\r\nHost: "+host+"\r\n\r\n")
	response, err := http.ReadResponse(bufio.NewReader(tlsClient), nil)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		return err
	}
	if string(body) != marker {
		return errors.New("response marker mismatch")
	}
	return <-done
}
func TestHandlerPanicAndHTTPContextCancelClose(t *testing.T) {
	started := make(chan struct{})
	for name, handler := range map[string]http.Handler{
		"panic":  http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("handler detail") }),
		"cancel": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }),
	} {
		t.Run(name, func(t *testing.T) {
			authority, caCert := newTestAuthority(t)
			session, err := New(Rules{Authority: authority, Handler: handler})
			must(t, err)
			server, client := net.Pipe()
			done := make(chan error, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { done <- session.Serve(ctx, server) }()
			go io.WriteString(client, "CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\n")
			if readHeader(t, client) != connectEstablished {
				t.Fatal("CONNECT denied")
			}
			tlsClient := tls.Client(client, &tls.Config{RootCAs: trust(caCert), ServerName: githubHost, NextProtos: []string{"http/1.1"}})
			if err := tlsClient.Handshake(); err != nil {
				t.Fatal(err)
			}
			go io.WriteString(tlsClient, "GET / HTTP/1.1\r\nHost: api.github.com\r\n\r\n")
			if name == "cancel" {
				<-started
				cancel()
			}
			if err := <-done; !errors.Is(err, ErrSession) {
				t.Fatalf("serve error=%v", err)
			}
		})
	}
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func TestTLSVersionAndALPNFailures(t *testing.T) {
	authority, caCert := newTestAuthority(t)
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
	must(t, err)
	for name, config := range map[string]*tls.Config{
		"tls 1.1":   {RootCAs: trust(caCert), ServerName: githubHost, MinVersion: tls.VersionTLS10, MaxVersion: tls.VersionTLS11, NextProtos: []string{"http/1.1"}},
		"no alpn":   {RootCAs: trust(caCert), ServerName: githubHost},
		"h2 only":   {RootCAs: trust(caCert), ServerName: githubHost, NextProtos: []string{"h2"}},
		"wrong sni": {RootCAs: trust(caCert), ServerName: openAIHost, NextProtos: []string{"http/1.1"}},
	} {
		t.Run(name, func(t *testing.T) {
			server, client := net.Pipe()
			done := make(chan error, 1)
			go func() { done <- session.Serve(context.Background(), server) }()
			go io.WriteString(client, "CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\n")
			if got := readHeader(t, client); got != connectEstablished {
				t.Fatalf("response=%q", got)
			}
			tlsClient := tls.Client(client, config)
			handshakeErr := tlsClient.Handshake()
			if name == "tls 1.1" && handshakeErr == nil {
				t.Fatal("TLS 1.1 unexpectedly succeeded")
			}
			if handshakeErr == nil {
				_, _ = io.WriteString(tlsClient, "GET / HTTP/1.1\r\nHost: api.github.com\r\n\r\n")
			}
			_ = client.SetReadDeadline(time.Now().Add(time.Second))
			var one [1]byte
			if n, _ := client.Read(one[:]); n != 0 {
				t.Fatal("TLS failure produced HTTP bytes")
			}
			if err := <-done; !errors.Is(err, ErrSession) {
				t.Fatalf("serve error=%v", err)
			}
		})
	}
}
func trust(c *x509.Certificate) *x509.CertPool { p := x509.NewCertPool(); p.AddCert(c); return p }
func TestConnectPhaseStallTimesOut(t *testing.T) {
	authority, _ := newTestAuthority(t)
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})})
	must(t, err)
	server, _ := net.Pipe()
	done := make(chan error, 1)
	started := time.Now()
	go func() { done <- session.Serve(context.Background(), server) }()
	select {
	case err := <-done:
		if !errors.Is(err, ErrSession) {
			t.Fatalf("serve error=%v", err)
		}
		elapsed := time.Since(started)
		if elapsed < phaseTimeout-200*time.Millisecond || elapsed > phaseTimeout+2*time.Second {
			t.Fatalf("stall elapsed=%v", elapsed)
		}
	case <-time.After(phaseTimeout + 2*time.Second):
		t.Fatal("phase stall did not stop")
	}
}
func TestValidationFormatAndIssueFailure(t *testing.T) {
	authority, _ := newTestAuthority(t)
	var nilAuthority *countingAuthority
	for name, rules := range map[string]Rules{
		"zero": {}, "typed nil authority": {Authority: nilAuthority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})},
		"typed nil handler": {Authority: authority, Handler: (*nilHandler)(nil)},
	} {
		t.Run(name, func(t *testing.T) {
			if session, err := New(rules); session != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("session=%p error=%v", session, err)
			}
		})
	}
	var zero Session
	if got := fmt.Sprintf("%+v", zero); got != "connectsession.Session" {
		t.Fatalf("format=%q", got)
	}
	if err := zero.Serve(context.Background(), nil); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("zero Serve error=%v", err)
	}
	authority.err = errors.New("issuer detail")
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})})
	must(t, err)
	server, client := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- session.Serve(context.Background(), server) }()
	go io.WriteString(client, "CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\n")
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	var one [1]byte
	if n, _ := client.Read(one[:]); n != 0 {
		t.Fatal("issuer failure produced a response")
	}
	if err := <-done; !errors.Is(err, ErrSession) || authority.count() != 1 {
		t.Fatalf("serve=%v calls=%d", err, authority.count())
	}
}

type nilHandler struct{}

func (*nilHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
func TestStrictFramingAndHeaderFailures(t *testing.T) {
	authority, _ := newTestAuthority(t)
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
	must(t, err)
	cases := []string{
		"CONNECT api.github.com:80 HTTP/1.1\r\nHost: api.github.com:80\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nX-Other: no\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nHost: api.github.com:443\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nProxy-Connection: close\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\nGET / HTTP/1.1",
		"GET api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.0\r\nHost: api.github.com:443\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nContent-Length: 0\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nTransfer-Encoding: chunked\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nTrailer: X\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nUpgrade: h2c\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nProxy-Authorization: x\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nUser-Agent: " + strings.Repeat("x", maxUserAgentBytes+1) + "\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nUser-Agent: bad\x7f\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nUser-Agent: one\r\nUser-Agent: two\r\n\r\n",
		"CONNECT api.github.com:443 HTTP/1.1\r\nHost: api.github.com:443\r\nProxy-Connection: keep-alive\r\nProxy-Connection: keep-alive\r\n\r\n",
		strings.Repeat("A", maxConnectHeader),
	}
	for _, input := range cases {
		server, client := net.Pipe()
		done := make(chan error, 1)
		go func() { done <- session.Serve(context.Background(), server) }()
		go io.WriteString(client, input)
		if got := readHeader(t, client); got != connectDenied {
			t.Fatalf("denial=%q input=%q", got, input[:24])
		}
		if err := <-done; !errors.Is(err, ErrDenied) {
			t.Fatalf("serve error=%v", err)
		}
	}
	if authority.count() != 0 {
		t.Fatalf("authority reached=%d", authority.count())
	}
}
func readHeader(t *testing.T, conn net.Conn) string {
	t.Helper()
	data, err := readHeaderNoFatal(conn)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func readHeaderNoFatal(conn net.Conn) (string, error) {
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	var data bytes.Buffer
	one := []byte{0}
	for !bytes.HasSuffix(data.Bytes(), []byte("\r\n\r\n")) {
		if _, err := conn.Read(one); err != nil {
			return "", err
		}
		data.WriteByte(one[0])
		if data.Len() > 4096 {
			return "", errors.New("header too long")
		}
	}
	return data.String(), nil
}
