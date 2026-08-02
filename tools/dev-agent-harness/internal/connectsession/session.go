package connectsession

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	phaseTimeout       = 5 * time.Second
	maxConnectHeader   = 16 << 10
	maxResponseBody    = 1 << 20
	maxResponseHeader  = 16 << 10
	maxHTTPInput       = (1 << 20) + maxConnectHeader
	maxUserAgentBytes  = 256
	githubHost         = "api.github.com"
	openAIHost         = "api.openai.com"
	connectDenied      = "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
	connectEstablished = "HTTP/1.1 200 Connection Established\r\nContent-Length: 0\r\n\r\n"
	controlNoContent   = "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"
	maxControlBody     = 512
)

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules   Error = "invalid-rules"
	ErrInvalidSession Error = "invalid-session"
	ErrSession        Error = "session-error"
	ErrDenied         Error = "session-denied"
)

type Authority interface {
	Issue(string) (tls.Certificate, error)
}

// CapabilityController applies peer-bound issuance and revocation policy.
// The session supplies only the minimum provider/repository request fields.
type CapabilityController interface {
	Issue(context.Context, string, string) (string, error)
	Revoke(context.Context, string) error
}
type Rules struct {
	Authority Authority
	Handler   http.Handler
	Control   CapabilityController
}
type Session struct {
	authority Authority
	handler   http.Handler
	control   CapabilityController
}

func New(r Rules) (*Session, error) {
	if isNil(r.Authority) || isNil(r.Handler) || isNil(r.Control) {
		return nil, ErrInvalidRules
	}
	return &Session{authority: r.Authority, handler: r.Handler, control: r.Control}, nil
}
func (s Session) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "connectsession.Session")
}
func (s *Session) Serve(ctx context.Context, conn net.Conn) (err error) {
	if !isNil(conn) {
		defer func() {
			_ = conn.Close()
		}()
	}
	defer func() {
		if recover() != nil {
			err = ErrSession
		}
	}()
	if s == nil || isNil(s.authority) || isNil(s.handler) || isNil(s.control) || isNil(ctx) || isNil(conn) {
		return ErrInvalidSession
	}
	if ctx.Err() != nil {
		return ErrSession
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()
	defer close(stop)
	connectCtx, cancelConnect := newPhaseContext(ctx)
	defer cancelConnect()
	if err := setDeadline(conn, connectCtx); err != nil {
		return ErrSession
	}
	header, ok := readInitialHeader(conn)
	if !ok {
		return denyInitial(conn, connectCtx)
	}
	if !isConnectHeader(header) {
		return s.serveControl(connectCtx, conn, header)
	}
	target := connectTarget(header)
	if !validConnect(header, target) || hasEarlyByte(conn) {
		if !writeFixed(conn, connectDenied) {
			return ErrSession
		}
		return ErrDenied
	}
	certificate, issueErr := s.authority.Issue(target)
	if issueErr != nil || !validCertificate(certificate) {
		return ErrSession
	}
	if err := setDeadline(conn, connectCtx); err != nil {
		return ErrSession
	}
	if !writeFixed(conn, connectEstablished) {
		return ErrSession
	}
	cancelConnect()
	tlsCtx, cancelTLS := newPhaseContext(ctx)
	defer cancelTLS()
	if err := setDeadline(conn, tlsCtx); err != nil {
		return ErrSession
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
		ClientAuth:   tls.NoClientCert,
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			if hello == nil || hello.ServerName != target {
				return nil, errors.New("tls name rejected")
			}
			return nil, nil
		},
	}
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.HandshakeContext(tlsCtx); err != nil {
		return ErrSession
	}
	state := tlsConn.ConnectionState()
	if state.Version < tls.VersionTLS12 || state.ServerName != target || state.NegotiatedProtocol != "http/1.1" {
		return ErrSession
	}
	cancelTLS()
	httpCtx, cancelHTTP := newPhaseContext(ctx)
	defer cancelHTTP()
	if err := setDeadline(tlsConn, httpCtx); err != nil {
		return ErrSession
	}
	limited := &io.LimitedReader{R: tlsConn, N: maxHTTPInput}
	reader := bufio.NewReader(limited)
	request, readErr := http.ReadRequest(reader)
	if readErr != nil || request == nil || !validInnerRequest(request, target) {
		return ErrSession
	}
	defer request.Body.Close()
	request = request.WithContext(httpCtx)
	response := newResponseBuffer()
	s.handler.ServeHTTP(response, request)
	if err := setDeadline(tlsConn, httpCtx); err != nil {
		return ErrSession
	}
	if err := response.flush(tlsConn); err != nil {
		return ErrSession
	}
	return nil
}

func denyInitial(conn net.Conn, ctx context.Context) error {
	if err := setDeadline(conn, ctx); err != nil || !writeFixed(conn, connectDenied) {
		return ErrSession
	}
	return ErrDenied
}

type controlRequest struct {
	issue      bool
	provider   string
	repository string
	handle     string
	bodyLength int
}

func (s *Session) serveControl(ctx context.Context, conn net.Conn, header []byte) error {
	request, ok := parseControlHeader(header)
	if !ok {
		return denyInitial(conn, ctx)
	}
	body := make([]byte, request.bodyLength)
	if request.bodyLength > 0 {
		if _, err := io.ReadFull(conn, body); err != nil {
			return denyInitial(conn, ctx)
		}
	}
	if hasControlExtraByte(conn) {
		return denyInitial(conn, ctx)
	}
	if request.issue {
		provider, repository, ok := decodeIssueBody(body)
		if !ok {
			return denyInitial(conn, ctx)
		}
		handle, err := s.control.Issue(ctx, provider, repository)
		if err != nil || !canonicalHandle(handle) {
			return denyInitial(conn, ctx)
		}
		responseBody, err := json.Marshal(struct {
			Handle string `json:"handle"`
		}{Handle: handle})
		if err != nil {
			return denyInitial(conn, ctx)
		}
		response := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(responseBody)) + "\r\nConnection: close\r\n\r\n" + string(responseBody)
		if err := setDeadline(conn, ctx); err != nil || !writeFixed(conn, response) {
			return ErrSession
		}
		return nil
	}
	if err := s.control.Revoke(ctx, request.handle); err != nil {
		return denyInitial(conn, ctx)
	}
	if err := setDeadline(conn, ctx); err != nil || !writeFixed(conn, controlNoContent) {
		return ErrSession
	}
	return nil
}

func hasControlExtraByte(conn net.Conn) bool {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		return true
	}
	var one [1]byte
	n, err := conn.Read(one[:])
	if n != 0 {
		return true
	}
	if errors.Is(err, io.EOF) {
		return false
	}
	if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
		return false
	}
	return err != nil
}
func newPhaseContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx, phaseDeadline(ctx))
}
func phaseDeadline(ctx context.Context) time.Time {
	deadline := time.Now().Add(phaseTimeout)
	if caller, ok := ctx.Deadline(); ok && caller.Before(deadline) {
		return caller
	}
	return deadline
}

func setDeadline(conn net.Conn, ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	deadline := phaseDeadline(ctx)
	return conn.SetDeadline(deadline)
}

func readInitialHeader(conn net.Conn) ([]byte, bool) {
	data := make([]byte, 0, maxConnectHeader)
	var one [1]byte
	for len(data) < maxConnectHeader {
		n, err := conn.Read(one[:])
		if n == 1 {
			data = append(data, one[0])
			if len(data) >= 4 && bytes.Equal(data[len(data)-4:], []byte("\r\n\r\n")) {
				return data, true
			}
		}
		if err != nil {
			return nil, false
		}
		if n != 1 {
			return nil, false
		}
	}
	return nil, false
}

func isConnectHeader(header []byte) bool {
	line := firstRequestLine(header)
	return strings.HasPrefix(line, "CONNECT ")
}

func connectTarget(header []byte) string {
	line := strings.Split(firstRequestLine(header), " ")
	if len(line) != 3 || line[0] != "CONNECT" || line[2] != "HTTP/1.1" {
		return ""
	}
	switch line[1] {
	case githubHost + ":443":
		return githubHost
	case openAIHost + ":443":
		return openAIHost
	default:
		return ""
	}
}

func firstRequestLine(header []byte) string {
	if index := bytes.Index(header, []byte("\r\n")); index >= 0 {
		return string(header[:index])
	}
	return ""
}

func parseControlHeader(header []byte) (controlRequest, bool) {
	lines := bytes.Split(header, []byte("\r\n"))
	if len(lines) < 4 || len(lines[len(lines)-1]) != 0 || len(lines[len(lines)-2]) != 0 {
		return controlRequest{}, false
	}
	request := controlRequest{}
	line := string(lines[0])
	switch {
	case line == "POST /v1/capabilities HTTP/1.1":
		request.issue = true
	case strings.HasPrefix(line, "DELETE /v1/capabilities/") && strings.HasSuffix(line, " HTTP/1.1"):
		path := strings.TrimSuffix(strings.TrimPrefix(line, "DELETE /v1/capabilities/"), " HTTP/1.1")
		if strings.ContainsAny(path, "/?#") || !canonicalHandle(path) {
			return controlRequest{}, false
		}
		request.handle = path
	default:
		return controlRequest{}, false
	}
	seenLength, seenType := false, false
	for _, raw := range lines[1 : len(lines)-2] {
		if len(raw) == 0 || !validHeaderBytes(raw) {
			return controlRequest{}, false
		}
		colon := bytes.IndexByte(raw, ':')
		if colon <= 0 || !headerToken(raw[:colon]) || colon+2 > len(raw) || raw[colon+1] != ' ' {
			return controlRequest{}, false
		}
		value := string(raw[colon+2:])
		if value == "" || value[0] == ' ' || value[0] == '\t' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t' {
			return controlRequest{}, false
		}
		switch {
		case strings.EqualFold(string(raw[:colon]), "Content-Length"):
			if seenLength {
				return controlRequest{}, false
			}
			length, ok := canonicalContentLength(value)
			if !ok {
				return controlRequest{}, false
			}
			request.bodyLength, seenLength = length, true
		case strings.EqualFold(string(raw[:colon]), "Content-Type"):
			if seenType || !request.issue || value != "application/json" {
				return controlRequest{}, false
			}
			seenType = true
		default:
			return controlRequest{}, false
		}
	}
	if !seenLength {
		return controlRequest{}, false
	}
	if request.issue {
		return request, seenType && request.bodyLength >= 1 && request.bodyLength <= maxControlBody
	}
	return request, !seenType && request.bodyLength == 0
}

func headerToken(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
			continue
		}
		return false
	}
	return true
}

func canonicalContentLength(value string) (int, bool) {
	if value == "" || len(value) > 3 || len(value) > 1 && value[0] == '0' {
		return 0, false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func decodeIssueBody(body []byte) (string, string, bool) {
	if len(body) == 0 || len(body) > maxControlBody || !utf8.Valid(body) {
		return "", "", false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return "", "", false
	}
	values := make(map[string]string, 2)
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, ok := keyToken.(string)
		if err != nil || !ok || key != "provider" && key != "repository" {
			return "", "", false
		}
		if _, duplicate := values[key]; duplicate {
			return "", "", false
		}
		var value string
		if err := decoder.Decode(&value); err != nil || value == "" {
			return "", "", false
		}
		values[key] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return "", "", false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return "", "", false
	}
	provider, present := values["provider"]
	if !present {
		return "", "", false
	}
	repository, hasRepository := values["repository"]
	if provider == "github" {
		return provider, repository, hasRepository && len(values) == 2
	}
	if provider == "openai" {
		return provider, "", !hasRepository && len(values) == 1
	}
	return "", "", false
}

func canonicalHandle(handle string) bool {
	if len(handle) != len("cap_")+43 || !strings.HasPrefix(handle, "cap_") {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(handle, "cap_"))
	return err == nil && len(raw) == 32 && "cap_"+base64.RawURLEncoding.EncodeToString(raw) == handle
}

func validConnect(header []byte, target string) bool {
	if len(header) == 0 || (target != githubHost && target != openAIHost) {
		return false
	}
	lines := bytes.Split(header, []byte("\r\n"))
	if len(lines) < 3 || len(lines[len(lines)-1]) != 0 {
		return false
	}
	lines = lines[:len(lines)-1]
	requestLine := strings.Split(string(lines[0]), " ")
	if len(requestLine) != 3 || requestLine[0] != "CONNECT" || requestLine[1] != target+":443" || requestLine[2] != "HTTP/1.1" {
		return false
	}
	seenHost, seenAgent, seenProxy := false, false, false
	for _, line := range lines[1 : len(lines)-1] {
		if len(line) == 0 || !validHeaderBytes(line) {
			return false
		}
		colon := bytes.IndexByte(line, ':')
		if colon <= 0 {
			return false
		}
		name := string(line[:colon])
		if colon+1 >= len(line) || line[colon+1] != ' ' || colon+2 >= len(line) {
			return false
		}
		value := string(line[colon+2:])
		if value[0] == ' ' || value[0] == '\t' || value[len(value)-1] == ' ' || value[len(value)-1] == '\t' {
			return false
		}
		switch {
		case strings.EqualFold(name, "Host"):
			if seenHost || value != target+":443" {
				return false
			}
			seenHost = true
		case strings.EqualFold(name, "User-Agent"):
			if seenAgent || len(value) > maxUserAgentBytes || !visibleASCII(value) {
				return false
			}
			seenAgent = true
		case strings.EqualFold(name, "Proxy-Connection"):
			if seenProxy || !strings.EqualFold(value, "keep-alive") {
				return false
			}
			seenProxy = true
		default:
			return false
		}
	}
	return seenHost
}

func validHeaderBytes(line []byte) bool {
	for _, value := range line {
		if value < 0x20 || value == 0x7f || value >= 0x80 {
			return false
		}
	}
	return true
}

func visibleASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] > 0x7e {
			return false
		}
	}
	return true
}

func hasEarlyByte(conn net.Conn) bool {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond)); err != nil {
		return true
	}
	var one [1]byte
	n, err := conn.Read(one[:])
	if n != 0 {
		return true
	}
	if err != nil {
		if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
			return false
		}
		return true
	}
	return false
}

func validCertificate(certificate tls.Certificate) bool {
	return len(certificate.Certificate) != 0 && certificate.PrivateKey != nil
}

func validInnerRequest(request *http.Request, target string) bool {
	if request.Proto != "HTTP/1.1" || request.ProtoMajor != 1 || request.ProtoMinor != 1 || request.URL == nil {
		return false
	}
	if request.Host != target && request.Host != target+":443" {
		return false
	}
	for _, value := range request.Header.Values("Connection") {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "keep-alive") {
				return false
			}
		}
	}
	return true
}

func writeFixed(conn net.Conn, response string) bool {
	data := []byte(response)
	for len(data) != 0 {
		n, err := conn.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil || n <= 0 {
			return false
		}
	}
	return true
}

type responseBuffer struct {
	header http.Header
	status int
	wrote  bool
	failed bool
	body   bytes.Buffer
}

func newResponseBuffer() *responseBuffer {
	return &responseBuffer{header: make(http.Header)}
}

func (w *responseBuffer) Header() http.Header { return w.header }

func (w *responseBuffer) WriteHeader(status int) {
	if w.wrote {
		return
	}
	if status < 100 || status > 999 {
		status = http.StatusInternalServerError
	}
	w.status, w.wrote = status, true
}

func (w *responseBuffer) Write(data []byte) (int, error) {
	if w.failed {
		return 0, ErrSession
	}
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if len(data) > maxResponseBody-w.body.Len() {
		w.failed = true
		return 0, ErrSession
	}
	return w.body.Write(data)
}

func (w *responseBuffer) flush(conn net.Conn) error {
	if w.failed {
		return ErrSession
	}
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if !validResponseHeaders(w.header, w.body.Len()) {
		return ErrSession
	}
	headers := make(http.Header, len(w.header)+2)
	for key, values := range w.header {
		headers[key] = append([]string(nil), values...)
	}
	headers.Del("Transfer-Encoding")
	headers.Set("Content-Length", fmt.Sprintf("%d", w.body.Len()))
	headers.Set("Connection", "close")
	var output bytes.Buffer
	text := http.StatusText(w.status)
	if text == "" {
		text = ""
	}
	fmt.Fprintf(&output, "HTTP/1.1 %d %s\r\n", w.status, text)
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range headers[key] {
			fmt.Fprintf(&output, "%s: %s\r\n", key, value)
		}
	}
	output.WriteString("\r\n")
	output.Write(w.body.Bytes())
	return writeAll(conn, output.Bytes())
}

func validResponseHeaders(headers http.Header, bodyLen int) bool {
	remaining := maxResponseHeader - 2
	if !consumeHeaderBudget(&remaining, "Content-Length", fmt.Sprintf("%d", bodyLen)) || !consumeHeaderBudget(&remaining, "Connection", "close") {
		return false
	}
	for key, values := range headers {
		if key != http.CanonicalHeaderKey(key) || !validHeaderName(key) {
			return false
		}
		if len(values) == 0 {
			if !consumeHeaderBudget(&remaining, key, "") {
				return false
			}
			continue
		}
		for _, value := range values {
			if !visibleASCII(value) || !consumeHeaderBudget(&remaining, key, value) {
				return false
			}
		}
	}
	return true
}

func consumeHeaderBudget(remaining *int, key, value string) bool {
	if len(key) > *remaining || len(value) > *remaining-len(key) || 4 > *remaining-len(key)-len(value) {
		return false
	}
	*remaining -= len(key) + len(value) + 4
	return true
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		value := name[i]
		if !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(value))) {
			return false
		}
	}
	return true
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) != 0 {
		n, err := conn.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil || n <= 0 {
			return ErrSession
		}
	}
	return nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
