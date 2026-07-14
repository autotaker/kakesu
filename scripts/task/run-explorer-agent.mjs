import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { parseArgs } from "./lib.mjs";
import {
  buildLaunchEvidence,
  codexCommand,
  readCanonicalContracts,
  validateExplorerQuestion,
} from "./agent-routing.mjs";

const PRODUCT_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const SUPPORTED_ARGS = new Set(["root", "question", "dry_run"]);

export function parseExplorerArgs(argv) {
  if (argv.filter((argument) => argument === "--question").length !== 1) {
    throw new Error("ROUTING_BOUNDED_QUESTION_REQUIRED");
  }
  const args = parseArgs(argv);
  const unknown = Object.keys(args).filter((key) => !SUPPORTED_ARGS.has(key));
  if (unknown.length) throw new Error(`EXPLORER_ARGUMENT_UNKNOWN: ${unknown.join(",")}`);
  validateExplorerQuestion(args.question);
  if (args.dry_run && !new Set(["true", "false"]).has(args.dry_run)) {
    throw new Error("EXPLORER_DRY_RUN_INVALID");
  }
  return args;
}

export function buildExplorerInvocation({ repository, question, productRoot = PRODUCT_ROOT }) {
  validateExplorerQuestion(question);
  const cwd = path.resolve(repository);
  if (!fs.existsSync(cwd) || !fs.statSync(cwd).isDirectory()) {
    throw new Error(`EXPLORER_ROOT_INVALID: ${cwd}`);
  }
  const route = { role: "explorer", ...readCanonicalContracts(productRoot).explorer };
  const prompt = [
    "Act as the Explorer Agent for one bounded local repository question.",
    `Question: ${JSON.stringify(question)}`,
    "Use targeted read-only searches and file reads only.",
    "Do not browse, edit files, use Git write operations, expand scope, or delegate to another agent.",
    "Return only a concise evidence summary with file references. The launcher records the effective execution contract.",
  ].join(" ");
  return { route, cwd, command: codexCommand(route, cwd, prompt) };
}

export function runExplorer({ repository, question, productRoot = PRODUCT_ROOT, spawn = spawnSync }) {
  const invocation = buildExplorerInvocation({ repository, question, productRoot });
  const result = spawn("codex", invocation.command, {
    cwd: invocation.cwd,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  const exitCode = result.status ?? 1;
  const error = result.error?.message ?? (exitCode === 0 ? null : `CODEX_EXIT_${exitCode}`);
  const evidence = buildLaunchEvidence({
    route: invocation.route,
    cwd: invocation.cwd,
    allowedPaths: [],
    childResult: { exit_code: exitCode },
    commit: null,
    error,
  });
  return { ...invocation, result, evidence };
}

function emitEvidence(evidence) {
  process.stdout.write(`${JSON.stringify(evidence)}\n`);
}

const isMain = process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (isMain) {
  try {
    const args = parseExplorerArgs(process.argv.slice(2));
    const repository = path.resolve(args.root || process.cwd());
    if (args.dry_run === "true") {
      const invocation = buildExplorerInvocation({ repository, question: args.question });
      emitEvidence(buildLaunchEvidence({ route: invocation.route, cwd: invocation.cwd, allowedPaths: [] }));
    } else {
      const launched = runExplorer({ repository, question: args.question });
      if (launched.result.stdout) {
        process.stdout.write(launched.result.stdout);
        if (!launched.result.stdout.endsWith("\n")) process.stdout.write("\n");
      }
      if (launched.result.stderr) process.stderr.write(launched.result.stderr);
      emitEvidence(launched.evidence);
      if (launched.evidence.child_result.exit_code !== 0) process.exitCode = launched.evidence.child_result.exit_code;
    }
  } catch (error) {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  }
}
