# Package Manifest

## Core documents

| File | Role |
|---|---|
| `README.md` | Reading order and invariant summary |
| `docs/00-agent-harness.md` | Main integrated design |
| `docs/01-domain-model.md` | Agent / Task / Workspace model and ER diagram |
| `docs/02-task-lifecycle.md` | Task creation, waiting, review, cancellation, terminal states |
| `docs/03-runtime-and-responses-api.md` | Coroutine runtime and Responses API mapping |
| `docs/04-tools-and-async.md` | LLM tools, timeout promotion, async IDs, mailbox |
| `docs/05-governance.md` | Sandbox boundary, Policy Judge, Effect Gateway |
| `docs/06-long-term-memory.md` | Task Episode and Wiki Agent memory architecture |
| `docs/07-semantic-wiki-schema.md` | Markdown semantic memory schema |
| `docs/08-running-example.md` | End-to-end root Task simulation |
| `docs/09-data-model.md` | Persistence model, constraints, transactions |
| `docs/10-implementation-and-tests.md` | Implementation phases and test plan |

## Schemas and examples

| Path | Role |
|---|---|
| `schemas/work-agent-tools.json` | Responses API function definitions for Work Agents |
| `schemas/judge-tools.json` | Function definitions for reviewers and policy/authority judges |
| `schemas/domain-types.ts` | Canonical logical TypeScript types |
| `examples/semantic/` | Concept / Schema / Script / Case Pattern examples |
| `examples/episodic/T-110.md` | Task Episode example |
| `sources/OPENAI_API_NOTES.md` | Official API assumptions checked on 2026-07-11 |
| `SHA256SUMS` | File integrity list |
