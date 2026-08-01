// Package peerbinder binds one accepted Unix connection to a listener-owned
// subject after checking the kernel-reported peer UID.
package peerbinder

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/autotaker/kakesu/tools/dev-agent-harness/internal/brokerlistener"
)

const (
	minIdentifier = 1
	maxIdentifier = 128
)

// Error is the fixed diagnostic surface of this package. It never includes
// identity, socket, operating-system, or lower-level error details.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRules Error = "invalid-rules"
	ErrDenied       Error = "peer-bind-denied"
)

// Rules fixes the one subject and expected UID for a listener lifetime.
// Subject.UID must equal ExpectedUID; no UID-to-subject mapping is retained.
type Rules struct {
	ExpectedUID int
	Subject     brokerlistener.Subject
}

// Binder is immutable after New. It never owns or closes a connection.
type Binder struct {
	expectedUID int
	subject     brokerlistener.Subject
	reader      peerReader
}

type peerReader func(*net.UnixConn) (int, error)

var _ brokerlistener.PeerBinder = (*Binder)(nil)

// New validates and copies one listener binding. The production reader is the
// platform adapter; tests use newWithReader to exercise the pure core.
func New(r Rules) (*Binder, error) {
	return newWithReader(r, readPeerUID)
}

func newWithReader(r Rules, reader peerReader) (*Binder, error) {
	if !validUID(r.ExpectedUID) || r.Subject.UID != r.ExpectedUID ||
		!validIdentifier(r.Subject.AgentInstanceID) || !validIdentifier(r.Subject.WorkspaceID) || reader == nil {
		return nil, ErrInvalidRules
	}
	return &Binder{
		expectedUID: r.ExpectedUID,
		subject: brokerlistener.Subject{
			AgentInstanceID: cloneString(r.Subject.AgentInstanceID),
			UID:             r.Subject.UID,
			WorkspaceID:     cloneString(r.Subject.WorkspaceID),
		},
		reader: reader,
	}, nil
}

// Format keeps Binder diagnostics fixed for every value and formatting verb.
func (b *Binder) Format(state fmt.State, verb rune) {
	_, _ = io.WriteString(state, "peerbinder.Binder")
}

// Bind accepts only a concrete Unix connection and performs one synchronous
// kernel UID read. Every rejection is an empty subject and ErrDenied.
func (b *Binder) Bind(ctx context.Context, conn net.Conn) (brokerlistener.Subject, error) {
	if b == nil || b.reader == nil || !validUID(b.expectedUID) ||
		b.subject.UID != b.expectedUID || !validIdentifier(b.subject.AgentInstanceID) ||
		!validIdentifier(b.subject.WorkspaceID) || isNil(ctx) || ctx.Err() != nil {
		return brokerlistener.Subject{}, ErrDenied
	}
	unixConn, ok := conn.(*net.UnixConn)
	if !ok || unixConn == nil {
		return brokerlistener.Subject{}, ErrDenied
	}
	uid, err := b.reader(unixConn)
	if err != nil || ctx.Err() != nil || uid != b.expectedUID {
		return brokerlistener.Subject{}, ErrDenied
	}
	return brokerlistener.Subject{
		AgentInstanceID: cloneString(b.subject.AgentInstanceID),
		UID:             b.subject.UID,
		WorkspaceID:     cloneString(b.subject.WorkspaceID),
	}, nil
}

func validUID(uid int) bool {
	return uid > 0 && uint64(uid) <= uint64(^uint32(0))
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

func cloneString(value string) string {
	return string(append([]byte(nil), value...))
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
