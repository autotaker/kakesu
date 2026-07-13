import fs from "node:fs";
import path from "node:path";
import { acquireWorkRepoLock, escapeHtml, parseArgs, readYaml, workRoot, writeFileAtomic } from "./lib.mjs";

const args = parseArgs(process.argv.slice(2));
const root = workRoot(args.work_root);
const release = acquireWorkRepoLock(root);
try {
const backlog = readYaml(path.join(root, "backlog.yaml"));
const project = readYaml(path.join(root, "project.yaml"));
const tasks = backlog.tasks ?? [];
const statuses = ["backlog", "plan", "dev", "qa", "blocked", "done", "cancelled"];

function epicProgress(epicId) {
  const children = tasks.filter((task) => task.epic === epicId && task.status !== "cancelled");
  const total = children.reduce((sum, task) => sum + task.estimate_points, 0);
  const done = children.filter((task) => task.status === "done").reduce((sum, task) => sum + task.estimate_points, 0);
  return { children, total, done, percent: total ? Math.round((done / total) * 100) : 0 };
}

const kanban = statuses.map((status) => {
  const cards = tasks.filter((task) => task.status === status).map((task) => `
    <article class="card priority-${escapeHtml(task.priority)}">
      <div class="card-top"><span>${escapeHtml(task.id)}</span><span>${escapeHtml(task.estimate_points)} pt</span></div>
      <h3>${escapeHtml(task.title)}</h3>
      <p>${escapeHtml(task.epic)} · ${escapeHtml(task.type)} · ${escapeHtml(task.priority)}</p>
      ${(task.depends_on ?? []).length ? `<small>depends: ${task.depends_on.map(escapeHtml).join(", ")}</small>` : ""}
    </article>`).join("");
  return `<section class="column"><header><h2>${escapeHtml(status)}</h2><span>${tasks.filter((task) => task.status === status).length}</span></header>${cards || '<p class="empty">No tasks</p>'}</section>`;
}).join("");

const roadmap = (backlog.epics ?? []).map((epic) => {
  const progress = epicProgress(epic.id);
  const distribution = statuses.filter((status) => status !== "cancelled").map((status) => {
    const count = progress.children.filter((task) => task.status === status).length;
    return count ? `${status} ${count}` : null;
  }).filter(Boolean).join(" / ");
  return `<article class="epic">
    <div class="epic-title"><div><span>${escapeHtml(epic.id)}</span><h3>${escapeHtml(epic.title)}</h3></div><strong>${progress.percent}%</strong></div>
    <div class="bar"><i style="width:${progress.percent}%"></i></div>
    <p>${escapeHtml(epic.target_start)} → ${escapeHtml(epic.target_end)} · ${progress.done}/${progress.total} pt</p>
    <small>${escapeHtml(distribution || "No active tasks")}</small>
  </article>`;
}).join("");

const generatedAt = new Intl.DateTimeFormat("sv-SE", {
  timeZone: project.timezone || "UTC",
  dateStyle: "short",
  timeStyle: "medium",
  hour12: false,
}).format(new Date());
const html = `<!doctype html>
<html lang="ja"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>${escapeHtml(backlog.project)} development board</title>
<style>
:root{color-scheme:dark;--bg:#0c1117;--panel:#151d27;--line:#293442;--ink:#eef5fb;--muted:#93a4b7;--accent:#57d6ad;--warm:#ffbd69}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 10% 0,#182a35 0,transparent 36%),var(--bg);color:var(--ink);font:14px/1.5 ui-sans-serif,system-ui,sans-serif}main{max-width:1600px;margin:auto;padding:32px}h1{font-size:30px;margin:0}.eyebrow{color:var(--accent);letter-spacing:.14em;text-transform:uppercase}.meta{color:var(--muted);margin:4px 0 28px}.roadmap{display:grid;grid-template-columns:repeat(auto-fit,minmax(320px,1fr));gap:16px;margin-bottom:30px}.epic,.column{background:color-mix(in srgb,var(--panel) 94%,transparent);border:1px solid var(--line);border-radius:16px}.epic{padding:18px}.epic-title{display:flex;justify-content:space-between;gap:12px}.epic-title span,.card-top{color:var(--muted);font-size:12px}.epic h3{margin:2px 0}.epic strong{font-size:24px;color:var(--accent)}.bar{height:8px;background:#27313c;border-radius:999px;overflow:hidden;margin:16px 0}.bar i{display:block;height:100%;background:linear-gradient(90deg,var(--accent),#87e8c9)}.epic p,.epic small{color:var(--muted)}.board{display:grid;grid-template-columns:repeat(7,minmax(160px,1fr));gap:12px;overflow-x:auto;padding-bottom:18px}.column{min-height:360px;padding:12px}.column>header{display:flex;justify-content:space-between;align-items:center;text-transform:uppercase}.column>header h2{font-size:13px;letter-spacing:.09em}.column>header span{background:#263442;border-radius:999px;padding:2px 8px}.card{background:#1b2632;border:1px solid #314050;border-left:4px solid var(--accent);border-radius:12px;padding:13px;margin:10px 0}.card h3{font-size:14px;margin:9px 0}.card p,.card small,.empty{color:var(--muted)}.card-top{display:flex;justify-content:space-between}.priority-P0{border-left-color:#ff6b6b}.priority-P1{border-left-color:var(--warm)}.empty{text-align:center;padding:36px 0}@media(max-width:800px){main{padding:20px}.board{grid-template-columns:repeat(7,260px)}}
</style></head><body><main>
<p class="eyebrow">${escapeHtml(backlog.project)} / delivery</p><h1>Development board</h1><p class="meta">Generated ${escapeHtml(generatedAt)} ${escapeHtml(project.timezone || "UTC")} · completed points only count toward progress</p>
<h2>Epic roadmap</h2><section class="roadmap">${roadmap}</section>
<h2>Task kanban</h2><section class="board">${kanban}</section>
</main></body></html>`;

const output = path.join(root, "viewer", "index.html");
fs.mkdirSync(path.dirname(output), { recursive: true });
writeFileAtomic(output, html);
process.stdout.write(`${output}\n`);
} finally {
  release();
}
