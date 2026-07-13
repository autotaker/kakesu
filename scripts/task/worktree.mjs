import path from "node:path";
import { spawnSync } from "node:child_process";
import {
  acquireWorkRepoLock,
  assertTaskId,
  git,
  parseArgs,
  readYaml,
  resolveInside,
  taskById,
  workRoot,
  writeYaml,
} from "./lib.mjs";

const args = parseArgs(process.argv.slice(2));
assertTaskId(args.task);
if (!new Set(["create", "remove"]).has(args.action)) throw new Error("--action must be create or remove");
const root = workRoot(args.work_root);
const backlogFile = path.join(root, "backlog.yaml");
const backlog = readYaml(backlogFile);
const project = readYaml(path.join(root, "project.yaml"));
const task = taskById(backlog, args.task);
const product = path.resolve(root, project.repository_path);
const slug = task.task_dir.replace(new RegExp(`^tasks/${task.id}-`), "");
const branch = `task/${task.id}-${slug}`;
const relativeWorktree = path.posix.join(project.worktree_root || "worktrees", `${task.id}-${slug}`);
const absoluteWorktree = resolveInside(root, relativeWorktree, "worktree path");

if (args.dry_run === "true") {
  process.stdout.write(`${JSON.stringify({ action: args.action, task: task.id, branch, worktree: absoluteWorktree }, null, 2)}\n`);
} else {
  const release = acquireWorkRepoLock(root);
  let created = false;
  let removed = false;
  let removedBranch = false;
  let branchHead;
  try {
    if (args.action === "create") {
      if (task.status !== "plan") throw new Error(`${task.id}: worktree creation requires status plan`);
      if (task.branch || task.worktree) throw new Error(`${task.id}: branch or worktree is already assigned`);
      for (const role of ["main", "planner", "dev", "reviewer", "qa"]) {
        if (!task.assignees?.[role]) throw new Error(`${task.id}: missing assignees.${role}`);
      }
      if (task.assignees.dev === task.assignees.reviewer || task.assignees.dev === task.assignees.qa) {
        throw new Error(`${task.id}: DEV Agent must differ from Reviewer and QA Agents`);
      }
      git(product, ["worktree", "add", "-b", branch, absoluteWorktree, project.default_branch]);
      created = true;
      task.branch = branch;
      task.worktree = relativeWorktree;
      task.status = "dev";
      writeYaml(backlogFile, backlog);
      git(root, ["add", "backlog.yaml"]);
      git(root, ["commit", "-m", `task: allocate worktree ${task.id}`]);
      process.stdout.write(`${absoluteWorktree}\n`);
    } else {
      if (!new Set(["done", "cancelled"]).has(task.status)) {
        throw new Error(`${task.id}: worktree removal requires status done or cancelled`);
      }
      if (!task.branch || !task.worktree) throw new Error(`${task.id}: no branch and worktree are assigned`);
      const assigned = resolveInside(root, task.worktree, `${task.id} worktree`);
      branchHead = git(product, ["rev-parse", task.branch]);
      if (task.status === "done") {
        git(product, ["merge-base", "--is-ancestor", task.branch, project.default_branch]);
      }
      git(product, ["worktree", "remove", assigned]);
      removed = true;
      git(product, ["branch", task.status === "cancelled" ? "-D" : "-d", task.branch]);
      removedBranch = true;
      delete task.branch;
      delete task.worktree;
      writeYaml(backlogFile, backlog);
      git(root, ["add", "backlog.yaml"]);
      git(root, ["commit", "-m", `task: release worktree ${task.id}`]);
    }
  } catch (error) {
    if (git(root, ["status", "--porcelain", "backlog.yaml"])) {
      const restore = spawnSync("git", ["restore", "--staged", "--worktree", "backlog.yaml"], { cwd: root, encoding: "utf8" });
      if (restore.status !== 0) process.stderr.write(restore.stderr);
    }
    if (created) {
      spawnSync("git", ["worktree", "remove", "--force", absoluteWorktree], { cwd: product });
      spawnSync("git", ["branch", "-d", branch], { cwd: product });
    }
    if (removed) {
      if (removedBranch) spawnSync("git", ["branch", branch, branchHead], { cwd: product });
      spawnSync("git", ["worktree", "add", absoluteWorktree, branch], { cwd: product });
    }
    throw error;
  } finally {
    release();
  }
}
