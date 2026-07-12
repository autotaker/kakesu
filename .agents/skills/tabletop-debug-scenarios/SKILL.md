---
name: tabletop-debug-scenarios
description: Create, extend, or review schema-backed tabletop E2E scenarios for agent-harness. Use when a specification or JSON Schema change affects Plane-to-Plane messages, Task or Agent Run state, Authority, Incident containment, canonical domain payloads, sequence requirements, validator mutations, or the HTML tabletop Viewer.
---

# Tabletop scenario debugging

Simulate a representative workflow as concrete component messages, validate every payload mechanically, and independently review whether the workflow is executable without hidden responsibility gaps.

## Establish scope

1. Read the affected design documents and the nearest `AGENTS.md` files completely.
2. Inspect `examples/e2e-tabletop/README.md`, active scenario files, `sequence-requirements.json`, canonical payload files, and both validator scripts.
3. Identify the four Plane responsibilities and the intended terminal outcome before editing.
4. Treat `scenario_id` as globally unique. Never rely on file load order to supersede another active scenario.

## Build the tabletop trace

Model each meaningful component exchange as a sequence step containing:

- stable message, Task, Workspace, correlation, causation, idempotency, and entity IDs;
- explicit source/destination Plane and component;
- state transition and timestamp;
- references needed to join the next message.

Cover the complete path from trigger through durable outcome. Include failure containment, ACKs, Authority gates, stop/resume, Mailbox delivery, or Episode creation when the design requires them. Do not omit a transition merely because prose implies it.

Human or external Authority communication must enter and return through the Control Plane Authority Gateway.

For MVP Incident Task-tree containment, encode these normative invariants explicitly:

- expand `all ancestors + source Task + all descendants` from one fixed Task graph revision;
- exclude siblings and other branches, and include a positive witness that an excluded sibling continues;
- close the Action gate for the full containment set before requesting any Run stop;
- stop and suspend `descendant -> source -> ancestor`, confirming each Run stop before its Task becomes suspended;
- after authorized containment release, start and restore `ancestor -> source -> descendant`;
- restore each Task to its captured prior state rather than forcing every Task to `running`.

Add focused mutations for set mismatch, sibling inclusion, stale graph revision, and reversed stop/resume order when those constraints are introduced or changed.

## Add canonical domain payloads

Create a canonical payload for every Command, Event, state change, immutable snapshot, Request, and Decision. Generic sequence projections are acceptable only for views that do not independently change domain state.

For every cross-record relationship, add a machine-checked join using the relevant ID, version, digest, Task, Workspace, or Run ID. Ensure:

- every active canonical record is used;
- every required state-changing message has a canonical record;
- renamed concrete message types cannot evade canonical checks;
- old scenarios, bindings, and payloads are removed after migration.

## Encode sequence requirements

Update `examples/e2e-tabletop/sequence-requirements.json` with:

- required message order;
- direct causation pairs;
- field correlations;
- scenario-specific invariants that JSON Schema cannot express.

Put single-payload constraints in JSON Schema. Put set equality, graph closure, ordering, exclusion, and cross-message joins in `scripts/validate-tabletop-scenarios.mjs`.

## Add negative mutations

For each new invariant, add a focused mutation to `scripts/validate-tabletop-scenarios.mjs` and register it in `scripts/test-tabletop-validator.mjs`. A mutation must change one relevant property and must make validation fail. Retain positive redelivery and nested-correlation tests.

## Validate and inspect

Run from the repository root:

```sh
node scripts/build-tabletop-viewer-data.mjs
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
git diff --check
```

Confirm the reported canonical count equals the canonical records used by active traces. Inspect the regenerated Viewer data and verify the changed scenario has the intended message count, order, Plane routing, state transitions, and canonical details.

## Perform independent review

Use separate reviewers when available:

- Sequence reviewer: simulate the workflow, check responsibility boundaries, causation, ordering, stop/resume behavior, sibling or descendant effects, and bypass paths.
- Schema reviewer: check canonical coverage, conditional fields, state transitions, ID joins, unused records, and false-pass paths in the validator.

Do not commit with any P0 finding. Fix P0 findings and rerun the same reviewers. Record nonblocking P1 findings and why they do not invalidate the scenario.

## Completion criteria

Finish only when:

- all active scenarios are unique and executable;
- Schema, canonical payload, sequence projection, requirements, and Viewer agree;
- baseline and all negative mutations pass;
- both independent review perspectives pass when the change warrants them;
- obsolete artifacts and references are removed.
