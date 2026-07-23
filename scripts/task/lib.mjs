import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import YAML from "yaml";

export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
export const REQUIRED_TASK_FILES = [
  "TASK.md",
  "PLAN.md",
  "REVIEW_RESULT.md",
  "QA_PLAN.md",
  "QA_RESULT.md",
  "HANDOVER.md",
];
export const TASK_STATUSES = new Set(["backlog", "plan", "dev", "qa", "blocked", "done", "cancelled"]);
export const POINT_SCALE = [1, 2, 3, 5, 8, 13];
export const MAIN_MANAGED_PATHS = [
  "backlog.yaml",
  "project.yaml",
  "tasks/",
  "wiki/",
  "lap30/",
  "viewer/index.html",
];

export function isMainManagedPath(file) {
  return MAIN_MANAGED_PATHS.some((entry) => entry.endsWith("/") ? file.startsWith(entry) : file === entry);
}

export function parseArgs(argv) {
  const args = {};
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (!argument.startsWith("--")) {
      throw new Error(`Unexpected argument: ${argument}`);
    }
    const key = argument.slice(2).replaceAll("-", "_");
    const value = argv[index + 1];
    if (!value || value.startsWith("--")) {
      throw new Error(`Missing value for ${argument}`);
    }
    args[key] = value;
    index += 1;
  }
  return args;
}

export function workRoot(explicitRoot) {
  return path.resolve(explicitRoot || REPO_ROOT);
}

export function assertTaskId(taskId) {
  if (!/^TASK-\d{4}$/.test(taskId ?? "")) {
    throw new Error(`Invalid task ID: ${taskId ?? "<missing>"}`);
  }
}

export function assertSlug(slug) {
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug ?? "")) {
    throw new Error(`Invalid slug: ${slug ?? "<missing>"}`);
  }
}

export function readYaml(file) {
  return YAML.parse(fs.readFileSync(file, "utf8"));
}

export function writeYaml(file, value) {
  writeFileAtomic(file, YAML.stringify(value, { lineWidth: 120 }));
}

export function writeFileAtomic(file, content) {
  const temporary = `${file}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, content, "utf8");
  fs.renameSync(temporary, file);
}

export function resolveInside(root, relativePath, label = "path") {
  if (!relativePath || path.isAbsolute(relativePath)) {
    throw new Error(`${label} must be a non-empty relative path`);
  }
  const resolvedRoot = path.resolve(root);
  const resolved = path.resolve(resolvedRoot, relativePath);
  if (resolved !== resolvedRoot && !resolved.startsWith(`${resolvedRoot}${path.sep}`)) {
    throw new Error(`${label} escapes the repository root: ${relativePath}`);
  }
  return resolved;
}

export function git(root, args, options = {}) {
  const result = spawnSync("git", args, {
    cwd: root,
    encoding: "utf8",
    ...options,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error((result.stderr || result.stdout || `git ${args.join(" ")} failed`).trim());
  }
  return result.stdout?.trim() ?? "";
}

export function workRepoLockDir(root) {
  const commonDir = git(root, ["rev-parse", "--path-format=absolute", "--git-common-dir"]);
  return path.join(commonDir, "agent-harness-locks", "work-repository.lock");
}

export function acquireWorkRepoLock(root, { requireClean = true, requireMain = true } = {}) {
  const lockDir = workRepoLockDir(root);
  const lockRoot = path.dirname(lockDir);
  fs.mkdirSync(lockRoot, { recursive: true });
  const createLock = () => {
    try {
      fs.mkdirSync(lockDir);
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      const ownerFile = path.join(lockDir, "owner.json");
      let owner;
      try {
        owner = JSON.parse(fs.readFileSync(ownerFile, "utf8"));
      } catch {
        throw new Error(`Work repository lock has no readable owner; inspect and remove if stale: ${lockDir}`);
      }
      try {
        process.kill(owner.pid, 0);
        throw new Error(`Another work repository writer (pid ${owner.pid}) holds ${lockDir}`);
      } catch (processError) {
        if (processError.code !== "ESRCH") throw processError;
        fs.rmSync(lockDir, { recursive: true, force: true });
        fs.mkdirSync(lockDir);
      }
    }
  };
  createLock();
  let released = false;
  const release = () => {
    if (released) return;
    fs.rmSync(lockDir, { recursive: true, force: true });
    released = true;
  };
  try {
    fs.writeFileSync(path.join(lockDir, "owner.json"), `${JSON.stringify({ pid: process.pid, started_at: new Date().toISOString() })}\n`);
    if (requireMain && git(root, ["branch", "--show-current"]) !== "main") {
      throw new Error("Work repository writes are only allowed on main");
    }
    if (requireClean && git(root, ["status", "--porcelain"])) {
      throw new Error("Work repository must be clean before a writer starts");
    }
    return release;
  } catch (error) {
    release();
    throw error;
  }
}

export function findMainWorktree(root = REPO_ROOT) {
  const records = git(root, ["worktree", "list", "--porcelain"]).split("\n\n");
  for (const record of records) {
    const lines = record.split("\n");
    if (!lines.includes("branch refs/heads/main")) continue;
    const location = lines.find((line) => line.startsWith("worktree "))?.slice("worktree ".length);
    if (location) return path.resolve(location);
  }
  throw new Error("The repository has no registered main worktree");
}

export function parseFrontmatter(file) {
  const content = fs.readFileSync(file, "utf8");
  const match = content.match(/^---\n([\s\S]*?)\n---\n/);
  if (!match) {
    throw new Error(`${file}: missing YAML frontmatter`);
  }
  return YAML.parse(match[1]);
}

export function taskById(backlog, taskId) {
  const task = backlog.tasks?.find((candidate) => candidate.id === taskId);
  if (!task) {
    throw new Error(`Task is not registered in backlog.yaml: ${taskId}`);
  }
  return task;
}

export function estimatePoints(fileCount, lineCount) {
  if (!Number.isInteger(fileCount) || fileCount < 0 || !Number.isInteger(lineCount) || lineCount < 0) {
    throw new Error("Implementation file and line estimates must be non-negative integers");
  }
  const raw = Math.max(1, Math.ceil(fileCount / 3), Math.ceil(lineCount / 200));
  const point = POINT_SCALE.find((candidate) => candidate >= raw);
  if (!point) {
    throw new Error(`Task exceeds the 13 point scale (raw score: ${raw}); split the task or record a main Agent exception`);
  }
  return point;
}

export function replaceTemplate(content, values) {
  return content.replace(/\{\{([A-Z_]+)\}\}/g, (_, key) => {
    if (!(key in values)) {
      throw new Error(`Missing template value: ${key}`);
    }
    return values[key];
  });
}

export function dateInTimezone(timezone) {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${values.year}-${values.month}-${values.day}`;
}

export function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
