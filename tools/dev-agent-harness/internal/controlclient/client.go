// Package controlclient speaks the deliberately small capability-control
// protocol exposed on the fixed egress Unix socket.
package controlclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	phaseTimeout      = 5 * time.Second
	maxResponseHeader = 1024
	issueBodyLength   = 60
)

// Error is intentionally context-free. It never retains a socket path,
// repository, handle, wire byte, or lower-level error.
type Error string

func (e Error) Error() string { return string(e) }

const ErrControl Error = "capability-control-client-failed"

var dialControl = func(ctx context.Context, network, address string) (net.Conn, error) {
	var dialer net.Dialer
	return dialer.DialContext(ctx, network, address)
}

// Issue obtains one GitHub Git-read capability for repository.
func Issue(socketPath, repository string) (string, error) {
	if !validSocket(socketPath) || !canonicalRepository(repository) {
		return "", ErrControl
	}
	body := `{"provider":"github","repository":"` + repository + `","operation":"github-git-read"}`
	request := "POST /v1/capabilities HTTP/1.1\r\nContent-Length: " + strconv.Itoa(len(body)) +
		"\r\nContent-Type: application/json\r\n\r\n" + body
	response, err := exchange(socketPath, request, true)
	if err != nil {
		return "", ErrControl
	}
	return response, nil
}

// Revoke invalidates exactly one canonical opaque handle.
func Revoke(socketPath, handle string) error {
	if !validSocket(socketPath) || !canonicalHandle(handle) {
		return ErrControl
	}
	request := "DELETE /v1/capabilities/" + handle + " HTTP/1.1\r\nContent-Length: 0\r\n\r\n"
	if _, err := exchange(socketPath, request, false); err != nil {
		return ErrControl
	}
	return nil
}

func exchange(socketPath, request string, issue bool) (result string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), phaseTimeout)
	conn, dialErr := dialControl(ctx, "unix", socketPath)
	cancel()
	if dialErr != nil || conn == nil {
		if conn != nil {
			_ = conn.Close()
		}
		return "", ErrControl
	}
	closed := false
	defer func() {
		if !closed {
			_ = conn.Close()
		}
	}()

	if conn.SetWriteDeadline(time.Now().Add(phaseTimeout)) != nil || !writeAll(conn, []byte(request)) {
		return "", ErrControl
	}
	if conn.SetReadDeadline(time.Now().Add(phaseTimeout)) != nil {
		return "", ErrControl
	}
	header, body, ok := readResponse(conn)
	if !ok {
		return "", ErrControl
	}
	if issue {
		length, ok := issueHeaderLength(header)
		if !ok || len(body) != length {
			return "", ErrControl
		}
		if len(body) != issueBodyLength || !bytes.HasPrefix(body, []byte(`{"handle":"`)) || !bytes.HasSuffix(body, []byte(`"}`)) {
			return "", ErrControl
		}
		handle := string(body[len(`{"handle":"`) : len(body)-len(`"}`)])
		if !canonicalHandle(handle) || string(body) != `{"handle":"`+handle+`"}` {
			return "", ErrControl
		}
		result = handle
	} else if string(header) != "HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: close\r\n\r\n" || len(body) != 0 {
		return "", ErrControl
	}
	closeErr := conn.Close()
	closed = true
	if closeErr != nil {
		return "", ErrControl
	}
	return result, nil
}

func writeAll(writer io.Writer, data []byte) bool {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil || n == 0 {
			return false
		}
	}
	return true
}

func readResponse(reader io.Reader) ([]byte, []byte, bool) {
	data, err := io.ReadAll(io.LimitReader(reader, maxResponseHeader+issueBodyLength+1))
	if err != nil || len(data) > maxResponseHeader+issueBodyLength {
		return nil, nil, false
	}
	marker := []byte("\r\n\r\n")
	index := bytes.Index(data, marker)
	if index < 0 || index+len(marker) > maxResponseHeader {
		return nil, nil, false
	}
	headerEnd := index + len(marker)
	return data[:headerEnd], data[headerEnd:], true
}

func issueHeaderLength(header []byte) (int, bool) {
	prefix := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: "
	suffix := "\r\nConnection: close\r\n\r\n"
	text := string(header)
	if !strings.HasPrefix(text, prefix) || !strings.HasSuffix(text, suffix) {
		return 0, false
	}
	value := strings.TrimSuffix(strings.TrimPrefix(text, prefix), suffix)
	if value == "" || len(value) > 3 || len(value) > 1 && value[0] == '0' {
		return 0, false
	}
	for i := range value {
		if value[i] < '0' || value[i] > '9' {
			return 0, false
		}
	}
	length, parseErr := strconv.Atoi(value)
	return length, parseErr == nil && length == issueBodyLength
}

func validSocket(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path
}

func canonicalRepository(repository string) bool {
	parts := strings.Split(repository, "/")
	return len(parts) == 2 && canonicalRepositoryPart(parts[0]) && canonicalRepositoryPart(parts[1])
}

func canonicalRepositoryPart(part string) bool {
	if part == "" || part == "." || part == ".." || !lowerAlphaNumeric(part[0]) {
		return false
	}
	for i := 1; i < len(part); i++ {
		if c := part[i]; !lowerAlphaNumeric(c) && c != '.' && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func lowerAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func canonicalHandle(handle string) bool {
	if len(handle) != len("cap_")+43 || !strings.HasPrefix(handle, "cap_") {
		return false
	}
	raw, decodeErr := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(handle, "cap_"))
	return decodeErr == nil && len(raw) == 32 && "cap_"+base64.RawURLEncoding.EncodeToString(raw) == handle
}
