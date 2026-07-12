#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..");
const validator = path.join(repoRoot, "scripts/validate-tabletop-scenarios.mjs");
const baseline = spawnSync(process.execPath, [validator], { cwd: repoRoot, encoding: "utf8" });
if (baseline.status !== 0) {
  process.stderr.write(baseline.stdout + baseline.stderr);
  process.exit(1);
}

const mutations = ["missing-message", "missing-correlation-path", "state-gap", "bad-causation", "wrong-prior-causation", "duplicate-idempotency", "missing-domain", "projection-domain-mismatch", "illegal-state", "authority-bypasses-control", "incident-cascade-wrong-order"];
const falseNegatives = [];
for (const mutation of mutations) {
  const result = spawnSync(process.execPath, [validator, `--mutation=${mutation}`], { cwd: repoRoot, encoding: "utf8" });
  if (result.status === 0) falseNegatives.push(mutation);
}
if (falseNegatives.length > 0) {
  console.error(`FAILED: validator accepted mutations: ${falseNegatives.join(", ")}`);
  process.exit(1);
}
const redelivery = spawnSync(process.execPath, [validator, "--mutation=idempotent-redelivery"], { cwd: repoRoot, encoding: "utf8" });
if (redelivery.status !== 0) {
  process.stderr.write(redelivery.stdout + redelivery.stderr);
  process.exit(1);
}
const nestedCorrelation = spawnSync(process.execPath, [validator, "--mutation=nested-correlation"], { cwd: repoRoot, encoding: "utf8" });
if (nestedCorrelation.status !== 0) {
  process.stderr.write(nestedCorrelation.stdout + nestedCorrelation.stderr);
  process.exit(1);
}
console.log(`PASS: baseline, ${mutations.length} negative mutations, idempotent redelivery, and nested correlation`);
