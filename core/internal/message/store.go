package message

import "context"

// Store is the durable Inbox/Outbox port implemented by control.db.
// Transport code must ACK only after PutInbox commits successfully.
type Store interface {
	PutOutbox(context.Context, Envelope) error
	ClaimOutbox(context.Context, int) ([]Envelope, error)
	MarkDelivered(context.Context, string) error
	PutInbox(context.Context, Envelope) (inserted bool, err error)
}
