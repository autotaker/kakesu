package control

import "kakesu/core/internal/plane"

func NewService() plane.Service {
	return plane.IdleService{ServiceName: "control-plane"}
}
