import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import YAML from "yaml";
import {
  REPO_ROOT, REQUIRED_TASK_FILES, acquireWorkRepoLock, assertSlug, assertTaskId, dateInTimezone,
  findMainWorktree, git, isMainManagedPath, parseArgs, parseFrontmatter, readYaml, replaceTemplate,
  resolveInside, taskById, writeYaml,
} from "./lib.mjs";

const ACTION_FILES = {
  task: ["TASK.md"], plan: ["PLAN.md"], "qa-plan": ["QA_PLAN.md"], review: ["REVIEW_RESULT.md"],
  "qa-result": ["QA_RESULT.md"], handover: ["HANDOVER.md"],
};

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
  if (action in ACTION_FILES) return ACTION_FILES[action].map((file) => `${taskDir}/${file}`);
  if (action === "task-start") return ["backlog.yaml", `${taskDir}/`];
  if (action === "main-transition") return ["backlog.yaml", `${taskDir}/TASK.md`, `${taskDir}/HANDOVER.md`];
  if (action === "sync") return ["backlog.yaml", "wiki/", "viewer/index.html", "tasks/"];
  throw new Error(`Unknown evidence action: ${action}`);
}

function matches(file, rules) { return rules.some((rule) => rule.endsWith("/") || rule.endsWith("-") ? file.startsWith(rule) : file === rule); }

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
  const candidateTree = git(candidateRoot, ["rev-parse", "HEAD^{tree}"]);
  if (git(candidateRoot, ["branch", "--show-current"]) !== task.branch) throw new Error("validator must run from the recorded candidate branch");
  if (git(root, ["rev-parse", task.branch]) !== candidateCommit) throw new Error("validator candidate differs from the recorded branch head");
  if (git(candidateRoot, ["status", "--porcelain"])) throw new Error("validator candidate worktree must be clean");
  const base = git(root, ["merge-base", "main", candidateCommit]);
  const digest = managedDigest(root, base, candidateCommit);
  if (handover.candidate_commit !== candidateCommit || handover.candidate_tree !== candidateTree || handover.managed_path_digest !== digest) {
    throw new Error(`validator candidate is not bound by HANDOVER: commit=${handover.candidate_commit ?? "missing"}/${candidateCommit}, tree=${handover.candidate_tree ?? "missing"}/${candidateTree}, digest=${handover.managed_path_digest ?? "missing"}/${digest}`);
  }
  if (!/^[0-9a-f]{40}$/.test(handover.bootstrap_evidence_commit ?? "") || !/^[0-9a-f]{64}$/.test(handover.bootstrap_evidence_digest ?? "")) {
    throw new Error("HANDOVER is missing the bootstrap evidence binding");
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
  try {
    const changed = changedFiles(root);
    if (!changed.length) return { commit: null, pushed: false, changed: [] };
    const forbidden = changed.filter((file) => !matches(file, rules));
    if (forbidden.length) throw new Error(`Evidence scope violation for ${action}: ${forbidden.join(", ")}`);
    // Bootstrap runs before this product change is merged, so its verifier and
    // schemas must come from the approved Task worktree, not the old main tree.
    if (validate) validateEvidenceAction(root, action, taskId, candidateRoot);
    git(root, ["add", "--", ...changed]);
    git(root, ["commit", "-m", message], { env: {
      ...process.env, WORK_REPO_LOCK_HELD: "1", WORK_PARENT_COMMIT: "1", WORK_ACTION: action,
      WORK_ALLOWED_PATHS: JSON.stringify(rules),
    } });
    let commit = git(root, ["rev-parse", "HEAD"]);
    if (!push) return { commit, pushed: false, changed };
    for (let retry = 0; retry <= 2; retry += 1) {
      const pushed = output("git", ["push", "origin", "HEAD:main"], root, true);
      if (pushed.status === 0) return { commit, pushed: true, changed, retries: retry };
      if (retry === 2) throw new Error(`Evidence push retry limit reached: ${(pushed.stderr || pushed.stdout).trim()}`);
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
    const published = evidenceCommit({ root, action: "task-start", taskId: args.id, message: `task: start ${args.id}`, push: args.push !== "false" });
    git(worktree, ["reset", "--hard", "main"]);
    return { ...published, branch, worktree };
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
  const tree = git(root, ["rev-parse", `${head}^{tree}`]);
  const base = git(root, ["merge-base", "main", head]);
  const digest = managedDigest(root, base, head);
  for (const [label, evidence] of [["review", review], ["qa", qa], ["handover", handover]]) {
    if (evidence.candidate_commit !== head || evidence.candidate_tree !== tree || evidence.managed_path_digest !== digest) {
      throw new Error(`${label} is not bound to the current candidate`);
    }
    if (!/^[0-9a-f]{40}$/.test(evidence.bootstrap_evidence_commit ?? "") || !/^[0-9a-f]{64}$/.test(evidence.bootstrap_evidence_digest ?? "")) {
      throw new Error(`${label} is missing the bootstrap evidence binding`);
    }
  }
  if (review.decision !== "pass" || review.make_check !== "pass") throw new Error("Independent REVIEW PASS is required");
  if (!new Set(["pass", "accepted_with_bugs"]).has(qa.decision)) throw new Error("Independent QA PASS is required");
  const bindings = [review, qa, handover].map((evidence) => `${evidence.bootstrap_evidence_commit}:${evidence.bootstrap_evidence_digest}`);
  if (new Set(bindings).size !== 1) throw new Error("Composite bootstrap bindings differ");
  const forbidden = diffPaths(root, "main", head).filter(isMainManagedPath);
  if (forbidden.length) throw new Error(`PR contains main-managed paths: ${forbidden.join(", ")}`);
  const [bootstrapCommit, bootstrapDigest] = bindings[0].split(":");
  assertBootstrapBinding(root, handover, [head, "main"]);
  return { task, head, tree, digest, bootstrap_commit: bootstrapCommit, bootstrap_digest: bootstrapDigest };
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
      const matches = (backlog.tasks ?? []).filter((task) => {
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
      if (matches.length !== 1) throw new Error(`Merge commit is not bound to exactly one Task HANDOVER candidate: merge_candidate=${candidateCommit}, observed=${observedCandidates.join(",") || "none"}`);
      const task = matches[0];
      const content = git(root, ["show", `${firstParent}:${task.task_dir}/HANDOVER.md`]);
      const handover = YAML.parse(content.match(/^---\r?\n([\s\S]*?)\r?\n---(?:\r?\n|$)/)[1]);
      if (git(root, ["rev-parse", `${candidateCommit}^{tree}`]) !== handover.candidate_tree) throw new Error("Merge candidate tree differs from HANDOVER");
      if (managedDigest(root, firstParent, args.head) !== handover.managed_path_digest) throw new Error("Merge managed-path digest differs from HANDOVER");
      const forbidden = diffPaths(root, firstParent, candidateCommit).filter(isMainManagedPath);
      if (forbidden.length) throw new Error(`Merge candidate contains main-managed paths: ${forbidden.join(", ")}`);
      const mergeEvidenceChanges = diffPaths(root, firstParent, args.head).filter(isMainManagedPath);
      if (mergeEvidenceChanges.length) throw new Error(`Merge commit changes main-managed paths: ${mergeEvidenceChanges.join(", ")}`);
      assertBootstrapBinding(root, handover, [firstParent, candidateCommit]);
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
  if (managedDigest(root, parents[1], args.merged_commit) !== handover.managed_path_digest) throw new Error("Merged managed-path digest differs from the candidate");
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
  const newlyDone = (backlog.tasks ?? []).filter((task) => task.merged_commit && task.status === "qa");
  for (const task of newlyDone) {
    const receipt = path.join(root, "wiki/ingestions", `${task.id}.json`);
    if (!fs.existsSync(receipt)) {
      run(process.execPath, [path.join(REPO_ROOT, "scripts/task/run-wiki-agent.mjs"), "--work-root", root, "--task", task.id, "--action", "ingest", "--commit", "false"], { cwd: root });
    }
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
  else if (args.action === "task-start") result = taskStart(args, root);
  else if (args.action === "task-pr") result = taskPr(args, root);
  else if (args.action === "scope-check") result = scopeCheck(args, root);
  else if (args.action === "post-merge") result = postMerge(args, root);
  else if (args.action === "sync") result = syncMain(args, root);
  else if (args.action === "candidate") result = assertComposite(root, args.task);
  else throw new Error("Unknown --action");
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
}
