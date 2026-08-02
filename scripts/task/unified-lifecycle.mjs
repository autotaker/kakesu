import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import YAML from "yaml";
import {
  REPO_ROOT, REQUIRED_TASK_FILES, acquireWorkRepoLock, assertSlug, assertTaskId, dateInTimezone,
  changedContentDigest, findMainWorktree, git, isMainManagedPath, parseArgs, parseFrontmatter, readYaml, replaceTemplate,
  resolveInside, taskById, writeFileAtomic, writeYaml,
} from "./lib.mjs";
import { validateDevSelection } from "./agent-routing.mjs";
import { buildWikiIndex } from "./wiki-index.mjs";

const ACTION_FILES = {
  task: ["TASK.md"], plan: ["PLAN.md"], "qa-plan": ["QA_PLAN.md"], review: ["REVIEW_RESULT.md"],
  "qa-result": ["QA_RESULT.md"], handover: ["HANDOVER.md"],
};
const PLANNING_FILES = ["TASK.md", "PLAN.md", "QA_PLAN.md", "REVIEW_RESULT.md", "QA_RESULT.md", "HANDOVER.md"];

function run(command, argv, options = {}) {
  const result = spawnSync(command, argv, { encoding: "utf8", ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error((result.stderr || result.stdout || `${command} failed`).trim());
  return result.stdout.trim();
}

function output(command, argv, cwd, allowFailure = false) {
  const result = spawnSync(command, argv, { cwd, encoding: "utf8" });
  if (result.error) throw result.error;
  if (result.status !== 0 && !allowFailure) throw new Error((result.stderr || result.stdout).trim());
  return result;
}

function lines(value) { return value.split("\n").filter(Boolean); }

function changedFiles(root) {
  return [...new Set([
    ...lines(git(root, ["diff", "--name-only", "--diff-filter=ACMRD"])),
    ...lines(git(root, ["diff", "--cached", "--name-only", "--diff-filter=ACMRD"])),
    ...lines(git(root, ["ls-files", "--others", "--exclude-standard"])),
  ])].sort();
}

function taskContext(root, taskId) {
  assertTaskId(taskId);
  const backlog = readYaml(path.join(root, "backlog.yaml"));
  const task = taskById(backlog, taskId);
  return { backlog, task, taskDir: task.task_dir };
}

function allowedFor(root, action, taskId) {
  if (action === "bootstrap") {
    if (taskId !== "TASK-0033" || !fs.existsSync(path.join(root, "tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json"))) {
      throw new Error("bootstrap evidence action requires TASK-0033 and its validated manifest");
    }
    return ["backlog.yaml", "project.yaml", "tasks/", "wiki/", "lap30/", "viewer/index.html"];
  }
  if (action === "task-start-rollback") return ["backlog.yaml", `tasks/${taskId}-`];
  if (action === "wiki") return ["wiki/semantic/", "wiki/decisions/", "wiki/ingestions/", "wiki/index.json"];
  const { taskDir } = taskContext(root, taskId);
  if (action === "planning-gate") return ["backlog.yaml", ...PLANNING_FILES.map((file) => `${taskDir}/${file}`)];
  if (action in ACTION_FILES) return ACTION_FILES[action].map((file) => `${taskDir}/${file}`);
  if (action === "task-start") return ["backlog.yaml", `${taskDir}/`];
  if (action === "main-transition") return ["backlog.yaml", `${taskDir}/TASK.md`, `${taskDir}/HANDOVER.md`];
  if (action === "sync") return ["backlog.yaml", "wiki/", "viewer/index.html", "tasks/"];
  throw new Error(`Unknown evidence action: ${action}`);
}

function matches(file, rules) { return rules.some((rule) => rule.endsWith("/") || rule.endsWith("-") ? file.startsWith(rule) : file === rule); }

function validatePlanningState(root, taskId, changed) {
  const { task, taskDir } = taskContext(root, taskId);
  const planning = new Set(["backlog.yaml", `${taskDir}/TASK.md`, `${taskDir}/PLAN.md`, `${taskDir}/QA_PLAN.md`]);
  if (!changed.some((file) => planning.has(file))) throw new Error("planning-gate requires a non-empty planning subset");
  const forbidden = changed.filter((file) => !matches(file, allowedFor(root, "planning-gate", taskId)));
  if (forbidden.length) throw new Error(`planning-gate scope violation: ${forbidden.join(", ")}`);
  const plan = parseFrontmatter(path.join(root, taskDir, "PLAN.md"));
  const qaPlan = parseFrontmatter(path.join(root, taskDir, "QA_PLAN.md"));
  const taskFile = parseFrontmatter(path.join(root, taskDir, "TASK.md"));
  if (plan.task_id !== taskId || qaPlan.task_id !== taskId || taskFile.task_id !== taskId) throw new Error("planning-gate task identity mismatch");
  if (plan.status !== "approved" || plan.approved_by !== task.assignees?.main || plan.planner_agent !== task.assignees?.planner) throw new Error("planning-gate PLAN approval or role mismatch");
  if (qaPlan.status !== "approved" || qaPlan.approved_by !== task.assignees?.main || qaPlan.qa_agent !== task.assignees?.qa) throw new Error("planning-gate QA PLAN approval or role mismatch");
  validateDevSelection(plan);
  if (!new Set(["dev", "plan"]).has(task.status)) throw new Error("planning-gate backlog status is invalid");
}

function restoreMerge(root) {
  output("git", ["merge", "--abort"], root, true);
}

function fastForwardPlanningWorktree(root, taskId, commit) {
  const { task } = taskContext(root, taskId);
  const worktree = resolveInside(root, task.worktree, `${taskId} worktree`);
  if (!fs.existsSync(worktree) || git(worktree, ["branch", "--show-current"]) !== task.branch) {
    throw new Error(`${taskId}: planning-gate requires the recorded Task worktree`);
  }
  if (git(worktree, ["status", "--porcelain"])) throw new Error(`${taskId}: planning worktree must be clean before fast-forward`);
  git(worktree, ["merge", "--ff-only", commit]);
  if (git(worktree, ["rev-parse", "HEAD"]) !== commit) throw new Error(`${taskId}: planning worktree did not fast-forward`);
}

function assertMain(root) {
  if (git(root, ["branch", "--show-current"]) !== "main") throw new Error("Evidence writes require the explicit main worktree");
  const registered = findMainWorktree(root);
  if (fs.realpathSync(registered) !== fs.realpathSync(root)) throw new Error(`Explicit root is not the registered main worktree: ${root}`);
}

const OPERATION_SCHEMAS = ["backlog", "decision", "ingestion-receipt", "bootstrap-manifest"];
const PRE_MERGE_EVIDENCE_ACTIONS = new Set(["handover", "review", "qa-result"]);
const BOOTSTRAP_MANIFEST_PATH = "tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json";

function hasOperationSchemas(root) {
  return OPERATION_SCHEMAS.every((name) => fs.existsSync(path.join(root, "schemas", "operations", `${name}.schema.json`)));
}

export function resolveOperationsValidation(root, action, taskId, candidateRoot = REPO_ROOT) {
  if (hasOperationSchemas(root)) return { mode: "main", validatorRoot: root, schemaRoot: root };
  if (!PRE_MERGE_EVIDENCE_ACTIONS.has(action)) {
    throw new Error(`main operations schemas are unavailable for ${action}`);
  }
  if (!hasOperationSchemas(candidateRoot)) throw new Error("candidate operations schemas are incomplete");
  const { task, taskDir } = taskContext(root, taskId);
  const handover = parseFrontmatter(path.join(root, taskDir, "HANDOVER.md"));
  const candidateCommit = git(candidateRoot, ["rev-parse", "HEAD"]);
  if (git(candidateRoot, ["branch", "--show-current"]) !== task.branch) throw new Error("validator must run from the recorded candidate branch");
  if (git(root, ["rev-parse", task.branch]) !== candidateCommit) throw new Error("validator candidate differs from the recorded branch head");
  if (git(candidateRoot, ["status", "--porcelain"])) throw new Error("validator candidate worktree must be clean");
  if (!/^[0-9a-f]{40}$/.test(handover.candidate_commit ?? "") || handover.candidate_commit !== candidateCommit) {
    throw new Error(`validator candidate is not bound by HANDOVER: candidate=${handover.candidate_commit ?? "missing"}/${candidateCommit}`);
  }
  return { mode: "pre-merge-candidate", validatorRoot: candidateRoot, schemaRoot: candidateRoot };
}

function validateOperations(root, action, taskId, candidateRoot = REPO_ROOT) {
  const validation = resolveOperationsValidation(root, action, taskId, candidateRoot);
  run(process.execPath, [
    path.join(validation.validatorRoot, "scripts/task/validate-work.mjs"),
    "--work-root", root,
    "--schema-root", validation.schemaRoot,
  ], { cwd: root });
}

function validateEvidenceAction(root, action, taskId, candidateRoot = REPO_ROOT) {
  if (action === "bootstrap") {
    run(process.execPath, [
      path.join(REPO_ROOT, "scripts/task/migrate-operations.mjs"),
      "--mode", "verify",
      "--target", root,
    ], { cwd: root });
    return;
  }
  validateOperations(root, action, taskId, candidateRoot);
}

export function evidenceCommit({ root, action, taskId, message, push = true, validate = true, candidateRoot = REPO_ROOT }) {
  assertMain(root);
  const rules = allowedFor(root, action, taskId);
  const release = acquireWorkRepoLock(root, { requireClean: false });
  const before = git(root, ["rev-parse", "HEAD"]);
  let prevalidatedDigest = null;
  let commitCreated = false;
  let pushed = false;
  let planningBranchBefore = null;
  let planningWorktree = null;
  try {
    const inputChanged = changedFiles(root);
    if (!inputChanged.length) return { commit: null, pushed: false, changed: [] };
    if (action === "planning-gate") validatePlanningState(root, taskId, inputChanged);
    else {
      const forbidden = inputChanged.filter((file) => !matches(file, rules));
      if (forbidden.length) throw new Error(`Evidence scope violation for ${action}: ${forbidden.join(", ")}`);
    }
    if (action === "wiki") {
      const output = path.join(root, "wiki", "index.json");
      writeFileAtomic(output, `${JSON.stringify(buildWikiIndex(root), null, 2)}\n`);
    }
    const changed = changedFiles(root);
    if (!changed.length) return { commit: null, pushed: false, changed: [] };
    if (action === "wiki") {
      const forbidden = changed.filter((file) => !matches(file, rules));
      if (forbidden.length) throw new Error(`Evidence scope violation for ${action} after index generation: ${forbidden.join(", ")}`);
    }
    prevalidatedDigest = validate ? changedContentDigest(root, { cached: false }) : null;
    if (action === "planning-gate") {
      const { task } = taskContext(root, taskId);
      planningWorktree = resolveInside(root, task.worktree, `${taskId} worktree`);
      if (!fs.existsSync(planningWorktree) || git(planningWorktree, ["branch", "--show-current"]) !== task.branch) {
        throw new Error(`${taskId}: planning-gate requires the recorded Task worktree`);
      }
      planningBranchBefore = git(root, ["rev-parse", task.branch]);
      if (git(planningWorktree, ["status", "--porcelain"])) throw new Error(`${taskId}: planning worktree must be clean before planning-gate`);
    }
    // Bootstrap runs before this product change is merged, so its verifier and
    // schemas must come from the approved Task worktree, not the old main tree.
    if (validate) validateOperations(root, action === "planning-gate" ? "plan" : action, taskId, candidateRoot);
    git(root, ["add", "--", ...changed]);
    git(root, ["commit", "-m", message], { env: {
      ...process.env, WORK_REPO_LOCK_HELD: "1", WORK_PARENT_COMMIT: "1", WORK_ACTION: action,
      WORK_ALLOWED_PATHS: JSON.stringify(rules),
      ...(prevalidatedDigest ? { WORK_VALIDATED_DIGEST: prevalidatedDigest } : {}),
    } });
    commitCreated = true;
    let commit = git(root, ["rev-parse", "HEAD"]);
    if (action === "planning-gate") fastForwardPlanningWorktree(root, taskId, commit);
    if (!push) return { commit, pushed: false, changed };
    for (let retry = 0; retry <= 2; retry += 1) {
      const pushResult = output("git", ["push", "origin", "HEAD:main"], root, true);
      if (pushResult.status === 0) {
        pushed = true;
        return { commit, pushed: true, changed, retries: retry };
      }
      if (retry === 2) throw new Error(`Evidence push retry limit reached: ${(pushResult.stderr || pushResult.stdout).trim()}`);
      git(root, ["fetch", "origin", "main"]);
      const rebase = output("git", ["rebase", "origin/main"], root, true);
      if (rebase.status !== 0) {
        output("git", ["rebase", "--abort"], root, true);
        throw new Error(`Evidence rebase conflict; manual reconciliation required: ${(rebase.stderr || rebase.stdout).trim()}`);
      }
      if (validate) validateEvidenceAction(root, action, taskId, candidateRoot);
      commit = git(root, ["rev-parse", "HEAD"]);
    }
  } catch (error) {
    if (action === "planning-gate" && commitCreated && !pushed && git(root, ["rev-parse", "HEAD"]) !== before) {
      if (planningWorktree && planningBranchBefore) output("git", ["reset", "--hard", planningBranchBefore], planningWorktree, true);
      output("git", ["reset", "--mixed", before], root, true);
    }
    throw new Error(`${error.message}; transaction_start=${before}`);
  } finally {
    release();
  }
}

export function sparsePatterns() {
  return ["/*", "!/backlog.yaml", "!/project.yaml", "!/tasks/", "!/wiki/", "!/lap30/", "!/viewer/index.html"];
}

export function createSparseWorktree(root, branch, worktree) {
  git(root, ["worktree", "add", "-b", branch, worktree, "main"]);
  try {
    git(worktree, ["sparse-checkout", "init", "--no-cone"]);
    const sparseFile = git(worktree, ["rev-parse", "--git-path", "info/sparse-checkout"]);
    fs.writeFileSync(sparseFile, `${sparsePatterns().join("\n")}\n`, "utf8");
    git(worktree, ["read-tree", "-mu", "HEAD"]);
    for (const forbidden of ["backlog.yaml", "project.yaml", "tasks", "wiki", "lap30", "viewer/index.html"]) {
      if (fs.existsSync(path.join(worktree, forbidden))) throw new Error(`Sparse worktree contains main-managed path: ${forbidden}`);
    }
  } catch (error) {
    output("git", ["worktree", "remove", "--force", worktree], root, true);
    output("git", ["branch", "-D", branch], root, true);
    throw error;
  }
}

export function taskStart(args, root, allocate = createSparseWorktree) {
  assertMain(root); assertTaskId(args.id); assertSlug(args.slug);
  if (!args.title) throw new Error("--title is required");
  if (git(root, ["status", "--porcelain"])) throw new Error("task-start requires a clean main worktree");
  git(root, ["fetch", "origin", "main"]);
  if (git(root, ["rev-parse", "HEAD"]) !== git(root, ["rev-parse", "origin/main"])) throw new Error("task-start requires current main");
  const startHead = git(root, ["rev-parse", "HEAD"]);
  const backlogFile = path.join(root, "backlog.yaml");
  const originalBacklog = fs.readFileSync(backlogFile, "utf8");
  const backlog = readYaml(backlogFile);
  if ((backlog.tasks ?? []).some((task) => task.id === args.id)) throw new Error(`Task already exists: ${args.id}`);
  const epic = args.epic ?? backlog.epics?.[0]?.id;
  if (!backlog.epics?.some((candidate) => candidate.id === epic)) throw new Error(`Unknown epic: ${epic}`);
  const relativeTaskDir = `tasks/${args.id}-${args.slug}`;
  const relativeWorktree = `worktrees/${args.id}-${args.slug}`;
  const branch = `task/${args.id}-${args.slug}`;
  const taskDir = resolveInside(root, relativeTaskDir);
  const worktree = resolveInside(root, relativeWorktree);
  let allocated = false;
  const removeAllocation = () => {
    if (!allocated) return;
    output("git", ["worktree", "remove", "--force", worktree], root, true);
    output("git", ["branch", "-D", branch], root, true);
    allocated = false;
  };
  fs.mkdirSync(path.dirname(taskDir), { recursive: true });
  fs.mkdirSync(taskDir, { recursive: false });
  try {
    const values = { TASK_ID: args.id, TITLE: args.title, TITLE_YAML: JSON.stringify(args.title), DATE: dateInTimezone(readYaml(path.join(root, "project.yaml")).timezone) };
    for (const filename of REQUIRED_TASK_FILES) {
      const template = fs.readFileSync(path.join(REPO_ROOT, "templates/task", filename), "utf8");
      fs.writeFileSync(path.join(taskDir, filename), replaceTemplate(template, values));
    }
    backlog.tasks ??= [];
    backlog.tasks.push({
      id: args.id, title: args.title, type: args.type ?? "feature", change_class: "product", epic, status: "plan",
      priority: args.priority ?? "P2", estimate_points: 1, task_dir: relativeTaskDir, depends_on: [], branch, worktree: relativeWorktree,
      assignees: {
        main: args.main ?? "main-agent-sol-high", planner: args.planner ?? "planner-agent-terra-medium",
        dev: args.dev ?? "dev-agent-sol-high", reviewer: args.reviewer ?? "reviewer-agent-terra-medium", qa: args.qa ?? "qa-agent-terra-medium",
      },
    });
    writeYaml(backlogFile, backlog);
    allocate(root, branch, worktree);
    allocated = true;
    // task-start is an allocation transaction only.  The first main commit is
    // planning-gate, so the scaffold remains dirty for the planner to finish.
    git(worktree, ["reset", "--hard", "main"]);
    return { commit: null, pushed: false, changed: changedFiles(root), branch, worktree };
  } catch (error) {
    const currentHead = git(root, ["rev-parse", "HEAD"]);
    if (currentHead !== startHead) {
      const remote = output("git", ["ls-remote", "origin", "refs/heads/main"], root, true);
      const remoteHead = remote.status === 0 ? remote.stdout.trim().split(/\s+/)[0] : null;
      if (remoteHead !== startHead) {
        throw new Error(`${error.message}; publish outcome is not the starting remote, so allocated resources and local evidence were retained for reconciliation`);
      }
      removeAllocation();
      git(root, ["reset", "--hard", startHead]);
      throw new Error(`${error.message}; remote remained unchanged and invocation-created resources were removed (branch, worktree, and local evidence)`);
    }
    removeAllocation();
    git(root, ["reset", "--mixed", startHead]);
    if (fs.existsSync(taskDir)) fs.rmSync(taskDir, { recursive: true, force: true });
    if (git(root, ["status", "--porcelain", "backlog.yaml"])) fs.writeFileSync(backlogFile, originalBacklog);
    throw new Error(`${error.message}; remote was not published and invocation-created resources were removed`);
  }
}

function diffPaths(root, base, head) { return lines(git(root, ["diff", "--name-only", `${base}...${head}`])); }

export function managedDigest(root, base, head) {
  const files = diffPaths(root, base, head).filter((file) => !isMainManagedPath(file)).sort();
  const records = files.map((file) => {
    const result = output("git", ["rev-parse", `${head}:${file}`], root, true);
    return `${file}\0${result.status === 0 ? result.stdout.trim() : "DELETED"}\n`;
  }).join("");
  return crypto.createHash("sha256").update(records).digest("hex");
}

function candidatePaths(root, task) {
  const candidateRoot = resolveInside(root, task.worktree, `${task.id} candidate worktree`);
  if (!fs.existsSync(candidateRoot)) throw new Error(`${task.id}: candidate worktree is missing`);
  if (git(candidateRoot, ["branch", "--show-current"]) !== task.branch) throw new Error(`${task.id}: candidate worktree branch mismatch`);
  return candidateRoot;
}

export function candidateCommit({ root, taskId, candidateRoot, message = `task: candidate ${taskId}` }) {
  assertMain(root);
  const { task } = taskContext(root, taskId);
  const worktree = candidateRoot ? path.resolve(candidateRoot) : candidatePaths(root, task);
  const release = acquireWorkRepoLock(root, { requireClean: false });
  try {
    if (git(worktree, ["branch", "--show-current"]) !== task.branch) throw new Error("candidate commit requires the Task branch");
    const base = git(root, ["merge-base", "main", task.branch]);
    const prior = Number(git(root, ["rev-list", "--count", `${base}..${task.branch}`]));
    if (prior !== 0) throw new Error("candidate branch must contain only the planning commit before DEV");
    const changed = changedFiles(worktree);
    if (!changed.length) throw new Error("candidate commit requires non-empty product changes");
    const forbidden = changed.filter(isMainManagedPath);
    if (forbidden.length) throw new Error(`candidate commit contains main-managed paths: ${forbidden.join(", ")}`);

    // Freeze the exact working bytes before the single DEV quality run.  The
    // pre-commit hook compares this digest with the staged bytes, so any
    // validation/build mutation causes the trusted fast path to fail closed.
    const digest = changedContentDigest(worktree, { cached: false });
    const check = spawnSync("make", ["check"], { cwd: worktree, encoding: "utf8" });
    const diagnostics = [check.stderr, check.stdout].filter(Boolean).join("\n").trim();
    if (check.status !== 0) throw new Error(diagnostics || "make check failed");
    const afterCheck = changedFiles(worktree);
    if (afterCheck.join("\0") !== changed.join("\0")) throw new Error("make check changed the candidate working set");
    if (changedContentDigest(worktree, { cached: false }) !== digest) throw new Error("make check changed candidate bytes");
    git(worktree, ["add", "--", ...changed]);
    let commitCreated = false;
    try {
      git(worktree, ["commit", "-m", message], { env: {
        ...process.env,
        WORK_REPO_LOCK_HELD: "1",
        WORK_PARENT_COMMIT: "1",
        WORK_ACTION: "candidate-commit",
        WORK_ALLOWED_PATHS: JSON.stringify(changed),
        WORK_VALIDATED_DIGEST: digest,
      } });
      commitCreated = true;
      const commit = git(worktree, ["rev-parse", "HEAD"]);
      if (Number(git(root, ["rev-list", "--count", `${base}..${commit}`])) !== 1) throw new Error("candidate must be one commit after planning");
      return { candidate_commit: commit, base, changed };
    } catch (error) {
      if (commitCreated) output("git", ["reset", "--mixed", base], worktree, true);
      throw error;
    }
  } finally {
    release();
  }
}

function completionAllowed(root, taskId) {
  const { taskDir } = taskContext(root, taskId);
  return ["backlog.yaml", ...PLANNING_FILES.filter((file) => file !== "PLAN.md").map((file) => `${taskDir}/${file}`), "wiki/"];
}

function snapshotBytes(root, files) {
  return new Map(files.map((file) => {
    const absolute = path.join(root, file);
    if (!fs.existsSync(absolute)) return [file, { exists: false }];
    const stat = fs.statSync(absolute);
    return [file, { exists: true, mode: stat.mode & 0o7777, bytes: fs.readFileSync(absolute) }];
  }));
}

function restoreBytes(root, snapshot) {
  for (const [file, state] of snapshot) {
    const absolute = path.join(root, file);
    if (!state.exists) {
      if (fs.existsSync(absolute)) fs.rmSync(absolute, { force: true });
      continue;
    }
    fs.mkdirSync(path.dirname(absolute), { recursive: true });
    fs.writeFileSync(absolute, state.bytes);
    fs.chmodSync(absolute, state.mode);
  }
}

function removeUnknownUntracked(root, snapshot, allowed) {
  for (const file of changedFiles(root)) {
    if (snapshot.has(file) || !matches(file, allowed)) continue;
    const tracked = output("git", ["ls-files", "--error-unmatch", "--", file], root, true).status === 0;
    if (!tracked) fs.rmSync(path.join(root, file), { force: true });
  }
}

export function completionGate({ root, taskId, message = `task: complete ${taskId}`, validate = true, push = false }) {
  assertMain(root);
  const { task, taskDir } = taskContext(root, taskId);
  const stagedBefore = lines(git(root, ["diff", "--cached", "--name-only", "--diff-filter=ACMRD"]));
  if (stagedBefore.length) throw new Error(`completion requires an unstaged quality evidence set; staged=${stagedBefore.join(", ")}`);
  const candidate = parseFrontmatter(path.join(root, taskDir, "HANDOVER.md")).candidate_commit;
  if (!/^[0-9a-f]{40}$/.test(candidate ?? "")) throw new Error("completion requires a main-side HANDOVER candidate_commit");
  if (git(root, ["branch", "--show-current"]) !== "main") throw new Error("completion requires main");
  if (output("git", ["cat-file", "-e", `${candidate}^{commit}`], root, true).status !== 0) throw new Error("candidate commit does not exist");
  if (git(root, ["rev-parse", task.branch]) !== candidate) throw new Error("HANDOVER candidate is not the Task branch head");
  const base = git(root, ["merge-base", "main", candidate]);
  if (Number(git(root, ["rev-list", "--count", `${base}..${candidate}`])) !== 1) throw new Error("candidate must be one commit after merge-base");
  const product = diffPaths(root, base, candidate);
  const forbidden = product.filter(isMainManagedPath);
  if (forbidden.length) throw new Error(`candidate is not product-only: ${forbidden.join(", ")}`);
  const candidateEvidence = output("git", ["cat-file", "-e", `${candidate}:${taskDir}/HANDOVER.md`], root, true);
  if (candidateEvidence.status === 0 && diffPaths(root, base, candidate).includes(`${taskDir}/HANDOVER.md`)) throw new Error("candidate-side HANDOVER is forbidden");
  const review = parseFrontmatter(path.join(root, taskDir, "REVIEW_RESULT.md"));
  const qa = parseFrontmatter(path.join(root, taskDir, "QA_RESULT.md"));
  const qaPlan = parseFrontmatter(path.join(root, taskDir, "QA_PLAN.md"));
  if (review.decision !== "pass" || review.reviewer_agent !== task.assignees?.reviewer) throw new Error("completion requires independent REVIEW PASS with reviewer identity");
  const reviewText = fs.readFileSync(path.join(root, taskDir, "REVIEW_RESULT.md"), "utf8");
  if (!/candidate/i.test(reviewText) || !/(?:DEV|make\s+check)/i.test(reviewText)) throw new Error("completion requires REVIEW candidate diff and DEV check audit");
  if (!new Set(["pass", "accepted_with_bugs"]).has(qa.decision) || qa.qa_agent !== task.assignees?.qa) throw new Error("completion requires independent QA PASS");
  if (qaPlan.status !== "approved") throw new Error("completion requires the approved QA PLAN");
  const allowed = completionAllowed(root, taskId);
  const preHead = git(root, ["rev-parse", "HEAD"]);
  const beforeChanged = changedFiles(root);
  const preForbidden = beforeChanged.filter((file) => !matches(file, allowed));
  if (preForbidden.length) throw new Error(`completion evidence scope violation: ${preForbidden.join(", ")}`);
  const snapshot = snapshotBytes(root, beforeChanged);
  const indexSnapshot = output("git", ["diff", "--cached", "--binary"], root, true).stdout;
  const release = acquireWorkRepoLock(root, { requireClean: false });
  let result;
  try {
    git(root, ["merge", "--no-ff", "--no-commit", candidate]);
    if (git(root, ["rev-parse", "MERGE_HEAD"]) !== candidate) throw new Error("completion merge second parent mismatch");
    const afterChanged = changedFiles(root);
    const invalid = afterChanged.filter((file) => !matches(file, allowed) && !product.includes(file));
    if (invalid.length) throw new Error(`completion scope violation: ${invalid.join(", ")}`);
    // Snapshot before validation and before staging; the hook binds the exact
    // post-validation bytes to this transaction's trusted parent.
    const digest = changedContentDigest(root, { cached: false });
    if (validate) validateOperations(root, "handover", taskId, root);
    const unstagedAfterValidation = lines(git(root, ["diff", "--name-only", "--diff-filter=ACMRD"]));
    if (unstagedAfterValidation.length) git(root, ["add", "-A", "--", ...unstagedAfterValidation]);
    const commitAllowed = [...new Set([...allowed, ...product])];
    git(root, ["commit", "-m", message], { env: {
      ...process.env,
      WORK_REPO_LOCK_HELD: "1",
      WORK_PARENT_COMMIT: "1",
      WORK_ACTION: "completion-gate",
      WORK_ALLOWED_PATHS: JSON.stringify(commitAllowed),
      WORK_VALIDATED_DIGEST: digest,
    } });
    const merge = git(root, ["rev-parse", "HEAD"]);
    result = { commit: merge, candidate_commit: candidate, changed: afterChanged, pushed: false };
  } catch (error) {
    restoreMerge(root);
    // --abort normally restores the pre-merge index and bytes.  Re-apply the
    // explicit snapshot as a final guard for conflicts or hook failures.
    restoreBytes(root, snapshot);
    removeUnknownUntracked(root, snapshot, allowed);
    output("git", ["reset", "--mixed", "HEAD"], root, true);
    if (indexSnapshot) {
      const patchFile = path.join(root, `.completion-index-${process.pid}.patch`);
      fs.writeFileSync(patchFile, indexSnapshot);
      try { output("git", ["apply", "--cached", patchFile], root, true); } finally { fs.rmSync(patchFile, { force: true }); }
    }
    throw new Error(`${error.message}; completion transaction aborted`);
  } finally {
    release();
  }
  if (!push) return result;
  const pushResult = output("git", ["push", "origin", "HEAD:main"], root, true);
  if (pushResult.status === 0) return { ...result, pushed: true, retries: 0 };
  const diagnostics = [pushResult.stderr, pushResult.stdout].filter(Boolean).join("\n").trim();
  const remote = output("git", ["ls-remote", "origin", "refs/heads/main"], root, true);
  const remoteHead = remote.status === 0 ? (lines(remote.stdout)[0]?.split(/\s+/)[0] ?? null) : null;
  if (remoteHead === result.commit) return { ...result, pushed: true, retries: 0, reconciled: true };
  if (remoteHead !== preHead) {
    throw new Error(`Completion merge push failed; reconciliation required (remote main=${remoteHead ?? "unknown"}, local merge=${result.commit})${diagnostics ? `: ${diagnostics}` : ""}`);
  }
  const rollbackRelease = acquireWorkRepoLock(root, { requireClean: false });
  try {
    if (git(root, ["rev-parse", "HEAD"]) !== result.commit || git(root, ["status", "--porcelain"])) {
      throw new Error(`Completion merge push failed; reconciliation required before rollback (local merge=${result.commit})`);
    }
    output("git", ["reset", "--mixed", preHead], root, true);
    for (const file of product) {
      const absolute = path.join(root, file);
      if (output("git", ["cat-file", "-e", `${preHead}:${file}`], root, true).status === 0) {
        output("git", ["restore", "--source", preHead, "--worktree", "--", file], root, true);
      } else if (fs.existsSync(absolute)) {
        fs.rmSync(absolute, { force: true, recursive: true });
      }
    }
    restoreBytes(root, snapshot);
    for (const file of changedFiles(root)) {
      if (snapshot.has(file)) continue;
      const tracked = output("git", ["ls-files", "--error-unmatch", "--", file], root, true).status === 0;
      if (!tracked) fs.rmSync(path.join(root, file), { force: true });
    }
    if (indexSnapshot) {
      const patchFile = path.join(root, `.completion-index-${process.pid}.patch`);
      fs.writeFileSync(patchFile, indexSnapshot);
      try { output("git", ["apply", "--cached", patchFile], root, true); } finally { fs.rmSync(patchFile, { force: true }); }
    }
  } finally {
    rollbackRelease();
  }
  throw new Error(`Completion merge push failed; remote main unchanged at ${preHead}; local merge commit rolled back and pre-merge evidence restored; completion can be retried${diagnostics ? `: ${diagnostics}` : ""}`);
}

function assertBootstrapBinding(root, evidence, refs) {
  const commit = evidence.bootstrap_evidence_commit ?? "";
  const digest = evidence.bootstrap_evidence_digest ?? "";
  if (!/^[0-9a-f]{40}$/.test(commit) || !/^[0-9a-f]{64}$/.test(digest)) {
    throw new Error("Evidence is missing the bootstrap evidence binding");
  }
  for (const ref of refs) git(root, ["merge-base", "--is-ancestor", commit, ref]);
  let manifest;
  try {
    manifest = JSON.parse(git(root, ["show", `${commit}:${BOOTSTRAP_MANIFEST_PATH}`]));
  } catch (error) {
    throw new Error(`Bootstrap binding does not resolve to its manifest: ${error.message}`);
  }
  const { manifest_sha256: recordedDigest, ...manifestBody } = manifest;
  const computedDigest = crypto.createHash("sha256").update(`${JSON.stringify(manifestBody)}\n`).digest("hex");
  if (recordedDigest !== digest || computedDigest !== digest) throw new Error("Bootstrap binding digest differs from the bound manifest");
}

export function assertComposite(root, taskId) {
  const { task, taskDir } = taskContext(root, taskId);
  const review = parseFrontmatter(path.join(root, taskDir, "REVIEW_RESULT.md"));
  const qa = parseFrontmatter(path.join(root, taskDir, "QA_RESULT.md"));
  const handover = parseFrontmatter(path.join(root, taskDir, "HANDOVER.md"));
  const head = git(root, ["rev-parse", task.branch]);
  if (handover.candidate_commit !== head) throw new Error("HANDOVER candidate does not match the Task branch head");
  const base = git(root, ["merge-base", "main", head]);
  if (Number(git(root, ["rev-list", "--count", `${base}..${head}`])) !== 1) throw new Error("Task branch must contain exactly one candidate commit after planning");
  const candidateFiles = diffPaths(root, base, head);
  if (candidateFiles.includes(`${taskDir}/HANDOVER.md`)) throw new Error("candidate-side HANDOVER is forbidden");
  const forbidden = candidateFiles.filter(isMainManagedPath);
  if (forbidden.length) throw new Error(`PR contains main-managed paths: ${forbidden.join(", ")}`);
  if (review.decision !== "pass" || review.reviewer_agent !== task.assignees?.reviewer) throw new Error("Independent REVIEW PASS is required");
  if (!new Set(["pass", "accepted_with_bugs"]).has(qa.decision) || qa.qa_agent !== task.assignees?.qa) throw new Error("Independent QA PASS is required");
  return { task, head, base, candidate_commit: head, files: candidateFiles };
}

function taskPr(args, root) {
  const candidate = assertComposite(root, args.task);
  if (args.dry_run === "true") return candidate;
  git(root, ["push", "-u", "origin", candidate.task.branch]);
  const url = run("gh", ["pr", "create", "--repo", args.repo ?? "autotaker/kakesu", "--base", "main", "--head", candidate.task.branch, "--title", `${args.task}: ${candidate.task.title}`, "--body", `Composite candidate ${candidate.head}`], { cwd: root });
  run("gh", ["pr", "merge", url, "--auto", "--merge"], { cwd: root });
  return { ...candidate, url };
}

export function scopeCheck(args, root) {
  const before = /^0+$/.test(args.base ?? "") ? `${args.head}^` : args.base;
  const files = args.event === "pr"
    ? diffPaths(root, before, args.head)
    : lines(git(root, ["diff", "--name-only", before, args.head]));
  if (args.event === "pr") {
    const forbidden = files.filter(isMainManagedPath);
    if (forbidden.length) throw new Error(`PR scope contains main-managed paths: ${forbidden.join(", ")}`);
  } else if (args.event === "main") {
    const parents = git(root, ["rev-list", "--parents", "-n", "1", args.head]).split(" ");
    if (args.allow_merge === "true" && parents.length === 3) {
      const firstParent = parents[1];
      const candidateCommit = parents[2];
      const backlog = YAML.parse(git(root, ["show", `${firstParent}:backlog.yaml`]));
      const observedCandidates = [];
      const boundTasks = (backlog.tasks ?? []).filter((task) => {
        try {
          const content = git(root, ["show", `${firstParent}:${task.task_dir}/HANDOVER.md`]);
          const frontmatter = content.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/);
          const boundCandidate = frontmatter ? YAML.parse(frontmatter[1]).candidate_commit : null;
          if (boundCandidate) observedCandidates.push(`${task.id}:${boundCandidate}`);
          return boundCandidate === candidateCommit;
        } catch {
          return false;
        }
      });
      if (boundTasks.length !== 1) throw new Error(`Merge commit is not bound to exactly one Task HANDOVER candidate: merge_candidate=${candidateCommit}, observed=${observedCandidates.join(",") || "none"}`);
      const task = boundTasks[0];
      const candidateBase = git(root, ["merge-base", firstParent, candidateCommit]);
      if (Number(git(root, ["rev-list", "--count", `${candidateBase}..${candidateCommit}`])) !== 1) throw new Error("Merge candidate must be one commit after planning");
      const candidateFiles = diffPaths(root, candidateBase, candidateCommit);
      if (candidateFiles.includes(`${task.task_dir}/HANDOVER.md`)) throw new Error("Merge candidate contains candidate-side HANDOVER");
      const forbidden = candidateFiles.filter(isMainManagedPath);
      if (forbidden.length) throw new Error(`Merge candidate contains main-managed paths: ${forbidden.join(", ")}`);
      const allowedEvidence = ["backlog.yaml", ...PLANNING_FILES.filter((file) => file !== "PLAN.md").map((file) => `${task.task_dir}/${file}`), "wiki/"];
      const mergeEvidenceChanges = diffPaths(root, firstParent, args.head).filter((file) => isMainManagedPath(file) && !matches(file, allowedEvidence));
      if (mergeEvidenceChanges.length) throw new Error(`Merge commit changes unapproved main-managed paths: ${mergeEvidenceChanges.join(", ")}`);
      return { files, merge_commit: true, task: task.id, candidate_commit: candidateCommit };
    }
    const forbidden = files.filter((file) => !isMainManagedPath(file));
    if (forbidden.length) throw new Error(`Direct main push contains product paths: ${forbidden.join(", ")}`);
  } else throw new Error("--event must be pr or main");
  return { files };
}

function postMerge(args, root) {
  assertTaskId(args.task);
  const { backlog, task, taskDir } = taskContext(root, args.task);
  if (!/^[0-9a-f]{40}$/.test(args.merged_commit ?? "")) throw new Error("--merged-commit must be a full commit SHA");
  git(root, ["merge-base", "--is-ancestor", args.merged_commit, "main"]);
  const handover = parseFrontmatter(path.join(root, taskDir, "HANDOVER.md"));
  const parents = git(root, ["rev-list", "--parents", "-n", "1", args.merged_commit]).split(" ");
  if (parents.length !== 3 || parents[2] !== handover.candidate_commit) throw new Error("Merged commit is not the recorded candidate merge commit");
  if (task.merged_commit) {
    if (task.merged_commit !== args.merged_commit) throw new Error("Task already records a different merged commit");
    return { no_op: true, merged_commit: task.merged_commit };
  }
  task.merged_commit = args.merged_commit;
  task.merge_pr = Number(args.pr);
  task.status = "qa";
  writeYaml(path.join(root, "backlog.yaml"), backlog);
  return evidenceCommit({ root, action: "main-transition", taskId: args.task, message: `task: record merge ${args.task} (#${args.pr})`, push: args.push !== "false" });
}

export function syncMain(args, root) {
  assertMain(root);
  if (git(root, ["status", "--porcelain"])) throw new Error("sync requires a clean main worktree");
  git(root, ["fetch", "--prune", "origin"]);
  git(root, ["merge", "--ff-only", "origin/main"]);
  const ci = output("gh", ["run", "list", "--repo", args.repo ?? "autotaker/kakesu", "--branch", "main", "--limit", "1", "--json", "conclusion", "--jq", ".[0].conclusion"], root, true);
  if (ci.status !== 0 || !["success", "neutral", "skipped"].includes(ci.stdout.trim())) throw new Error("sync stops while main CI is unavailable or red");
  if (args.fast === "1") return { fast: true };
  let backlog = readYaml(path.join(root, "backlog.yaml"));
  // Completion-gate records a final done state in the same no-ff merge. This
  // recovery path only handles legacy qa transitions; Wiki is optional and
  // must not be implicitly invoked during cleanup.
  const newlyDone = (backlog.tasks ?? []).filter((task) => task.merged_commit && task.status === "qa");
  for (const task of newlyDone) {
    if (task.worktree && fs.existsSync(resolveInside(root, task.worktree)) && git(resolveInside(root, task.worktree), ["status", "--porcelain"])) {
      throw new Error(`${task.id}: dirty worktree blocks cleanup`);
    }
    task.status = "done";
  }
  if (newlyDone.length) {
    writeYaml(path.join(root, "backlog.yaml"), backlog);
    evidenceCommit({ root, action: "sync", taskId: newlyDone[0].id, message: `task: complete ${newlyDone.map((task) => task.id).join(",")}`, push: args.push !== "false" });
  }
  backlog = readYaml(path.join(root, "backlog.yaml"));
  const cleanup = (backlog.tasks ?? []).filter((task) => task.status === "done" && task.branch && task.worktree);
  for (const task of cleanup) {
    const worktree = resolveInside(root, task.worktree);
    if (fs.existsSync(worktree) && git(worktree, ["status", "--porcelain"])) throw new Error(`${task.id}: dirty worktree blocks cleanup`);
    git(root, ["merge-base", "--is-ancestor", task.branch, "main"]);
    if (fs.existsSync(worktree)) git(root, ["worktree", "remove", worktree]);
    git(root, ["branch", "-d", task.branch]);
    delete task.branch; delete task.worktree;
  }
  if (!cleanup.length) return { fast: false, no_op: !newlyDone.length };
  writeYaml(path.join(root, "backlog.yaml"), backlog);
  return evidenceCommit({ root, action: "sync", taskId: cleanup[0].id, message: `task: clean ${cleanup.map((task) => task.id).join(",")}`, push: args.push !== "false" });
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === new URL(import.meta.url).pathname;
if (isMain) {
  const args = parseArgs(process.argv.slice(2));
  const root = path.resolve(args.main_root ?? (args.action === "task-pr" ? REPO_ROOT : findMainWorktree(REPO_ROOT)));
  let result;
  if (args.action === "evidence-commit") result = evidenceCommit({ root, action: args.evidence_action, taskId: args.task, message: args.message ?? `task: ${args.evidence_action} ${args.task}`, push: args.push !== "false" });
  else if (args.action === "planning-gate") result = evidenceCommit({ root, action: "planning-gate", taskId: args.task, message: args.message ?? `task: planning ${args.task}`, push: args.push !== "false" });
  else if (args.action === "candidate-commit") result = candidateCommit({ root, taskId: args.task, candidateRoot: args.candidate_root, message: args.message ?? `task: candidate ${args.task}` });
  else if (args.action === "completion-gate") result = completionGate({ root, taskId: args.task, message: args.message ?? `task: complete ${args.task}`, validate: args.validate !== "false", push: args.push !== "false" });
  else if (args.action === "task-start") result = taskStart(args, root);
  else if (args.action === "task-pr") result = taskPr(args, root);
  else if (args.action === "scope-check") result = scopeCheck(args, root);
  else if (args.action === "post-merge") result = postMerge(args, root);
  else if (args.action === "sync") result = syncMain(args, root);
  else if (args.action === "candidate") result = assertComposite(root, args.task);
  else throw new Error("Unknown --action");
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
}
