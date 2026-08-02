package proxybridge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testSocketPath = "/run/dev-agent-harness/egress.sock"

func TestNewUsesOnlyFixedLoopbackAndReturnsCanonicalEndpoint(t *testing.T) {
	listener := newFakeListener(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 43219})
	var calls atomic.Int32
	server, endpoint, err := newWithDependencies(validRulesForTest(), func(network, address string) (net.Listener, error) {
		calls.Add(1)
		if network != "tcp4" || address != "127.0.0.1:0" {
			t.Fatalf("listen called with %q, %q", network, address)
		}
		return listener, nil
	}, dialFunc(func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("unused")
	}))
	if err != nil || server == nil {
		t.Fatalf("newWithDependencies() = %#v, %q, %v", server, endpoint, err)
	}
	if calls.Load() != 1 || endpoint != "http://127.0.0.1:43219" {
		t.Fatalf("listen calls=%d endpoint=%q", calls.Load(), endpoint)
	}
	if got := fmt.Sprintf("%v", server); got != "proxybridge.Server" || strings.Contains(got, testSocketPath) {
		t.Fatalf("server format = %q", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := server.Serve(ctx); err != nil {
		t.Fatalf("cancelled Serve() = %v", err)
	}
	if listener.closeCount.Load() != 1 || listener.acceptCount.Load() != 0 {
		t.Fatalf("listener closes=%d accepts=%d", listener.closeCount.Load(), listener.acceptCount.Load())
	}
}

func TestNewRejectsInvalidRulesBeforeListen(t *testing.T) {
	typedNilDialer := (*recordingDialer)(nil)
	tests := map[string]struct {
		rules  Rules
		dialer contextDialer
	}{
		"empty":         {rules: Rules{MaxConcurrent: 1}, dialer: &recordingDialer{}},
		"relative":      {rules: Rules{UnixSocketPath: "run/egress.sock", MaxConcurrent: 1}, dialer: &recordingDialer{}},
		"unclean":       {rules: Rules{UnixSocketPath: "/run/../run/egress.sock", MaxConcurrent: 1}, dialer: &recordingDialer{}},
		"nul":           {rules: Rules{UnixSocketPath: "/run/egress\x00.sock", MaxConcurrent: 1}, dialer: &recordingDialer{}},
		"zero capacity": {rules: Rules{UnixSocketPath: testSocketPath}, dialer: &recordingDialer{}},
		"high capacity": {rules: Rules{UnixSocketPath: testSocketPath, MaxConcurrent: 65}, dialer: &recordingDialer{}},
		"nil dialer":    {rules: validRulesForTest(), dialer: nil},
		"typed dialer":  {rules: validRulesForTest(), dialer: typedNilDialer},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			server, endpoint, err := newWithDependencies(test.rules, func(string, string) (net.Listener, error) {
				calls.Add(1)
				return newValidListener(), nil
			}, test.dialer)
			if server != nil || endpoint != "" || !errors.Is(err, ErrInvalidRules) || calls.Load() != 0 {
				t.Fatalf("result=%#v, %q, %v; listens=%d", server, endpoint, err, calls.Load())
			}
		})
	}

	server, endpoint, err := newWithDependencies(validRulesForTest(), nil, &recordingDialer{})
	if server != nil || endpoint != "" || !errors.Is(err, ErrInvalidRules) {
		t.Fatalf("nil listen function = %#v, %q, %v", server, endpoint, err)
	}
}

func TestNewRejectsBadListenerAndSanitizesFailure(t *testing.T) {
	lower := "secret-path-and-address"
	typedNil := (*fakeListener)(nil)
	tests := map[string]func() (net.Listener, error){
		"error": func() (net.Listener, error) { return nil, errors.New(lower) },
		"listener and error": func() (net.Listener, error) {
			return newValidListener(), errors.New(lower)
		},
		"nil":       func() (net.Listener, error) { return nil, nil },
		"typed nil": func() (net.Listener, error) { return typedNil, nil },
		"panic":     func() (net.Listener, error) { panic(lower) },
	}
	for name, result := range tests {
		t.Run(name, func(t *testing.T) {
			var returned *fakeListener
			server, endpoint, err := newWithDependencies(validRulesForTest(), func(string, string) (net.Listener, error) {
				listener, listenErr := result()
				returned, _ = listener.(*fakeListener)
				return listener, listenErr
			}, &recordingDialer{})
			if server != nil || endpoint != "" || !errors.Is(err, ErrListener) || strings.Contains(err.Error(), lower) {
				t.Fatalf("result=%#v, %q, %v", server, endpoint, err)
			}
			if returned != nil && returned.closeCount.Load() != 1 {
				t.Fatalf("returned listener closes = %d", returned.closeCount.Load())
			}
		})
	}
}

func TestNewValidatesReturnedAddressAndClosesListener(t *testing.T) {
	typedNilAddress := (*net.TCPAddr)(nil)
	tests := map[string]net.Addr{
		"nil":          nil,
		"typed nil":    typedNilAddress,
		"wrong type":   fixedAddr("127.0.0.1:4000"),
		"wildcard":     &net.TCPAddr{IP: net.IPv4zero, Port: 4000},
		"ipv6":         &net.TCPAddr{IP: net.ParseIP("::1"), Port: 4000},
		"non loopback": &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 4000},
		"zero port":    &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0},
		"high port":    &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 65536},
		"zone":         &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4000, Zone: "zone"},
	}
	for name, address := range tests {
		t.Run(name, func(t *testing.T) {
			listener := newFakeListener(address)
			server, endpoint, err := newWithDependencies(validRulesForTest(), func(string, string) (net.Listener, error) {
				return listener, nil
			}, &recordingDialer{})
			if server != nil || endpoint != "" || !errors.Is(err, ErrListener) {
				t.Fatalf("result=%#v, %q, %v", server, endpoint, err)
			}
			if listener.closeCount.Load() != 1 {
				t.Fatalf("listener closes = %d", listener.closeCount.Load())
			}
		})
	}

	listener := newFakeListener(panicAddr{})
	server, endpoint, err := newWithDependencies(validRulesForTest(), func(string, string) (net.Listener, error) {
		return listener, nil
	}, &recordingDialer{})
	if server != nil || endpoint != "" || !errors.Is(err, ErrListener) || listener.closeCount.Load() != 1 {
		t.Fatalf("panicking Addr result=%#v, %q, %v closes=%d", server, endpoint, err, listener.closeCount.Load())
	}
}

func TestServeRejectsInvalidStateAndSecondRun(t *testing.T) {
	var zero Server
	if err := zero.Serve(context.Background()); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("zero Serve() = %v", err)
	}
	server, _, listener := newTestServer(t, validRulesForTest(), &recordingDialer{})
	if err := server.Serve(nil); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("nil context Serve() = %v", err)
	}
	if listener.closeCount.Load() != 1 || listener.acceptCount.Load() != 0 {
		t.Fatalf("closes=%d accepts=%d", listener.closeCount.Load(), listener.acceptCount.Load())
	}
	if err := server.Serve(context.Background()); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("second Serve() = %v", err)
	}
}

func TestServeDialsOnceWithFixedUnixPathAndDeadline(t *testing.T) {
	upstreamPeers := make(chan net.Conn, 1)
	dialer := &recordingDialer{dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > dialTimeout {
			return nil, errors.New("missing fixed deadline")
		}
		bridge, peer := net.Pipe()
		upstreamPeers <- peer
		return bridge, nil
	}}
	server, _, listener := newTestServer(t, validRulesForTest(), dialer)
	bridgeClient, client := net.Pipe()
	listener.add(bridgeClient, nil)
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := serveAsync(server, ctx)
	upstreamPeer := <-upstreamPeers
	cancel()
	if err := waitError(t, done); err != nil {
		t.Fatalf("Serve() = %v", err)
	}
	calls := dialer.snapshot()
	if len(calls) != 1 || calls[0].network != unixNetwork || calls[0].address != testSocketPath {
		t.Fatalf("dial calls = %#v", calls)
	}
	assertConnectionClosed(t, client)
	assertConnectionClosed(t, upstreamPeer)
}

func TestDialFailureIsLocalNoReadRetryOrLeak(t *testing.T) {
	lower := "dial leaked " + testSocketPath
	dialer := &recordingDialer{dial: func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New(lower)
	}}
	server, _, listener := newTestServer(t, validRulesForTest(), dialer)
	client := newScriptConn(nil, nil)
	listener.add(client, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := serveAsync(server, ctx)
	waitClosed(t, client.closed)
	if client.readCount.Load() != 0 || dialer.count() != 1 {
		t.Fatalf("client reads=%d dials=%d", client.readCount.Load(), dialer.count())
	}
	cancel()
	if err := waitError(t, done); err != nil || strings.Contains(fmt.Sprint(err), lower) {
		t.Fatalf("Serve() = %v", err)
	}
}

func TestCapacityIsAcquiredBeforeAcceptAndDial(t *testing.T) {
	entered := make(chan struct{}, 2)
	releaseFirst := make(chan struct{})
	var sequence atomic.Int32
	dialer := &recordingDialer{dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		call := sequence.Add(1)
		entered <- struct{}{}
		if call == 1 {
			select {
			case <-releaseFirst:
			case <-ctx.Done():
			}
		}
		return nil, errors.New("connection denied")
	}}
	rules := validRulesForTest()
	rules.MaxConcurrent = 1
	server, _, listener := newTestServer(t, rules, dialer)
	first, firstPeer := net.Pipe()
	second, secondPeer := net.Pipe()
	defer firstPeer.Close()
	defer secondPeer.Close()
	listener.add(first, nil)
	listener.add(second, nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := serveAsync(server, ctx)
	waitSignal(t, entered, "first dial")
	time.Sleep(30 * time.Millisecond)
	if listener.acceptCount.Load() != 1 || dialer.count() != 1 {
		t.Fatalf("while saturated: accepts=%d dials=%d", listener.acceptCount.Load(), dialer.count())
	}
	close(releaseFirst)
	waitSignal(t, entered, "second dial")
	if dialer.count() != 2 {
		t.Fatalf("dials after release = %d", dialer.count())
	}
	cancel()
	if err := waitError(t, done); err != nil {
		t.Fatalf("Serve() = %v", err)
	}
}

func TestUnexpectedAcceptFailureCancelsAndDrainsActiveConnection(t *testing.T) {
	upstreamBridge, upstreamPeer := net.Pipe()
	dialed := make(chan struct{})
	dialer := &recordingDialer{dial: func(context.Context, string, string) (net.Conn, error) {
		close(dialed)
		return upstreamBridge, nil
	}}
	rules := validRulesForTest()
	rules.MaxConcurrent = 2
	server, _, listener := newTestServer(t, rules, dialer)
	clientBridge, clientPeer := net.Pipe()
	listener.add(clientBridge, nil)
	listener.add(nil, errors.New("accept lower detail"))
	done := serveAsync(server, context.Background())
	waitSignal(t, dialed, "dial")
	if err := waitError(t, done); !errors.Is(err, ErrServer) || err.Error() != "server-error" {
		t.Fatalf("Serve() = %v", err)
	}
	if listener.closeCount.Load() != 1 {
		t.Fatalf("listener closes = %d", listener.closeCount.Load())
	}
	assertConnectionClosed(t, clientPeer)
	assertConnectionClosed(t, upstreamPeer)
}

func TestAcceptNilAndPanicAreFixedServerFailures(t *testing.T) {
	var typedNil *scriptConn
	for name, result := range map[string]acceptResult{
		"nil":       {},
		"typed nil": {conn: typedNil},
		"panic":     {panicValue: "private accept panic"},
	} {
		t.Run(name, func(t *testing.T) {
			server, _, listener := newTestServer(t, validRulesForTest(), &recordingDialer{})
			listener.results <- result
			if err := server.Serve(context.Background()); !errors.Is(err, ErrServer) || err.Error() != "server-error" {
				t.Fatalf("Serve() = %v", err)
			}
		})
	}
}

func TestRawBidirectionalStreamAndBothHalfCloses(t *testing.T) {
	request := []byte{'C', 'O', 'N', 'N', 'E', 'C', 'T', 0, 0xff, '\n'}
	response := []byte{0x16, 0x03, 0x03, 0, 5, 'r', 'e', 'p', 'l', 'y'}
	client := newScriptConn(request, nil)
	upstream := newScriptConn(response, nil)
	// Upstream response becomes readable only after its write half is closed,
	// proving that client EOF does not discard the reverse direction.
	upstream.readGate = upstream.closeWriteCalled

	done := make(chan struct{})
	go func() {
		bridgeStreams(context.Background(), client, upstream)
		close(done)
	}()
	waitSignal(t, done, "bidirectional stream")
	if got := upstream.writtenBytes(); !bytes.Equal(got, request) {
		t.Fatalf("upstream bytes = %v", got)
	}
	if got := client.writtenBytes(); !bytes.Equal(got, response) {
		t.Fatalf("client bytes = %v", got)
	}
	if upstream.closeWriteCount.Load() != 1 || client.closeWriteCount.Load() != 1 {
		t.Fatalf("CloseWrite upstream=%d client=%d", upstream.closeWriteCount.Load(), client.closeWriteCount.Load())
	}
}

func TestStreamCancellationAndCopyFailureCloseBothEnds(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		client := newScriptConn(nil, make(chan struct{}))
		upstream := newScriptConn(nil, make(chan struct{}))
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { bridgeStreams(ctx, client, upstream); close(done) }()
		cancel()
		waitSignal(t, done, "cancelled stream")
		waitClosed(t, client.closed)
		waitClosed(t, upstream.closed)
	})
	t.Run("copy error", func(t *testing.T) {
		client := newScriptConn(nil, nil)
		client.readErr = errors.New("private copy error")
		upstream := newScriptConn(nil, make(chan struct{}))
		upstream.closeErr = errors.New("private close error")
		done := make(chan struct{})
		go func() { bridgeStreams(context.Background(), client, upstream); close(done) }()
		waitSignal(t, done, "failed stream")
		waitClosed(t, client.closed)
		waitClosed(t, upstream.closed)
	})
	t.Run("half-close error", func(t *testing.T) {
		client := newScriptConn([]byte("request"), nil)
		upstream := newScriptConn(nil, make(chan struct{}))
		upstream.closeWriteErr = errors.New("private half-close error")
		done := make(chan struct{})
		go func() { bridgeStreams(context.Background(), client, upstream); close(done) }()
		waitSignal(t, done, "half-close failed stream")
		waitClosed(t, client.closed)
		waitClosed(t, upstream.closed)
	})
}

func validRulesForTest() Rules {
	return Rules{UnixSocketPath: testSocketPath, MaxConcurrent: 2}
}

func newValidListener() *fakeListener {
	return newFakeListener(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 41001})
}

func newTestServer(t *testing.T, rules Rules, dialer contextDialer) (*Server, string, *fakeListener) {
	t.Helper()
	listener := newValidListener()
	server, endpoint, err := newWithDependencies(rules, func(network, address string) (net.Listener, error) {
		if network != listenNetwork || address != listenAddress {
			t.Fatalf("listen = %q, %q", network, address)
		}
		return listener, nil
	}, dialer)
	if err != nil {
		t.Fatal(err)
	}
	return server, endpoint, listener
}

func serveAsync(server *Server, ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	return done
}

func waitError(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Serve")
		return nil
	}
}

func waitSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func waitClosed(t *testing.T, closed <-chan struct{}) {
	t.Helper()
	waitSignal(t, closed, "connection close")
}

func assertConnectionClosed(t *testing.T, conn net.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, err := conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("connection remained open")
	}
	_ = conn.Close()
}

type dialCall struct {
	network string
	address string
}

type dialFunc func(context.Context, string, string) (net.Conn, error)

func (f dialFunc) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return f(ctx, network, address)
}

type recordingDialer struct {
	mu    sync.Mutex
	calls []dialCall
	dial  dialFunc
}

func (d *recordingDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, dialCall{network: network, address: address})
	dial := d.dial
	d.mu.Unlock()
	if dial == nil {
		return nil, errors.New("dial not configured")
	}
	return dial(ctx, network, address)
}

func (d *recordingDialer) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.calls)
}

func (d *recordingDialer) snapshot() []dialCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]dialCall(nil), d.calls...)
}

type acceptResult struct {
	conn       net.Conn
	err        error
	panicValue any
}

type fakeListener struct {
	address     net.Addr
	results     chan acceptResult
	closed      chan struct{}
	closeOnce   sync.Once
	closeCount  atomic.Int32
	acceptCount atomic.Int32
}

func newFakeListener(address net.Addr) *fakeListener {
	return &fakeListener{address: address, results: make(chan acceptResult, 16), closed: make(chan struct{})}
}

func (l *fakeListener) Accept() (net.Conn, error) {
	l.acceptCount.Add(1)
	select {
	case result := <-l.results:
		if result.panicValue != nil {
			panic(result.panicValue)
		}
		return result.conn, result.err
	case <-l.closed:
		return nil, errors.New("listener closed")
	}
}

func (l *fakeListener) Close() error {
	l.closeCount.Add(1)
	l.closeOnce.Do(func() { close(l.closed) })
	return nil
}

func (l *fakeListener) Addr() net.Addr { return l.address }

func (l *fakeListener) add(conn net.Conn, err error) {
	l.results <- acceptResult{conn: conn, err: err}
}

type fixedAddr string

func (a fixedAddr) Network() string { return "tcp" }
func (a fixedAddr) String() string  { return string(a) }

type panicAddr struct{}

func (panicAddr) Network() string { panic("private address") }
func (panicAddr) String() string  { panic("private address") }

// scriptConn is an in-memory half-close-capable connection. It can gate reads
// to make direction ordering observable without opening a real socket.
type scriptConn struct {
	readData []byte
	readGate <-chan struct{}
	readErr  error

	mu      sync.Mutex
	readPos int
	written bytes.Buffer

	closed           chan struct{}
	closeOnce        sync.Once
	closeWriteCalled chan struct{}
	closeWriteOnce   sync.Once
	closeErr         error
	closeWriteErr    error
	readCount        atomic.Int32
	closeWriteCount  atomic.Int32
}

func newScriptConn(readData []byte, readGate <-chan struct{}) *scriptConn {
	return &scriptConn{
		readData:         append([]byte(nil), readData...),
		readGate:         readGate,
		closed:           make(chan struct{}),
		closeWriteCalled: make(chan struct{}),
	}
}

func (c *scriptConn) Read(p []byte) (int, error) {
	c.readCount.Add(1)
	if c.readGate != nil {
		select {
		case <-c.readGate:
		case <-c.closed:
			return 0, net.ErrClosed
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.readErr != nil {
		return 0, c.readErr
	}
	if c.readPos == len(c.readData) {
		return 0, io.EOF
	}
	n := copy(p, c.readData[c.readPos:])
	c.readPos += n
	return n, nil
}

func (c *scriptConn) Write(p []byte) (int, error) {
	select {
	case <-c.closed:
		return 0, net.ErrClosed
	default:
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.written.Write(p)
}

func (c *scriptConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.closeErr
}

func (c *scriptConn) CloseWrite() error {
	c.closeWriteCount.Add(1)
	c.closeWriteOnce.Do(func() { close(c.closeWriteCalled) })
	return c.closeWriteErr
}

func (c *scriptConn) LocalAddr() net.Addr              { return fixedAddr("local") }
func (c *scriptConn) RemoteAddr() net.Addr             { return fixedAddr("remote") }
func (c *scriptConn) SetDeadline(time.Time) error      { return nil }
func (c *scriptConn) SetReadDeadline(time.Time) error  { return nil }
func (c *scriptConn) SetWriteDeadline(time.Time) error { return nil }

func (c *scriptConn) writtenBytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.written.Bytes()...)
}
