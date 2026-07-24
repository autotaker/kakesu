import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import Ajv2020 from "ajv/dist/2020.js";
import addFormats from "ajv-formats";
import { checkTask } from "./check-task.mjs";
import { POINT_SCALE, TASK_STATUSES, parseArgs, parseFrontmatter, readYaml, resolveInside, workRoot } from "./lib.mjs";
import { buildWikiIndex } from "./wiki-index.mjs";

const args = parseArgs(process.argv.slice(2));
const root = workRoot(args.work_root);
const schemaRoot = workRoot(args.schema_root ?? args.work_root);
const errors = [];
const projectFile = path.join(root, "project.yaml");
const backlogFile = path.join(root, "backlog.yaml");
const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);
const validators = new Map();

function validateSchema(schemaFile, value, label) {
  if (!fs.existsSync(schemaFile)) {
    errors.push(`missing ${path.relative(schemaRoot, schemaFile)}`);
    return;
  }
  let validate = validators.get(schemaFile);
  if (!validate) {
    validate = ajv.compile(JSON.parse(fs.readFileSync(schemaFile, "utf8")));
    validators.set(schemaFile, validate);
  }
  if (!validate(value)) {
    for (const issue of validate.errors ?? []) {
      errors.push(`${label}${issue.instancePath || ""}: ${issue.message}`);
    }
  }
}

if (!fs.existsSync(projectFile)) errors.push("missing project.yaml");
if (!fs.existsSync(backlogFile)) errors.push("missing backlog.yaml");

const project = fs.existsSync(projectFile) ? readYaml(projectFile) : {};
const backlog = fs.existsSync(backlogFile) ? readYaml(backlogFile) : {};
validateSchema(path.join(schemaRoot, "schemas", "operations", "backlog.schema.json"), backlog, "backlog.yaml");

function markdownFiles(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) return markdownFiles(target);
    return entry.isFile() && entry.name.endsWith(".md") ? [target] : [];
  });
}

function validateLinks(file) {
  const content = fs.readFileSync(file, "utf8");
  for (const match of content.matchAll(/\[[^\]]+\]\(([^)]+)\)/g)) {
    const link = match[1].split("#", 1)[0];
    if (!link || /^[a-z][a-z0-9+.-]*:/i.test(link) || path.isAbsolute(link)) continue;
    const target = path.resolve(path.dirname(file), decodeURIComponent(link));
    if (!fs.existsSync(target)) {
      errors.push(`${path.relative(root, file)}: broken link ${match[1]}`);
    }
  }
}
if (project.version !== 2) errors.push("project.yaml version must be 2");
if (backlog.version !== 1) errors.push("backlog.yaml version must be 1");
if (project.default_branch !== "main") errors.push("project default_branch must be main");
if (project.repository_path !== "." || project.evidence_root !== ".") errors.push("project repository_path and evidence_root must be the unified root");
if (!fs.existsSync(path.join(root, ".git"))) errors.push(`unified root is not a Git worktree: ${root}`);

const epics = backlog.epics ?? [];
const tasks = backlog.tasks ?? [];
const epicIds = new Set();
for (const epic of epics) {
  if (!/^EPIC-\d{3}$/.test(epic.id ?? "")) errors.push(`invalid epic ID: ${epic.id}`);
  if (epicIds.has(epic.id)) errors.push(`duplicate epic ID: ${epic.id}`);
  epicIds.add(epic.id);
  if (!epic.title || !/^\d{4}-\d{2}-\d{2}$/.test(epic.target_start ?? "") || !/^\d{4}-\d{2}-\d{2}$/.test(epic.target_end ?? "")) {
    errors.push(`${epic.id}: title, target_start, and target_end are required`);
  }
}

const taskIds = new Set();
for (const task of tasks) {
  try {
    resolveInside(root, task.task_dir, `${task.id} task_dir`);
  } catch (error) {
    errors.push(error.message);
  }
  if (!/^TASK-\d{4}$/.test(task.id ?? "")) errors.push(`invalid task ID: ${task.id}`);
  if (taskIds.has(task.id)) errors.push(`duplicate task ID: ${task.id}`);
  taskIds.add(task.id);
  if (!epicIds.has(task.epic)) errors.push(`${task.id}: unknown epic ${task.epic}`);
  if (!TASK_STATUSES.has(task.status)) errors.push(`${task.id}: invalid status ${task.status}`);
  if (!new Set(["feature", "bug", "chore", "research"]).has(task.type)) errors.push(`${task.id}: invalid type ${task.type}`);
  if (!new Set(["P0", "P1", "P2", "P3"]).has(task.priority)) errors.push(`${task.id}: invalid priority ${task.priority}`);
  if (!POINT_SCALE.includes(task.estimate_points)) errors.push(`${task.id}: invalid estimate_points ${task.estimate_points}`);
  if (task.type === "bug" && !task.origin_task) errors.push(`${task.id}: bug requires origin_task`);
  if (task.bootstrap_exception && task.id !== "TASK-0001") errors.push(`${task.id}: bootstrap_exception is reserved for TASK-0001`);
}
for (const task of tasks) {
  for (const dependency of task.depends_on ?? []) {
    if (!taskIds.has(dependency)) errors.push(`${task.id}: unknown dependency ${dependency}`);
    if (dependency === task.id) errors.push(`${task.id}: task cannot depend on itself`);
  }
  errors.push(...checkTask(root, backlog, task.id));
}

const visiting = new Set();
const visited = new Set();
function visit(taskId, chain = []) {
  if (visiting.has(taskId)) {
    errors.push(`dependency cycle: ${[...chain, taskId].join(" -> ")}`);
    return;
  }
  if (visited.has(taskId)) return;
  visiting.add(taskId);
  const task = tasks.find((candidate) => candidate.id === taskId);
  for (const dependency of task?.depends_on ?? []) visit(dependency, [...chain, taskId]);
  visiting.delete(taskId);
  visited.add(taskId);
}
for (const task of tasks) visit(task.id);

const wikiIndex = buildWikiIndex(root);
const semanticKinds = new Set(["concept", "schema", "script", "case-pattern"]);
const decisionIds = new Set();
const decisions = [];
for (const page of wikiIndex.pages) {
  if (!page.title) errors.push(`${page.path}: title is required`);
  if (page.path.startsWith("wiki/semantic/") && !semanticKinds.has(page.kind)) {
    errors.push(`${page.path}: invalid semantic kind ${page.kind}`);
  }
  if (page.path.startsWith("wiki/decisions/")) {
    const metadata = parseFrontmatter(path.join(root, page.path));
    validateSchema(path.join(schemaRoot, "schemas", "operations", "decision.schema.json"), metadata, page.path);
    decisions.push({ ...metadata, path: page.path });
    if (page.kind !== "decision" || !/^DECISION-\d{4}$/.test(page.decision_id ?? "")) {
      errors.push(`${page.path}: invalid Decision metadata`);
    }
    if (!new Set(["accepted", "superseded"]).has(page.status)) {
      errors.push(`${page.path}: Decision status must be accepted or superseded`);
    }
    if (decisionIds.has(page.decision_id)) errors.push(`${page.path}: duplicate Decision ID ${page.decision_id}`);
    decisionIds.add(page.decision_id);
  }
}
const decisionById = new Map(decisions.map((decision) => [decision.decision_id, decision]));
for (const decision of decisions) {
  for (const superseded of decision.supersedes ?? []) {
    if (superseded === decision.decision_id) errors.push(`${decision.path}: Decision cannot supersede itself`);
    if (!decisionIds.has(superseded)) errors.push(`${decision.path}: unknown supersedes Decision ${superseded}`);
    if (decision.status === "accepted" && decisionById.get(superseded)?.status !== "superseded") {
      errors.push(`${decision.path}: superseded Decision ${superseded} must have status superseded`);
    }
  }
}
for (const decision of decisions.filter((candidate) => candidate.status === "superseded")) {
  const successors = decisions.filter((candidate) => candidate.status === "accepted" && candidate.supersedes?.includes(decision.decision_id));
  if (!successors.length) errors.push(`${decision.path}: superseded Decision requires an accepted successor`);
}
for (const file of [...markdownFiles(path.join(root, "tasks")), ...markdownFiles(path.join(root, "wiki"))]) {
  validateLinks(file);
}

const indexFile = path.join(root, "wiki", "index.json");
if (!fs.existsSync(indexFile)) {
  errors.push("missing wiki/index.json");
} else {
  const expected = `${JSON.stringify(wikiIndex, null, 2)}\n`;
  if (fs.readFileSync(indexFile, "utf8") !== expected) errors.push("wiki/index.json is stale; run make wiki-index");
}

const ingestionDir = path.join(root, "wiki", "ingestions");
if (fs.existsSync(ingestionDir)) {
  for (const entry of fs.readdirSync(ingestionDir)) {
    if (!entry.endsWith(".json")) continue;
    let receipt;
    try {
      receipt = JSON.parse(fs.readFileSync(path.join(ingestionDir, entry), "utf8"));
    } catch (error) {
      errors.push(`wiki/ingestions/${entry}: invalid JSON: ${error.message}`);
      continue;
    }
    validateSchema(path.join(schemaRoot, "schemas", "operations", "ingestion-receipt.schema.json"), receipt, `wiki/ingestions/${entry}`);
    if (!/^TASK-\d{4}$/.test(receipt.task_id ?? "") || !/^[a-f0-9]{64}$/.test(receipt.handover_sha256 ?? "")) {
      errors.push(`wiki/ingestions/${entry}: invalid task_id or handover_sha256`);
    }
    if (!Array.isArray(receipt.updated_pages) || !Array.isArray(receipt.created_decisions)) {
      errors.push(`wiki/ingestions/${entry}: updated_pages and created_decisions must be arrays`);
    }
    if (entry !== `${receipt.task_id}.json`) {
      errors.push(`wiki/ingestions/${entry}: filename must match task_id`);
    }
    const task = tasks.find((candidate) => candidate.id === receipt.task_id);
    if (!task) {
      errors.push(`wiki/ingestions/${entry}: unknown task_id`);
    } else {
      const handover = path.join(root, task.task_dir, "HANDOVER.md");
      const actualDigest = crypto.createHash("sha256").update(fs.readFileSync(handover)).digest("hex");
      if (receipt.handover_sha256 !== actualDigest) {
        errors.push(`wiki/ingestions/${entry}: handover_sha256 does not match HANDOVER.md`);
      }
    }
    for (const page of receipt.updated_pages ?? []) {
      try {
        const target = resolveInside(root, page, `wiki/ingestions/${entry} updated_pages entry`);
        if (!target.startsWith(`${path.join(root, "wiki", "semantic")}${path.sep}`)) {
          errors.push(`wiki/ingestions/${entry}: updated page must be under wiki/semantic: ${page}`);
        } else if (!fs.existsSync(target)) {
          errors.push(`wiki/ingestions/${entry}: missing updated page ${page}`);
        }
      } catch (error) {
        errors.push(error.message);
      }
    }
    for (const decisionId of receipt.created_decisions ?? []) {
      if (!decisionIds.has(decisionId)) errors.push(`wiki/ingestions/${entry}: unknown Decision ${decisionId}`);
    }
  }
}

const bootstrapManifest = path.join(root, "tasks", "TASK-0033-unify-work-repository", "BOOTSTRAP_MANIFEST.json");
if (fs.existsSync(bootstrapManifest)) {
  try {
    const manifest = JSON.parse(fs.readFileSync(bootstrapManifest, "utf8"));
    validateSchema(path.join(schemaRoot, "schemas", "operations", "bootstrap-manifest.schema.json"), manifest, "BOOTSTRAP_MANIFEST.json");
    const { manifest_sha256: recorded, ...body } = manifest;
    const actual = crypto.createHash("sha256").update(`${JSON.stringify(body)}\n`).digest("hex");
    if (recorded !== actual) errors.push("BOOTSTRAP_MANIFEST.json: self-digest mismatch");
  } catch (error) {
    errors.push(`BOOTSTRAP_MANIFEST.json: invalid JSON: ${error.message}`);
  }
}

if (errors.length) {
  process.stderr.write(`${[...new Set(errors)].map((error) => `- ${error}`).join("\n")}\n`);
  process.exit(1);
}
process.stdout.write(`Validated ${epics.length} epic(s), ${tasks.length} task(s), and ${wikiIndex.pages.length} Wiki page(s).\n`);
