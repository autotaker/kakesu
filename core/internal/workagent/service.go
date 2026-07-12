package workagent

import "kakesu/core/internal/plane"

func NewService() plane.Service {
	return plane.IdleService{ServiceName: "work-agent-plane"}
}
