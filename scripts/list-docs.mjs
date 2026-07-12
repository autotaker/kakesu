import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import YAML from "yaml";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const glossary = YAML.parse(fs.readFileSync(path.join(ROOT, "docs", "glossary.yml"), "utf8"));

function globToRegExp(glob) {
  let expression = "^";
  for (let index = 0; index < glob.length; index += 1) {
    const character = glob[index];
    if (character === "*" && glob[index + 1] === "*") {
      if (glob[index + 2] === "/") {
        expression += "(?:.*/)?";
        index += 2;
      } else {
        expression += ".*";
        index += 1;
      }
    } else if (character === "*") {
      expression += "[^/]*";
    } else if (character === "?") {
      expression += "[^/]";
    } else {
      expression += character.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    }
  }
  return new RegExp(`${expression}$`);
}

export function documentationFiles() {
  const scope = glossary.scope;
  const sourcePatterns = scope.sources.map(globToRegExp);
  const excludePatterns = scope.excludes.map(globToRegExp);
  const japanese = /[ぁ-んァ-ヶ一-龯]/;
  const files = execFileSync(
    "git",
    ["ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", "*.md"],
    { cwd: ROOT, encoding: "utf8" },
  )
    .split("\0")
    .filter(Boolean)
    .filter((file) => sourcePatterns.some((pattern) => pattern.test(file)))
    .filter((file) => !excludePatterns.some((pattern) => pattern.test(file)))
    .filter((file) => {
      if (scope.language !== "japanese") {
        return true;
      }
      return japanese.test(fs.readFileSync(path.join(ROOT, file), "utf8"));
    })
    .sort();
  return files;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (process.argv.includes("--json")) {
    process.stdout.write(`${JSON.stringify(documentationFiles())}\n`);
  } else {
    process.stdout.write(`${documentationFiles().join("\n")}\n`);
  }
}
