import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
  ROLE_CONTRACTS,
  assertFixedOverrides,
  buildLaunchEvidence,
  canonicalDigest,
  readCanonicalContracts,
  renderWorkAdapter,
  syncWorkAdapter,
  validateDelegation,
  validateDevSelection,
  validateChildOutcome,
} from "./agent-routing.mjs";

const productRoot = path.resolve(import.meta.dirname, "../..");

test("role routing matches the canonical model and effort contracts", () => {
  assert.deepEqual(readCanonicalContracts(productRoot), ROLE_CONTRACTS);
  assert.deepEqual(ROLE_CONTRACTS.main, { profile: "sol-high", model: "gpt-5.6-sol", effort: "high", sandbox: "workspace-write" });
  for (const role of ["planner", "qa", "reviewer"]) assert.equal(ROLE_CONTRACTS[role].model, "gpt-5.6-terra");
  assert.deepEqual(ROLE_CONTRACTS.explorer, { profile: "luna-medium", model: "gpt-5.6-luna", effort: "medium", sandbox: "read-only" });
});

test("fixed role overrides fail closed and redundant canonical values pass", () => {
  assert.doesNotThrow(() => assertFixedOverrides(ROLE_CONTRACTS.planner, { profile: "terra-medium", model: "gpt-5.6-terra", effort: "medium" }));
  assert.throws(() => assertFixedOverrides(ROLE_CONTRACTS.planner, { model: "gpt-5.6-sol" }), /ROUTING_OVERRIDE_MISMATCH/);
  assert.throws(() => assertFixedOverrides(ROLE_CONTRACTS.main, { effort: "medium" }), /ROUTING_OVERRIDE_MISMATCH/);
});

test("DEV selection permits Luna only without risk and validates promotion", () => {
  assert.equal(validateDevSelection({ approved_dev_profile: "luna-xhigh", approved_dev_profile_reason: "local", approved_dev_profile_risk_signals: [] }), "luna-xhigh");
  assert.equal(validateDevSelection({ approved_dev_profile: "sol-high", approved_dev_profile_reason: "contract", approved_dev_profile_risk_signals: ["contract"] }), "sol-high");
  assert.throws(() => validateDevSelection({ approved_dev_profile: "luna-xhigh", approved_dev_profile_reason: "bad", approved_dev_profile_risk_signals: ["security"] }), /DEV_LUNA_HAS_RISK_SIGNAL/);
  assert.equal(validateDevSelection({
    approved_dev_profile: "sol-high",
    approved_dev_profile_reason: "promoted",
    approved_dev_profile_risk_signals: ["cross_cutting"],
    dev_profile_promotions: [{ from: "luna-xhigh", to: "sol-high", signal: "cross_cutting", reason: "expanded", approved_by: "main", approved_at: "2026-07-14" }],
  }), "sol-high");
});

test("explorer delegation accepts only depth two, two threads, and one question", () => {
  assert.equal(validateDelegation({ chain: ["root", "explorer"], questions: ["Where is routing defined?"] }), true);
  assert.equal(validateDelegation({ chain: ["root", "planner", "explorer"], questions: ["Which file owns the gate?"], threads: 2 }), true);
  assert.throws(() => validateDelegation({ chain: ["root", "planner", "qa", "explorer"], questions: ["x"] }), /MAX_DEPTH/);
  assert.throws(() => validateDelegation({ chain: ["root", "explorer", "explorer"], questions: ["x"] }), /EXPLORER_SPAWN/);
  assert.throws(() => validateDelegation({ chain: ["root", "explorer"], questions: ["x", "y"] }), /BOUNDED_QUESTION/);
  assert.throws(() => validateDelegation({ chain: ["root", "explorer"], questions: ["x"], threads: 3 }), /MAX_THREADS/);
});

test("work adapter generation is deterministic and detects drift", () => {
  const adapterRoot = fs.mkdtempSync(path.join(os.tmpdir(), "routing-adapter-"));
  try {
    const first = syncWorkAdapter({ productRoot, adapterRoot });
    assert.equal(first.changed, true);
    assert.equal(first.digest, canonicalDigest(productRoot));
    assert.equal(syncWorkAdapter({ productRoot, adapterRoot }).changed, false);
    assert.doesNotThrow(() => syncWorkAdapter({ productRoot, adapterRoot, check: true }));
    const target = path.join(adapterRoot, ".codex", "config.toml");
    fs.appendFileSync(target, "# drift\n");
    assert.throws(() => syncWorkAdapter({ productRoot, adapterRoot, check: true }), /ROUTING_ADAPTER_DRIFT/);
    assert.match(renderWorkAdapter(productRoot, adapterRoot), /max_depth = 2/);
  } finally {
    fs.rmSync(adapterRoot, { recursive: true, force: true });
  }
});

test("launch evidence is concise, closed-stdin, and redacts secret-like values", () => {
  const evidence = buildLaunchEvidence({
    route: { role: "planner", ...ROLE_CONTRACTS.planner }, cwd: productRoot, allowedPaths: ["tasks/TASK-0002/PLAN.md"],
    childResult: { exit_code: 1 }, error: "Bearer abcdef123456", commit: null,
  });
  assert.equal(evidence.stdin, "closed");
  assert.equal(evidence.commit, null);
  assert.equal(evidence.error, "[REDACTED]");
  assert.ok(JSON.stringify(evidence).length < 1000);
  assert.equal("raw_log" in evidence, false);
});

test("parent commit preconditions reject child commit, stage, failure, and scope drift", () => {
  const base = { childExit: 0, beforeHead: "a", afterHead: "a", stagedFiles: [], changedFiles: ["tasks/TASK-0002/PLAN.md"], allowedPaths: ["tasks/TASK-0002/PLAN.md"] };
  assert.deepEqual(validateChildOutcome(base), base.changedFiles);
  assert.throws(() => validateChildOutcome({ ...base, childExit: 1 }), /WORK_CHILD_FAILED/);
  assert.throws(() => validateChildOutcome({ ...base, afterHead: "b" }), /WORK_CHILD_COMMIT_FORBIDDEN/);
  assert.throws(() => validateChildOutcome({ ...base, stagedFiles: base.changedFiles }), /WORK_CHILD_STAGE_FORBIDDEN/);
  assert.throws(() => validateChildOutcome({ ...base, changedFiles: ["backlog.yaml"] }), /WORK_SCOPE_VIOLATION/);
});
