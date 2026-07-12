# Core Runtime scaffold

Go implementation boundary for the CLI, Control Plane, Work Agent Plane, and Execution Plane.

- `cmd/kakesu`: user-facing CLI process
- `internal/plane`: supervised service lifecycle primitives
- `internal/control`: Task, Mailbox, Async, Authority boundary
- `internal/workagent`: Responses API coroutine runtime
- `internal/execution`: workspace and process lifecycle
- `internal/message`: cross-plane message envelope

Durable state and messages will live in `control.db`; channels are limited to wake-up and bounded in-process handoff.
