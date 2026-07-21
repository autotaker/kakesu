import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { git, parseFrontmatter, writeFileAtomic } from "./lib.mjs";

const MODULE_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

export const ROLE_CONTRACTS = Object.freeze({
  main: { profile: "sol-high", model: "gpt-5.6-sol", effort: "high", sandbox: "workspace-write" },
  planner: { profile: "terra-medium", model: "gpt-5.6-terra", effort: "medium", sandbox: "workspace-write" },
  qa: { profile: "terra-medium", model: "gpt-5.6-terra", effort: "medium", sandbox: "workspace-write" },
  reviewer: { profile: "terra-medium", model: "gpt-5.6-terra", effort: "medium", sandbox: "workspace-write" },
  "dev-luna": { profile: "luna-xhigh", model: "gpt-5.6-luna", effort: "xhigh", sandbox: "workspace-write" },
  "dev-sol": { profile: "sol-high", model: "gpt-5.6-sol", effort: "high", sandbox: "workspace-write" },
  explorer: { profile: "luna-medium", model: "gpt-5.6-luna", effort: "medium", sandbox: "read-only" },
});

export const ACTION_ROLES = Object.freeze({
  task: "main",
  plan: "planner",
  "qa-plan": "qa",
  review: "reviewer",
  "qa-result": "qa",
  "main-transition": "main",
  governance: "main",
});

export const MAX_EXPLORER_QUESTION_LENGTH = 500;

const FIXED_KEYS = ["model", "model_reasoning_effort", "sandbox_mode"];
const PROJECT_CONFIG = Object.freeze({
  model: "gpt-5.6-sol",
  model_reasoning_effort: "high",
  sandbox_mode: "workspace-write",
});
const PROJECT_ROLE_REGISTRY = Object.freeze({
  main: { description: "Root orchestration and approval using Sol/high.", config_file: "agents/main.toml" },
  planner: { description: "PLAN authoring using Terra/medium.", config_file: "agents/planner.toml" },
  qa: { description: "QA planning and execution using Terra/medium.", config_file: "agents/qa.toml" },
  reviewer: { description: "Independent review using Terra/medium.", config_file: "agents/reviewer.toml" },
  "dev-luna": { description: "Low-risk DEV implementation using Luna/xhigh.", config_file: "agents/dev-luna.toml" },
  "dev-sol": { description: "High or unknown-risk DEV implementation using Sol/high.", config_file: "agents/dev-sol.toml" },
  explorer: { description: "Bounded read-only repository research using Luna/medium.", config_file: "agents/explorer.toml" },
});
const CODEX_DEFAULT_MAX_THREADS = 6;
const CODEX_DEFAULT_MAX_DEPTH = 1;
const PROHIBITED_PROJECT_KEYS = new Set(["hide_spawn_agent_metadata", "tool_namespace"]);

function tomlSection(content, name) {
  const lines = content.split(/\r?\n/);
  const start = lines.findIndex((line) => line.trim() === `[${name}]`);
  if (start === -1) return null;
  const next = lines.findIndex((line, index) => index > start && /^\s*\[/.test(line));
  return lines.slice(start + 1, next === -1 ? undefined : next).join("\n");
}

function quotedValue(content, key) {
  const match = content.match(new RegExp(`^${key}\\s*=\\s*"([^"]+)"\\s*$`, "m"));
  if (!match) throw new Error(`ROUTING_CONFIG_INVALID: missing ${key}`);
  return match[1];
}

function readProjectConfig(productRoot) {
  const file = path.join(productRoot, ".codex", "config.toml");
  const content = fs.readFileSync(file, "utf8");
  const topLevel = {};
  const registry = {};
  const seenSections = new Set();
  let section = null;

  for (const [index, sourceLine] of content.split(/\r?\n/).entries()) {
    const line = sourceLine.trim();
    if (!line || line.startsWith("#")) continue;

    const sectionMatch = line.match(/^\[([^\]]+)]$/);
    if (sectionMatch) {
      section = sectionMatch[1].trim();
      if (section === "features.multi_agent_v2") throw new Error("ROUTING_PROJECT_FEATURE_FORBIDDEN");
      if (section === "agents") throw new Error("ROUTING_PROJECT_AGENTS_HEADER_FORBIDDEN");
      if (!section.startsWith("agents.")) {
        throw new Error(`ROUTING_PROJECT_SECTION_UNKNOWN: ${section}`);
      }
      if (seenSections.has(section)) throw new Error(`ROUTING_PROJECT_SECTION_DUPLICATE: ${section}`);
      seenSections.add(section);
      if (section.startsWith("agents.")) {
        const role = section.slice("agents.".length);
        if (!Object.hasOwn(PROJECT_ROLE_REGISTRY, role)) throw new Error(`ROUTING_PROJECT_ROLE_UNKNOWN: ${role}`);
        registry[role] = {};
      }
      continue;
    }

    const assignment = line.match(/^([A-Za-z0-9_.-]+)\s*=\s*(.+)$/);
    if (!assignment) throw new Error(`ROUTING_PROJECT_CONFIG_INVALID: line ${index + 1}`);
    const [, key, rawValue] = assignment;
    if (PROHIBITED_PROJECT_KEYS.has(key)) throw new Error(`ROUTING_PROJECT_KEY_FORBIDDEN: ${key}`);
    if (key === "features.multi_agent_v2" || key.startsWith("features.multi_agent_v2.")) {
      throw new Error("ROUTING_PROJECT_FEATURE_FORBIDDEN");
    }
    if (section === null && (key === "agents" || key.startsWith("agents."))) {
      if (/^agents\.(?:max_threads|max_depth)$/.test(key)) {
        throw new Error(`ROUTING_PROJECT_GLOBAL_OVERRIDE_FORBIDDEN: ${key.slice("agents.".length)}`);
      }
      throw new Error(`ROUTING_PROJECT_KEY_UNKNOWN: ${key}`);
    }
    const expected = section === null
      ? PROJECT_CONFIG
      : PROJECT_ROLE_REGISTRY[section.slice("agents.".length)];
    if (!expected || !Object.hasOwn(expected, key)) throw new Error(`ROUTING_PROJECT_KEY_UNKNOWN: ${key}`);
    const destination = section === null ? topLevel : registry[section.slice("agents.".length)];
    if (Object.hasOwn(destination, key)) throw new Error(`ROUTING_PROJECT_KEY_DUPLICATE: ${key}`);
    const valueMatch = rawValue.match(/^"([^"]*)"$/);
    if (!valueMatch) throw new Error(`ROUTING_PROJECT_CONFIG_INVALID: line ${index + 1}`);
    const value = valueMatch[1];
    if (value !== expected[key]) {
      const error = key === "config_file" ? "ROUTING_PROJECT_ROLE_MAPPING_MISMATCH" : "ROUTING_PROJECT_VALUE_MISMATCH";
      throw new Error(`${error}: ${section ?? key}`);
    }
    destination[key] = value;
  }

  for (const key of FIXED_KEYS) {
    if (!Object.hasOwn(topLevel, key)) throw new Error(`ROUTING_CONFIG_INVALID: missing ${key}`);
  }
  for (const [role, expected] of Object.entries(PROJECT_ROLE_REGISTRY)) {
    if (!Object.hasOwn(registry, role)) throw new Error(`ROUTING_PROJECT_ROLE_MISSING: ${role}`);
    for (const key of Object.keys(expected)) {
      if (!Object.hasOwn(registry[role], key)) throw new Error(`ROUTING_PROJECT_ROLE_FIELD_MISSING: ${role}.${key}`);
    }
  }
  return {
    top_level: Object.fromEntries(FIXED_KEYS.map((key) => [key, topLevel[key]])),
    agents: Object.fromEntries(Object.keys(PROJECT_ROLE_REGISTRY).map((role) => [role, registry[role]])),
  };
}

export function readCanonicalContracts(productRoot = MODULE_ROOT) {
  readProjectConfig(productRoot);
  const contracts = {};
  for (const [role, expected] of Object.entries(ROLE_CONTRACTS)) {
    const file = path.join(productRoot, ".codex", "agents", `${role}.toml`);
    if (!fs.existsSync(file)) throw new Error(`ROUTING_CONFIG_MISSING: ${file}`);
    const content = fs.readFileSync(file, "utf8");
    const actual = {
      profile: expected.profile,
      model: quotedValue(content, "model"),
      effort: quotedValue(content, "model_reasoning_effort"),
      sandbox: quotedValue(content, "sandbox_mode"),
    };
    if (quotedValue(content, "name") !== role) throw new Error(`ROUTING_ROLE_NAME_MISMATCH: ${role}`);
    for (const key of ["model", "effort", "sandbox"]) {
      if (actual[key] !== expected[key]) throw new Error(`ROUTING_CONTRACT_MISMATCH: ${role}.${key}`);
    }
    if (role === "explorer") {
      if (!/exactly one bounded/i.test(content) || !/Do not .*spawn/i.test(content)) {
        throw new Error("ROUTING_EXPLORER_POLICY_MISSING");
      }
      if (!/^max_depth\s*=\s*0\s*$/m.test(content) || !/^max_threads\s*=\s*1\s*$/m.test(content)) {
        throw new Error("ROUTING_EXPLORER_CHILD_POLICY_MISSING");
      }
    } else {
      const agents = tomlSection(content, "agents");
      if (agents === null || !/^max_threads\s*=\s*2\s*$/m.test(agents)) {
        throw new Error(`ROUTING_ROLE_THREAD_POLICY_MISMATCH: ${role}`);
      }
      if (/^max_depth\s*=/m.test(agents)) {
        throw new Error(`ROUTING_ROLE_LOCAL_DEPTH_FORBIDDEN: ${role}`);
      }
    }
    contracts[role] = actual;
  }
  return contracts;
}

export function canonicalDigest(productRoot = MODULE_ROOT) {
  const canonical = { project: readProjectConfig(productRoot), roles: readCanonicalContracts(productRoot) };
  return crypto.createHash("sha256").update(JSON.stringify(canonical)).digest("hex");
}

export function renderWorkAdapter(productRoot, adapterRoot) {
  readCanonicalContracts(productRoot);
  const relativeAgents = path.relative(path.join(adapterRoot, ".codex"), path.join(productRoot, ".codex", "agents"));
  const lines = [
    "# Generated by make work-config-sync. Do not edit by hand.",
    `# canonical_digest = "${canonicalDigest(productRoot)}"`,
    ...Object.entries(PROJECT_CONFIG).map(([key, value]) => `${key} = ${JSON.stringify(value)}`),
  ];
  for (const role of Object.keys(ROLE_CONTRACTS)) {
    lines.push(
      "",
      `[agents.${role}]`,
      `description = "${role} canonical role contract"`,
      `config_file = "${path.posix.join(relativeAgents.split(path.sep).join("/"), `${role}.toml`)}"`,
    );
  }
  return `${lines.join("\n")}\n`;
}

export function syncWorkAdapter({ productRoot = MODULE_ROOT, adapterRoot, check = false }) {
  const target = path.join(adapterRoot, ".codex", "config.toml");
  const expected = renderWorkAdapter(productRoot, adapterRoot);
  if (check) {
    if (!fs.existsSync(target) || fs.readFileSync(target, "utf8") !== expected) {
      throw new Error("ROUTING_ADAPTER_DRIFT: run make work-config-sync");
    }
    return { target, digest: canonicalDigest(productRoot), changed: false };
  }
  fs.mkdirSync(path.dirname(target), { recursive: true });
  const changed = !fs.existsSync(target) || fs.readFileSync(target, "utf8") !== expected;
  if (changed) writeFileAtomic(target, expected);
  return { target, digest: canonicalDigest(productRoot), changed };
}

export function validateDevSelection(plan) {
  const profile = plan.approved_dev_profile;
  if (!new Set(["luna-xhigh", "sol-high"]).has(profile)) throw new Error("DEV_PROFILE_UNKNOWN");
  if (!plan.approved_dev_profile_reason || typeof plan.approved_dev_profile_reason !== "string") {
    throw new Error("DEV_PROFILE_REASON_MISSING");
  }
  const signals = plan.approved_dev_profile_risk_signals;
  if (!Array.isArray(signals)) throw new Error("DEV_PROFILE_RISK_SIGNALS_MISSING");
  if (profile === "luna-xhigh" && signals.length) throw new Error("DEV_LUNA_HAS_RISK_SIGNAL");
  if (profile === "sol-high" && signals.length === 0) throw new Error("DEV_SOL_RISK_SIGNAL_MISSING");
  const promotions = plan.dev_profile_promotions ?? [];
  let previous = promotions.length ? promotions[0].from : profile;
  for (const promotion of promotions) {
    if (promotion.from !== "luna-xhigh" || promotion.to !== "sol-high" || previous !== "luna-xhigh") {
      throw new Error("DEV_PROFILE_DOWNGRADE_OR_INVALID_PROMOTION");
    }
    for (const key of ["signal", "reason", "approved_by", "approved_at"]) {
      if (!promotion[key]) throw new Error(`DEV_PROFILE_PROMOTION_${key.toUpperCase()}_MISSING`);
    }
    previous = "sol-high";
  }
  if (promotions.length > 0 && profile !== previous) {
    throw new Error("DEV_PROFILE_PROMOTION_RESULT_MISMATCH");
  }
  return profile;
}

export function roleForAction(action, planFile) {
  if (action !== "handover") return ACTION_ROLES[action] ?? null;
  if (!planFile) throw new Error("DEV_PLAN_REQUIRED");
  const profile = validateDevSelection(parseFrontmatter(planFile));
  return profile === "luna-xhigh" ? "dev-luna" : "dev-sol";
}

export function assertFixedOverrides(contract, args) {
  const supplied = {
    profile: args.profile,
    model: args.model,
    effort: args.effort ?? args.model_reasoning_effort,
  };
  for (const [key, value] of Object.entries(supplied)) {
    if (value && value !== contract[key]) throw new Error(`ROUTING_OVERRIDE_MISMATCH: ${key}`);
  }
}

export function resolveFixedRoute({ action, planFile, args = {}, productRoot = MODULE_ROOT }) {
  const role = roleForAction(action, planFile);
  if (!role) throw new Error(`ROUTING_ROLE_UNKNOWN: ${action}`);
  const contract = readCanonicalContracts(productRoot)[role];
  assertFixedOverrides(contract, args);
  return { role, ...contract };
}

export function validateExplorerQuestion(question) {
  if (typeof question !== "string" || !question.trim()) {
    throw new Error("ROUTING_BOUNDED_QUESTION_REQUIRED");
  }
  if (question !== question.trim() || /[\r\n]/.test(question) || question.length > MAX_EXPLORER_QUESTION_LENGTH) {
    throw new Error("ROUTING_BOUNDED_QUESTION_INVALID");
  }
  return question;
}

export function validateDelegation({ chain, questions = [], threads = 1 }) {
  if (!Array.isArray(chain) || chain.length < 2 || chain[0] !== "root" || chain.at(-1) !== "explorer") {
    throw new Error("ROUTING_DELEGATION_INVALID");
  }
  if (chain.slice(0, -1).includes("explorer")) throw new Error("ROUTING_EXPLORER_SPAWN_FORBIDDEN");
  if (chain.length - 1 > CODEX_DEFAULT_MAX_DEPTH) throw new Error("ROUTING_MAX_DEPTH_EXCEEDED");
  if (!Number.isInteger(threads) || threads < 1 || threads > CODEX_DEFAULT_MAX_THREADS) {
    throw new Error("ROUTING_MAX_THREADS_EXCEEDED");
  }
  if (questions.length !== 1) {
    throw new Error("ROUTING_BOUNDED_QUESTION_REQUIRED");
  }
  validateExplorerQuestion(questions[0]);
  return true;
}

export function validateChildOutcome({ childExit, beforeHead, afterHead, stagedFiles = [], changedFiles = [], allowedPaths = [] }) {
  if (childExit !== 0) throw new Error("WORK_CHILD_FAILED");
  if (beforeHead !== afterHead) throw new Error("WORK_CHILD_COMMIT_FORBIDDEN");
  if (stagedFiles.length) throw new Error("WORK_CHILD_STAGE_FORBIDDEN");
  if (!changedFiles.length) throw new Error("WORK_NO_CHANGES");
  const matches = (file) => allowedPaths.some((rule) => rule.endsWith("/**") ? file.startsWith(rule.slice(0, -2)) : file === rule);
  const forbidden = changedFiles.filter((file) => !matches(file));
  if (forbidden.length) throw new Error(`WORK_SCOPE_VIOLATION:${forbidden.join(",")}`);
  return changedFiles;
}

export function rollbackWorkRepository(root, beforeHead) {
  if (!beforeHead) throw new Error("WORK_ROLLBACK_HEAD_MISSING");
  git(root, ["reset", "--hard", beforeHead]);
  git(root, ["clean", "-ffd"]);
  if (git(root, ["rev-parse", "HEAD"]) !== beforeHead) throw new Error("WORK_ROLLBACK_HEAD_MISMATCH");
  const status = git(root, ["status", "--porcelain"]);
  if (status) throw new Error(`WORK_ROLLBACK_DIRTY:${status.replaceAll("\n", ",")}`);
}

export function buildLaunchEvidence({ route, cwd, allowedPaths, childResult = null, commit = null, error = null, legacy = false }) {
  return {
    event: "agent_launch",
    route: legacy ? "legacy" : "fixed-role",
    role: route?.role ?? null,
    profile: route?.profile ?? null,
    model: route?.model ?? null,
    effort: route?.effort ?? null,
    cwd: path.resolve(cwd),
    sandbox: route?.sandbox ?? "workspace-write",
    write_scope: route?.sandbox === "read-only" ? "none" : "allowed-paths",
    allowed_paths: [...allowedPaths],
    stdin: "closed",
    child_result: childResult,
    commit,
    error: error ? String(error).slice(0, 240).replace(/(?:sk-|Bearer\s+)[A-Za-z0-9._-]+/gi, "[REDACTED]") : null,
  };
}

export function codexCommand(route, cwd, prompt) {
  return [
    "exec",
    "-C",
    cwd,
    "--sandbox",
    route.sandbox,
    "-m",
    route.model,
    "-c",
    `model_reasoning_effort=${JSON.stringify(route.effort)}`,
    prompt,
  ];
}

export function assertRoleFilesHaveOnlyKnownContractKeys(productRoot = MODULE_ROOT) {
  for (const role of Object.keys(ROLE_CONTRACTS)) {
    const content = fs.readFileSync(path.join(productRoot, ".codex", "agents", `${role}.toml`), "utf8");
    for (const key of FIXED_KEYS) quotedValue(content, key);
  }
}
