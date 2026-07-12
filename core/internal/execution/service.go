package execution

import "kakesu/core/internal/plane"

func NewService() plane.Service {
	return plane.IdleService{ServiceName: "execution-plane"}
}
