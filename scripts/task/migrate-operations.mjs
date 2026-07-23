import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import YAML from "yaml";
import { REPO_ROOT, parseArgs, parseFrontmatter, resolveInside, writeFileAtomic } from "./lib.mjs";

const BASELINE_REF = "d030db5dc2974056387616d047197823b94602ce";
const TASK_ID = "TASK-0033";
const MANIFEST_PATH = "tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json";
const BASELINE_PREFIXES = ["tasks/", "wiki/", "lap30/"];
const BASELINE_FILES = new Set(["backlog.yaml", "viewer/index.html"]);

function runGit(root, argv, { allowFailure = false } = {}) {
  const result = spawnSync("git", argv, { cwd: root, encoding: null });
  if (result.error) throw result.error;
  if (result.status !== 0 && !allowFailure) {
    throw new Error(Buffer.from(result.stderr || result.stdout || "git failed").toString().trim());
  }
  return result;
}

function sha256(content) {
  return crypto.createHash("sha256").update(content).digest("hex");
}

function sourceFiles(source, sourceRef) {
  const output = runGit(source, ["ls-tree", "-r", "--name-only", "-z", sourceRef]).stdout;
  return output.toString().split("\0").filter(Boolean).filter((file) =>
    BASELINE_FILES.has(file) || BASELINE_PREFIXES.some((prefix) => file.startsWith(prefix)));
}

function readRefFile(source, sourceRef, file) {
  return runGit(source, ["show", `${sourceRef}:${file}`]).stdout;
}

function listFiles(root, relative) {
  const start = resolveInside(root, relative, "overlay path");
  if (!fs.existsSync(start)) throw new Error(`Missing overlay path: ${relative}`);
  const visit = (absolute) => fs.readdirSync(absolute, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(absolute, entry.name);
    if (entry.isSymbolicLink()) throw new Error(`Migration refuses symlink: ${path.relative(root, target)}`);
    if (entry.isDirectory()) return visit(target);
    return entry.isFile() ? [path.relative(root, target).split(path.sep).join("/")] : [];
  });
  return visit(start).sort();
}

function taskDirFor(backlog, taskId) {
  const matches = (backlog.tasks ?? []).filter((task) => task.id === taskId);
  if (matches.length !== 1) throw new Error(`Expected exactly one ${taskId} in source backlog`);
  return matches[0].task_dir;
}

function assertOverlay(backlogAtRef, currentBacklog) {
  const baseline = backlogAtRef.tasks ?? [];
  const current = currentBacklog.tasks ?? [];
  if (baseline.length !== 32) throw new Error(`REF-2 must contain 32 tasks, found ${baseline.length}`);
  if (current.length !== 33) throw new Error(`Current source must contain 32 historical tasks plus ${TASK_ID}`);
  if (JSON.stringify(current.slice(0, 32)) !== JSON.stringify(baseline)) {
    throw new Error("Historical backlog entries differ from REF-2; migration is append-only");
  }
  if (current[32]?.id !== TASK_ID) throw new Error(`${TASK_ID} must be the only backlog overlay`);
}

function buildManifest(source, target, sourceRef) {
  const resolvedRef = runGit(source, ["rev-parse", `${sourceRef}^{commit}`]).stdout.toString().trim();
  if (resolvedRef !== sourceRef) throw new Error(`Source ref mismatch: expected ${sourceRef}, got ${resolvedRef}`);
  const baselineBacklogBytes = readRefFile(source, sourceRef, "backlog.yaml");
  const baselineBacklog = YAML.parse(baselineBacklogBytes.toString());
  const currentBacklogBytes = fs.readFileSync(path.join(source, "backlog.yaml"));
  const currentBacklog = YAML.parse(currentBacklogBytes.toString());
  assertOverlay(baselineBacklog, currentBacklog);
  const taskDir = taskDirFor(currentBacklog, TASK_ID);
  const baselinePaths = sourceFiles(source, sourceRef).filter((file) => file !== "backlog.yaml");
  const overlayPaths = ["backlog.yaml", ...listFiles(source, taskDir)];
  const entries = [
    ...baselinePaths.map((file) => ({ file, origin: "REF-2", content: readRefFile(source, sourceRef, file) })),
    ...overlayPaths.map((file) => ({ file, origin: file === "backlog.yaml" ? "TASK-0033-overlay" : "TASK-0033-evidence", content: fs.readFileSync(resolveInside(source, file)) })),
  ].sort((left, right) => left.file.localeCompare(right.file));
  const duplicates = entries.filter((entry, index) => entries.findIndex((candidate) => candidate.file === entry.file) !== index);
  if (duplicates.length) throw new Error(`Duplicate migration paths: ${duplicates.map((entry) => entry.file).join(",")}`);
  const records = entries.map(({ file, origin, content }) => ({ file, origin, bytes: content.length, sha256: sha256(content) }));
  const categoryCounts = {
    historical_tasks: baselineBacklog.tasks.length,
    current_tasks: 1,
    task_files: records.filter((entry) => entry.file.startsWith("tasks/")).length,
    wiki_files: records.filter((entry) => entry.file.startsWith("wiki/")).length,
    lap30_files: records.filter((entry) => entry.file.startsWith("lap30/")).length,
  };
  const projectTemplate = path.join(REPO_ROOT, "project.yaml");
  const projectBytes = fs.existsSync(path.join(target, "project.yaml")) ? fs.readFileSync(path.join(target, "project.yaml")) : fs.readFileSync(projectTemplate);
  const manifest = {
    version: 1,
    task_id: TASK_ID,
    source_ref: sourceRef,
    source_tree: runGit(source, ["rev-parse", `${sourceRef}^{tree}`]).stdout.toString().trim(),
    target_repository: path.basename(target),
    project_sha256: sha256(projectBytes),
    category_counts: categoryCounts,
    entries: records,
  };
  const digestInput = `${JSON.stringify(manifest)}\n`;
  return { entries, projectBytes, manifest: { ...manifest, manifest_sha256: sha256(digestInput) } };
}

function writeEntry(target, entry) {
  const output = resolveInside(target, entry.file, "migration target");
  fs.mkdirSync(path.dirname(output), { recursive: true });
  if (fs.existsSync(output) && sha256(fs.readFileSync(output)) !== sha256(entry.content)) {
    throw new Error(`Refusing to overwrite non-identical target: ${entry.file}`);
  }
  if (!fs.existsSync(output)) fs.writeFileSync(output, entry.content);
}

function verify(target, manifest, readEntry = (file) => fs.readFileSync(resolveInside(target, file, "manifest entry"))) {
  const errors = [];
  const { manifest_sha256: recordedManifestDigest, ...manifestBody } = manifest;
  if (sha256(`${JSON.stringify(manifestBody)}\n`) !== recordedManifestDigest) errors.push("manifest self-digest mismatch");
  for (const entry of manifest.entries ?? []) {
    try {
      const content = readEntry(entry.file);
      if (content.length !== entry.bytes || sha256(content) !== entry.sha256) errors.push(`digest mismatch ${entry.file}`);
    } catch {
      errors.push(`missing ${entry.file}`);
    }
  }
  try {
    if (sha256(readEntry("project.yaml")) !== manifest.project_sha256) errors.push("project.yaml digest mismatch");
  } catch {
    errors.push("missing project.yaml");
  }
  if (manifest.category_counts?.historical_tasks !== 32 || manifest.category_counts?.current_tasks !== 1) errors.push("task count mismatch");
  if (errors.length) throw new Error(`Bootstrap verification failed:\n${errors.map((error) => `- ${error}`).join("\n")}`);
}

function quarantinePaths(source, target) {
  const targetCommon = path.resolve(target, runGit(target, ["rev-parse", "--git-common-dir"]).stdout.toString().trim());
  const key = sha256(path.resolve(source)).slice(0, 16);
  return {
    marker: path.join(source, ".agent-harness-evidence-quarantine.json"),
    pending: path.join(source, ".agent-harness-evidence-quarantine.pending.json"),
    quarantine: path.join(targetCommon, "agent-harness-quarantine", `${key}.git`),
  };
}

function freezeSource(source, target, sourceRef, expectedHead) {
  const paths = quarantinePaths(source, target);
  if (fs.existsSync(paths.marker) || fs.existsSync(paths.pending)) throw new Error("Source repository is already quarantined");
  const gitDirValue = runGit(source, ["rev-parse", "--git-dir"]).stdout.toString().trim();
  const gitDir = path.resolve(source, gitDirValue);
  if (gitDir !== path.join(source, ".git") || !fs.statSync(gitDir).isDirectory()) throw new Error("Quarantine requires a standalone .git directory");
  const worktrees = runGit(source, ["worktree", "list", "--porcelain"]).stdout.toString().split("\n").filter((line) => line.startsWith("worktree "));
  const registeredWorktree = worktrees.length === 1 ? fs.realpathSync(path.resolve(worktrees[0].slice(9))) : null;
  if (worktrees.length !== 1 || registeredWorktree !== fs.realpathSync(source)) throw new Error("Quarantine requires exactly the source main worktree");
  if (runGit(source, ["status", "--porcelain"]).stdout.toString().trim()) throw new Error("Quarantine requires a clean source repository");
  const head = runGit(source, ["rev-parse", "HEAD"]).stdout.toString().trim();
  if (!expectedHead || head !== expectedHead) throw new Error(`Source HEAD mismatch: expected ${expectedHead || "<required>"}, got ${head}`);
  const oldMarker = path.join(gitDir, "agent-harness-evidence-frozen");
  const legacy = fs.existsSync(oldMarker) ? JSON.parse(fs.readFileSync(oldMarker, "utf8")) : null;
  if (legacy && (legacy.authority !== path.resolve(target) || legacy.source_ref !== sourceRef)) throw new Error("Legacy freeze authority or source ref mismatch");
  const priorResult = runGit(source, ["config", "--local", "--get", "core.hooksPath"], { allowFailure: true });
  const state = {
    version: 2, authority: path.resolve(target), source: path.resolve(source), source_ref: sourceRef,
    expected_head: head, quarantine: paths.quarantine,
    prior_hooks_path: legacy?.prior_hooks_path ?? (priorResult.status === 0 ? priorResult.stdout.toString().trim() : null),
    legacy_frozen_hooks_path: legacy?.frozen_hooks_path ?? null,
  };
  if (fs.existsSync(paths.quarantine)) throw new Error(`Quarantine destination already exists: ${paths.quarantine}`);
  fs.mkdirSync(path.dirname(paths.quarantine), { recursive: true });
  writeFileAtomic(paths.pending, `${JSON.stringify(state)}\n`);
  try {
    fs.renameSync(gitDir, paths.quarantine);
  } catch (error) {
    fs.rmSync(paths.pending, { force: true });
    throw error;
  }
  fs.renameSync(paths.pending, paths.marker);
  process.stdout.write(`${paths.marker}\n`);
}

function unfreezeSource(source, target) {
  const paths = quarantinePaths(source, target);
  const stateFile = fs.existsSync(paths.marker) ? paths.marker : paths.pending;
  if (!fs.existsSync(stateFile)) throw new Error("Source repository is not quarantined");
  const state = JSON.parse(fs.readFileSync(stateFile, "utf8"));
  if (state.authority !== path.resolve(target) || state.source !== path.resolve(source) || state.quarantine !== paths.quarantine) throw new Error("Quarantine marker mismatch");
  const sourceGit = path.join(source, ".git");
  if (fs.existsSync(sourceGit) && fs.existsSync(paths.quarantine)) throw new Error("Both source and quarantined Git metadata exist; refusing ambiguous recovery");
  if (!fs.existsSync(sourceGit)) {
    const quarantinedHead = runGit(source, ["--git-dir", paths.quarantine, "rev-parse", "HEAD"]).stdout.toString().trim();
    if (quarantinedHead !== state.expected_head) throw new Error("Quarantined source HEAD mismatch");
    fs.renameSync(paths.quarantine, sourceGit);
  }
  if (runGit(source, ["rev-parse", "HEAD"]).stdout.toString().trim() !== state.expected_head) throw new Error("Restored source HEAD mismatch");
  if (state.prior_hooks_path) runGit(source, ["config", "--local", "core.hooksPath", state.prior_hooks_path]);
  else runGit(source, ["config", "--local", "--unset-all", "core.hooksPath"], { allowFailure: true });
  fs.rmSync(path.join(source, ".git", "agent-harness-evidence-frozen"), { force: true });
  if (state.legacy_frozen_hooks_path) fs.rmSync(state.legacy_frozen_hooks_path.replace(paths.quarantine, path.join(source, ".git")), { recursive: true, force: true });
  fs.rmSync(stateFile, { force: true });
}

const args = parseArgs(process.argv.slice(2));
const mode = args.mode ?? "plan";
if (!args.source && mode !== "verify") throw new Error("--source is required outside verify mode");
const source = path.resolve(args.source ?? REPO_ROOT);
const target = path.resolve(args.target ?? REPO_ROOT);
const sourceRef = args.source_ref ?? BASELINE_REF;
if (sourceRef !== BASELINE_REF && args.fixture !== "true") throw new Error(`Production migration requires REF-2 ${BASELINE_REF}`);
if (!["plan", "apply", "verify", "freeze", "unfreeze"].includes(mode)) throw new Error("--mode must be plan, apply, verify, freeze, or unfreeze");

if (mode === "freeze" || mode === "unfreeze") {
  if (mode === "freeze") freezeSource(source, target, sourceRef, args.expected_head);
  else unfreezeSource(source, target);
} else if (mode === "verify") {
  const handoverFile = path.join(target, "tasks/TASK-0033-unify-work-repository/HANDOVER.md");
  const binding = fs.existsSync(handoverFile) ? parseFrontmatter(handoverFile) : {};
  let manifest;
  if (/^[0-9a-f]{40}$/.test(binding.bootstrap_evidence_commit ?? "") && /^[0-9a-f]{64}$/.test(binding.bootstrap_evidence_digest ?? "")) {
    runGit(target, ["merge-base", "--is-ancestor", binding.bootstrap_evidence_commit, "main"]);
    const readCommit = (file) => runGit(target, ["show", `${binding.bootstrap_evidence_commit}:${file}`]).stdout;
    manifest = JSON.parse(readCommit(args.manifest ?? MANIFEST_PATH).toString());
    if (manifest.manifest_sha256 !== binding.bootstrap_evidence_digest) throw new Error("HANDOVER bootstrap digest does not match the bound manifest");
    verify(target, manifest, readCommit);
  } else {
    manifest = JSON.parse(fs.readFileSync(resolveInside(target, args.manifest ?? MANIFEST_PATH, "manifest"), "utf8"));
    verify(target, manifest);
  }
  process.stdout.write(`${manifest.manifest_sha256}\n`);
} else {
  const { entries, projectBytes, manifest } = buildManifest(source, target, sourceRef);
  if (mode === "apply") {
    const projectFile = path.join(target, "project.yaml");
    if (fs.existsSync(projectFile) && sha256(fs.readFileSync(projectFile)) !== sha256(projectBytes)) throw new Error("Refusing to overwrite non-identical target: project.yaml");
    if (!fs.existsSync(projectFile)) fs.writeFileSync(projectFile, projectBytes);
    for (const entry of entries) writeEntry(target, entry);
    const manifestFile = resolveInside(target, args.manifest ?? MANIFEST_PATH, "manifest");
    fs.mkdirSync(path.dirname(manifestFile), { recursive: true });
    writeFileAtomic(manifestFile, `${JSON.stringify(manifest, null, 2)}\n`);
    verify(target, manifest);
  }
  process.stdout.write(`${JSON.stringify(manifest, null, 2)}\n`);
}
