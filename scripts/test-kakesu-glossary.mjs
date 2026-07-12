import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(import.meta.url);
const textlintBin = require.resolve("textlint/bin/textlint.js");
const temporaryDirectory = fs.mkdtempSync(path.join(os.tmpdir(), "kakesu-glossary-"));
const fixture = path.join(temporaryDirectory, "fixture.md");

function lint(markdown) {
  fs.writeFileSync(fixture, markdown, "utf8");
  return spawnSync(
    process.execPath,
    [
      textlintBin,
      "--config",
      path.join(ROOT, ".textlintrc.json"),
      "--rulesdir",
      path.join(ROOT, "scripts"),
      fixture,
      "--format",
      "compact",
      "--no-color",
    ],
    { cwd: ROOT, encoding: "utf8" },
  );
}

try {
  const invalid = lint(
    "これはmessageとmessage_idの例である。Tokio taskは別概念である。\n\n" +
      "*message*\n\n> message\n\n[message](https://example.com/api)\n",
  );
  const invalidOutput = `${invalid.stdout}${invalid.stderr}`;
  if (invalid.status === 0) {
    throw new Error("invalid terminology fixture unexpectedly passed");
  }
  if (!invalidOutput.includes("message => メッセージ")) {
    throw new Error(`Japanese replacement violation was not detected:\n${invalidOutput}`);
  }
  if (!invalidOutput.includes("message_id must be enclosed in backticks")) {
    throw new Error("identifier formatting violation was not detected");
  }
  if (invalidOutput.includes("Tokio task")) {
    throw new Error("generic Tokio task was incorrectly treated as the Kakesu Task term");
  }
  if (invalidOutput.includes("api => API")) {
    throw new Error("a link destination was incorrectly linted as prose");
  }
  if ((invalidOutput.match(/message => メッセージ/g) || []).length !== 4) {
    throw new Error(`emphasis, blockquote, or link-label terminology was skipped:\n${invalidOutput}`);
  }

  const valid = lint("これはメッセージと`message_id`の例である。Tokio taskは別概念である。\n");
  if (valid.status !== 0) {
    throw new Error(`valid terminology fixture failed:\n${valid.stdout}${valid.stderr}`);
  }
  console.log("PASS: glossary replacement, identifier formatting, and task context safety");
} finally {
  fs.rmSync(temporaryDirectory, { recursive: true, force: true });
}
