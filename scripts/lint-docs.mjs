import { spawnSync } from "node:child_process";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { documentationFiles } from "./list-docs.mjs";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(import.meta.url);
const textlintBin = require.resolve("textlint/bin/textlint.js");
const markdownFiles = documentationFiles();

const forwardedArgs = process.argv.slice(2);
const batchSize = 40;
for (let offset = 0; offset < markdownFiles.length; offset += batchSize) {
  const files = markdownFiles.slice(offset, offset + batchSize);
  const result = spawnSync(
    process.execPath,
    [textlintBin, "--config", ".textlintrc.json", "--rulesdir", "scripts", ...files, ...forwardedArgs],
    { cwd: ROOT, stdio: "inherit" },
  );
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}
