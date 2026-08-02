package controlclient

import (
	"context"
	"errors"
	"io"
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
	if calls != 0 {
		t.Fatalf("dial calls=%d", calls)
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
	if strings.HasPrefix(text, "DELETE ") {
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

func issueResponse(handle string) string {
	return issueRawResponse(`{"handle":"` + handle + `"}`)
}

func issueRawResponse(body string) string {
	return "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " +
		strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n" + body
}
