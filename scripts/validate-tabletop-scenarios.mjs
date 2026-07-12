#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import net from "node:net";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const scenarioArgument = process.argv.slice(2).find((argument) => !argument.startsWith("--"));
const scenarioPaths = scenarioArgument
  ? [path.resolve(repoRoot, scenarioArgument)]
  : [
      path.join(repoRoot, "examples/e2e-tabletop/executable-scenarios.json"),
      path.join(repoRoot, "examples/e2e-tabletop/executable-scenario-002.json")
    ];
const requirementPath = path.join(repoRoot, "examples/e2e-tabletop/sequence-requirements.json");
const domainBindingPath = path.join(repoRoot, "examples/e2e-tabletop/domain-payload-bindings.json");
const documentCache = new Map();

function readJson(file) {
  const absolute = path.resolve(file);
  if (!documentCache.has(absolute)) {
    documentCache.set(absolute, JSON.parse(fs.readFileSync(absolute, "utf8")));
  }
  return documentCache.get(absolute);
}

function resolvePointer(document, fragment) {
  if (!fragment || fragment === "#") return document;
  if (!fragment.startsWith("#/")) throw new Error(`unsupported fragment: ${fragment}`);
  return fragment.slice(2).split("/").reduce((value, token) => {
    const key = token.replace(/~1/g, "/").replace(/~0/g, "~");
    return value[key];
  }, document);
}

function resolveRef(ref, baseFile) {
  if (ref.startsWith("urn:")) throw new Error(`external URN ref is not resolvable: ${ref}`);
  const [filePart, fragmentPart] = ref.split("#", 2);
  const targetFile = filePart ? path.resolve(path.dirname(baseFile), filePart) : baseFile;
  const document = readJson(targetFile);
  return { schema: resolvePointer(document, fragmentPart === undefined ? "" : `#${fragmentPart}`), file: targetFile };
}

function typeMatches(value, type) {
  if (type === "null") return value === null;
  if (type === "array") return Array.isArray(value);
  if (type === "object") return value !== null && typeof value === "object" && !Array.isArray(value);
  if (type === "integer") return Number.isInteger(value);
  if (type === "number") return typeof value === "number" && Number.isFinite(value);
  return typeof value === type;
}

function deepEqual(a, b) {
  return JSON.stringify(a) === JSON.stringify(b);
}

function validateFormat(value, format) {
  if (typeof value !== "string") return true;
  if (format === "date-time") return !Number.isNaN(Date.parse(value)) && value.includes("T");
  if (format === "ipv4") return net.isIP(value) === 4;
  if (format === "ipv6") return net.isIP(value) === 6;
  return true;
}

function collectProperties(schema, baseFile, seen = new Set()) {
  if (!schema || typeof schema !== "object") return new Set();
  if (schema.$ref) {
    const key = `${baseFile}|${schema.$ref}`;
    if (seen.has(key)) return new Set();
    seen.add(key);
    const resolved = resolveRef(schema.$ref, baseFile);
    return collectProperties(resolved.schema, resolved.file, seen);
  }
  const result = new Set(Object.keys(schema.properties ?? {}));
  for (const child of schema.allOf ?? []) {
    for (const key of collectProperties(child, baseFile, seen)) result.add(key);
  }
  return result;
}

function validate(value, schema, baseFile, location = "$", options = {}) {
  const errors = [];
  if (schema === true || (schema && Object.keys(schema).length === 0)) return errors;
  if (schema === false) return [`${location}: schema is false`];
  if (schema.$ref) {
    const resolved = resolveRef(schema.$ref, baseFile);
    return validate(value, resolved.schema, resolved.file, location, options);
  }

  if (schema.allOf) {
    for (const child of schema.allOf) errors.push(...validate(value, child, baseFile, location, { ...options, ignoreUnevaluated: true }));
    if (!options.ignoreUnevaluated && schema.allOf.some((child) => child.unevaluatedProperties === false) && typeMatches(value, "object")) {
      const allowed = collectProperties(schema, baseFile);
      for (const key of Object.keys(value)) if (!allowed.has(key)) errors.push(`${location}.${key}: unevaluated property`);
    }
  }
  if (schema.anyOf) {
    const matches = schema.anyOf.map((child) => validate(value, child, baseFile, location));
    if (!matches.some((result) => result.length === 0)) errors.push(`${location}: does not match anyOf (${matches.map((x) => x.join("; ")).join(" | ")})`);
  }
  if (schema.oneOf) {
    const matches = schema.oneOf.map((child) => validate(value, child, baseFile, location));
    const count = matches.filter((result) => result.length === 0).length;
    if (count !== 1) errors.push(`${location}: expected exactly one oneOf match, got ${count} (${matches.map((x) => x.join("; ")).join(" | ")})`);
  }
  if (schema.const !== undefined && !deepEqual(value, schema.const)) errors.push(`${location}: must equal const ${JSON.stringify(schema.const)}`);
  if (schema.enum && !schema.enum.some((candidate) => deepEqual(value, candidate))) errors.push(`${location}: not in enum`);
  if (schema.type) {
    const types = Array.isArray(schema.type) ? schema.type : [schema.type];
    if (!types.some((type) => typeMatches(value, type))) errors.push(`${location}: expected type ${types.join("|")}`);
  }
  if (typeof value === "string") {
    if (schema.minLength !== undefined && value.length < schema.minLength) errors.push(`${location}: shorter than minLength`);
    if (schema.maxLength !== undefined && value.length > schema.maxLength) errors.push(`${location}: longer than maxLength`);
    if (schema.pattern && !new RegExp(schema.pattern).test(value)) errors.push(`${location}: does not match pattern ${schema.pattern}`);
    if (schema.format && !validateFormat(value, schema.format)) errors.push(`${location}: invalid ${schema.format}`);
  }
  if (typeof value === "number") {
    if (schema.minimum !== undefined && value < schema.minimum) errors.push(`${location}: below minimum`);
    if (schema.maximum !== undefined && value > schema.maximum) errors.push(`${location}: above maximum`);
  }
  if (Array.isArray(value)) {
    if (schema.minItems !== undefined && value.length < schema.minItems) errors.push(`${location}: fewer than minItems`);
    if (schema.maxItems !== undefined && value.length > schema.maxItems) errors.push(`${location}: more than maxItems`);
    if (schema.uniqueItems && new Set(value.map((item) => JSON.stringify(item))).size !== value.length) errors.push(`${location}: items are not unique`);
    if (schema.items) value.forEach((item, index) => errors.push(...validate(item, schema.items, baseFile, `${location}[${index}]`)));
  }
  if (typeMatches(value, "object")) {
    for (const key of schema.required ?? []) if (!(key in value)) errors.push(`${location}.${key}: required property missing`);
    for (const [key, child] of Object.entries(schema.properties ?? {})) if (key in value) errors.push(...validate(value[key], child, baseFile, `${location}.${key}`));
    if (schema.additionalProperties === false) {
      const allowed = new Set(Object.keys(schema.properties ?? {}));
      for (const key of Object.keys(value)) if (!allowed.has(key)) errors.push(`${location}.${key}: additional property`);
    } else if (schema.additionalProperties && typeof schema.additionalProperties === "object") {
      const declared = new Set(Object.keys(schema.properties ?? {}));
      for (const [key, childValue] of Object.entries(value)) {
        if (!declared.has(key)) errors.push(...validate(childValue, schema.additionalProperties, baseFile, `${location}.${key}`));
      }
    }
  }
  return errors;
}

function validateScenarioEnvelope(document) {
  const errors = [];
  if (document.schema_version !== "draft-v0") errors.push("root.schema_version must be draft-v0");
  if (!Array.isArray(document.scenarios) || document.scenarios.length === 0) errors.push("root.scenarios must be non-empty");
  const scenarioIds = new Set();
  for (const scenario of document.scenarios ?? []) {
    if (scenarioIds.has(scenario.scenario_id)) errors.push(`duplicate scenario_id ${scenario.scenario_id}`);
    scenarioIds.add(scenario.scenario_id);
    if (!Array.isArray(scenario.steps) || scenario.steps.length === 0) errors.push(`${scenario.scenario_id}: steps must be non-empty`);
    const stepIds = new Set();
    for (const step of scenario.steps ?? []) {
      if (stepIds.has(step.step_id)) errors.push(`${scenario.scenario_id}: duplicate step_id ${step.step_id}`);
      stepIds.add(step.step_id);
      for (const field of ["step_id", "from_plane", "to_plane", "from_component", "to_component", "message_type", "schema_ref", "payload"]) {
        if (!(field in step)) errors.push(`${scenario.scenario_id}/${step.step_id}: missing ${field}`);
      }
    }
  }
  return errors;
}

function validateSequenceInvariants(scenario) {
  const errors = [];
  const byType = new Map();
  for (const step of scenario.steps) {
    const list = byType.get(step.message_type) ?? [];
    list.push(step);
    byType.set(step.message_type, list);
  }

  const field = (payload, name) => payload[name] ?? payload.refs?.[name];
  const memoryRequests = new Map((byType.get("MemoryContextRequest") ?? []).map((step) => [field(step.payload, "request_id"), step.payload]));
  for (const step of byType.get("MemoryContext") ?? []) {
    const request = memoryRequests.get(field(step.payload, "request_id"));
    if (!request) errors.push(`${scenario.scenario_id}/${step.step_id}: MemoryContext has no matching request_id`);
    else if (field(request, "task_id") !== field(step.payload, "task_id")) errors.push(`${scenario.scenario_id}/${step.step_id}: MemoryContext task_id differs from request`);
  }

  for (const step of byType.get("OutboundTransaction") ?? []) {
    const payload = step.payload;
    if (Date.parse(payload.completed_at) < Date.parse(payload.started_at)) errors.push(`${scenario.scenario_id}/${step.step_id}: transaction completes before it starts`);
  }

  for (const step of byType.get("PolicyRevision") ?? []) {
    const payload = step.payload;
    if (payload.new_policy_version !== payload.previous_policy_version + 1) errors.push(`${scenario.scenario_id}/${step.step_id}: policy version is not monotonic by one`);
    if (payload.base_policy_version !== payload.previous_policy_version) errors.push(`${scenario.scenario_id}/${step.step_id}: base and previous policy versions differ`);
    if (payload.status === "active" && (!payload.activation_ack_ref || !payload.activated_at)) errors.push(`${scenario.scenario_id}/${step.step_id}: active revision lacks activation ACK/time`);
  }

  for (const step of byType.get("CompletionReviewOutput") ?? []) {
    if (!step.payload.entity?.id && (!step.payload.review_job_id || !step.payload.task_id || !step.payload.input_digest)) errors.push(`${scenario.scenario_id}/${step.step_id}: completion output lacks correlation fields`);
  }

  const seenMessages = new Set();
  const seenIdempotencyKeys = new Map();
  const entityStates = new Map();
  let scenarioCorrelationId = null;
  let previousTime = -Infinity;
  const allowedTransitions = new Map([
    ["task", new Set(["null->ready", "null->running", "ready->running", "running->waiting", "waiting->running", "running->reviewing_completion", "reviewing_completion->completed"])],
    ["policy-grant", new Set(["null->pending_activation", "pending_activation->active", "active->revoked"])],
    ["policy-revision", new Set(["null->pending_activation", "pending_activation->active", "active->superseded"])],
    ["async-operation", new Set(["null->running", "running->completed", "running->failed", "running->cancelled"])],
    ["mailbox-event", new Set(["null->deliverable", "null->delivered", "delivered->consumed"])],
    ["memory-request", new Set(["null->requested", "requested->completed"])],
    ["completion-review", new Set(["null->reviewing", "reviewing->accepted", "reviewing->rejected"])],
    ["episode-job", new Set(["null->running", "running->completed"])],
    ["task-episode", new Set(["null->committed"])],
    ["tool-call", new Set(["null->running", "running->completed", "running->failed"])],
    ["policy-activation", new Set(["null->acked"])]
  ]);
  for (const step of scenario.steps) {
    const payload = step.payload;
    if (payload.message_id) {
      if (seenMessages.has(payload.message_id)) errors.push(`${scenario.scenario_id}/${step.step_id}: duplicate message_id`);
      if (payload.causation_message_id && !seenMessages.has(payload.causation_message_id)) errors.push(`${scenario.scenario_id}/${step.step_id}: causation does not reference an earlier message`);
      seenMessages.add(payload.message_id);
      const operationFingerprint = JSON.stringify({ task_id: payload.task_id, workspace_id: payload.workspace_id, entity: payload.entity, refs: payload.refs });
      const priorOperation = seenIdempotencyKeys.get(payload.idempotency_key);
      const isRedelivery = payload.delivery_attempt >= 2 && payload.redelivery_of_message_id;
      if (priorOperation) {
        if (!isRedelivery) errors.push(`${scenario.scenario_id}/${step.step_id}: reused idempotency_key lacks redelivery metadata`);
        if (priorOperation.fingerprint !== operationFingerprint) errors.push(`${scenario.scenario_id}/${step.step_id}: idempotency key reused with different operation fingerprint`);
        if (payload.redelivery_of_message_id !== priorOperation.message_id) errors.push(`${scenario.scenario_id}/${step.step_id}: redelivery does not reference first delivery`);
      } else seenIdempotencyKeys.set(payload.idempotency_key, { fingerprint: operationFingerprint, message_id: payload.message_id });
      if (scenarioCorrelationId === null) scenarioCorrelationId = payload.correlation_id;
      else if (payload.correlation_id !== scenarioCorrelationId) errors.push(`${scenario.scenario_id}/${step.step_id}: correlation_id leaves scenario chain`);
      if (payload.causation_message_id) {
        const cause = scenario.steps.find((candidate) => candidate.payload.message_id === payload.causation_message_id)?.payload;
        if (cause && cause.correlation_id !== payload.correlation_id) errors.push(`${scenario.scenario_id}/${step.step_id}: cause belongs to another correlation`);
      }
      const entity = payload.entity;
      if (entity && !isRedelivery) {
        const key = `${entity.kind}:${entity.id}`;
        const transition = `${entity.from_state === null ? "null" : entity.from_state}->${entity.to_state === null ? "null" : entity.to_state}`;
        const allowed = allowedTransitions.get(entity.kind);
        if (allowed && !allowed.has(transition)) errors.push(`${scenario.scenario_id}/${step.step_id}: illegal ${entity.kind} transition ${transition}`);
        if (entityStates.has(key) && entity.from_state !== entityStates.get(key)) errors.push(`${scenario.scenario_id}/${step.step_id}: entity ${key} state discontinuity, expected ${entityStates.get(key)}, got ${entity.from_state}`);
        entityStates.set(key, entity.to_state);
      }
      const time = Date.parse(payload.occurred_at);
      if (time < previousTime) errors.push(`${scenario.scenario_id}/${step.step_id}: occurred_at is out of order`);
      previousTime = time;
    }
  }
  return errors;
}

const scenarioDocuments = scenarioPaths.map(readJson);
const document = {
  schema_version: scenarioDocuments[0]?.schema_version,
  scenarios: scenarioDocuments.flatMap((item) => item.scenarios ?? [])
};
const mutationArgument = process.argv.find((argument) => argument.startsWith("--mutation="))?.slice("--mutation=".length);
if (mutationArgument === "missing-message") document.scenarios[0].steps.splice(5, 1);
if (mutationArgument === "missing-correlation-path") delete document.scenarios[0].steps.find((step) => step.message_type === "CompletionReviewOutput").payload.refs.input_digest;
if (mutationArgument === "state-gap") document.scenarios.find((item) => item.scenario_id === "E2E-003").steps.find((step) => step.message_type === "TaskCompleted").payload.entity.from_state = "running";
if (mutationArgument === "bad-causation") document.scenarios[0].steps[1].payload.causation_message_id = "missing-message";
if (mutationArgument === "duplicate-idempotency") document.scenarios[0].steps[1].payload.idempotency_key = document.scenarios[0].steps[0].payload.idempotency_key;
if (mutationArgument === "missing-domain") document.scenarios[0].steps.find((step) => step.message_type === "EgressChallenge").payload.message_id = "unbound-message";
if (mutationArgument === "projection-domain-mismatch") document.scenarios[0].steps.find((step) => step.message_type === "EgressChallenge").payload.refs.challenge_id = "challenge-forged";
if (mutationArgument === "illegal-state") document.scenarios.find((item) => item.scenario_id === "E2E-003").steps.find((step) => step.message_type === "TaskWaiting").payload.entity.to_state = "teleported";
if (mutationArgument === "wrong-prior-causation") document.scenarios.find((item) => item.scenario_id === "E2E-002").steps.find((step) => step.message_type === "ParentIntegration").payload.causation_message_id = "e2-m01";
if (mutationArgument === "idempotent-redelivery") {
  const scenario = document.scenarios[0];
  const index = scenario.steps.findIndex((step) => step.message_type === "PolicyGrantReady");
  const duplicate = structuredClone(scenario.steps[index]);
  duplicate.step_id = `${duplicate.step_id}-REDLIVERY`;
  duplicate.payload.message_id = "e1-m14-redelivery";
  duplicate.payload.causation_message_id = duplicate.payload.redelivery_of_message_id = "e1-m14";
  duplicate.payload.delivery_attempt = 2;
  duplicate.payload.occurred_at = "2026-07-12T01:02:34Z";
  scenario.steps.splice(index + 1, 0, duplicate);
}
if (mutationArgument === "nested-correlation") {
  const scenario = document.scenarios.find((item) => item.scenario_id === "E2E-002");
  for (const step of scenario.steps) {
    if (step.payload.task_id === "task-child-002") step.payload.local_correlation_id = "local-child-002";
    else if (step.payload.task_id === "task-parent-002") step.payload.local_correlation_id = "local-parent-002";
  }
}
const failures = validateScenarioEnvelope(document);
const requirementDocument = readJson(requirementPath);
const domainBindingDocument = readJson(domainBindingPath);
const domainPayloadDocuments = fs.readdirSync(path.join(repoRoot, "examples/e2e-tabletop"))
  .filter((name) => /^domain-payloads-.*\.json$/.test(name))
  .sort()
  .map((name) => readJson(path.join(repoRoot, "examples/e2e-tabletop", name)));
const legacyScenarioDocument = readJson(path.join(repoRoot, "examples/e2e-tabletop/scenarios.json"));
const requirementSchemaFile = path.join(repoRoot, "schemas/draft-v0/common/sequence-invariant.schema.json");
const requirementSchema = readJson(requirementSchemaFile);
for (const [index, invariant] of (requirementDocument.requirements ?? []).entries()) {
  failures.push(...validate(invariant, requirementSchema, requirementSchemaFile, `requirements[${index}]`));
}
let checked = 0;
let domainChecked = 0;
const boundMessageIds = new Set();
const canonicalPayloads = new Map();
for (const binding of domainBindingDocument.bindings ?? []) {
  if (boundMessageIds.has(binding.message_id)) failures.push(`duplicate domain binding for ${binding.message_id}`);
  boundMessageIds.add(binding.message_id);
  const match = /^scenarios\.json#([^/]+)\/(.+)$/.exec(binding.source);
  if (!match) {
    failures.push(`${binding.message_id}: invalid domain source ${binding.source}`);
    continue;
  }
  const sourceScenario = legacyScenarioDocument.scenarios?.find((item) => item.scenario_id === match[1]);
  const sourceStep = sourceScenario?.steps?.find((item) => item.step_id === match[2]);
  if (!sourceStep) {
    failures.push(`${binding.message_id}: domain source not found ${binding.source}`);
    continue;
  }
  const [schemaFilePart, fragmentPart] = sourceStep.schema_ref.split("#", 2);
  const schemaFile = path.resolve(repoRoot, schemaFilePart);
  const schema = resolvePointer(readJson(schemaFile), fragmentPart === undefined ? "" : `#${fragmentPart}`);
  failures.push(...validate(sourceStep.payload, schema, schemaFile, `domain/${binding.message_id}`));
  canonicalPayloads.set(binding.message_id, sourceStep.payload);
  domainChecked += 1;
}
for (const record of domainBindingDocument.additional_records ?? []) {
  if (boundMessageIds.has(record.message_id)) failures.push(`duplicate domain payload for ${record.message_id}`);
  boundMessageIds.add(record.message_id);
  const [schemaFilePart, fragmentPart] = record.schema_ref.split("#", 2);
  const schemaFile = path.resolve(repoRoot, schemaFilePart);
  const schema = resolvePointer(readJson(schemaFile), fragmentPart === undefined ? "" : `#${fragmentPart}`);
  failures.push(...validate(record.payload, schema, schemaFile, `domain/${record.message_id}`));
  canonicalPayloads.set(record.message_id, record.payload);
  domainChecked += 1;
}
for (const record of domainPayloadDocuments.flatMap((item) => item.records ?? [])) {
  if (boundMessageIds.has(record.message_id)) failures.push(`duplicate domain payload for ${record.message_id}`);
  boundMessageIds.add(record.message_id);
  const [schemaFilePart, fragmentPart] = record.schema_ref.split("#", 2);
  const schemaFile = path.resolve(repoRoot, schemaFilePart);
  const schema = resolvePointer(readJson(schemaFile), fragmentPart === undefined ? "" : `#${fragmentPart}`);
  failures.push(...validate(record.payload, schema, schemaFile, `domain/${record.message_id}`));
  canonicalPayloads.set(record.message_id, record.payload);
  domainChecked += 1;
}
const domainRequiredTypes = new Set([
  "MemoryContextRequest", "MemoryContext", "TestToolCall", "GitHubToolCall", "GitHubToolCallRetry", "RegistryToolCall", "RegistryToolCallRetry", "ToolCall",
  "TestToolResult", "ToolResultAccepted", "AsyncOperationRunning", "AsyncOperationCompleted", "Continuation", "ResumeContext",
  "EgressAttemptBlocked", "EgressAttemptAllowed", "EgressChallenge", "GrantRequest", "GrantDecisionRecord", "PolicyGrantPendingActivation", "PolicyGrantActive", "PolicyActivationAck", "OutboundTransaction",
  "CompletionReviewInput", "CompletionReviewOutput", "ChildCompletionReviewInput", "ChildCompletionReviewOutput", "ParentCompletionReviewInput", "ParentCompletionReviewOutput", "RemediationCompletionReviewInput", "RemediationCompletionReviewOutput",
  "EpisodeAgentInput", "ChildEpisodeAgentInput", "TaskEpisode", "ChildTaskEpisode",
  "MailboxAsyncCompleted", "MailboxChildCompleted", "EgressAuditInput", "EgressReviewFinding", "EvidenceRecord", "RemediationTaskCommand",
  "DelegateCommand", "TaskRunning", "TaskWaiting", "TaskResumed", "TaskReviewingCompletion", "TaskCompleted",
  "ChildTaskCreated", "ChildTaskStarted", "ChildTaskReviewingCompletion", "ChildTaskCompleted",
  "ParentTaskRunning", "ParentTaskReviewingCompletion", "ParentTaskCompleted",
  "RemediationTaskCreated", "RemediationTaskReviewingCompletion", "RemediationTaskCompleted",
  "AsyncCompleted", "PolicyGrantReady", "ChildWorkspaceCreated", "MailboxAsyncConsumed", "MailboxChildConsumed",
  "PolicyRevisionJob", "PolicyAgentRevisionInput", "PolicyCandidate", "PolicyRegressionResult", "PolicyRevisionDecision", "GovernanceAuthorityRequest", "GovernanceAuthorityDecision", "PolicyRevisionPendingActivation", "PolicyRevisionActive"
]);
function collectNamedValues(value, result = new Map()) {
  if (Array.isArray(value)) {
    for (const item of value) collectNamedValues(item, result);
  } else if (value && typeof value === "object") {
    for (const [key, child] of Object.entries(value)) {
      if (child === null || ["string", "number", "boolean"].includes(typeof child)) {
        const values = result.get(key) ?? [];
        values.push(child);
        result.set(key, values);
      } else collectNamedValues(child, result);
    }
  }
  return result;
}
const scenarioIdSet = new Set((document.scenarios ?? []).map((scenario) => scenario.scenario_id));
if (!process.argv.includes("--partial")) {
  for (const requirement of requirementDocument.requirements ?? []) {
    if (!scenarioIdSet.has(requirement.scenario_id)) failures.push(`required scenario ${requirement.scenario_id} missing`);
  }
}
for (const scenario of document.scenarios ?? []) {
  const planes = new Set();
  for (const step of scenario.steps ?? []) {
    planes.add(step.from_plane);
    planes.add(step.to_plane);
    const [schemaFilePart, fragmentPart] = step.schema_ref.split("#", 2);
    const schemaFile = path.resolve(repoRoot, schemaFilePart);
    const schema = resolvePointer(readJson(schemaFile), fragmentPart === undefined ? "" : `#${fragmentPart}`);
    const errors = validate(step.payload, schema, schemaFile, `${scenario.scenario_id}/${step.step_id}`);
    failures.push(...errors);
    checked += 1;
    const canonicalMessageId = step.payload.redelivery_of_message_id ?? step.payload.message_id;
    if (domainRequiredTypes.has(step.message_type) && !boundMessageIds.has(canonicalMessageId)) failures.push(`${scenario.scenario_id}/${step.step_id}: canonical domain payload missing for ${step.message_type}`);
    const canonical = canonicalPayloads.get(canonicalMessageId);
    if (canonical) {
      const named = collectNamedValues(canonical);
      const taskCandidates = [...(named.get("task_id") ?? []), ...(named.get("owner_task_id") ?? []), ...(named.get("source_task_id") ?? []), ...(named.get("subject_id") ?? [])];
      if (taskCandidates.length > 0 && !taskCandidates.includes(step.payload.task_id)) failures.push(`${scenario.scenario_id}/${step.step_id}: sequence task_id disagrees with canonical payload`);
      const workspaceCandidates = named.get("workspace_id") ?? [];
      if (workspaceCandidates.length > 0 && !workspaceCandidates.includes(step.payload.workspace_id)) failures.push(`${scenario.scenario_id}/${step.step_id}: sequence workspace_id disagrees with canonical payload`);
      for (const [key, value] of Object.entries(step.payload.refs ?? {})) {
        const candidates = named.get(key);
        if (candidates && candidates.length > 0 && !candidates.includes(value)) failures.push(`${scenario.scenario_id}/${step.step_id}: sequence refs.${key} disagrees with canonical payload`);
      }
      const canonicalFromStates = named.get("from_state") ?? [];
      const canonicalToStates = named.get("to_state") ?? [];
      if (canonicalFromStates.length > 0 && !canonicalFromStates.includes(step.payload.entity?.from_state)) failures.push(`${scenario.scenario_id}/${step.step_id}: sequence from_state disagrees with canonical payload`);
      if (canonicalToStates.length > 0 && !canonicalToStates.includes(step.payload.entity?.to_state)) failures.push(`${scenario.scenario_id}/${step.step_id}: sequence to_state disagrees with canonical payload`);
    }
  }
  for (const plane of ["control-plane", "execution-plane", "governance-plane", "memory-plane"]) {
    if (!planes.has(plane)) failures.push(`${scenario.scenario_id}: does not cross ${plane}`);
  }
  failures.push(...validateSequenceInvariants(scenario));

  const requirement = (requirementDocument.requirements ?? []).find((item) => item.scenario_id === scenario.scenario_id);
  if (!requirement) {
    failures.push(`${scenario.scenario_id}: sequence requirement missing`);
  } else {
    const stepsByType = new Map();
    for (const [index, step] of scenario.steps.entries()) {
      if (stepsByType.has(step.message_type)) {
        const first = stepsByType.get(step.message_type).step;
        if (step.payload.redelivery_of_message_id !== first.payload.message_id) failures.push(`${scenario.scenario_id}: duplicate message type ${step.message_type}; requirement matching would be ambiguous`);
      } else stepsByType.set(step.message_type, { step, index });
    }
    let previousIndex = -1;
    for (const type of requirement.ordered_message_types) {
      const found = stepsByType.get(type);
      if (!found) failures.push(`${scenario.scenario_id}: required message ${type} missing`);
      else if (found.index <= previousIndex) failures.push(`${scenario.scenario_id}: required message ${type} is out of order`);
      else previousIndex = found.index;
    }
    const readPath = (value, dotted) => dotted.split(".").reduce((current, key) => current?.[key], value);
    for (const correlation of requirement.correlations) {
      const left = stepsByType.get(correlation.left_type)?.step.payload;
      const right = stepsByType.get(correlation.right_type)?.step.payload;
      if (left && right) {
        const leftValue = readPath(left, correlation.left_path);
        const rightValue = readPath(right, correlation.right_path);
        if (leftValue === undefined) failures.push(`${scenario.scenario_id}: correlation path missing ${correlation.left_type}.${correlation.left_path}`);
        if (rightValue === undefined) failures.push(`${scenario.scenario_id}: correlation path missing ${correlation.right_type}.${correlation.right_path}`);
        if (leftValue !== undefined && rightValue !== undefined && leftValue !== rightValue) failures.push(`${scenario.scenario_id}: correlation mismatch ${correlation.left_type}.${correlation.left_path} -> ${correlation.right_type}.${correlation.right_path}`);
      }
    }
    for (const causation of requirement.causations) {
      const cause = stepsByType.get(causation.cause_type)?.step.payload;
      const effect = stepsByType.get(causation.effect_type)?.step.payload;
      if (!cause || !effect) continue;
      if (effect.causation_message_id !== cause.message_id) failures.push(`${scenario.scenario_id}: causation mismatch ${causation.cause_type} -> ${causation.effect_type}`);
    }
  }
}

if (failures.length > 0) {
  console.error(`FAILED: ${failures.length} error(s) across ${checked} payload(s)`);
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`PASS: ${document.scenarios.length} scenarios, ${checked} sequence payloads, ${domainChecked} canonical domain payloads, ${document.scenarios.length} applied sequence requirements, schema and cross-step invariants satisfied`);
