"use strict";

const fs = require("node:fs");
const path = require("node:path");
const YAML = require("yaml");

const ROOT = path.resolve(__dirname, "..");
const GLOSSARY = path.join(ROOT, "docs", "glossary.yml");
const glossary = YAML.parse(fs.readFileSync(GLOSSARY, "utf8"));
const categories = glossary.categories;

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const expressionFor = (pattern) => {
  const escaped = escapeRegExp(pattern);
  const prefix = /^[A-Za-z0-9_]/.test(pattern) ? "(?<![A-Za-z0-9_-])" : "";
  const suffix = /[A-Za-z0-9_]$/.test(pattern) ? "(?![A-Za-z0-9_-])" : "";
  return new RegExp(`${prefix}${escaped}${suffix}`, "g");
};

const terms = Object.values(categories)
  .flatMap((category) => category.terms || [])
  .filter((term) => term.lint?.enabled === true);

const replacementRules = terms
  .filter((term) => term.lint.mode === "replace")
  .flatMap((term) => {
    const lint = term.lint;
    return (lint.patterns || []).map((pattern) => {
      return {
        formalName: term.formal_name,
        replacement: lint.replacement,
        expression: expressionFor(pattern),
      };
    });
  });

const identifierRules = [
  ...new Map(
    terms
      .filter((term) => term.lint.mode === "identifier")
      .flatMap((term) =>
        (term.lint.patterns || []).map((pattern) => [
          pattern,
          {
            formalName: term.formal_name,
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
      for (const rule of replacementRules) {
        for (const match of source.matchAll(rule.expression)) {
          if (match[0] === rule.replacement) {
            continue;
          }
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
