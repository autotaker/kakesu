// Package brokerhttp maps strict HTTP/1.1 origin-form requests to the
// in-memory broker exchange. It owns no listener, TLS, identity production
// resolver, transport, retry, or response diagnostics.
package brokerhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerexchange"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresspolicy"
	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

const (
	maxRequestBodyBytes = 1 << 20
	maxResponseBytes    = 1 << 20
)

// Error values are fixed and contain no request, identity, response, or
// dependency detail.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
)

// Exchange is the already-composed in-memory broker boundary. Handler never
// implements policy, capability consumption, credential resolution, or
// forwarding itself.
type Exchange interface {
	Do(egresstransaction.Subject, egresstransaction.Request) (brokerexchange.Response, error)
}

// SubjectResolver obtains trusted caller identity from context only.
type SubjectResolver interface {
	Resolve(context.Context) (egresstransaction.Subject, error)
}

// SubjectResolverFunc adapts a context-only function to SubjectResolver.
type SubjectResolverFunc func(context.Context) (egresstransaction.Subject, error)

func (f SubjectResolverFunc) Resolve(ctx context.Context) (egresstransaction.Subject, error) {
	return f(ctx)
}

// Rules contains only the immutable exchange, context-only resolver, and
// bounded request body setting.
type Rules struct {
	Exchange     Exchange
	Resolver     SubjectResolver
	MaxBodyBytes int
}

// Handler is immutable after New returns and keeps no request-local state.
type Handler struct {
	exchange     Exchange
	resolver     SubjectResolver
	maxBodyBytes int
}

// Format keeps dependencies and configuration out of diagnostics.
func (h Handler) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "brokerhttp.Handler")
}

// New validates dependencies and the bounded body setting. Typed nil
// interfaces are rejected before request handling begins.
func New(r Rules) (*Handler, error) {
	if isNil(r.Exchange) || isNil(r.Resolver) || r.MaxBodyBytes < 1 || r.MaxBodyBytes > maxRequestBodyBytes {
		return nil, ErrInvalidRules
	}
	return &Handler{exchange: r.Exchange, resolver: r.Resolver, maxBodyBytes: r.MaxBodyBytes}, nil
}

// ServeHTTP accepts only the strict HTTP/1.1 origin-form surface. Every
// malformed request and dependency failure becomes the same empty 403.
func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if h == nil || isNil(h.exchange) || isNil(h.resolver) || h.maxBodyBytes < 1 || h.maxBodyBytes > maxRequestBodyBytes {
		writeDenied(writer)
		return
	}
	mapped, ok := mapRequest(request, h.maxBodyBytes)
	if !ok {
		writeDenied(writer)
		return
	}
	subject, err := h.resolver.Resolve(request.Context())
	if err != nil {
		writeDenied(writer)
		return
	}
	response, err := h.exchange.Do(subject, mapped)
	if err != nil || !validResponse(response) {
		writeDenied(writer)
		return
	}
	writeSuccess(writer, response)
}

func mapRequest(request *http.Request, maxBody int) (egresstransaction.Request, bool) {
	if request == nil || !validProtocolAndTarget(request) || !validFraming(request, maxBody) {
		return egresstransaction.Request{}, false
	}
	body, ok := readBody(request, maxBody)
	if !ok {
		return egresstransaction.Request{}, false
	}
	contentTypes := append([]string(nil), request.Header.Values("Content-Type")...)
	if len(contentTypes) > 1 {
		return egresstransaction.Request{}, false
	}
	authorization := append([]string(nil), request.Header.Values("Authorization")...)
	target := request.URL.Path
	if request.URL.RawQuery != "" {
		target += "?" + request.URL.RawQuery
	}
	return egresstransaction.Request{
		Method: request.Method, URL: "https://" + request.Host + target,
		ContentType: first(contentTypes), Body: body, Authorization: authorization,
	}, true
}

func validProtocolAndTarget(request *http.Request) bool {
	if request.Proto != "HTTP/1.1" || request.ProtoMajor != 1 || request.ProtoMinor != 1 {
		return false
	}
	if !validMethod(request.Method) {
		return false
	}
	if request.URL == nil || request.URL.Scheme != "" || request.URL.Host != "" || request.URL.User != nil ||
		request.URL.Opaque != "" || request.URL.Fragment != "" || request.URL.RawFragment != "" ||
		request.URL.ForceQuery || request.URL.RawPath != "" {
		return false
	}
	path := request.URL.Path
	target := path
	if request.URL.RawQuery != "" {
		target += "?" + request.URL.RawQuery
	}
	if path == "" || path[0] != '/' || strings.HasPrefix(path, "//") || request.RequestURI == "" || request.RequestURI != target ||
		strings.ContainsAny(path, "%?#") || !visibleASCII(path) {
		return false
	}
	if !validHost(request.Host) || len(request.Header.Values("Host")) != 0 {
		return false
	}
	if request.URL.RawQuery != "" && !validGitDiscoveryQuery(request) {
		return false
	}
	return true
}

func validGitDiscoveryQuery(request *http.Request) bool {
	return request != nil && request.URL != nil && request.Method == http.MethodGet &&
		(request.Host == egresspolicy.GitHubGitHost || request.Host == egresspolicy.GitHubGitHost+":443") &&
		request.URL.RawQuery == "service=git-upload-pack" &&
		strings.HasSuffix(request.URL.Path, ".git/info/refs") &&
		!strings.ContainsAny(request.URL.RawQuery, "%#&;") && visibleASCII(request.URL.RawQuery)
}

func validHost(host string) bool {
	if len(host) == 0 || len(host) > 253 || !visibleASCII(host) || strings.ContainsAny(host, "/?#@\\%,;") {
		return false
	}
	for _, r := range host {
		if r == ' ' || r == '\t' {
			return false
		}
	}
	if strings.Count(host, ":") == 0 {
		return true
	}
	if strings.Count(host, ":") != 1 {
		return false
	}
	colon := strings.IndexByte(host, ':')
	if colon == 0 || colon == len(host)-1 {
		return false
	}
	port, err := strconv.Atoi(host[colon+1:])
	return err == nil && port > 0 && port <= 65535
}

func validMethod(method string) bool {
	if method == "" || method == http.MethodConnect || !visibleASCII(method) {
		return false
	}
	for i := 0; i < len(method); i++ {
		c := method[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		if !strings.ContainsRune("!#$%&'*+-.^_`|~", rune(c)) {
			return false
		}
	}
	return true
}

func validFraming(request *http.Request, maxBody int) bool {
	if request.ContentLength < 0 || request.ContentLength > int64(maxBody) || len(request.TransferEncoding) != 0 ||
		len(request.Trailer) != 0 || len(request.Header.Values("Transfer-Encoding")) != 0 ||
		len(request.Header.Values("Trailer")) != 0 || len(request.Header.Values("Upgrade")) != 0 ||
		connectionHasUpgrade(request.Header.Values("Connection")) {
		return false
	}
	contentLengths := request.Header.Values("Content-Length")
	if len(contentLengths) > 1 {
		return false
	}
	if len(contentLengths) == 1 {
		parsed, ok := parseContentLength(contentLengths[0])
		if !ok || parsed != request.ContentLength {
			return false
		}
	}
	return true
}

func readBody(request *http.Request, maxBody int) ([]byte, bool) {
	if request.Body == nil {
		return nil, request.ContentLength == 0
	}
	data, err := io.ReadAll(io.LimitReader(request.Body, int64(maxBody)+1))
	if err != nil || len(data) > maxBody || int64(len(data)) != request.ContentLength {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

func parseContentLength(value string) (int64, bool) {
	if value == "" {
		return 0, false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	return parsed, err == nil
}

func connectionHasUpgrade(values []string) bool {
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
				return true
			}
		}
	}
	return false
}

func validResponse(response brokerexchange.Response) bool {
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices &&
		len(response.Body) <= maxResponseBytes && validResponseContentType(response.ContentType)
}

func validResponseContentType(value string) bool {
	switch value {
	case "", "application/json", egresspolicy.GitUploadPackAdvertise, egresspolicy.GitUploadPackResult:
		return true
	default:
		return false
	}
}

func writeDenied(writer http.ResponseWriter) {
	if writer == nil {
		return
	}
	clearHeaders(writer)
	setFixedHeaders(writer)
	writer.Header().Set("Content-Length", "0")
	writer.WriteHeader(http.StatusForbidden)
}

func writeSuccess(writer http.ResponseWriter, response brokerexchange.Response) {
	if writer == nil {
		return
	}
	clearHeaders(writer)
	setFixedHeaders(writer)
	if response.ContentType != "" {
		writer.Header().Set("Content-Type", response.ContentType)
	}
	body := append([]byte(nil), response.Body...)
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	writer.WriteHeader(response.StatusCode)
	if len(body) != 0 {
		_, _ = writer.Write(body)
	}
}

func clearHeaders(writer http.ResponseWriter) {
	if writer == nil {
		return
	}
	for key := range writer.Header() {
		delete(writer.Header(), key)
	}
}

func setFixedHeaders(writer http.ResponseWriter) {
	if writer == nil {
		return
	}
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func visibleASCII(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < 0x20 || value[i] == 0x7f || value[i] >= 0x80 {
			return false
		}
	}
	return true
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
