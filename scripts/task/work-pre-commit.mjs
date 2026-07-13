import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { REPO_ROOT, git, parseArgs, parseFrontmatter, workRoot } from "./lib.mjs";

const args = parseArgs(process.argv.slice(2));
const root = workRoot(args.work_root);
const staged = git(root, ["diff", "--cached", "--name-only", "--diff-filter=ACMRD"]).split("\n").filter(Boolean);
const action = process.env.WIKI_ACTION;
const target = process.env.WIKI_TARGET;
const workAction = process.env.WORK_ACTION;
const workAllowed = process.env.WORK_ALLOWED_PATHS ? JSON.parse(process.env.WORK_ALLOWED_PATHS) : null;
const unstaged = git(root, ["diff", "--name-only"]).split("\n").filter(Boolean);
const untracked = git(root, ["ls-files", "--others", "--exclude-standard"]).split("\n").filter(Boolean);
if (unstaged.length || untracked.length) {
  throw new Error(`Commit requires a complete staging set; unstaged=${unstaged.join(",") || "none"}; untracked=${untracked.join(",") || "none"}`);
}
if (workAction) {
  if (!Array.isArray(workAllowed) || !workAllowed.length) throw new Error(`Missing allowed paths for ${workAction}`);
  const matchesAllowed = (file) => workAllowed.some((rule) => rule.endsWith("/**") ? file.startsWith(rule.slice(0, -2)) : file === rule);
  const forbidden = staged.filter((file) => !matchesAllowed(file));
  if (forbidden.length) throw new Error(`Work Agent staged files outside ${workAction}: ${forbidden.join(", ")}`);
}

if (action) {
  const allowed = action === "ingest"
    ? (file) => /^wiki\/(semantic|decisions|ingestions)\//.test(file) || file === "wiki/index.json"
    : (file) => file === target;
  const forbidden = staged.filter((file) => !allowed(file));
  if (forbidden.length) throw new Error(`Wiki Agent staged files outside ${action}: ${forbidden.join(", ")}`);
  if (action !== "ingest" && staged.includes(target)) {
    const old = spawnSync("git", ["show", `HEAD:${target}`], { cwd: root, encoding: "utf8" });
    if (old.status !== 0) throw new Error(`${target}: context action requires an existing evidence file`);
    const current = fs.readFileSync(path.join(root, target), "utf8");
    const heading = action === "context-task" ? "## 関連コンテキスト" : "## 関連Wikiと判断";
    const maskSection = (value) => {
      const start = value.indexOf(`${heading}\n`);
      if (start < 0) throw new Error(`${target}: missing ${heading}`);
      const bodyStart = start + heading.length + 1;
      const next = value.indexOf("\n## ", bodyStart);
      return `${value.slice(0, bodyStart)}\n<CONTEXT>\n${next < 0 ? "" : value.slice(next + 1)}`;
    };
    if (maskSection(old.stdout) !== maskSection(current)) {
      throw new Error(`${target}: ${action} may only change ${heading}`);
    }
  }
}

for (const file of staged.filter((candidate) => candidate.startsWith("wiki/decisions/") && candidate.endsWith(".md"))) {
  const absolute = path.join(root, file);
  const old = spawnSync("git", ["show", `HEAD:${file}`], { cwd: root, encoding: "utf8" });
  if (old.status !== 0) continue;
  const temporary = path.join(root, ".locks", `old-decision-${process.pid}.md`);
  fs.mkdirSync(path.dirname(temporary), { recursive: true });
  fs.writeFileSync(temporary, old.stdout, "utf8");
  try {
    const oldMetadata = parseFrontmatter(temporary);
    if (!fs.existsSync(absolute)) throw new Error(`${file}: existing Decision cannot be deleted`);
    const current = fs.readFileSync(absolute, "utf8");
    if (oldMetadata.status === "superseded") {
      if (old.stdout !== current) throw new Error(`${file}: superseded Decision is immutable`);
      continue;
    }
    const normalizeStatus = (value) => value.replace(/^(status:\s*).+$/m, "$1<STATUS>");
    if (normalizeStatus(old.stdout) !== normalizeStatus(current)) {
      throw new Error(`${file}: accepted Decision content is immutable; only status may change`);
    }
    if (parseFrontmatter(absolute).status !== "superseded") {
      throw new Error(`${file}: an accepted Decision may only transition to superseded`);
    }
  } finally {
    fs.rmSync(temporary, { force: true });
  }
}

const validation = spawnSync(
  process.execPath,
  [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root],
  { cwd: REPO_ROOT, encoding: "utf8" },
);
if (validation.status !== 0) {
  process.stderr.write(validation.stderr);
  process.exit(1);
}
process.stdout.write(validation.stdout);
