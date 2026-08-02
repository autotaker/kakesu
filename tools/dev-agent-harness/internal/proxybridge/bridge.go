// Package proxybridge exposes one fixed IPv4 loopback endpoint and forwards
// each accepted connection to one trusted Unix socket. It is only a byte
// bridge: protocol parsing and authorization remain with the Unix service.
package proxybridge

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	listenNetwork = "tcp4"
	listenAddress = "127.0.0.1:0"
	unixNetwork   = "unix"

	minConcurrent = 1
	maxConcurrent = 64
	dialTimeout   = 5 * time.Second
)

// Error is a fixed public error. It never includes an address, Unix path, or
// lower-level error.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrListener     Error = "listener-error"
	ErrInvalidServe Error = "invalid-serve"
	ErrServer       Error = "server-error"
)

// Rules contains the only caller-supplied bridge settings. UnixSocketPath is
// trusted configuration, not Agent input. The TCP bind address and dial
// timeout are deliberately not configurable.
type Rules struct {
	UnixSocketPath string
	MaxConcurrent  int
}

// Server owns the listener created by New. A Server has exactly one Serve
// run; cancellation or any server-level failure closes that listener and
// drains all accepted connections.
type Server struct {
	listener       net.Listener
	dialer         contextDialer
	unixSocketPath string
	endpoint       string
	port           int
	maxConcurrent  int

	started   atomic.Bool
	closeOnce sync.Once
}

// Format keeps the retained path, listener, and dialer out of diagnostics.
func (s *Server) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "proxybridge.Server")
}

type listenFunc func(network, address string) (net.Listener, error)

type contextDialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

// New listens exactly once on an OS-assigned IPv4 loopback port. The returned
// endpoint is the only TCP address representation exposed to callers.
func New(r Rules) (*Server, string, error) {
	return newWithDependencies(r, net.Listen, &net.Dialer{})
}

// newWithDependencies is the hermetic test seam. Production always reaches it
// through New with net.Listen and net.Dialer.
func newWithDependencies(r Rules, listen listenFunc, dialer contextDialer) (*Server, string, error) {
	if !validRules(r) || listen == nil || isNil(dialer) {
		return nil, "", ErrInvalidRules
	}

	listener, err, panicked := safeListen(listen)
	if panicked || err != nil || isNil(listener) {
		if !isNil(listener) {
			safeClose(listener)
		}
		return nil, "", ErrListener
	}

	port, valid := listenerPort(listener)
	if !valid {
		safeClose(listener)
		return nil, "", ErrListener
	}
	endpoint := "http://127.0.0.1:" + strconv.Itoa(port)
	server := &Server{
		listener:       listener,
		dialer:         dialer,
		unixSocketPath: strings.Clone(r.UnixSocketPath),
		endpoint:       endpoint,
		port:           port,
		maxConcurrent:  r.MaxConcurrent,
	}
	return server, endpoint, nil
}

func validRules(r Rules) bool {
	return r.MaxConcurrent >= minConcurrent && r.MaxConcurrent <= maxConcurrent &&
		r.UnixSocketPath != "" && !strings.ContainsRune(r.UnixSocketPath, '\x00') &&
		filepath.IsAbs(r.UnixSocketPath) && filepath.Clean(r.UnixSocketPath) == r.UnixSocketPath
}

func safeListen(listen listenFunc) (listener net.Listener, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			listener = nil
			err = nil
			panicked = true
		}
	}()
	listener, err = listen(listenNetwork, listenAddress)
	return listener, err, false
}

func listenerPort(listener net.Listener) (port int, valid bool) {
	defer func() {
		if recover() != nil {
			port = 0
			valid = false
		}
	}()
	address := listener.Addr()
	if isNil(address) {
		return 0, false
	}
	tcpAddress, ok := address.(*net.TCPAddr)
	if !ok || tcpAddress == nil || tcpAddress.Zone != "" || tcpAddress.Port < 1 || tcpAddress.Port > 65535 {
		return 0, false
	}
	ipv4 := tcpAddress.IP.To4()
	if ipv4 == nil || !ipv4.Equal(net.IPv4(127, 0, 0, 1)) {
		return 0, false
	}
	return tcpAddress.Port, true
}

// Serve accepts connections until ctx is cancelled or the listener fails.
// Capacity is acquired before Accept, so no connection above the configured
// bound can reach the Unix dial phase.
func (s *Server) Serve(ctx context.Context) error {
	if s == nil {
		return ErrInvalidServe
	}
	if !s.started.CompareAndSwap(false, true) {
		return ErrInvalidServe
	}
	if !s.valid() || isNil(ctx) {
		s.closeListener()
		return ErrInvalidServe
	}
	if ctx.Err() != nil {
		s.closeListener()
		return nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	watcherStop := make(chan struct{})
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			s.closeListener()
			cancel()
		case <-watcherStop:
		}
	}()

	slots := make(chan struct{}, s.maxConcurrent)
	var connections sync.WaitGroup
	serverFailure := false

acceptLoop:
	for {
		select {
		case slots <- struct{}{}:
		case <-runCtx.Done():
			break acceptLoop
		}
		if runCtx.Err() != nil {
			<-slots
			break acceptLoop
		}

		conn, err, panicked := safeAccept(s.listener)
		if panicked || err != nil || isNil(conn) {
			<-slots
			if !isNil(conn) {
				safeClose(conn)
			}
			if ctx.Err() == nil {
				serverFailure = true
			}
			break acceptLoop
		}

		connections.Add(1)
		go s.serveConnection(runCtx, conn, slots, &connections)
	}

	// Stop new work first, then cancel active dials/copies and wait until every
	// connection has released its slot.
	s.closeListener()
	cancel()
	close(watcherStop)
	<-watcherDone
	connections.Wait()
	if serverFailure {
		return ErrServer
	}
	return nil
}

func (s *Server) valid() bool {
	return !isNil(s.listener) && !isNil(s.dialer) &&
		s.maxConcurrent >= minConcurrent && s.maxConcurrent <= maxConcurrent &&
		s.port >= 1 && s.port <= 65535 &&
		s.endpoint == "http://127.0.0.1:"+strconv.Itoa(s.port) && validRules(Rules{
		UnixSocketPath: s.unixSocketPath,
		MaxConcurrent:  s.maxConcurrent,
	})
}

func (s *Server) closeListener() {
	s.closeOnce.Do(func() { safeClose(s.listener) })
}

func safeAccept(listener net.Listener) (conn net.Conn, err error, panicked bool) {
	defer func() {
		if recover() != nil {
			conn = nil
			err = nil
			panicked = true
		}
	}()
	conn, err = listener.Accept()
	return conn, err, false
}

func (s *Server) serveConnection(runCtx context.Context, client net.Conn, slots chan struct{}, connections *sync.WaitGroup) {
	defer connections.Done()
	defer func() { <-slots }()
	defer safeClose(client)
	defer func() { _ = recover() }()

	dialCtx, cancelDial := context.WithTimeout(runCtx, dialTimeout)
	upstream, ok := safeDial(s.dialer, dialCtx, s.unixSocketPath)
	cancelDial()
	if !ok {
		if !isNil(upstream) {
			safeClose(upstream)
		}
		return
	}
	defer safeClose(upstream)

	bridgeStreams(runCtx, client, upstream)
}

func safeDial(dialer contextDialer, ctx context.Context, path string) (conn net.Conn, ok bool) {
	defer func() {
		if recover() != nil {
			conn = nil
			ok = false
		}
	}()
	var err error
	conn, err = dialer.DialContext(ctx, unixNetwork, path)
	return conn, err == nil && !isNil(conn)
}

type copyResult struct {
	failed bool
}

// bridgeStreams waits for both directions after normal EOF. A failed copy,
// failed half-close, or cancellation closes both endpoints to unblock the
// other worker before it is drained.
func bridgeStreams(ctx context.Context, client, upstream net.Conn) {
	results := make(chan copyResult, 2)
	go copyDirection(upstream, client, results)
	go copyDirection(client, upstream, results)

	remaining := 2
	failed := false
	for remaining > 0 {
		if failed {
			<-results
			remaining--
			continue
		}
		select {
		case result := <-results:
			remaining--
			if result.failed && remaining > 0 {
				failed = true
				safeClose(client)
				safeClose(upstream)
			}
		case <-ctx.Done():
			failed = true
			safeClose(client)
			safeClose(upstream)
		}
	}
}

func copyDirection(destination, source net.Conn, results chan<- copyResult) {
	result := copyResult{}
	defer func() {
		if recover() != nil {
			result.failed = true
		}
		results <- result
	}()
	if _, err := io.Copy(destination, source); err != nil {
		result.failed = true
		return
	}
	if !closeWrite(destination) {
		result.failed = true
	}
}

func closeWrite(conn net.Conn) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	writer, supported := conn.(interface{ CloseWrite() error })
	if !supported || isNil(writer) {
		safeClose(conn)
		return false
	}
	return writer.CloseWrite() == nil
}

func safeClose(closer io.Closer) {
	if isNil(closer) {
		return
	}
	defer func() { _ = recover() }()
	_ = closer.Close()
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
