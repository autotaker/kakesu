import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { rollbackWorkRepository, syncWorkAdapter } from "./agent-routing.mjs";
import { REPO_ROOT, acquireWorkRepoLock, git, parseArgs, workRoot } from "./lib.mjs";

export const WORK_CONFIG_ALLOWED_PATHS = Object.freeze([".codex/config.toml"]);

function changedFiles(repository) {
  return [...new Set([
    ...git(repository, ["diff", "--name-only"]).split("\n").filter(Boolean),
    ...git(repository, ["diff", "--cached", "--name-only"]).split("\n").filter(Boolean),
    ...git(repository, ["ls-files", "--others", "--exclude-standard"]).split("\n").filter(Boolean),
  ])];
}

function validateWorkRepository(root) {
  const result = spawnSync(
    process.execPath,
    [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root],
    { cwd: REPO_ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] },
  );
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error("WORK_VALIDATION_FAILED");
}

function conciseError(error) {
  return String(error).slice(0, 240).replace(/(?:sk-|Bearer\s+)[A-Za-z0-9._-]+/gi, "[REDACTED]");
}

export function buildWorkConfigSyncEvidence({ root, mode, result = null, commit = null, error = null }) {
  return {
    event: "work_config_sync",
    mode,
    owner: "lock-owning-parent",
    target: path.join(root, ".codex", "config.toml"),
    digest: result?.digest ?? null,
    changed: result?.changed ?? false,
    commit,
    error: error ? conciseError(error) : null,
  };
}

export function runWorkConfigSync({
  productRoot = REPO_ROOT,
  adapterRoot,
  mode = "sync",
  validateWork = validateWorkRepository,
  emit = (evidence) => process.stdout.write(`${JSON.stringify(evidence)}\n`),
} = {}) {
  if (!adapterRoot) throw new Error("WORK_CONFIG_ROOT_REQUIRED");
  if (!new Set(["sync", "check"]).has(mode)) throw new Error("WORK_CONFIG_MODE_INVALID");

  const root = path.resolve(adapterRoot);
  let release = null;
  let beforeHead = null;
  let result = null;
  let commit = null;
  try {
    release = acquireWorkRepoLock(root);
    beforeHead = git(root, ["rev-parse", "HEAD"]);

    if (mode === "sync") {
      if (git(root, ["config", "--get", "core.hooksPath"]) !== ".githooks") {
        throw new Error("WORK_HOOKS_PATH_INVALID");
      }
      const hook = path.join(root, ".githooks", "pre-commit");
      if (!fs.existsSync(hook) || !fs.statSync(hook).isFile() || (fs.statSync(hook).mode & 0o111) === 0) {
        throw new Error("WORK_PRE_COMMIT_HOOK_INVALID");
      }

      result = syncWorkAdapter({ productRoot, adapterRoot: root });
      if (result.changed) {
        const changed = changedFiles(root);
        if (changed.length !== 1 || changed[0] !== WORK_CONFIG_ALLOWED_PATHS[0]) {
          throw new Error(`WORK_CONFIG_SCOPE_VIOLATION:${changed.join(",")}`);
        }
        git(root, ["add", "--", ...WORK_CONFIG_ALLOWED_PATHS]);
        validateWork(root);
        git(root, ["commit", "-m", "governance: sync work adapter"], { env: {
          ...process.env,
          WORK_REPO_LOCK_HELD: "1",
          WORK_PARENT_COMMIT: "1",
          WORK_ACTION: "work-config-sync",
          WORK_ALLOWED_PATHS: JSON.stringify(WORK_CONFIG_ALLOWED_PATHS),
        } });
        commit = git(root, ["rev-parse", "HEAD"]);
      }
    } else {
      result = syncWorkAdapter({ productRoot, adapterRoot: root, check: true });
    }

    syncWorkAdapter({ productRoot, adapterRoot: root, check: true });
    validateWork(root);
    if (git(root, ["rev-parse", "HEAD"]) !== (commit ?? beforeHead)) throw new Error("WORK_CONFIG_HEAD_CHANGED");
    if (changedFiles(root).length) throw new Error("WORK_PARENT_LEFT_DIRTY");
    const evidence = buildWorkConfigSyncEvidence({ root, mode, result, commit });
    emit(evidence);
    return evidence;
  } catch (error) {
    let failure = error;
    if (beforeHead && mode === "sync") {
      try {
        rollbackWorkRepository(root, beforeHead);
      } catch (rollbackError) {
        failure = new Error(`${error.message};WORK_ROLLBACK_FAILED:${rollbackError.message}`);
      }
    }
    emit(buildWorkConfigSyncEvidence({ root, mode, result, commit: null, error: failure.message }));
    throw failure;
  } finally {
    release?.();
  }
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (isMain) {
  const args = parseArgs(process.argv.slice(2));
  runWorkConfigSync({
    adapterRoot: workRoot(args.work_root),
    mode: args.mode ?? "sync",
  });
}
