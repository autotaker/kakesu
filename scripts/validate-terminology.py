#!/usr/bin/env python3
"""Extract and validate the English-term inventory used by the documentation glossary."""

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
TEXTLINT_CONFIG = ROOT / ".textlintrc.json"
PACKAGE_JSON = ROOT / "package.json"
TOKEN_RE = re.compile(r"[A-Za-z][A-Za-z0-9_./:+-]*")
FENCED_RE = re.compile(r"```.*?```|~~~.*?~~~", re.DOTALL)
INLINE_CODE_RE = re.compile(r"`[^`\n]+`")
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


def glossary_terms(glossary: dict) -> tuple[dict[str, dict], set[str]]:
    known: dict[str, dict] = {}
    identifier_values: set[str] = set()
    for category, definition in glossary["categories"].items():
        for term in definition.get("terms", []):
            if not isinstance(term, dict):
                continue
            canonical = dict(term)
            canonical["category"] = category
            for value in [term.get("formal_name"), *term.get("variants", []), *term.get("abbreviations", [])]:
                if isinstance(value, str):
                    known[value] = canonical
                    if category == "identifier":
                        identifier_values.add(value)
    return known, identifier_values


def classify(
    token: str,
    known: dict[str, dict],
    identifier_values: set[str],
    code_count: int,
    total_count: int,
) -> tuple[str, dict]:
    existing = known.get(token)
    if existing:
        category = existing["category"]
    elif token in identifier_values or re.search(r"[_/.:]", token) or any(char.isdigit() for char in token):
        category = "identifier"
    elif code_count == total_count and code_count > 0:
        category = "identifier"
    elif token.isupper() and len(token) > 1:
        category = "english_preferred"
    else:
        category = "japanese_translation"

    if existing:
        formal_name = existing["formal_name"]
        japanese = existing.get("japanese")
        abbreviations = list(existing.get("abbreviations", []))
        variants = list(existing.get("variants", []))
        description = existing.get("description", "")
    else:
        formal_name = token
        japanese = None if category != "japanese_translation" else "訳語を検討"
        abbreviations = []
        variants = []
        if category == "identifier":
            description = "本文から抽出したコード識別子。本文ではバッククォートで囲む。"
        elif category == "english_preferred":
            description = "本文から抽出した製品名、仕様名、固有名、または略語。英語表記を維持する。"
        else:
            description = "本文から抽出した一般技術語。自然な日本語の訳語を優先する。"

    return category, {
        "token": token,
        "category": category,
        "formal_name": formal_name,
        "japanese": japanese,
        "abbreviations": abbreviations,
        "variants": variants,
        "description": description,
    }


def inventory(glossary: dict) -> tuple[list[dict], Counter[str], dict[str, set[str]]]:
    counts, token_files, code_counts = extract_terms()
    known, identifier_values = glossary_terms(glossary)
    records = []
    for token in sorted(counts, key=lambda value: (value.lower(), value)):
        category, record = classify(token, known, identifier_values, code_counts[token], counts[token])
        record["count"] = counts[token]
        record["prose_count"] = counts[token] - code_counts[token]
        record["code_count"] = code_counts[token]
        record["source_count"] = len(token_files[token])
        records.append(record)
    return records, counts, token_files


def update_glossary() -> None:
    glossary = yaml.safe_load(GLOSSARY.read_text(encoding="utf-8"))
    records, counts, _ = inventory(glossary)
    glossary["extraction_inventory"] = {
        "source": "日本語を含む**/*.md",
        "raw_token_count": sum(counts.values()),
        "unique_token_count": len(counts),
        "classification": "all_extracted_termsの各レコードにcategoryを付ける。",
    }
    glossary["all_extracted_terms"] = records
    GLOSSARY.write_text(
        yaml.safe_dump(glossary, allow_unicode=True, sort_keys=False, width=120),
        encoding="utf-8",
    )
    print(f"updated {GLOSSARY}: {sum(counts.values())} tokens, {len(counts)} unique")


def validate() -> list[str]:
    errors: list[str] = []
    glossary = yaml.safe_load(GLOSSARY.read_text(encoding="utf-8"))
    records, counts, _ = inventory(glossary)
    expected_records = glossary.get("all_extracted_terms", [])
    actual_by_token = {record["token"]: record for record in records}
    expected_by_token = {record.get("token"): record for record in expected_records}

    if glossary.get("scope", {}).get("sources") != ["**/*.md"]:
        errors.append("docs/glossary.yml scope.sources must cover all Markdown files")
    if glossary.get("scope", {}).get("language") != "japanese":
        errors.append("docs/glossary.yml scope.language must limit terminology lint to Japanese Markdown")
    if len(expected_records) != len(expected_by_token):
        errors.append("docs/glossary.yml all_extracted_terms contains duplicate tokens")
    if set(actual_by_token) != set(expected_by_token):
        missing = sorted(set(actual_by_token) - set(expected_by_token))
        extra = sorted(set(expected_by_token) - set(actual_by_token))
        if missing:
            errors.append(f"glossary is missing extracted tokens: {', '.join(missing[:10])}")
        if extra:
            errors.append(f"glossary contains stale extracted tokens: {', '.join(extra[:10])}")
    for token, actual in actual_by_token.items():
        expected = expected_by_token.get(token)
        if not expected:
            continue
        if expected != actual:
            differing_fields = sorted(
                field
                for field in set(expected) | set(actual)
                if expected.get(field) != actual.get(field)
            )
            errors.append(
                f"glossary record drift for {token!r}: fields {', '.join(differing_fields)}"
            )
        if expected.get("category") not in {"japanese_translation", "english_preferred", "identifier"}:
            errors.append(f"invalid category for {token!r}: {expected.get('category')}")

    if glossary.get("extraction_inventory", {}).get("raw_token_count") != sum(counts.values()):
        errors.append("glossary raw_token_count is stale")
    if glossary.get("extraction_inventory", {}).get("unique_token_count") != len(counts):
        errors.append("glossary unique_token_count is stale")

    config = json.loads(TEXTLINT_CONFIG.read_text(encoding="utf-8"))
    if config.get("rules", {}).get("kakesu-glossary") is not True:
        errors.append(".textlintrc.json must enable kakesu-glossary")
    package = json.loads(PACKAGE_JSON.read_text(encoding="utf-8"))
    lint_command = package.get("scripts", {}).get("lint:docs", "")
    if lint_command != "node scripts/lint-docs.mjs":
        errors.append("package.json lint:docs must use the safe Markdown file enumerator")

    for category, definition in glossary["categories"].items():
        for term in definition.get("terms", []):
            lint = term.get("lint")
            if not isinstance(lint, dict):
                errors.append(f"missing lint policy for glossary term: {term.get('formal_name')}")
                continue
            if category == "japanese_translation":
                if lint.get("enabled") is not True or lint.get("mode") != "replace":
                    errors.append(f"Japanese translation is not lint-enabled: {term.get('formal_name')}")
                if lint.get("replacement") != term.get("japanese"):
                    errors.append(f"lint replacement does not match japanese term: {term.get('formal_name')}")
                if not lint.get("patterns"):
                    errors.append(f"Japanese translation has no lint pattern: {term.get('formal_name')}")
            if category == "identifier":
                if lint.get("enabled") is not True or lint.get("mode") != "identifier":
                    errors.append(f"identifier term is not lint-enabled: {term.get('formal_name')}")
                if set(lint.get("patterns", [])) != set(term.get("variants", [])):
                    errors.append(f"identifier lint patterns do not match variants: {term.get('formal_name')}")
            for pattern in lint.get("patterns", []):
                if pattern == "task":
                    errors.append("generic task must not be an automatic replacement pattern")

    return errors


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--write", action="store_true", help="regenerate all_extracted_terms in the glossary")
    args = parser.parse_args()
    if args.write:
        update_glossary()
        return 0
    errors = validate()
    if errors:
        for error in errors:
            print(f"terminology: {error}", file=sys.stderr)
        return 1
    print("terminology: glossary, extraction inventory, and direct textlint rule are synchronized")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
