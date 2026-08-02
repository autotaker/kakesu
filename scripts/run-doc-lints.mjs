import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const DEFAULT_UV_CACHE_DIR = path.join(ROOT, ".build", "uv-cache");

export function buildDocLintCommands({ uv = "uv", pnpm = "pnpm" } = {}) {
  return [
    {
      command: uv,
      args: ["run", "--project", "memory", "python", "scripts/validate-terminology.py"],
      usesUvCache: true,
    },
    { command: pnpm, args: ["lint:docs"] },
    { command: "git", args: ["diff", "--check"] },
  ];
}

export function runDocLints({
  spawn = spawnSync,
  cwd = ROOT,
  env = process.env,
  uv = env.UV ?? "uv",
  pnpm = env.PNPM ?? "pnpm",
  uvCacheDir = env.UV_CACHE_DIR ?? DEFAULT_UV_CACHE_DIR,
  report = () => console.error("doc lint command failed to start"),
} = {}) {
  let failed = false;

  for (const { command, args, usesUvCache } of buildDocLintCommands({ uv, pnpm })) {
    const options = {
      cwd,
      env: usesUvCache ? { ...env, UV_CACHE_DIR: uvCacheDir } : env,
      shell: false,
      stdio: "inherit",
    };
    try {
      const result = spawn(command, args, options);
      if (result?.error) {
        report();
        failed = true;
      } else if (result?.status !== 0) {
        failed = true;
      }
    } catch {
      report();
      failed = true;
    }
  }

  return failed ? 1 : 0;
}

if (path.resolve(process.argv[1] ?? "") === fileURLToPath(import.meta.url)) {
  process.exitCode = runDocLints();
}
