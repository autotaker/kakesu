import fs, { chmodSync } from "node:fs";
import path from "node:path";
import { git, parseArgs, workRoot } from "./lib.mjs";

const args = parseArgs(process.argv.slice(2));
const root = workRoot(args.work_root);
const hook = path.join(root, ".githooks", "pre-commit");
if (!fs.existsSync(path.join(root, ".git"))) throw new Error(`${root} is not a Git repository`);
if (!fs.existsSync(hook)) throw new Error(`${hook} is missing`);
if (git(root, ["branch", "--show-current"]) !== "main") throw new Error("Work repository must use main");
chmodSync(hook, 0o755);
git(root, ["config", "core.hooksPath", ".githooks"]);
process.stdout.write(`Configured ${root} with core.hooksPath=.githooks\n`);
