import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import {
  REQUIRED_TASK_FILES,
  TASK_STATUSES,
  assertTaskId,
  estimatePoints,
  git,
  parseArgs,
  parseFrontmatter,
  readYaml,
  resolveInside,
  taskById,
  workRoot,
} from "./lib.mjs";
import { validateDevSelection } from "./agent-routing.mjs";

const CHANGE_CLASSES = new Set(["product", "safety_contract"]);
const SAFETY_CONTRACT_PATHS = [
  /^AGENTS\.md$/,
  /^\.agents\/skills\//,
  /^docs\/development\//,
  /^docs\/glossary\.yml$/,
  /^templates\/task\//,
];
const SAFETY_CONTRACT_GENERATED_PATHS = new Set(["docs/99-glossary-index.md"]);
const SAFETY_CONTRACT_V2_FIELDS = ["safety_contract_planned_paths", "safety_contract_generated_paths"];
const SAFETY_CONTRACT_EXCLUSION = /製品コード[^\n]*(?:test|テスト)[^\n]*runtime\/build設定[^\n]*Schema[^\n]*製品依存[^\n]*(?:生成製品入力\/成果物[^\n]*)?(?:外部観測可能な)?(?:製品)?挙動/;
const LEGACY_TASK_0024_EXCLUSION = /製品コード、製品test、runtime\/build設定、製品Schema、製品依存、製品挙動/;
const SAFETY_CHECK_KEYS = ["process_tests", "contract_scope", "docs_lint", "make_check"];

function isTimestamp(value) {
  return typeof value === "string" && value.trim() !== "" && !Number.isNaN(Date.parse(value));
}

function safetyCheckDigest(candidateTree, mergeTree, checks) {
  const normalized = [
    `candidate_tree=${candidateTree}`,
    `merge_tree=${mergeTree}`,
    ...SAFETY_CHECK_KEYS.map((key) => `${key}=${checks[key]}`),
  ].join("\n");
  return createHash("sha256").update(`${normalized}\n`).digest("hex");
}

function hasOwn(value, key) {
  return Object.prototype.hasOwnProperty.call(value, key);
}

function isRepositoryFilePath(value) {
  return typeof value === "string"
    && value.length > 0
    && !path.posix.isAbsolute(value)
    && path.posix.normalize(value) === value
    && !value.endsWith("/")
    && path.posix.extname(value) !== ""
    && !value.includes("\\")
    && !/[*?[\]{}]/.test(value)
    && !value.split("/").includes("..");
}

function validateSafetyContractPlan(plan, changeClass, taskId) {
  const errors = [];
  const hasVersion = hasOwn(plan, "safety_contract_version");
  const presentV2Fields = SAFETY_CONTRACT_V2_FIELDS.filter((field) => hasOwn(plan, field));
  if (changeClass !== "safety_contract") {
    if (hasVersion || presentV2Fields.length) {
      errors.push(`${taskId}: safety_contract v2 fields require change_class safety_contract`);
    }
    return { errors, version: undefined, plannedPaths: [], generatedPaths: [] };
  }
  if (!hasVersion) {
    if (presentV2Fields.length) {
      errors.push(`${taskId}: safety_contract v2 path fields require safety_contract_version: 2`);
    }
    return { errors, version: undefined, plannedPaths: [], generatedPaths: [] };
  }
  if (plan.safety_contract_version !== 2) {
    errors.push(`${taskId}: unsupported safety_contract_version ${String(plan.safety_contract_version)}`);
    return { errors, version: plan.safety_contract_version, plannedPaths: [], generatedPaths: [] };
  }

  const values = {};
  for (const field of SAFETY_CONTRACT_V2_FIELDS) {
    if (!hasOwn(plan, field) || !Array.isArray(plan[field])) {
      errors.push(`${taskId}: safety_contract v2 requires ${field} as an array`);
      values[field] = [];
    } else {
      values[field] = plan[field];
    }
  }
  const plannedPaths = values.safety_contract_planned_paths;
  const generatedPaths = values.safety_contract_generated_paths;
  for (const [field, paths, allowed] of [
    ["safety_contract_planned_paths", plannedPaths, (candidate) => SAFETY_CONTRACT_PATHS.some((pattern) => pattern.test(candidate))],
    ["safety_contract_generated_paths", generatedPaths, (candidate) => SAFETY_CONTRACT_GENERATED_PATHS.has(candidate)],
  ]) {
    for (const candidate of paths) {
      if (!isRepositoryFilePath(candidate)) {
        errors.push(`${taskId}: ${field} contains an invalid repository file path: ${String(candidate)}`);
      } else if (!allowed(candidate)) {
        errors.push(`${taskId}: ${field} contains an unapproved path: ${candidate}`);
      }
    }
  }
  const allPaths = [...plannedPaths, ...generatedPaths];
  const duplicate = allPaths.find((candidate, index) => allPaths.indexOf(candidate) !== index);
  if (duplicate !== undefined) {
    errors.push(`${taskId}: safety_contract v2 path declarations contain a duplicate: ${String(duplicate)}`);
  }
  return { errors, version: 2, plannedPaths, generatedPaths };
}

function checkSafetyContractDone({ root, taskDir, task, taskId, planContract }) {
  const errors = [];
  const plan = parseFrontmatter(path.join(taskDir, "PLAN.md"));
  const qaPlan = parseFrontmatter(path.join(taskDir, "QA_PLAN.md"));
  const handover = parseFrontmatter(path.join(taskDir, "HANDOVER.md"));
  const taskContract = fs.readFileSync(path.join(taskDir, "TASK.md"), "utf8");
  const explicitExclusion = SAFETY_CONTRACT_EXCLUSION.test(taskContract)
    || (taskId === "TASK-0024" && LEGACY_TASK_0024_EXCLUSION.test(taskContract));
  if (!explicitExclusion) {
    errors.push(`${taskId}: safety_contract requires an explicit product-artifact exclusion in TASK.md`);
  }
  if (plan.change_class !== "safety_contract" || qaPlan.change_class !== "safety_contract") {
    errors.push(`${taskId}: safety_contract change_class must match in PLAN.md and QA_PLAN.md`);
  }
  if (plan.planning_reviewed_by !== task.assignees?.reviewer
      || plan.planning_review_decision !== "pass"
      || !isTimestamp(plan.planning_reviewed_at)) {
    errors.push(`${taskId}: safety_contract requires the assigned Reviewer planning review PASS`);
  }
  if (plan.classification_approved_by !== task.assignees?.main
      || !String(plan.classification_approval_reason ?? "").trim()
      || !isTimestamp(plan.classification_approved_at)) {
    errors.push(`${taskId}: safety_contract classification requires approval by the assigned main Agent`);
  }
  if (qaPlan.qa_agent !== task.assignees?.qa || qaPlan.approved_by !== task.assignees?.main) {
    errors.push(`${taskId}: safety_contract requires a TASK-first QA PLAN approved by the assigned main Agent`);
  }
  const approvalTimes = [plan.planning_reviewed_at, plan.approved_at, qaPlan.approved_at, plan.classification_approved_at]
    .map((value) => Date.parse(value));
  if (approvalTimes.some(Number.isNaN)
      || approvalTimes[0] > approvalTimes[1]
      || approvalTimes[1] > approvalTimes[3]
      || approvalTimes[2] > approvalTimes[3]) {
    errors.push(`${taskId}: safety_contract approval timestamps are inconsistent`);
  }
  const safetyChecks = handover.safety_checks;
  const safetyCheckKeys = safetyChecks && !Array.isArray(safetyChecks) && typeof safetyChecks === "object"
    ? Object.keys(safetyChecks).sort()
    : [];
  if (safetyCheckKeys.length !== SAFETY_CHECK_KEYS.length
      || safetyCheckKeys.some((key, index) => key !== [...SAFETY_CHECK_KEYS].sort()[index])
      || SAFETY_CHECK_KEYS.some((key) => safetyChecks[key] !== "pass")
      || !isTimestamp(handover.safety_checked_at)
      || !/^[a-f0-9]{64}$/.test(handover.safety_check_digest ?? "")) {
    errors.push(`${taskId}: safety_contract requires the exact passed safety_checks, checked_at, and SHA-256 digest`);
  }
  if (!task.merged_commit) {
    errors.push(`${taskId}: safety_contract done requires merged_commit`);
    return errors;
  }
  try {
    const project = readYaml(path.join(root, "project.yaml"));
    const repository = path.resolve(root, project.repository_path);
    git(repository, ["cat-file", "-e", `${task.merged_commit}^{commit}`]);
    git(repository, ["merge-base", "--is-ancestor", task.merged_commit, project.default_branch]);
    const [merge, firstParent, secondParent, ...extraParents] = git(repository, ["rev-list", "--parents", "-n", "1", task.merged_commit]).split(" ");
    if (merge !== task.merged_commit || !firstParent || !secondParent || extraParents.length) {
      throw new Error("merged_commit is not an exact two-parent no-ff merge");
    }
    const candidateTree = git(repository, ["rev-parse", `${secondParent}^{tree}`]);
    const mergeTree = git(repository, ["rev-parse", `${task.merged_commit}^{tree}`]);
    if (handover.safety_candidate_tree !== candidateTree
        || handover.safety_merge_tree !== mergeTree
        || candidateTree !== mergeTree) {
      throw new Error("safety candidate and merge trees do not match recorded Git trees");
    }
    if (handover.safety_check_digest !== safetyCheckDigest(candidateTree, mergeTree, safetyChecks)) {
      throw new Error("safety_check_digest does not match the canonical safety evidence");
    }
    const changedEntries = git(repository, ["diff", "--name-status", "--find-renames", "--find-copies-harder", firstParent, secondParent])
      .split("\n").filter(Boolean).map((line) => line.split("\t"));
    if (changedEntries.length === 0
        || changedEntries.some(([status]) => /^R|^C/.test(status))
        || changedEntries.some(([status, ...changedPaths]) => !/^[AMD]$/.test(status) || changedPaths.length !== 1)) {
      throw new Error("safety_contract includes a product or unapproved path");
    }
    if (planContract.version === 2 && planContract.errors.length === 0) {
      const declaredPaths = new Set([...planContract.plannedPaths, ...planContract.generatedPaths]);
      const undeclared = changedEntries.find(([, changedPath]) => !declaredPaths.has(changedPath));
      if (undeclared) {
        throw new Error(`safety_contract candidate diff includes an undeclared path: ${undeclared[1]}`);
      }
      const changedStatuses = new Map(changedEntries.map(([status, changedPath]) => [changedPath, status]));
      const missingGenerated = planContract.generatedPaths.find((generatedPath) => !/[AM]/.test(changedStatuses.get(generatedPath) ?? ""));
      if (missingGenerated) {
        throw new Error(`declared generated path is missing from the candidate diff: ${missingGenerated}`);
      }
    } else if (changedEntries.some(([, changedPath]) => !SAFETY_CONTRACT_PATHS.some((pattern) => pattern.test(changedPath)))) {
      throw new Error("safety_contract includes a product or unapproved path");
    }
  } catch (error) {
    errors.push(`${taskId}: safety_contract Git evidence is invalid: ${error.message}`);
  }
  return errors;
}

export function checkTask(root, backlog, taskId, { phase = "full" } = {}) {
  const errors = [];
  try {
    assertTaskId(taskId);
    const task = taskById(backlog, taskId);
    const declaredChangeClass = task.change_class;
    const changeClass = declaredChangeClass === undefined ? "product" : declaredChangeClass;
    if (!CHANGE_CLASSES.has(changeClass)) {
      errors.push(`${taskId}: change_class must be product or safety_contract`);
    }
    const safetyContract = changeClass === "safety_contract";
    if (!new Set(["full", "preflight"]).has(phase)) {
      errors.push(`${taskId}: unsupported check phase ${phase}`);
    }
    if (!TASK_STATUSES.has(task.status)) {
      errors.push(`${taskId}: invalid status ${task.status}`);
    }
    if (task.status === "blocked" && !["plan", "dev", "qa"].includes(task.resume_status)) {
      errors.push(`${taskId}: blocked task requires resume_status plan, dev, or qa`);
    }
    if (task.status !== "blocked" && task.resume_status) {
      errors.push(`${taskId}: resume_status is only allowed while blocked`);
    }
    const effectivePhase = task.status === "blocked" ? task.resume_status : task.status;
    if (task.bootstrap_exception && taskId !== "TASK-0001") {
      errors.push(`${taskId}: bootstrap_exception is reserved for TASK-0001`);
    }
    const taskDir = resolveInside(root, task.task_dir, `${taskId} task_dir`);
    for (const filename of REQUIRED_TASK_FILES) {
      const file = path.join(taskDir, filename);
      if (!fs.existsSync(file)) {
        errors.push(`${taskId}: missing ${filename}`);
        continue;
      }
      const frontmatter = parseFrontmatter(file);
      if (frontmatter.task_id !== taskId) {
        errors.push(`${taskId}: ${filename} has mismatched task_id`);
      }
    }
    const planContract = validateSafetyContractPlan(
      parseFrontmatter(path.join(taskDir, "PLAN.md")),
      changeClass,
      taskId,
    );
    errors.push(...planContract.errors);
    if (phase === "preflight") return errors;

    if (["dev", "qa", "done"].includes(effectivePhase)) {
      const plan = parseFrontmatter(path.join(taskDir, "PLAN.md"));
      const qaPlan = parseFrontmatter(path.join(taskDir, "QA_PLAN.md"));
      if (plan.status !== "approved" || !plan.approved_by || !plan.approved_at) {
        errors.push(`${taskId}: DEV gate requires an approved PLAN.md`);
      }
      if (qaPlan.status !== "approved" || !qaPlan.approved_by || !qaPlan.approved_at) {
        errors.push(`${taskId}: DEV gate requires an approved QA_PLAN.md`);
      }
      try {
        const expected = estimatePoints(plan.planned_implementation_files, plan.planned_implementation_lines);
        if (plan.estimate_points !== expected || task.estimate_points !== expected) {
          errors.push(`${taskId}: estimate_points must be ${expected} from approved PLAN.md`);
        }
      } catch (error) {
        errors.push(`${taskId}: ${error.message}`);
      }
      if (!task.bootstrap_exception) {
        try {
          const selectedProfile = validateDevSelection(plan);
          const expectedAgentMarker = selectedProfile === "luna-xhigh" ? "luna-xhigh" : "sol-high";
          if (!String(task.assignees?.dev ?? "").includes(expectedAgentMarker)) {
            errors.push(`${taskId}: assignees.dev must match approved DEV profile ${selectedProfile}`);
          }
        } catch (error) {
          errors.push(`${taskId}: ${error.message}`);
        }
      }
      const assignees = task.assignees ?? {};
      for (const role of ["main", "planner", "dev", "reviewer", "qa"]) {
        if (!assignees[role]) errors.push(`${taskId}: DEV gate requires assignees.${role}`);
      }
      if (!task.bootstrap_exception) {
        if (plan.planner_agent !== assignees.planner || plan.approved_by !== assignees.main) {
          errors.push(`${taskId}: PLAN author and approver must match assigned Planner and main Agents`);
        }
        if (qaPlan.qa_agent !== assignees.qa || qaPlan.approved_by !== assignees.main) {
          errors.push(`${taskId}: QA PLAN author and approver must match assigned QA and main Agents`);
        }
        if (assignees.dev === assignees.reviewer) {
          errors.push(`${taskId}: DEV Agent and Reviewer Agent must be different`);
        }
        if (assignees.dev === assignees.qa) {
          errors.push(`${taskId}: DEV Agent and QA Agent must be different`);
        }
        if (["dev", "qa"].includes(effectivePhase)) {
          const validBranch = new RegExp(`^task/${taskId}-[a-z0-9]+(?:-[a-z0-9]+)*$`).test(task.branch ?? "");
          const validWorktree = new RegExp(`^worktrees/${taskId}-[a-z0-9]+(?:-[a-z0-9]+)*$`).test(task.worktree ?? "");
          if (!validBranch) errors.push(`${taskId}: DEV gate requires a task branch`);
          if (!validWorktree) errors.push(`${taskId}: DEV gate requires a task worktree`);
          if (validBranch && validWorktree) {
            try {
              const project = readYaml(path.join(root, "project.yaml"));
              const repository = path.resolve(root, project.repository_path);
              const assignedWorktree = resolveInside(root, task.worktree, `${taskId} worktree`);
              git(repository, ["show-ref", "--verify", `refs/heads/${task.branch}`]);
              const records = git(repository, ["worktree", "list", "--porcelain"]);
              const assignedRealpath = fs.realpathSync(assignedWorktree);
              const registered = records.split("\n\n").some((record) => {
                const worktreeLine = record.split("\n").find((line) => line.startsWith("worktree "));
                if (!worktreeLine || !record.includes(`branch refs/heads/${task.branch}`)) return false;
                try {
                  return fs.realpathSync(worktreeLine.slice("worktree ".length)) === assignedRealpath;
                } catch {
                  return false;
                }
              });
              if (!fs.existsSync(assignedWorktree) || !registered) throw new Error("missing registered worktree");
            } catch {
              errors.push(`${taskId}: assigned branch and worktree must exist in the product repository`);
            }
          }
        }
      }
    }

    if (["qa", "done"].includes(effectivePhase) && !safetyContract) {
      const review = parseFrontmatter(path.join(taskDir, "REVIEW_RESULT.md"));
      if (review.decision !== "pass" || review.make_check !== "pass" || !review.reviewed_commit) {
        errors.push(`${taskId}: QA gate requires review PASS, make check PASS, and reviewed_commit`);
      }
      if (review.reviewer_agent !== task.assignees?.reviewer) {
        errors.push(`${taskId}: REVIEW_RESULT reviewer_agent must match assignees.reviewer`);
      }
      const qaPlan = parseFrontmatter(path.join(taskDir, "QA_PLAN.md"));
      if (!qaPlan.implementation_reviewed_at || typeof qaPlan.expectation_changed !== "boolean") {
        errors.push(`${taskId}: QA gate requires the post-implementation QA PLAN review`);
      }
      if (qaPlan.expectation_changed && qaPlan.expectation_change_approved_by !== task.assignees?.main) {
        errors.push(`${taskId}: changed QA expectations require approval by the assigned main Agent`);
      }
      if (!task.merged_commit) errors.push(`${taskId}: QA gate requires merged_commit`);
      const project = readYaml(path.join(root, "project.yaml"));
      const repository = path.resolve(root, project.repository_path);
      for (const [label, commit] of [["reviewed_commit", review.reviewed_commit], ["merged_commit", task.merged_commit]]) {
        if (!commit) continue;
        try {
          git(repository, ["cat-file", "-e", `${commit}^{commit}`]);
        } catch {
          errors.push(`${taskId}: ${label} is not a product repository commit`);
        }
      }
      if (review.reviewed_commit && task.merged_commit) {
        try {
          git(repository, ["merge-base", "--is-ancestor", task.merged_commit, project.default_branch]);
          if (task.bootstrap_exception) {
            if (review.reviewed_commit !== task.merged_commit) throw new Error("bootstrap commit mismatch");
          } else {
            const [merge, firstParent, secondParent, ...extraParents] = git(repository, ["rev-list", "--parents", "-n", "1", task.merged_commit]).split(" ");
            if (merge !== task.merged_commit || !firstParent || secondParent !== review.reviewed_commit || extraParents.length) {
              throw new Error("not an exact two-parent no-ff merge of reviewed_commit");
            }
          }
        } catch {
          errors.push(`${taskId}: merged_commit must be on main and exactly merge the reviewed commit with --no-ff`);
        }
      }
    }

    if (task.status === "done") {
      if (safetyContract) {
        errors.push(...checkSafetyContractDone({ root, taskDir, task, taskId, planContract }));
      } else {
        const qa = parseFrontmatter(path.join(taskDir, "QA_RESULT.md"));
        const handover = parseFrontmatter(path.join(taskDir, "HANDOVER.md"));
        if (!new Set(["pass", "accepted_with_bugs"]).has(qa.decision)) {
          errors.push(`${taskId}: done requires QA pass or accepted_with_bugs`);
        }
        if (handover.status !== "complete" || !handover.completed_at) {
          errors.push(`${taskId}: done requires a complete HANDOVER`);
        }
        if (!fs.existsSync(path.join(root, "wiki", "ingestions", `${taskId}.json`))) {
          errors.push(`${taskId}: done requires a Wiki ingestion receipt`);
        }
        if (!qa.qa_agent || qa.qa_agent !== task.assignees?.qa || !qa.tested_commit || !qa.tested_at) {
          errors.push(`${taskId}: done requires QA agent identity, tested commit, and tested_at`);
        } else {
          const project = readYaml(path.join(root, "project.yaml"));
          const repository = path.resolve(root, project.repository_path);
          try {
            git(repository, ["cat-file", "-e", `${qa.tested_commit}^{commit}`]);
            git(repository, ["merge-base", "--is-ancestor", task.merged_commit, qa.tested_commit]);
            git(repository, ["merge-base", "--is-ancestor", qa.tested_commit, project.default_branch]);
          } catch {
            errors.push(`${taskId}: tested_commit must be on main at or after merged_commit`);
          }
        }
      }
    }
  } catch (error) {
    errors.push(error.message);
  }
  return errors;
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === new URL(import.meta.url).pathname;
if (isMain) {
  const args = parseArgs(process.argv.slice(2));
  const root = workRoot(args.work_root);
  const backlog = readYaml(path.join(root, "backlog.yaml"));
  const taskIds = args.task ? [args.task] : (backlog.tasks ?? []).map((task) => task.id);
  const errors = taskIds.flatMap((taskId) => checkTask(root, backlog, taskId, { phase: args.phase ?? "full" }));
  if (errors.length) {
    process.stderr.write(`${errors.map((error) => `- ${error}`).join("\n")}\n`);
    process.exit(1);
  }
  process.stdout.write(`Validated ${taskIds.length} task(s).\n`);
}
