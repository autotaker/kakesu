package app

import (
	"context"

	"kakesu/core/internal/control"
	"kakesu/core/internal/execution"
	"kakesu/core/internal/plane"
	"kakesu/core/internal/workagent"
)

type App struct {
	group *plane.RunGroup
}

func New() *App {
	return &App{group: plane.NewRunGroup(
		control.NewService(),
		workagent.NewService(),
		execution.NewService(),
	)}
}

func (a *App) Run(ctx context.Context) error {
	return a.group.Run(ctx)
}
