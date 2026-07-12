package plane

import "context"

// IdleService marks a scaffold boundary. Each Plane will replace this wait loop
// with a durable Inbox/Outbox dispatcher and bounded workers.
type IdleService struct {
	ServiceName string
}

func (s IdleService) Name() string { return s.ServiceName }

func (s IdleService) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
