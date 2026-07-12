package plane

import (
	"context"
	"fmt"
	"sync"
)

// RunGroup owns all long-lived goroutines. No service may be started detached.
type RunGroup struct {
	services []Service
}

func NewRunGroup(services ...Service) *RunGroup {
	return &RunGroup{services: services}
}

func (g *RunGroup) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errors := make(chan error, len(g.services))
	var workers sync.WaitGroup
	for _, service := range g.services {
		service := service
		workers.Add(1)
		go func() {
			defer workers.Done()
			if err := service.Run(ctx); err != nil {
				errors <- fmt.Errorf("%s: %w", service.Name(), err)
				cancel()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()

	select {
	case err := <-errors:
		<-done
		return err
	case <-ctx.Done():
		<-done
		select {
		case err := <-errors:
			return err
		default:
			return nil
		}
	}
}
