#!/usr/bin/env python3
"""Validate the curated documentation glossary without persisting extraction data."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from collections import Counter, defaultdict
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[1]
GLOSSARY = ROOT / "docs" / "glossary.yml"
GLOSSARY_INDEX = ROOT / "docs" / "99-glossary-index.md"
TEXTLINT_CONFIG = ROOT / ".textlintrc.json"
PACKAGE_JSON = ROOT / "package.json"
MAX_GLOSSARY_LINES = 1000
TOKEN_RE = re.compile(r"[A-Za-z][A-Za-z0-9_./:+-]*")
FENCED_RE = re.compile(r"```.*?```|~~~.*?~~~", re.DOTALL)
INLINE_CODE_RE = re.compile(r"`[^`\n]+`")
LINK_DESTINATION_RE = re.compile(r"\]\(([^)\s]+)(?:\s+[^)]*)?\)")
TRAILING_PUNCTUATION = ".,;!?)]}"


def markdown_files() -> list[Path]:
    result = subprocess.run(
        ["node", "scripts/list-docs.mjs", "--json"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return [ROOT / relative for relative in json.loads(result.stdout)]


def code_spans(text: str) -> list[tuple[int, int]]:
    spans = [match.span() for match in FENCED_RE.finditer(text)]
    spans.extend(match.span() for match in INLINE_CODE_RE.finditer(text))
    spans.extend(match.span(1) for match in LINK_DESTINATION_RE.finditer(text))
    return spans


def in_span(position: int, spans: list[tuple[int, int]]) -> bool:
    return any(start <= position < end for start, end in spans)


def extract_terms() -> tuple[Counter[str], dict[str, set[str]], Counter[str]]:
    counts: Counter[str] = Counter()
    token_files: dict[str, set[str]] = defaultdict(set)
    code_counts: Counter[str] = Counter()

    for path in markdown_files():
        relative = path.relative_to(ROOT).as_posix()
        text = path.read_text(encoding="utf-8")
        spans = code_spans(text)
        for match in TOKEN_RE.finditer(text):
            token = match.group(0).rstrip(TRAILING_PUNCTUATION)
            if not token:
                continue
            counts[token] += 1
            token_files[token].add(relative)
            if in_span(match.start(), spans):
                code_counts[token] += 1

    return counts, token_files, code_counts


def rule_patterns(term: str, rule: dict, *, default_to_term: bool) -> list[str]:
    patterns = rule.get("match")
    if patterns is None and default_to_term:
        return [term]
    return patterns or []


def registered_tokens(glossary: dict) -> set[str]:
    registered: set[str] = set()
    for group_name in ("replacements", "preserved", "identifiers"):
        group = glossary.get("rules", {}).get(group_name, {})
        for term, rule in group.items():
            registered.add(term)
            registered.update(rule_patterns(term, rule, default_to_term=group_name != "identifiers"))
            registered.update(rule.get("aliases", []))
    return registered


def unregistered_frequent_terms(glossary: dict) -> list[tuple[str, int, int]]:
    counts, _, code_counts = extract_terms()
    registered = registered_tokens(glossary)
    threshold = glossary.get("review", {}).get("frequent_prose_threshold", 3)
    findings = []
    for token, total_count in counts.items():
        prose_count = total_count - code_counts[token]
        if token in registered or prose_count < threshold:
            continue
        if code_counts[token] == total_count:
            continue
        if token.isupper() and len(token) > 1:
            continue
        if re.search(r"[_/.:]", token) or any(character.isdigit() for character in token):
            continue
        findings.append((token, prose_count, total_count))
    return sorted(findings, key=lambda item: (-item[1], item[0].lower(), item[0]))


def render_glossary_index(glossary: dict) -> str:
    replacements = glossary.get("rules", {}).get("replacements", {})
    lines = [
        "# Kakesu用語索引",
        "",
        "<!-- このファイルはdocs/glossary.ymlから自動生成する。直接編集しない。 -->",
        "",
        "Kakesu内で固有の意味を持つ用語について、標準表記、短い説明、定義元をまとめる。表記規則と表記揺れの正本は[glossary.yml](glossary.yml)とする。",
        "",
        "| 標準表記 | 正式名 | 説明 | 定義 |",
        "|---|---|---|---|",
    ]
    for term, project_term in glossary.get("project_terms", {}).items():
        if "display" in project_term:
            display = project_term["display"]
        elif term in replacements:
            display = replacements[term]["to"]
        else:
            display = term
        lines.append(
            f"| {display} | `{term}` | {project_term['description']} | "
            f"[設計書]({project_term['definition']}) |"
        )
    return "\n".join(lines) + "\n"


def write_generated_index(glossary: dict) -> None:
    GLOSSARY_INDEX.write_text(render_glossary_index(glossary), encoding="utf-8")
    print(f"updated {GLOSSARY_INDEX}: {len(glossary.get('project_terms', {}))} project terms")


def validate_rule_group(errors: list[str], name: str, group: object) -> None:
    if not isinstance(group, dict) or not group:
        errors.append(f"glossary rules.{name} must be a non-empty mapping")
        return
    for term, rule in group.items():
        if not isinstance(term, str) or not term or not isinstance(rule, dict):
            errors.append(f"invalid rule in glossary rules.{name}: {term!r}")
            continue
        match = rule_patterns(term, rule, default_to_term=name != "identifiers")
        if name == "replacements" and not isinstance(rule.get("to"), str):
            errors.append(f"replacement rule has no string 'to' value: {term}")
        if name == "preserved" and "to" in rule and not isinstance(rule.get("to"), str):
            errors.append(f"preserved canonicalization has invalid 'to' value: {term}")
        if name == "identifiers" and not match:
            errors.append(f"identifier rule has no match patterns: {term}")
        if not all(isinstance(pattern, str) and pattern for pattern in match):
            errors.append(f"rule has invalid match patterns: {term}")
        aliases = rule.get("aliases", [])
        if not isinstance(aliases, list) or not all(isinstance(alias, str) and alias for alias in aliases):
            errors.append(f"rule has invalid aliases: {term}")
        if "note" in rule and not isinstance(rule["note"], str):
            errors.append(f"rule note must be a string: {term}")
        if "task" in match:
            errors.append("generic task must not be an automatic replacement pattern")


def validate() -> list[str]:
    errors: list[str] = []
    glossary_source = GLOSSARY.read_text(encoding="utf-8")
    glossary = yaml.safe_load(glossary_source)

    if glossary.get("version") != 2:
        errors.append("docs/glossary.yml must use compact schema version 2")
    if len(glossary_source.splitlines()) > MAX_GLOSSARY_LINES:
        errors.append(f"docs/glossary.yml must not exceed {MAX_GLOSSARY_LINES} lines")
    for removed_key in ("all_extracted_terms", "extraction_inventory", "categories"):
        if removed_key in glossary:
            errors.append(f"docs/glossary.yml must not persist generated or legacy field: {removed_key}")
    if glossary.get("scope", {}).get("sources") != ["**/*.md"]:
        errors.append("docs/glossary.yml scope.sources must cover all Markdown files")
    if glossary.get("scope", {}).get("language") != "japanese":
        errors.append("docs/glossary.yml scope.language must limit terminology lint to Japanese Markdown")

    rules = glossary.get("rules")
    if not isinstance(rules, dict):
        errors.append("docs/glossary.yml rules must be a mapping")
        rules = {}
    for group_name in ("replacements", "preserved", "identifiers"):
        validate_rule_group(errors, group_name, rules.get(group_name))

    replacement_patterns = {
        pattern
        for term, rule in rules.get("replacements", {}).items()
        for pattern in rule_patterns(term, rule, default_to_term=True)
    }
    replacement_patterns.update(
        pattern
        for term, rule in rules.get("preserved", {}).items()
        if isinstance(rule.get("to"), str)
        for pattern in rule_patterns(term, rule, default_to_term=True)
    )
    identifier_patterns = {
        pattern
        for term, rule in rules.get("identifiers", {}).items()
        for pattern in rule_patterns(term, rule, default_to_term=False)
    }
    collisions = sorted(replacement_patterns & identifier_patterns)
    if collisions:
        errors.append(f"replacement and identifier patterns overlap: {', '.join(collisions[:10])}")

    project_terms = glossary.get("project_terms")
    if not isinstance(project_terms, dict) or not project_terms:
        errors.append("docs/glossary.yml project_terms must be a non-empty mapping")
    else:
        for term, project_term in project_terms.items():
            if not isinstance(project_term, dict) or not all(
                project_term.get(field) for field in ("description", "definition")
            ):
                errors.append(f"project term must have description and definition: {term}")
                continue
            definition_path = project_term["definition"].split("#", 1)[0]
            if not (GLOSSARY.parent / definition_path).is_file():
                errors.append(f"project term definition does not exist: {project_term['definition']}")

    expected_index = render_glossary_index(glossary)
    if not GLOSSARY_INDEX.is_file() or GLOSSARY_INDEX.read_text(encoding="utf-8") != expected_index:
        errors.append("docs/99-glossary-index.md is stale; regenerate it with validate-terminology.py --write")

    review_threshold = glossary.get("review", {}).get("frequent_prose_threshold")
    if not isinstance(review_threshold, int) or review_threshold < 1:
        errors.append("glossary review.frequent_prose_threshold must be a positive integer")
    else:
        unregistered = unregistered_frequent_terms(glossary)
        if unregistered:
            summary = ", ".join(f"{term}({count})" for term, count, _ in unregistered[:10])
            errors.append(f"frequent prose terms require glossary review: {summary}")

    config = json.loads(TEXTLINT_CONFIG.read_text(encoding="utf-8"))
    if config.get("rules", {}).get("kakesu-glossary") is not True:
        errors.append(".textlintrc.json must enable kakesu-glossary")
    package = json.loads(PACKAGE_JSON.read_text(encoding="utf-8"))
    lint_command = package.get("scripts", {}).get("lint:docs", "")
    if lint_command != "node scripts/lint-docs.mjs":
        errors.append("package.json lint:docs must use the safe Markdown file enumerator")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--write", action="store_true", help="regenerate only the derived glossary index")
    parser.add_argument("--report", action="store_true", help="report extracted terms without modifying files")
    args = parser.parse_args()

    glossary = yaml.safe_load(GLOSSARY.read_text(encoding="utf-8"))
    if args.write:
        write_generated_index(glossary)
    if args.report:
        counts, _, _ = extract_terms()
        unregistered = unregistered_frequent_terms(glossary)
        print(f"extracted {sum(counts.values())} tokens, {len(counts)} unique")
        for term, prose_count, total_count in unregistered:
            print(f"{term}\tprose={prose_count}\ttotal={total_count}")
        return 0

    errors = validate()
    if errors:
        for error in errors:
            print(f"terminology: {error}", file=sys.stderr)
        return 1
    print("terminology: compact curated glossary and generated index are synchronized")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
