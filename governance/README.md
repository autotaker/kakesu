# Governance Plane scaffold

Rust service boundary for fail-closed Egress enforcement, policy, audit, and credentials.

Tokio tasks are execution resources only. Durable Governance state and Inbox/Outbox will live in `governance.db`; all long-lived tasks must be registered with the service supervisor.
