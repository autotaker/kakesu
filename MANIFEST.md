# Package Manifest

## Core documents

| File | Role |
|---|---|
| `README.md` | Reading order and invariant summary |
| `docs/00-kakesu.md` | Main integrated design |
| `docs/01-domain-model.md` | Agent / Task / Workspace model and ER diagram |
| `docs/02-task-lifecycle.md` | Task creation, waiting, review, cancellation, terminal states |
| `docs/03-agent-lifecycle.md` | Agent registration, ownership, runs, recovery, and release |
| `docs/04-built-in-agents.md` | Built-in LLM components outside Harness agent/run management |
| `docs/05-runtime-and-responses-api.md` | Coroutine runtime and Responses API mapping |
| `docs/06-tools-and-async.md` | LLM tools, timeout promotion, async IDs, mailbox |
| `docs/07-governance.md` | Sandbox boundary, CASB egress enforcement, and policy grants |
| `docs/08-long-term-memory.md` | Task Episode and Wiki Agent memory architecture |
| `docs/09-semantic-wiki-schema.md` | Markdown semantic memory schema |
| `docs/10-running-example.md` | End-to-end root Task simulation |
| `docs/11-data-model.md` | Persistence model, constraints, transactions |
| `docs/12-implementation-and-tests.md` | Implementation phases and test plan |
| `docs/13-technology-stack.md` | Selected languages, frameworks, stores, IPC, and build conventions |

## Implementation scaffolds

| Path | Role |
|---|---|
| `core/` | Go CLI, Control, Work Agent, and Execution runtime scaffold |
| `memory/` | Python and OpenAI Agents SDK Memory service scaffold |
| `governance/` | Rust Governance enforcement service scaffold |
| `Makefile` | Repository-wide build, test, lint, and validation entrypoint |

## Schemas and examples

| Path | Role |
|---|---|
| `schemas/README.md` | Plane ownership and draft-v0 schema versioning rules |
| `schemas/draft-v0/control-plane/README.md` | Control Plane canonical schema catalog |
| `schemas/draft-v0/execution-plane/README.md` | Execution Plane canonical schema catalog |
| `schemas/draft-v0/governance-plane/README.md` | Governance Plane canonical schema catalog |
| `schemas/draft-v0/memory-plane/README.md` | Memory Plane canonical schema catalog |
| `schemas/draft-v0/common/README.md` | Cross-plane primitive and envelope schema catalog |
| `schemas/draft-v0/api/work-agent-tools.json` | Responses API function bundle for Work Agents |
| `schemas/draft-v0/api/built-in-agent-outputs.json` | Structured Output bundle for built-in components |
| `schemas/draft-v0/api/authority-tools.json` | Authority adapter ingress function bundle |
| `schemas/domain-types.ts` | Canonical logical TypeScript types |
| `examples/semantic/` | Concept / Schema / Script / Case Pattern examples |
| `examples/episodic/T-110.md` | Documentation example for the Task Episode schema, not runtime storage |
| `sources/OPENAI_API_NOTES.md` | Official API assumptions checked on 2026-07-11 |
