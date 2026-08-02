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
	"strconv"
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
	mu          sync.Mutex
	cert        *proxyca.Authority
	public      []byte
	publicSet   bool
	calls       int
	publicCalls int
	err         error
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
func (a *countingAuthority) PublicCertificatePEM() []byte {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.publicCalls++
	if a.publicSet {
		return a.public
	}
	return a.cert.PublicCertificatePEM()
}
func (a *countingAuthority) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}
func (a *countingAuthority) publicCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.publicCalls
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

type allowControl struct{}

func (allowControl) Issue(context.Context, string, string, ...string) (string, error) {
	return "cap_" + strings.Repeat("A", 43), nil
}
func (allowControl) Revoke(context.Context, string) error { return nil }

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
	session, err := New(Rules{Authority: authority, Handler: handler, Control: allowControl{}})
	must(t, err)
	for _, host := range []string{githubHost, githubGitHost, openAIHost} {
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
	if authority.count() != 3 || exchange.calls != 3 || resolver.calls != 3 {
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
	session, err := New(Rules{Authority: authority, Handler: handler, Control: allowControl{}})
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
			session, err := New(Rules{Authority: authority, Control: allowControl{}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			session, err := New(Rules{Authority: authority, Handler: handler, Control: allowControl{}})
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
			session, err := New(Rules{Authority: authority, Handler: handler, Control: allowControl{}})
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
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") }), Control: allowControl{}})
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
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Control: allowControl{}})
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
		"zero": {}, "typed nil authority": {Authority: nilAuthority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Control: allowControl{}},
		"typed nil handler": {Authority: authority, Handler: (*nilHandler)(nil), Control: allowControl{}},
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
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Control: allowControl{}})
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
	session, err := New(Rules{Authority: authority, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") }), Control: allowControl{}})
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

type recordingControl struct {
	mu         sync.Mutex
	issues     int
	revokes    int
	provider   string
	repository string
	operation  string
	handle     string
	err        error
}

func (c *recordingControl) Issue(_ context.Context, provider, repository string, operations ...string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.issues++
	c.provider, c.repository = provider, repository
	if len(operations) == 1 {
		c.operation = operations[0]
	}
	if c.err != nil {
		return "", c.err
	}
	return "cap_" + strings.Repeat("A", 43), nil
}

func TestControlGitReadSelectorIsExplicit(t *testing.T) {
	authority, _ := newTestAuthority(t)
	control := &recordingControl{}
	session, err := New(Rules{Authority: authority, Control: control, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
	must(t, err)
	body := `{"provider":"github","repository":"octo/repo","operation":"github-git-read"}`
	response, serveErr := runControl(session, issueWire(body))
	if serveErr != nil || !strings.HasPrefix(response, "HTTP/1.1 200 OK\r\n") {
		t.Fatalf("response=%q err=%v", response, serveErr)
	}
	control.mu.Lock()
	if control.issues != 1 || control.provider != "github" || control.repository != "octo/repo" || control.operation != "github-git-read" {
		t.Fatalf("control=%+v", control)
	}
	control.mu.Unlock()

	for _, body := range []string{
		`{"provider":"github","repository":"octo/repo","operation":"git-read"}`,
		`{"provider":"github","repository":"octo/repo","operation":"github-rest-read"}`,
		`{"provider":"openai","operation":"github-git-read"}`,
		`{"provider":"github","repository":"octo/repo","operation":"github-git-read","host":"github.com"}`,
	} {
		bad := &recordingControl{}
		badSession, err := New(Rules{Authority: authority, Control: bad, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
		must(t, err)
		response, serveErr := runControl(badSession, issueWire(body))
		if response != connectDenied || !errors.Is(serveErr, ErrDenied) || bad.issues != 0 {
			t.Fatalf("body=%s response=%q err=%v issues=%d", body, response, serveErr, bad.issues)
		}
	}
}

func (c *recordingControl) Revoke(_ context.Context, handle string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revokes++
	c.handle = handle
	return c.err
}

func TestControlIssueAndRevokeExactResponses(t *testing.T) {
	authority, _ := newTestAuthority(t)
	control := &recordingControl{}
	session, err := New(Rules{
		Authority: authority, Control: control,
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") }),
	})
	must(t, err)
	body := `{"repository":"octo/repo","provider":"github"}`
	response, serveErr := runControl(session, "POST /v1/capabilities HTTP/1.1\r\nContent-Length: "+strconv.Itoa(len(body))+"\r\nContent-Type: application/json\r\n\r\n"+body)
	wantBody := `{"handle":"cap_` + strings.Repeat("A", 43) + `"}`
	want := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(wantBody)) + "\r\nConnection: close\r\n\r\n" + wantBody
	if serveErr != nil || response != want {
		t.Fatalf("issue err=%v response=%q", serveErr, response)
	}
	handle := "cap_" + strings.Repeat("A", 43)
	response, serveErr = runControl(session, "DELETE /v1/capabilities/"+handle+" HTTP/1.1\r\nContent-Length: 0\r\n\r\n")
	if serveErr != nil || response != controlNoContent {
		t.Fatalf("revoke err=%v response=%q", serveErr, response)
	}
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.issues != 1 || control.revokes != 1 || control.provider != "github" || control.repository != "octo/repo" || control.handle != handle || authority.count() != 0 {
		t.Fatalf("control=%+v authority=%d", control, authority.count())
	}
}

func TestControlStrictFramingJSONAndFixedDenial(t *testing.T) {
	authority, _ := newTestAuthority(t)
	for name, input := range map[string]string{
		"unknown method":     "GET /v1/capabilities HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"unknown path":       "POST /v2/capabilities HTTP/1.1\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"host":               "POST /v1/capabilities HTTP/1.1\r\nHost: local\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"duplicate length":   "POST /v1/capabilities HTTP/1.1\r\nContent-Length: 2\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"leading zero":       "POST /v1/capabilities HTTP/1.1\r\nContent-Length: 02\r\nContent-Type: application/json\r\n\r\n{}",
		"missing type":       "POST /v1/capabilities HTTP/1.1\r\nContent-Length: 2\r\n\r\n{}",
		"chunked":            "POST /v1/capabilities HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"keep alive":         "POST /v1/capabilities HTTP/1.1\r\nConnection: keep-alive\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"upgrade":            "POST /v1/capabilities HTTP/1.1\r\nUpgrade: h2c\r\nContent-Length: 2\r\nContent-Type: application/json\r\n\r\n{}",
		"duplicate json":     issueWire(`{"provider":"openai","provider":"github"}`),
		"unknown json":       issueWire(`{"provider":"openai","model":"secret-model"}`),
		"unknown operation":  issueWire(`{"provider":"github","repository":"octo/repo","operation":"receive-pack"}`),
		"subject json":       issueWire(`{"provider":"openai","subject":"agent"}`),
		"null":               issueWire(`{"provider":null}`),
		"missing repository": issueWire(`{"provider":"github"}`),
		"extra repository":   issueWire(`{"provider":"openai","repository":"octo/repo"}`),
		"body over limit":    issueWire(strings.Repeat("x", maxControlBody+1)),
		"invalid utf8":       issueWire(string([]byte{'{', '"', 'p', 'r', 'o', 'v', 'i', 'd', 'e', 'r', '"', ':', '"', 0xff, '"', '}'})),
		"early bytes":        issueWire(`{"provider":"openai"}`) + "GET /second HTTP/1.1\r\n\r\n",
		"revoke body":        "DELETE /v1/capabilities/cap_" + strings.Repeat("A", 43) + " HTTP/1.1\r\nContent-Length: 1\r\n\r\nx",
		"revoke query":       "DELETE /v1/capabilities/cap_" + strings.Repeat("A", 43) + "?x=1 HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"bad handle":         "DELETE /v1/capabilities/cap_bad HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			control := &recordingControl{}
			session, err := New(Rules{Authority: authority, Control: control, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
			must(t, err)
			response, serveErr := runControl(session, input)
			if response != connectDenied || !errors.Is(serveErr, ErrDenied) {
				t.Fatalf("response=%q err=%v", response, serveErr)
			}
			control.mu.Lock()
			defer control.mu.Unlock()
			if control.issues != 0 || control.revokes != 0 {
				t.Fatalf("controller reached: %+v", control)
			}
		})
	}
	if authority.count() != 0 {
		t.Fatalf("authority reached=%d", authority.count())
	}
}

func TestControlDependencyFailureIsFixedAndCloses(t *testing.T) {
	authority, _ := newTestAuthority(t)
	control := &recordingControl{err: errors.New("secret handle url allowlist lower error")}
	session, err := New(Rules{Authority: authority, Control: control, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
	must(t, err)
	response, serveErr := runControl(session, issueWire(`{"provider":"openai"}`))
	if response != connectDenied || !errors.Is(serveErr, ErrDenied) || strings.Contains(response, "secret") {
		t.Fatalf("response=%q err=%v", response, serveErr)
	}
}

func TestProxyCAExactResponseCopyAndRouteIsolation(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	public := makeProxyCAPEM(t, now, proxyCATestOptions{})
	authority := &countingAuthority{public: public, publicSet: true}
	control := &recordingControl{}
	handlerCalls := 0
	session, err := New(Rules{
		Authority: authority,
		Control:   control,
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			handlerCalls++
		}),
	})
	must(t, err)

	want := "HTTP/1.1 200 OK\r\nContent-Type: application/x-pem-file\r\nContent-Length: " +
		strconv.Itoa(len(public)) + "\r\nConnection: close\r\n\r\n" + string(public)
	first, serveErr := runControl(session, proxyCARequest)
	if serveErr != nil || first != want {
		t.Fatalf("proxy CA response mismatch: len=%d err=%v", len(first), serveErr)
	}
	mutated := []byte(first)
	mutated[len(mutated)-2] ^= 1
	second, serveErr := runControl(session, proxyCARequest)
	if serveErr != nil || second != want || first != want {
		t.Fatalf("response did not remain isolated: len=%d err=%v", len(second), serveErr)
	}
	control.mu.Lock()
	issues, revokes := control.issues, control.revokes
	control.mu.Unlock()
	if authority.count() != 0 || authority.publicCount() != 2 || issues != 0 || revokes != 0 || handlerCalls != 0 {
		t.Fatalf("route calls issueCert=%d public=%d control=%d/%d handler=%d", authority.count(), authority.publicCount(), issues, revokes, handlerCalls)
	}
	block, rest := pem.Decode(public)
	certificate, parseErr := x509.ParseCertificate(block.Bytes)
	if parseErr != nil || len(rest) != 0 || !certificate.IsCA || certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Fatal("success body was not a certificate-only CA")
	}
}

func TestProxyCARequestIsExactZeroBody(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	public := makeProxyCAPEM(t, now, proxyCATestOptions{})
	for name, input := range map[string]string{
		"method":            "POST /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"path":              "GET /v1/proxy-cas HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"query":             "GET /v1/proxy-ca?x=1 HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"fragment":          "GET /v1/proxy-ca#x HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
		"version":           "GET /v1/proxy-ca HTTP/1.0\r\nContent-Length: 0\r\n\r\n",
		"missing length":    "GET /v1/proxy-ca HTTP/1.1\r\n\r\n",
		"header case":       "GET /v1/proxy-ca HTTP/1.1\r\ncontent-length: 0\r\n\r\n",
		"leading zero":      "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 00\r\n\r\n",
		"space":             "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length:  0\r\n\r\n",
		"duplicate":         "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\nContent-Length: 0\r\n\r\n",
		"host":              "GET /v1/proxy-ca HTTP/1.1\r\nHost: local\r\nContent-Length: 0\r\n\r\n",
		"connection":        "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
		"content type":      "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\nContent-Type: application/x-pem-file\r\n\r\n",
		"chunked":           "GET /v1/proxy-ca HTTP/1.1\r\nTransfer-Encoding: chunked\r\nContent-Length: 0\r\n\r\n",
		"body":              "GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 1\r\n\r\nx",
		"early byte":        proxyCARequest + "x",
		"second request":    proxyCARequest + proxyCARequest,
		"noncanonical line": "GET  /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\n\r\n",
	} {
		t.Run(name, func(t *testing.T) {
			authority := &countingAuthority{public: public, publicSet: true}
			control := &recordingControl{}
			session, err := New(Rules{Authority: authority, Control: control, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
			must(t, err)
			response, serveErr := runControl(session, input)
			if response != connectDenied || !errors.Is(serveErr, ErrDenied) || authority.publicCount() != 0 || authority.count() != 0 {
				t.Fatalf("malformed GET accepted: response-len=%d err=%v public=%d issue=%d", len(response), serveErr, authority.publicCount(), authority.count())
			}
			control.mu.Lock()
			defer control.mu.Unlock()
			if control.issues != 0 || control.revokes != 0 {
				t.Fatal("malformed GET reached capability controller")
			}
		})
	}
}

func TestProxyCAAuthorityOutputValidationIsFixedAndNonLeaking(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	valid := makeProxyCAPEM(t, now, proxyCATestOptions{})
	private := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("PRIVATE-SENTINEL")})
	cases := map[string][]byte{
		"nil":              nil,
		"empty":            {},
		"malformed":        []byte("SECRET-SUBJECT malformed"),
		"private":          private,
		"wrong block":      pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte{1}}),
		"multiple":         append(append([]byte(nil), valid...), valid...),
		"trailing newline": append(append([]byte(nil), valid...), '\n'),
		"over limit":       bytes.Repeat([]byte("x"), maxProxyCAPEM+1),
		"not yet valid":    makeProxyCAPEM(t, now, proxyCATestOptions{notBefore: now.Add(time.Minute), notAfter: now.Add(time.Hour)}),
		"expired":          makeProxyCAPEM(t, now, proxyCATestOptions{notBefore: now.Add(-time.Hour), notAfter: now}),
		"non ca":           makeProxyCAPEM(t, now, proxyCATestOptions{nonCA: true}),
		"no constraints":   makeProxyCAPEM(t, now, proxyCATestOptions{noConstraints: true}),
		"no cert sign":     makeProxyCAPEM(t, now, proxyCATestOptions{noCertSign: true}),
		"p384":             makeProxyCAPEM(t, now, proxyCATestOptions{p384: true}),
		"not self signed":  makeProxyCAPEM(t, now, proxyCATestOptions{notSelfSigned: true}),
	}
	for name, output := range cases {
		t.Run(name, func(t *testing.T) {
			authority := &countingAuthority{public: output, publicSet: true}
			session, err := New(Rules{Authority: authority, Control: allowControl{}, Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("handler called") })})
			must(t, err)
			response, serveErr := runControl(session, proxyCARequest)
			if response != connectDenied || !errors.Is(serveErr, ErrDenied) || authority.publicCount() != 1 || authority.count() != 0 {
				t.Fatalf("invalid authority output accepted: response-len=%d err=%v calls=%d", len(response), serveErr, authority.publicCount())
			}
			for _, sentinel := range []string{"PRIVATE-SENTINEL", "SECRET-SUBJECT", "BEGIN CERTIFICATE"} {
				if strings.Contains(response, sentinel) || strings.Contains(serveErr.Error(), sentinel) {
					t.Fatal("authority material leaked")
				}
			}
		})
	}
}

type proxyCATestOptions struct {
	notBefore, notAfter                                   time.Time
	nonCA, noConstraints, noCertSign, p384, notSelfSigned bool
}

func makeProxyCAPEM(t *testing.T, now time.Time, options proxyCATestOptions) []byte {
	t.Helper()
	curve := elliptic.P256()
	if options.p384 {
		curve = elliptic.P384()
	}
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	must(t, err)
	notBefore, notAfter := options.notBefore, options.notAfter
	if notBefore.IsZero() {
		notBefore = now.Add(-time.Hour)
	}
	if notAfter.IsZero() {
		notAfter = now.Add(time.Hour)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{CommonName: "SECRET-SUBJECT"},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  !options.nonCA,
		BasicConstraintsValid: !options.noConstraints,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	if options.noCertSign {
		template.KeyUsage = x509.KeyUsageDigitalSignature
	}
	parent, signer := template, any(key)
	if options.notSelfSigned {
		parentKey, generateErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		must(t, generateErr)
		parent = &x509.Certificate{
			SerialNumber: big.NewInt(43), Subject: pkix.Name{CommonName: "other"},
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true,
			BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
		}
		signer = parentKey
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
	must(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func issueWire(body string) string {
	return "POST /v1/capabilities HTTP/1.1\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nContent-Type: application/json\r\n\r\n" + body
}

func runControl(session *Session, input string) (string, error) {
	server, client := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- session.Serve(context.Background(), server) }()
	go func() { _, _ = io.WriteString(client, input) }()
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	response, _ := io.ReadAll(client)
	_ = client.Close()
	return string(response), <-done
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
