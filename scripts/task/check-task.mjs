import fs from "node:fs";
import path from "node:path";
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

export function checkTask(root, backlog, taskId) {
  const errors = [];
  try {
    assertTaskId(taskId);
    const task = taskById(backlog, taskId);
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
              const registered = records.split("\n\n").some((record) => record.includes(`worktree ${assignedWorktree}\n`) && record.includes(`branch refs/heads/${task.branch}`));
              if (!fs.existsSync(assignedWorktree) || !registered) throw new Error("missing registered worktree");
            } catch {
              errors.push(`${taskId}: assigned branch and worktree must exist in the product repository`);
            }
          }
        }
      }
    }

    if (["qa", "done"].includes(effectivePhase)) {
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
  const errors = taskIds.flatMap((taskId) => checkTask(root, backlog, taskId));
  if (errors.length) {
    process.stderr.write(`${errors.map((error) => `- ${error}`).join("\n")}\n`);
    process.exit(1);
  }
  process.stdout.write(`Validated ${taskIds.length} task(s).\n`);
}
