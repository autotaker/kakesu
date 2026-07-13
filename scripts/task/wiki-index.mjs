import fs from "node:fs";
import path from "node:path";
import { acquireWorkRepoLock, git, parseArgs, parseFrontmatter, workRoot, writeFileAtomic } from "./lib.mjs";

function markdownFiles(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const target = path.join(directory, entry.name);
    if (entry.isDirectory()) return markdownFiles(target);
    return entry.isFile() && entry.name.endsWith(".md") ? [target] : [];
  });
}

export function buildWikiIndex(root) {
  const wikiRoot = path.join(root, "wiki");
  const files = [
    ...markdownFiles(path.join(wikiRoot, "semantic")),
    ...markdownFiles(path.join(wikiRoot, "decisions")),
  ].sort();
  return {
    version: 1,
    pages: files.map((file) => {
      const metadata = parseFrontmatter(file);
      const content = fs.readFileSync(file, "utf8");
      const links = [...content.matchAll(/\[[^\]]+\]\(([^)]+)\)/g)].map((match) => match[1]);
      return {
        path: path.relative(root, file),
        kind: metadata.kind,
        title: metadata.title,
        decision_id: metadata.decision_id,
        status: metadata.status,
        links,
      };
    }),
  };
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === new URL(import.meta.url).pathname;
if (isMain) {
  const args = parseArgs(process.argv.slice(2));
  const root = workRoot(args.work_root);
  const outerWriter = process.env.WORK_REPO_LOCK_HELD === "1";
  const release = outerWriter ? () => {} : acquireWorkRepoLock(root);
  try {
    const output = path.join(root, "wiki", "index.json");
    writeFileAtomic(output, `${JSON.stringify(buildWikiIndex(root), null, 2)}\n`);
    if (!outerWriter && git(root, ["status", "--porcelain", "wiki/index.json"])) {
      git(root, ["add", "wiki/index.json"]);
      git(root, ["commit", "-m", "wiki: refresh index"]);
    }
    process.stdout.write(`${output}\n`);
  } finally {
    release();
  }
}
