package controlclient

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const testSocket = "/run/dev-agent-harness/egress.sock"

var testHandle = "cap_" + strings.Repeat("A", 43)

type observedConn struct {
	net.Conn
	mu             sync.Mutex
	readDeadlines  int
	writeDeadlines int
	deadlines      int
	closes         int
	maxWrite       int
	closeErr       error
	deadlineErr    error
}

func (c *observedConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.readDeadlines++
	err := c.deadlineErr
	c.mu.Unlock()
	if err != nil {
		return err
	}
	return c.Conn.SetReadDeadline(deadline)
}

func (c *observedConn) SetWriteDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.writeDeadlines++
	err := c.deadlineErr
	c.mu.Unlock()
	if err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(deadline)
}

func (c *observedConn) SetDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadlines++
	err := c.deadlineErr
	c.mu.Unlock()
	if err != nil {
		return err
	}
	return c.Conn.SetDeadline(deadline)
}

func (c *observedConn) Write(data []byte) (int, error) {
	if c.maxWrite > 0 && len(data) > c.maxWrite {
		data = data[:c.maxWrite]
	}
	return c.Conn.Write(data)
}

func (c *observedConn) Close() error {
	c.mu.Lock()
	c.closes++
	err := c.closeErr
	c.mu.Unlock()
	underlyingErr := c.Conn.Close()
	if err != nil {
		return err
	}
	return underlyingErr
}

func (c *observedConn) counts() (int, int, int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readDeadlines, c.writeDeadlines, c.deadlines, c.closes
}

func TestIssueUsesExactSingleConnectionWire(t *testing.T) {
	body := `{"provider":"github","repository":"octo/repo","operation":"github-git-read"}`
	wantRequest := "POST /v1/capabilities HTTP/1.1\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nContent-Type: application/json\r\n\r\n" + body
	wantResponse := issueResponse(testHandle)
	conn, request, calls, done := installPipe(t, wantResponse, 7, nil)

	got, err := Issue(testSocket, "octo/repo")
	if err != nil || got != testHandle {
		t.Fatalf("handle shape mismatch: len=%d err=%v", len(got), err)
	}
	<-done
	if *request != wantRequest || *calls != 1 {
		t.Fatalf("request=%q calls=%d", *request, *calls)
	}
	reads, writes, deadlines, closes := conn.counts()
	if reads != 1 || writes != 1 || deadlines != 0 || closes != 1 {
		t.Fatalf("deadlines read=%d write=%d all=%d closes=%d", reads, writes, deadlines, closes)
	}
}

func TestRevokeUsesExactSingleConnectionWire(t *testing.T) {
	wantRequest := "DELETE /v1/capabilities/" + testHandle + " HTTP/1.1\r\nContent-Length: 0\r\n\r\n"
	response := "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
	_, request, calls, done := installPipe(t, response, 5, nil)

	err := Revoke(testSocket, testHandle)
	<-done
	if err != nil || *request != wantRequest || *calls != 1 {
		t.Fatalf("err=%v request=%q calls=%d", err, *request, *calls)
	}
}

func TestProxyCAUsesExactSingleConnectionWireAndReturnsCopy(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	public := makeClientProxyCAPEM(t, now, clientCATestOptions{})
	response := proxyCAResponse(public)
	conn, request, calls, done := installPipe(t, response, 3, nil)

	got, err := ProxyCA(testSocket)
	if err != nil || !bytes.Equal(got, public) {
		t.Fatalf("public CA mismatch: len=%d err=%v", len(got), err)
	}
	<-done
	if *request != proxyCARequest || *calls != 1 {
		t.Fatalf("request=%q calls=%d", *request, *calls)
	}
	reads, writes, deadlines, closes := conn.counts()
	if reads != 1 || writes != 1 || deadlines != 0 || closes != 1 {
		t.Fatalf("deadlines read=%d write=%d all=%d closes=%d", reads, writes, deadlines, closes)
	}
	got[0] ^= 1
	if bytes.Equal(got, public) || !bytes.Equal([]byte(response[len(response)-len(public):]), public) {
		t.Fatal("returned CA aliases response state")
	}
}

func TestInputIsRejectedBeforeDial(t *testing.T) {
	original := dialControl
	defer func() { dialControl = original }()
	calls := 0
	dialControl = func(context.Context, string, string) (net.Conn, error) {
		calls++
		return nil, errors.New("secret socket lower error")
	}

	for _, input := range []struct{ socket, repository string }{
		{"relative.sock", "octo/repo"},
		{"", "octo/repo"},
		{testSocket, "Octo/repo"},
		{testSocket, "octo/repo/extra"},
		{testSocket, "octo/../repo"},
		{testSocket, "octo/repo name"},
	} {
		if value, err := Issue(input.socket, input.repository); value != "" || err != ErrControl {
			t.Fatalf("input accepted: value=%q err=%v", value, err)
		}
	}
	for _, handle := range []string{"", "cap_bad", "cap_" + strings.Repeat("+", 43), testHandle + "x"} {
		if err := Revoke(testSocket, handle); err != ErrControl {
			t.Fatalf("handle accepted: %q", handle)
		}
	}
	for _, socket := range []string{"", "relative.sock", "/run/../tmp/egress.sock", "/run//egress.sock"} {
		if value, err := ProxyCA(socket); value != nil || err != ErrControl {
			t.Fatalf("proxy CA socket accepted: value-len=%d err=%v", len(value), err)
		}
	}
	if calls != 0 {
		t.Fatalf("dial calls=%d", calls)
	}
}

func TestProxyCAStrictResponseMatrix(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	public := makeClientProxyCAPEM(t, now, clientCATestOptions{})
	valid := proxyCAResponse(public)
	length := strconv.Itoa(len(public))
	cases := map[string]string{
		"informational":  strings.Replace(valid, "200 OK", "100 Continue", 1),
		"no content":     strings.Replace(valid, "200 OK", "204 No Content", 1),
		"redirect":       strings.Replace(valid, "200 OK", "302 Found", 1),
		"forbidden":      strings.Replace(valid, "200 OK", "403 Forbidden", 1),
		"server error":   strings.Replace(valid, "200 OK", "500 Internal Server Error", 1),
		"header order":   "HTTP/1.1 200 OK\r\nContent-Length: " + length + "\r\nContent-Type: application/x-pem-file\r\nConnection: close\r\n\r\n" + string(public),
		"header case":    strings.Replace(valid, "Content-Type", "content-type", 1),
		"header space":   strings.Replace(valid, "Content-Length: ", "Content-Length:  ", 1),
		"extra header":   strings.Replace(valid, "Connection: close", "X-Extra: no\r\nConnection: close", 1),
		"duplicate type": strings.Replace(valid, "Content-Length", "Content-Type: application/x-pem-file\r\nContent-Length", 1),
		"wrong type":     strings.Replace(valid, "application/x-pem-file", "text/plain", 1),
		"chunked":        strings.Replace(valid, "Content-Length: "+length, "Transfer-Encoding: chunked", 1),
		"leading zero":   strings.Replace(valid, "Content-Length: "+length, "Content-Length: 0"+length, 1),
		"missing close":  strings.Replace(valid, "Connection: close\r\n", "", 1),
		"wrong length":   strings.Replace(valid, "Content-Length: "+length, "Content-Length: 1", 1),
		"early eof":      valid[:len(valid)-20],
		"extra byte":     valid + "x",
		"second response": valid +
			"HTTP/1.1 200 OK\r\nContent-Type: application/x-pem-file\r\nContent-Length: 1\r\nConnection: close\r\n\r\nx",
		"header overflow": "HTTP/1.1 200 OK\r\nX: " + strings.Repeat("x", maxResponseHeader) + "\r\n\r\n",
	}
	for name, response := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, calls, done := installPipe(t, response, 0, nil)
			value, err := ProxyCA(testSocket)
			<-done
			if value != nil || err != ErrControl || *calls != 1 {
				t.Fatalf("invalid response accepted: value-len=%d err=%v calls=%d", len(value), err, *calls)
			}
		})
	}
}

func TestProxyCACertificateValidationAndNonLeak(t *testing.T) {
	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	originalClock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = originalClock })
	valid := makeClientProxyCAPEM(t, now, clientCATestOptions{})
	private := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: []byte("PRIVATE-SENTINEL")})
	cases := map[string][]byte{
		"empty":           {},
		"malformed":       []byte("LOWER-ERROR SECRET-SUBJECT"),
		"private":         private,
		"wrong block":     pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte{1}}),
		"multiple":        append(append([]byte(nil), valid...), valid...),
		"trailing":        append(append([]byte(nil), valid...), 'x'),
		"over limit":      bytes.Repeat([]byte("x"), maxProxyCAPEM+1),
		"not yet valid":   makeClientProxyCAPEM(t, now, clientCATestOptions{notBefore: now.Add(time.Second), notAfter: now.Add(time.Hour)}),
		"expired":         makeClientProxyCAPEM(t, now, clientCATestOptions{notBefore: now.Add(-time.Hour), notAfter: now}),
		"non ca":          makeClientProxyCAPEM(t, now, clientCATestOptions{nonCA: true}),
		"no constraints":  makeClientProxyCAPEM(t, now, clientCATestOptions{noConstraints: true}),
		"no cert sign":    makeClientProxyCAPEM(t, now, clientCATestOptions{noCertSign: true}),
		"p384":            makeClientProxyCAPEM(t, now, clientCATestOptions{p384: true}),
		"not self signed": makeClientProxyCAPEM(t, now, clientCATestOptions{notSelfSigned: true}),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, calls, done := installPipe(t, proxyCAResponse(body), 0, nil)
			value, err := ProxyCA(testSocket)
			<-done
			if value != nil || err != ErrControl || *calls != 1 {
				t.Fatalf("invalid certificate accepted: value-len=%d err=%v calls=%d", len(value), err, *calls)
			}
			for _, sentinel := range []string{"PRIVATE-SENTINEL", "LOWER-ERROR", "SECRET-SUBJECT", testSocket, "/v1/proxy-ca"} {
				if strings.Contains(err.Error(), sentinel) {
					t.Fatal("fixed error leaked certificate, socket, route, or lower error")
				}
			}
		})
	}
}

func TestProxyCATransportFailuresCloseAndStayFixed(t *testing.T) {
	original := dialControl
	t.Cleanup(func() { dialControl = original })
	sentinel := "PRIVATE-SENTINEL /secret/socket lower-error"
	dialControl = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New(sentinel)
	}
	if value, err := ProxyCA(testSocket); value != nil || err != ErrControl || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("dial failure value-len=%d err=%v", len(value), err)
	}

	server, client := net.Pipe()
	observed := &observedConn{Conn: client, deadlineErr: errors.New(sentinel)}
	dialControl = func(context.Context, string, string) (net.Conn, error) { return observed, nil }
	if value, err := ProxyCA(testSocket); value != nil || err != ErrControl {
		t.Fatalf("deadline failure value-len=%d err=%v", len(value), err)
	}
	_ = server.Close()
	_, _, _, closes := observed.counts()
	if closes != 1 {
		t.Fatalf("deadline failure closes=%d", closes)
	}

	now := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := proxyCAClock
	proxyCAClock = func() time.Time { return now }
	t.Cleanup(func() { proxyCAClock = clock })
	public := makeClientProxyCAPEM(t, now, clientCATestOptions{})
	conn, _, _, done := installPipe(t, proxyCAResponse(public), 0, errors.New(sentinel))
	value, err := ProxyCA(testSocket)
	<-done
	if value != nil || err != ErrControl || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("close failure value-len=%d err=%v", len(value), err)
	}
	_, _, _, closes = conn.counts()
	if closes != 1 {
		t.Fatalf("close failure closes=%d", closes)
	}
}

func TestIssueStrictResponseMatrix(t *testing.T) {
	validBody := `{"handle":"` + testHandle + `"}`
	cases := map[string]string{
		"status":         strings.Replace(issueResponse(testHandle), "200 OK", "403 Forbidden", 1),
		"header order":   "HTTP/1.1 200 OK\r\nContent-Length: 60\r\nContent-Type: application/json\r\nConnection: close\r\n\r\n" + validBody,
		"extra header":   strings.Replace(issueResponse(testHandle), "Connection: close", "X-Extra: no\r\nConnection: close", 1),
		"chunked":        strings.Replace(issueResponse(testHandle), "Content-Length: 60", "Transfer-Encoding: chunked", 1),
		"leading zero":   strings.Replace(issueResponse(testHandle), "Content-Length: 60", "Content-Length: 060", 1),
		"wrong type":     strings.Replace(issueResponse(testHandle), "application/json", "text/plain", 1),
		"unknown json":   issueRawResponse(`{"handle":"` + testHandle + `","other":"x"}`),
		"space json":     issueRawResponse(`{ "handle":"` + testHandle + `"}`),
		"bad handle":     issueRawResponse(`{"handle":"cap_bad"}`),
		"escaped handle": issueRawResponse(`{"handle":"cap_` + strings.Repeat(`\u0041`, 43) + `"}`),
		"early eof":      "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 60\r\nConnection: close\r\n\r\n" + validBody[:20],
		"extra bytes":    issueResponse(testHandle) + "x",
		"second response": issueResponse(testHandle) +
			"HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n",
	}
	for name, response := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, calls, done := installPipe(t, response, 0, nil)
			value, err := Issue(testSocket, "octo/repo")
			<-done
			if value != "" || err != ErrControl || *calls != 1 {
				t.Fatalf("value=%q err=%v calls=%d", value, err, *calls)
			}
		})
	}
}

func TestRevokeStrictResponseMatrix(t *testing.T) {
	valid := "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
	for name, response := range map[string]string{
		"status":        strings.Replace(valid, "204 No Content", "200 OK", 1),
		"missing close": strings.Replace(valid, "Connection: close\r\n", "", 1),
		"extra header":  strings.Replace(valid, "Connection: close", "X: y\r\nConnection: close", 1),
		"body":          valid + "x",
		"early eof":     strings.TrimSuffix(valid, "\r\n"),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, _, done := installPipe(t, response, 0, nil)
			err := Revoke(testSocket, testHandle)
			<-done
			if err != ErrControl {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestTransportFailuresAreFixedAndClose(t *testing.T) {
	original := dialControl
	defer func() { dialControl = original }()
	sentinel := "secret.sock octo/repo " + testHandle + " lower-error"
	dialControl = func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New(sentinel)
	}
	if value, err := Issue(testSocket, "octo/repo"); value != "" || err != ErrControl || strings.Contains(err.Error(), sentinel) {
		t.Fatalf("value=%q err=%v", value, err)
	}

	server, client := net.Pipe()
	observed := &observedConn{Conn: client, deadlineErr: errors.New(sentinel)}
	dialControl = func(context.Context, string, string) (net.Conn, error) { return observed, nil }
	if value, err := Issue(testSocket, "octo/repo"); value != "" || err != ErrControl {
		t.Fatalf("value=%q err=%v", value, err)
	}
	_ = server.Close()
	_, _, _, closes := observed.counts()
	if closes != 1 {
		t.Fatalf("closes=%d", closes)
	}
}

func TestCloseFailureRejectsOtherwiseValidResponse(t *testing.T) {
	conn, _, _, done := installPipe(t, issueResponse(testHandle), 0, errors.New("secret close error"))
	value, err := Issue(testSocket, "octo/repo")
	<-done
	if value != "" || err != ErrControl {
		t.Fatalf("value=%q err=%v", value, err)
	}
	_, _, _, closes := conn.counts()
	if closes != 1 {
		t.Fatalf("closes=%d", closes)
	}
}

func installPipe(t *testing.T, response string, maxWrite int, closeErr error) (*observedConn, *string, *int, <-chan struct{}) {
	t.Helper()
	original := dialControl
	server, client := net.Pipe()
	observed := &observedConn{Conn: client, maxWrite: maxWrite, closeErr: closeErr}
	calls := 0
	request := ""
	dialControl = func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		if network != "unix" || address != testSocket || ctx == nil {
			t.Errorf("dial network=%q address=%q", network, address)
		}
		return observed, nil
	}
	t.Cleanup(func() {
		dialControl = original
		_ = server.Close()
		_ = client.Close()
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer server.Close()
		_ = server.SetDeadline(time.Now().Add(2 * time.Second))
		data := make([]byte, 0, 256)
		var one [1]byte
		for {
			n, err := server.Read(one[:])
			if n == 1 {
				data = append(data, one[0])
				if bytesCompleteRequest(data) {
					break
				}
			}
			if err != nil {
				break
			}
		}
		request = string(data)
		_, _ = io.WriteString(server, response)
	}()
	return observed, &request, &calls, done
}

func bytesCompleteRequest(data []byte) bool {
	text := string(data)
	if strings.HasPrefix(text, "DELETE ") || strings.HasPrefix(text, "GET ") {
		return strings.HasSuffix(text, "\r\n\r\n")
	}
	marker := "\r\n\r\n"
	index := strings.Index(text, marker)
	if index < 0 {
		return false
	}
	lengthPrefix := "Content-Length: "
	lengthStart := strings.Index(text[:index], lengthPrefix)
	if lengthStart < 0 {
		return false
	}
	lengthStart += len(lengthPrefix)
	lengthEnd := strings.Index(text[lengthStart:index], "\r\n")
	if lengthEnd < 0 {
		return false
	}
	length, err := strconv.Atoi(text[lengthStart : lengthStart+lengthEnd])
	return err == nil && len(text[index+len(marker):]) == length
}

func proxyCAResponse(body []byte) string {
	return "HTTP/1.1 200 OK\r\nContent-Type: application/x-pem-file\r\nContent-Length: " +
		strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + string(body)
}

type clientCATestOptions struct {
	notBefore, notAfter                                   time.Time
	nonCA, noConstraints, noCertSign, p384, notSelfSigned bool
}

func makeClientProxyCAPEM(t *testing.T, now time.Time, options clientCATestOptions) []byte {
	t.Helper()
	curve := elliptic.P256()
	if options.p384 {
		curve = elliptic.P384()
	}
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	notBefore, notAfter := options.notBefore, options.notAfter
	if notBefore.IsZero() {
		notBefore = now.Add(-time.Hour)
	}
	if notAfter.IsZero() {
		notAfter = now.Add(time.Hour)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(52),
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
		if generateErr != nil {
			t.Fatal(generateErr)
		}
		parent = &x509.Certificate{
			SerialNumber: big.NewInt(53), Subject: pkix.Name{CommonName: "other"},
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true,
			BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
		}
		signer = parentKey
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func issueResponse(handle string) string {
	return issueRawResponse(`{"handle":"` + handle + `"}`)
}

func issueRawResponse(body string) string {
	return "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " +
		strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body
}
