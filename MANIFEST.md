# Package Manifest

## Core documents

| File | Role |
|---|---|
| `README.md` | Reading order and invariant summary |
| `docs/00-agent-harness.md` | Main integrated design |
| `docs/01-domain-model.md` | Agent / Task / Workspace model and ER diagram |
| `docs/02-task-lifecycle.md` | Task creation, waiting, review, cancellation, terminal states |
| `docs/03-agent-lifecycle.md` | Agent registration, ownership, runs, recovery, and release |
| `docs/04-built-in-agents.md` | Built-in LLM components outside Harness agent/run management |
| `docs/05-runtime-and-responses-api.md` | Coroutine runtime and Responses API mapping |
| `docs/06-tools-and-async.md` | LLM tools, timeout promotion, async IDs, mailbox |
| `docs/07-governance.md` | Sandbox boundary, Policy Judge, Effect Gateway |
| `docs/08-long-term-memory.md` | Task Episode and Wiki Agent memory architecture |
| `docs/09-semantic-wiki-schema.md` | Markdown semantic memory schema |
| `docs/10-running-example.md` | End-to-end root Task simulation |
| `docs/11-data-model.md` | Persistence model, constraints, transactions |
| `docs/12-implementation-and-tests.md` | Implementation phases and test plan |

## Schemas and examples

| Path | Role |
|---|---|
| `schemas/work-agent-tools.json` | Responses API function definitions for Work Agents |
| `schemas/built-in-agent-outputs.json` | Structured Output schemas for built-in Reviewer and Policy Judge components |
| `schemas/authority-tools.json` | Effect and Task Escalation ingress functions for external Authority adapters |
| `schemas/domain-types.ts` | Canonical logical TypeScript types |
| `examples/semantic/` | Concept / Schema / Script / Case Pattern examples |
| `examples/episodic/T-110.md` | Documentation example for the Task Episode schema, not runtime storage |
| `sources/OPENAI_API_NOTES.md` | Official API assumptions checked on 2026-07-11 |
| `SHA256SUMS` | File integrity list |
