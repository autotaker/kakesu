package brokerlistener

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

var testSubject = Subject{AgentInstanceID: "agent-1", UID: 1000, WorkspaceID: "workspace-1"}

func TestNewBoundsTypedNilAndFormat(t *testing.T) {
	var nilBinder *testBinder
	var nilSession *testSession
	valid := Rules{Binder: &testBinder{}, Session: &testSession{}, MaxConcurrent: 1}
	tests := map[string]Rules{
		"zero":          {},
		"nil binder":    {Session: valid.Session, MaxConcurrent: 1},
		"typed binder":  {Binder: nilBinder, Session: valid.Session, MaxConcurrent: 1},
		"nil session":   {Binder: valid.Binder, MaxConcurrent: 1},
		"typed session": {Binder: valid.Binder, Session: nilSession, MaxConcurrent: 1},
		"low":           {Binder: valid.Binder, Session: valid.Session, MaxConcurrent: 0},
		"high":          {Binder: valid.Binder, Session: valid.Session, MaxConcurrent: maxConcurrent + 1},
	}
	for name, rules := range tests {
		t.Run(name, func(t *testing.T) {
			server, err := New(rules)
			if server != nil || !errors.Is(err, ErrInvalidRules) {
				t.Fatalf("New() = %#v, %v", server, err)
			}
		})
	}
	server, err := New(valid)
	if err != nil || server == nil {
		t.Fatalf("valid New() = %#v, %v", server, err)
	}
	if got := fmt.Sprintf("%v", server); got != "brokerlistener.Server" {
		t.Fatalf("server format = %q", got)
	}
	if got := fmt.Sprintf("%v", Resolver{}); got != "brokerlistener.Resolver" {
		t.Fatalf("resolver format = %q", got)
	}
}

func TestResolverRejectsMissingWrongInvalidAndCancelled(t *testing.T) {
	resolver := Resolver{}
	contexts := map[string]context.Context{
		"nil":        nil,
		"background": context.Background(),
		"wrong type": context.WithValue(context.Background(), subjectContextKey{}, "subject"),
		"invalid":    context.WithValue(context.Background(), subjectContextKey{}, Subject{UID: 0, AgentInstanceID: "bad", WorkspaceID: "bad"}),
	}
	for name, ctx := range contexts {
		t.Run(name, func(t *testing.T) {
			if subject, err := resolver.Resolve(ctx); subject != (Subject{}) || !errors.Is(err, ErrResolver) {
				t.Fatalf("Resolve() = %#v, %v", subject, err)
			}
		})
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if subject, err := resolver.Resolve(cancelled); subject != (Subject{}) || !errors.Is(err, ErrResolver) {
		t.Fatalf("cancelled Resolve() = %#v, %v", subject, err)
	}
}

func TestResolverCopiesSubject(t *testing.T) {
	resolver := Resolver{}
	agentBytes := []byte("agent")
	workspaceBytes := []byte("workspace")
	input := Subject{
		AgentInstanceID: unsafe.String(&agentBytes[0], len(agentBytes)),
		UID:             1000,
		WorkspaceID:     unsafe.String(&workspaceBytes[0], len(workspaceBytes)),
	}
	ctx := context.WithValue(context.Background(), subjectContextKey{}, input)
	got, err := resolver.Resolve(ctx)
	if err != nil || got != input {
		t.Fatalf("Resolve() = %#v, %v", got, err)
	}
	if unsafe.StringData(got.AgentInstanceID) == unsafe.StringData(input.AgentInstanceID) ||
		unsafe.StringData(got.WorkspaceID) == unsafe.StringData(input.WorkspaceID) {
		t.Fatal("subject fields were not copied")
	}
	agentBytes[0] = 'X'
	workspaceBytes[0] = 'Y'
	if got.AgentInstanceID != "agent" || got.WorkspaceID != "workspace" {
		t.Fatalf("resolved copy changed after source mutation: %#v", got)
	}
}

func TestServeRejectsInvalidInputsWithoutPanic(t *testing.T) {
	server, err := New(Rules{Binder: &testBinder{}, Session: &testSession{}, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	listener := newFakeListener()
	defer listener.closeForTest()
	var nilContext context.Context
	var nilListener net.Listener
	if err := server.Serve(nilContext, listener); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("nil context error = %v", err)
	}
	if err := server.Serve(context.Background(), nilListener); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("nil listener error = %v", err)
	}
	var zero Server
	if err := zero.Serve(context.Background(), listener); !errors.Is(err, ErrInvalidServe) {
		t.Fatalf("zero server error = %v", err)
	}
}

func TestServeAlreadyCancelledDoesNotAccept(t *testing.T) {
	contexts := map[string]context.Context{}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	contexts["cancel"] = cancelled
	deadline, deadlineCancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer deadlineCancel()
	contexts["deadline"] = deadline
	for name, ctx := range contexts {
		t.Run(name, func(t *testing.T) {
			listener := newFakeListener()
			server, err := New(Rules{Binder: &testBinder{}, Session: &testSession{}, MaxConcurrent: 1})
			if err != nil {
				t.Fatal(err)
			}
			if err := server.Serve(ctx, listener); err != nil {
				t.Fatalf("already-stopped Serve = %v", err)
			}
			if listener.acceptCount() != 0 || !listener.wasClosed() {
				t.Fatalf("Accept=%d closed=%v", listener.acceptCount(), listener.wasClosed())
			}
		})
	}
}

func TestServeAcquiresSlotBeforeAcceptAndCapsSessions(t *testing.T) {
	listener := newFakeListener()
	firstServer, firstClient := net.Pipe()
	secondServer, secondClient := net.Pipe()
	defer firstClient.Close()
	defer secondClient.Close()
	listener.add(firstServer)
	listener.add(secondServer)

	entered := make(chan struct{})
	release := make(chan struct{})
	var active atomic.Int32
	var maximum atomic.Int32
	var sessions atomic.Int32
	session := &testSession{serve: func(ctx context.Context, conn net.Conn) error {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		sessions.Add(1)
		close(entered)
		<-release
		active.Add(-1)
		return nil
	}}
	binder := &testBinder{subject: testSubject}
	server, err := New(Rules{Binder: binder, Session: session, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(context.Background(), listener) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first session did not start")
	}
	// Consume the first accept notification before checking for a second.
	<-listener.accepted
	select {
	case <-listener.accepted:
		t.Fatal("second Accept happened while slot was occupied")
	case <-time.After(30 * time.Millisecond):
	}
	if got := sessions.Load(); got != 1 || maximum.Load() > 1 {
		t.Fatalf("sessions=%d maximum=%d", got, maximum.Load())
	}
	close(release)
	waitFor(t, func() bool { return listener.acceptCount() >= 2 })
	cancelListener(t, listener)
	if err := <-serveDone; err == nil {
		t.Fatal("external listener close should be a server error")
	}
	if got := sessions.Load(); got != 2 {
		t.Fatalf("sessions after release = %d", got)
	}
}

func TestServeBinderBeforeSessionAndPrivateIdentityIsolation(t *testing.T) {
	listener := newFakeListener()
	connections := make([]net.Conn, 3)
	clients := make([]net.Conn, 3)
	trackedConnections := make([]*trackedConn, 3)
	for i := range connections {
		raw, client := net.Pipe()
		trackedConnections[i] = &trackedConn{Conn: raw, closed: make(chan struct{})}
		connections[i], clients[i] = trackedConnections[i], client
		listener.add(connections[i])
		defer clients[i].Close()
	}
	var mu sync.Mutex
	order := make([]string, 0, len(connections)*2)
	bound := make(map[net.Conn]bool, len(connections))
	var sessionBeforeBind atomic.Bool
	seen := make(chan Subject, len(connections))
	var identity atomic.Int32
	binder := &testBinder{bind: func(ctx context.Context, conn net.Conn) (Subject, error) {
		mu.Lock()
		order = append(order, "bind")
		bound[conn] = true
		mu.Unlock()
		id := identity.Add(1)
		return Subject{AgentInstanceID: fmt.Sprintf("agent-%d", id), UID: 1000, WorkspaceID: "workspace"}, nil
	}}
	session := &testSession{serve: func(ctx context.Context, conn net.Conn) error {
		mu.Lock()
		order = append(order, "session")
		if !bound[conn] {
			sessionBeforeBind.Store(true)
		}
		mu.Unlock()
		subject, err := (Resolver{}).Resolve(ctx)
		if err != nil {
			return err
		}
		seen <- subject
		return nil
	}}
	server, err := New(Rules{Binder: binder, Session: session, MaxConcurrent: 3})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background(), listener) }()
	got := make([]Subject, 0, len(connections))
	for range connections {
		select {
		case subject := <-seen:
			got = append(got, subject)
		case <-time.After(time.Second):
			t.Fatal("session did not observe identity")
		}
	}
	if len(order) != 6 {
		t.Fatalf("callback order = %v", order)
	}
	if order[0] != "bind" {
		t.Fatalf("first callback was %q, want bind", order[0])
	}
	if sessionBeforeBind.Load() {
		t.Fatal("session ran before its connection was bound")
	}
	if len(got) != 3 {
		t.Fatalf("subjects = %#v", got)
	}
	if got[0] == got[1] || got[1] == got[2] || got[0] == got[2] {
		t.Fatal("parallel subjects were shared")
	}
	cancelListener(t, listener)
	if err := <-done; err == nil {
		t.Fatal("external listener close should fail Serve")
	}
	for _, conn := range trackedConnections {
		if !waitClosed(conn) {
			t.Fatal("successful session connection was not closed")
		}
	}
}

func TestServeRejectsInvalidOrBinderErrorsAndContinues(t *testing.T) {
	listener := newFakeListener()
	var clientConns []net.Conn
	var trackedConns []*trackedConn
	for range 4 {
		rawServer, clientConn := net.Pipe()
		serverConn := &trackedConn{Conn: rawServer, closed: make(chan struct{})}
		clientConns = append(clientConns, clientConn)
		trackedConns = append(trackedConns, serverConn)
		listener.add(serverConn)
		defer clientConn.Close()
	}
	var bindCalls atomic.Int32
	binder := &testBinder{bind: func(context.Context, net.Conn) (Subject, error) {
		call := bindCalls.Add(1)
		switch call {
		case 1:
			return Subject{}, errors.New("private binder detail")
		case 2:
			return Subject{AgentInstanceID: "bad space", UID: 1000, WorkspaceID: "workspace"}, nil
		case 3:
			panic("binder panic detail")
		default:
			return testSubject, nil
		}
	}}
	var sessions atomic.Int32
	session := &testSession{serve: func(context.Context, net.Conn) error {
		sessions.Add(1)
		return nil
	}}
	server, err := New(Rules{Binder: binder, Session: session, MaxConcurrent: 2})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background(), listener) }()
	waitFor(t, func() bool { return bindCalls.Load() == 4 })
	cancelListener(t, listener)
	if err := <-done; err == nil {
		t.Fatal("external listener close should fail Serve")
	}
	if got := sessions.Load(); got != 1 {
		t.Fatalf("session calls = %d, want one valid connection", got)
	}
	for _, conn := range trackedConns {
		if !waitClosed(conn) {
			t.Fatal("accepted connection was not closed")
		}
	}
}

func TestServeSessionErrorAndPanicAreConnectionLocal(t *testing.T) {
	listener := newFakeListener()
	trackedConnections := make([]*trackedConn, 0, 3)
	for range 3 {
		raw, clientConn := net.Pipe()
		serverConn := &trackedConn{Conn: raw, closed: make(chan struct{})}
		trackedConnections = append(trackedConnections, serverConn)
		listener.add(serverConn)
		defer clientConn.Close()
	}
	var calls atomic.Int32
	session := &testSession{serve: func(context.Context, net.Conn) error {
		switch calls.Add(1) {
		case 1:
			return errors.New("session detail")
		case 2:
			panic("session panic detail")
		default:
			return nil
		}
	}}
	server, err := New(Rules{Binder: &testBinder{subject: testSubject}, Session: session, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background(), listener) }()
	waitFor(t, func() bool { return calls.Load() == 3 })
	cancelListener(t, listener)
	if err := <-done; err == nil {
		t.Fatal("external listener close should fail Serve")
	}
	for _, conn := range trackedConnections {
		if !waitClosed(conn) {
			t.Fatal("panic/error session connection was not closed")
		}
	}
}

func TestServeUnexpectedAcceptFailureCancelsAndDrains(t *testing.T) {
	listener := newFakeListener()
	serverConn, clientConn := net.Pipe()
	listener.add(serverConn)
	defer clientConn.Close()
	binderEntered := make(chan struct{})
	sessionReturned := make(chan struct{})
	binder := &testBinder{bind: func(ctx context.Context, conn net.Conn) (Subject, error) {
		close(binderEntered)
		return testSubject, nil
	}}
	session := &testSession{serve: func(ctx context.Context, conn net.Conn) error {
		<-ctx.Done()
		close(sessionReturned)
		return nil
	}}
	server, err := New(Rules{Binder: binder, Session: session, MaxConcurrent: 2})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(context.Background(), listener) }()
	select {
	case <-binderEntered:
	case <-time.After(time.Second):
		t.Fatal("binder did not run")
	}
	listener.fail(errors.New("accept detail"))
	if err := <-done; !errors.Is(err, ErrServer) {
		t.Fatalf("Serve error = %v", err)
	}
	select {
	case <-sessionReturned:
	case <-time.After(time.Second):
		t.Fatal("session was not drained after accept failure")
	}
	if !listener.wasClosed() {
		t.Fatal("listener was not closed")
	}
}

func TestServeCallerCancelClosesAndDrains(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "deadline"}[deadline], func(t *testing.T) {
			listener := newFakeListener()
			serverConn, clientConn := net.Pipe()
			listener.add(serverConn)
			defer clientConn.Close()
			sessionStarted := make(chan struct{})
			sessionDone := make(chan struct{})
			session := &testSession{serve: func(ctx context.Context, conn net.Conn) error {
				close(sessionStarted)
				<-ctx.Done()
				close(sessionDone)
				return nil
			}}
			server, err := New(Rules{Binder: &testBinder{subject: testSubject}, Session: session, MaxConcurrent: 1})
			if err != nil {
				t.Fatal(err)
			}
			var ctx context.Context
			var cancel context.CancelFunc
			if deadline {
				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
			} else {
				ctx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- server.Serve(ctx, listener) }()
			select {
			case <-sessionStarted:
			case <-time.After(time.Second):
				t.Fatal("session did not start")
			}
			if !deadline {
				cancel()
			}
			select {
			case <-sessionDone:
			case <-time.After(time.Second):
				t.Fatal("session did not observe cancellation")
			}
			if err := <-done; err != nil {
				t.Fatalf("cancelled Serve = %v", err)
			}
			if !listener.wasClosed() {
				t.Fatal("listener was not closed on cancellation")
			}
		})
	}
}

func TestServeCancelsCooperativeBinderAndClosesConn(t *testing.T) {
	listener := newFakeListener()
	rawServer, client := net.Pipe()
	tracked := &trackedConn{Conn: rawServer, closed: make(chan struct{})}
	listener.add(tracked)
	defer client.Close()
	binderDone := make(chan struct{})
	binder := &testBinder{bind: func(ctx context.Context, conn net.Conn) (Subject, error) {
		<-ctx.Done()
		close(binderDone)
		return Subject{}, errors.New("cooperative binder stopped")
	}}
	server, err := New(Rules{Binder: binder, Session: &testSession{}, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, listener) }()
	waitFor(t, func() bool { return binder.count() == 1 })
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("cancelled Serve = %v", err)
	}
	select {
	case <-binderDone:
	case <-time.After(time.Second):
		t.Fatal("binder did not drain")
	}
	if !waitClosed(tracked) {
		t.Fatal("connection was not closed")
	}
}

func TestSubjectIdentifierBoundaries(t *testing.T) {
	valid := []string{"a", "A9", "agent-1", "a_b.c", strings.Repeat("x", maxIdentifier)}
	for _, value := range valid {
		if !validIdentifier(value) {
			t.Errorf("valid identifier %q rejected", value)
		}
	}
	invalid := []string{"", "-agent", "_agent", ".agent", "agent space", "agent/slash", strings.Repeat("x", maxIdentifier+1), "é"}
	for _, value := range invalid {
		if validIdentifier(value) {
			t.Errorf("invalid identifier %q accepted", value)
		}
	}
	for _, subject := range []Subject{
		{AgentInstanceID: "agent", UID: 0, WorkspaceID: "workspace"},
		{AgentInstanceID: "agent", UID: -1, WorkspaceID: "workspace"},
		{AgentInstanceID: "agent", UID: 1, WorkspaceID: "workspace"},
	} {
		if subject.UID <= 0 && validSubject(subject) {
			t.Errorf("invalid UID accepted: %#v", subject)
		}
	}
}

type testBinder struct {
	mu      sync.Mutex
	calls   int
	subject Subject
	bind    func(context.Context, net.Conn) (Subject, error)
}

func (b *testBinder) Bind(ctx context.Context, conn net.Conn) (Subject, error) {
	b.mu.Lock()
	b.calls++
	bind, subject := b.bind, b.subject
	b.mu.Unlock()
	if bind != nil {
		return bind(ctx, conn)
	}
	return subject, nil
}

func (b *testBinder) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls
}

type testSession struct {
	serve func(context.Context, net.Conn) error
}

func (s *testSession) Serve(ctx context.Context, conn net.Conn) error {
	if s.serve == nil {
		return nil
	}
	return s.serve(ctx, conn)
}

type fakeListener struct {
	mu         sync.Mutex
	queue      chan net.Conn
	closed     chan struct{}
	accepted   chan struct{}
	failures   chan error
	closeOnce  sync.Once
	closeCount int
	failure    error
	accepts    atomic.Int32
}

func newFakeListener() *fakeListener {
	return &fakeListener{
		queue: make(chan net.Conn, 32), closed: make(chan struct{}), accepted: make(chan struct{}, 32), failures: make(chan error, 1),
	}
}

func (l *fakeListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	failure := l.failure
	l.mu.Unlock()
	if failure != nil {
		return nil, failure
	}
	select {
	case conn := <-l.queue:
		l.accepts.Add(1)
		select {
		case l.accepted <- struct{}{}:
		default:
		}
		return conn, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case err := <-l.failures:
		return nil, err
	}
}

func (l *fakeListener) Close() error {
	l.closeOnce.Do(func() {
		l.mu.Lock()
		l.closeCount++
		l.mu.Unlock()
		close(l.closed)
	})
	return nil
}

func (l *fakeListener) Addr() net.Addr { return testAddr("fake") }

func (l *fakeListener) add(conn net.Conn) {
	l.queue <- conn
}

func (l *fakeListener) fail(err error) {
	l.mu.Lock()
	l.failure = err
	l.mu.Unlock()
	l.failures <- err
}

func (l *fakeListener) closeForTest() {
	_ = l.Close()
}

func (l *fakeListener) wasClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closeCount > 0
}

func (l *fakeListener) acceptCount() int {
	return int(l.accepts.Load())
}

type testAddr string

func (a testAddr) Network() string { return string(a) }
func (a testAddr) String() string  { return string(a) }

func cancelListener(t *testing.T, listener *fakeListener) {
	t.Helper()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not become true")
}

func waitClosed(conn net.Conn) bool {
	if tracked, ok := conn.(*trackedConn); ok {
		select {
		case <-tracked.closed:
			return true
		case <-time.After(time.Second):
			return false
		}
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond)); err == nil {
			var one [1]byte
			_, readErr := conn.Read(one[:])
			if readErr != nil {
				return true
			}
		}
	}
	return false
}

type trackedConn struct {
	net.Conn
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *trackedConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}
