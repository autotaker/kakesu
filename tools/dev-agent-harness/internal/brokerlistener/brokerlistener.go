// Package brokerlistener owns an injected listener and turns each accepted
// connection into one trusted, context-bound session.  It deliberately does
// not create listeners or infer identity from connection metadata.
package brokerlistener

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/egresstransaction"
)

const (
	minConcurrent = 1
	maxConcurrent = 64
	minIdentifier = 1
	maxIdentifier = 128
)

// Error is a fixed public error.  It never includes dependency, identity,
// address, or lower-level error details.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrInvalidServe Error = "invalid-serve"
	ErrResolver     Error = "resolver-error"
	ErrServer       Error = "server-error"
)

// Subject is the identity that a trusted PeerBinder supplies.  It is an
// alias so the listener and downstream transaction use exactly one type.
type Subject = egresstransaction.Subject

// PeerBinder is a trusted, context-cooperative identity producer.  A binder
// is called synchronously once for each accepted connection.
type PeerBinder interface {
	Bind(context.Context, net.Conn) (Subject, error)
}

// PeerBinderFunc adapts a function to PeerBinder.
type PeerBinderFunc func(context.Context, net.Conn) (Subject, error)

func (f PeerBinderFunc) Bind(ctx context.Context, conn net.Conn) (Subject, error) {
	return f(ctx, conn)
}

// Session is the already-constructed one-connection session boundary.  It
// must cooperate with context cancellation and return without being killed by
// the listener.
type Session interface {
	Serve(context.Context, net.Conn) error
}

// SessionFunc adapts a function to Session.
type SessionFunc func(context.Context, net.Conn) error

func (f SessionFunc) Serve(ctx context.Context, conn net.Conn) error {
	return f(ctx, conn)
}

// Rules contains the only long-lived dependencies retained by Server.
type Rules struct {
	Binder        PeerBinder
	Session       Session
	MaxConcurrent int
}

// Server is immutable after New.  Listener and all connection lifecycle
// state belong to one Serve call and are not retained here.
type Server struct {
	binder        PeerBinder
	session       Session
	maxConcurrent int
}

// Format intentionally exposes only the stable type name.
func (s Server) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "brokerlistener.Server")
}

// New validates trusted dependencies and the bounded concurrency setting.
func New(r Rules) (*Server, error) {
	if isNil(r.Binder) || isNil(r.Session) ||
		r.MaxConcurrent < minConcurrent || r.MaxConcurrent > maxConcurrent {
		return nil, ErrInvalidRules
	}
	return &Server{
		binder:        r.Binder,
		session:       r.Session,
		maxConcurrent: r.MaxConcurrent,
	}, nil
}

// Resolver obtains the private identity installed by Server.  It has no
// setter: the only producer is the trusted binder path in this package.
type Resolver struct{}

// Format keeps Resolver diagnostics fixed and dependency-free.
func (Resolver) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "brokerlistener.Resolver")
}

// Resolve returns a fresh copy of the subject bound to ctx.  Missing,
// malformed, or cancelled contexts are all deliberately collapsed to the
// same fixed error.
func (Resolver) Resolve(ctx context.Context) (Subject, error) {
	if isNil(ctx) || ctx.Err() != nil {
		return Subject{}, ErrResolver
	}
	value := ctx.Value(subjectContextKey{})
	subject, ok := value.(Subject)
	if !ok || !validSubject(subject) || ctx.Err() != nil {
		return Subject{}, ErrResolver
	}
	return copySubject(subject), nil
}

// Serve owns listener until it returns.  Slots are acquired before Accept,
// and every accepted connection is drained before shutdown completes.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	if s == nil || isNil(s.binder) || isNil(s.session) ||
		s.maxConcurrent < minConcurrent || s.maxConcurrent > maxConcurrent ||
		isNil(ctx) || isNil(listener) {
		return ErrInvalidServe
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var closeOnce sync.Once
	closeListener := func() {
		closeOnce.Do(func() {
			// A standard net.Listener does not panic from Close.  The recover
			// keeps a faulty test double from bypassing connection draining.
			defer func() { _ = recover() }()
			_ = listener.Close()
		})
	}

	watcherStop := make(chan struct{})
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			closeListener()
			cancel()
		case <-watcherStop:
		}
	}()

	slots := make(chan struct{}, s.maxConcurrent)
	var connections sync.WaitGroup
	failure := false

acceptLoop:
	for {
		select {
		case slots <- struct{}{}:
		case <-runCtx.Done():
			break acceptLoop
		}
		// Re-check cancellation before entering Accept (fail closed).
		if runCtx.Err() != nil {
			<-slots
			break acceptLoop
		}

		conn, err := listener.Accept()
		if err != nil {
			<-slots
			if ctx.Err() == nil {
				failure = true
			}
			break acceptLoop
		}
		if isNil(conn) {
			<-slots
			if ctx.Err() == nil {
				failure = true
			}
			break acceptLoop
		}

		// Add happens before the goroutine is started and Wait is reached only
		// after this accept loop has stopped, preventing Add/Wait races.
		connections.Add(1)
		go s.serveConnection(runCtx, conn, slots, &connections)
	}

	// Both cancellation and failure close the listener.  The once also makes
	// this safe when the context watcher already performed the close.
	closeListener()
	cancel()
	close(watcherStop)
	<-watcherDone
	connections.Wait()
	if failure {
		return ErrServer
	}
	return nil
}

func (s *Server) serveConnection(runCtx context.Context, conn net.Conn, slots chan struct{}, connections *sync.WaitGroup) {
	defer connections.Done()
	defer func() {
		// slots contains exactly one token for this connection.
		<-slots
	}()
	defer func() {
		defer func() { _ = recover() }()
		_ = conn.Close()
	}()
	defer func() { _ = recover() }()

	connCtx, cancel := context.WithCancel(runCtx)
	defer cancel()

	subject, err := s.binder.Bind(connCtx, conn)
	if err != nil || !validSubject(subject) {
		return
	}
	subject = copySubject(subject)

	// The private key is installed only after validation and copying.  The
	// session receives one context value that cannot be forged by its caller.
	_ = s.session.Serve(context.WithValue(connCtx, subjectContextKey{}, subject), conn)
}

type subjectContextKey struct{}

func validSubject(subject Subject) bool {
	return subject.UID > 0 && validIdentifier(subject.AgentInstanceID) && validIdentifier(subject.WorkspaceID)
}

func validIdentifier(value string) bool {
	if len(value) < minIdentifier || len(value) > maxIdentifier {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if index == 0 {
			if !isASCIIAlphaNumeric(char) {
				return false
			}
			continue
		}
		if !isASCIIAlphaNumeric(char) && char != '.' && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

func isASCIIAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9'
}

func copySubject(subject Subject) Subject {
	return Subject{
		AgentInstanceID: strings.Clone(subject.AgentInstanceID),
		UID:             subject.UID,
		WorkspaceID:     strings.Clone(subject.WorkspaceID),
	}
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
