package plane

import "context"

// Service is a long-lived Core Runtime component owned by the application run group.
type Service interface {
	Name() string
	Run(context.Context) error
}
