"use strict";

const fs = require("node:fs");
const path = require("node:path");
const YAML = require("yaml");

const ROOT = path.resolve(__dirname, "..");
const GLOSSARY = path.join(ROOT, "docs", "glossary.yml");
const glossary = YAML.parse(fs.readFileSync(GLOSSARY, "utf8"));
const rules = glossary.rules;

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const expressionFor = (pattern) => {
  const escaped = escapeRegExp(pattern);
  const prefix = /^[A-Za-z0-9_]/.test(pattern) ? "(?<![A-Za-z0-9_-])" : "";
  const suffix = /[A-Za-z0-9_]$/.test(pattern) ? "(?![A-Za-z0-9_-])" : "";
  return new RegExp(`${prefix}${escaped}${suffix}`, "g");
};

const replacementRules = [
  ...Object.entries(rules.replacements || {}),
  ...Object.entries(rules.preserved || {}).filter(([, rule]) => typeof rule.to === "string"),
]
  .flatMap(([formalName, rule]) => {
    const patterns = [...new Set([...(rule.match || [formalName]), rule.to])];
    return patterns.map((pattern) => {
      return {
        formalName,
        pattern,
        replacement: rule.to,
        expression: expressionFor(pattern),
      };
    });
  })
  .sort((left, right) => right.pattern.length - left.pattern.length);

const identifierRules = [
  ...new Map(
    Object.entries(rules.identifiers || {})
      .flatMap(([formalName, rule]) =>
        (rule.match || []).map((pattern) => [
          pattern,
          {
            formalName,
            pattern,
            expression: expressionFor(pattern),
          },
        ]),
      ),
  ).values(),
];

function reporter(context) {
  const { Syntax, getSource, report, fixer, RuleError } = context;

  function isLinkDestination(node) {
    let parent = node.parent;
    while (parent) {
      if (parent.type === Syntax.Link) {
        return node.value === parent.url || /^[A-Za-z][A-Za-z0-9+.-]*:\/\//.test(node.value);
      }
      parent = parent.parent;
    }
    return false;
  }

  return {
    [Syntax.Str](node) {
      if (isLinkDestination(node)) {
        return;
      }
      const source = getSource(node);
      const occupied = [];
      const overlaps = (start, end) =>
        occupied.some(([occupiedStart, occupiedEnd]) => start < occupiedEnd && end > occupiedStart);
      for (const rule of replacementRules) {
        for (const match of source.matchAll(rule.expression)) {
          const end = match.index + match[0].length;
          if (overlaps(match.index, end)) {
            continue;
          }
          if (match[0] === rule.replacement) {
            occupied.push([match.index, end]);
            continue;
          }
          occupied.push([match.index, end]);
          report(
            node,
            new RuleError(`${match[0]} => ${rule.replacement} (glossary: ${rule.formalName})`, {
              index: match.index,
              fix: fixer.replaceTextRange(
                [match.index, match.index + match[0].length],
                rule.replacement,
              ),
            }),
          );
        }
      }
      for (const rule of identifierRules) {
        for (const match of source.matchAll(rule.expression)) {
          const end = match.index + match[0].length;
          if (overlaps(match.index, end)) {
            continue;
          }
          occupied.push([match.index, end]);
          report(
            node,
            new RuleError(
              `${match[0]} must be enclosed in backticks (glossary: ${rule.formalName})`,
              {
                index: match.index,
                fix: fixer.replaceTextRange(
                  [match.index, match.index + match[0].length],
                  `\`${match[0]}\``,
                ),
              },
            ),
          );
        }
      }
    },
  };
}

module.exports = {
  linter: reporter,
  fixer: reporter,
};
