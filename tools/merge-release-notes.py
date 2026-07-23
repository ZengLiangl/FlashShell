#!/usr/bin/env python3
"""Merge base GitHub Release notes with newly generated changelog by section type.

Used by .github/workflows/release.yml when bump-tag.sh --base is set.
Avoids embedding a large Python heredoc in YAML (breaks indentation).
"""

from __future__ import annotations

import argparse
import re
from collections import OrderedDict
from pathlib import Path

SECTION_ORDER = [
    ("## ✨ 新功能", ("新功能", "feat")),
    ("## 🐛 问题修复", ("问题修复", "fix", "bug")),
    ("## ⚡ 性能优化", ("性能优化", "perf")),
    ("## ♻️ 重构", ("重构", "refactor")),
    ("## 🌐 国际化", ("国际化", "i18n")),
    ("## 🔧 其他变更", ("其他变更", "chore", "其他")),
]

SKIP_SECTION = re.compile(r"本次更新|完整变更", re.I)
COMPARE_RE = re.compile(
    r"\*\*完整变更\*\*:\s*\[([^\]]+)\]\(([^)]+)\)",
    re.I,
)
COMPARE_RANGE_RE = re.compile(
    r"^(v[^.\s][^\s.]*(?:\.[^\s.]+)*)\.\.\.(v[^.\s][^\s.]*(?:\.[^\s.]+)*)$"
)


def normalize_heading(line: str) -> str:
    s = line.strip()
    s = re.sub(r"^#+\s*", "", s)
    s = re.sub(r"[\U0001F300-\U0001FAFF\u2600-\u27BF]", "", s)
    return s.strip().lower()


def match_section_key(heading_line: str):
    norm = normalize_heading(heading_line)
    if SKIP_SECTION.search(norm):
        return None
    for title, aliases in SECTION_ORDER:
        for a in aliases:
            if a.lower() in norm:
                return title
    if heading_line.strip().startswith("##"):
        return heading_line.strip()
    return None


def parse_sections(md: str) -> OrderedDict:
    """Parse markdown into {section_title: [item_blocks]}.

    Skips headings like「本次更新」; nested category sections under it still merge.
    """
    sections: OrderedDict[str, list[str]] = OrderedDict()
    current = None
    buf: list[str] = []

    def flush_item() -> None:
        nonlocal buf
        if not current:
            buf = []
            return
        text = "\n".join(buf).rstrip()
        buf = []
        if not text.strip():
            return
        sections.setdefault(current, [])
        if text not in sections[current]:
            sections[current].append(text)

    for raw in (md or "").replace("\r\n", "\n").replace("\r", "\n").split("\n"):
        line = raw.rstrip()
        stripped = line.strip()
        if stripped.startswith("## "):
            flush_item()
            current = match_section_key(stripped)
            continue
        if stripped == "---":
            flush_item()
            current = None
            continue
        if COMPARE_RE.search(stripped):
            flush_item()
            current = None
            continue
        if current is None:
            continue
        # Top-level "- " starts a new item; indented bullets are children.
        if stripped.startswith("- ") and (not line or not line[0].isspace()):
            flush_item()
            buf = [line]
        elif buf:
            if not stripped:
                continue
            buf.append(line)
        elif stripped:
            buf = [f"- {stripped}"]
    flush_item()
    return sections


def extract_compare_start(md: str) -> str:
    m = COMPARE_RE.search(md or "")
    if not m:
        return ""
    label = m.group(1).strip()
    rm = COMPARE_RANGE_RE.match(label)
    if rm:
        return rm.group(1)
    if "..." in label:
        return label.split("...", 1)[0].strip()
    return ""


def merge_sections(new_secs: OrderedDict, base_secs: OrderedDict) -> OrderedDict:
    """New commits first, then historical items; dedupe by first line."""
    out: OrderedDict[str, list[str]] = OrderedDict()
    keys = [title for title, _ in SECTION_ORDER]
    for k in list(new_secs.keys()) + list(base_secs.keys()):
        if k not in keys:
            keys.append(k)
    for k in keys:
        items: list[str] = []
        seen: set[str] = set()
        for src in (new_secs, base_secs):
            for item in src.get(k, []):
                head = item.split("\n", 1)[0].strip()
                if head in seen:
                    continue
                seen.add(head)
                items.append(item)
        if items:
            out[k] = items
    return out


def merge_release_notes(
    *,
    base_body: str,
    new_body: str,
    tag: str,
    repo_url: str,
    prev_tag: str = "",
) -> str:
    merged = merge_sections(parse_sections(new_body), parse_sections(base_body))
    lines: list[str] = []
    for title, _ in SECTION_ORDER:
        items = merged.pop(title, None)
        if not items:
            continue
        lines.append(title)
        lines.append("")
        for item in items:
            lines.append(item.rstrip())
            lines.append("")
    for title, items in merged.items():
        lines.append(title)
        lines.append("")
        for item in items:
            lines.append(item.rstrip())
            lines.append("")

    compare_start = extract_compare_start(base_body) or (prev_tag or "").strip()
    if compare_start and compare_start != tag:
        lines.append("---")
        lines.append("")
        lines.append(
            f"**完整变更**: [{compare_start}...{tag}]({repo_url}/compare/{compare_start}...{tag})"
        )
        lines.append("")
    return "\n".join(lines).rstrip() + "\n"


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--base", required=True, help="Path to base release body markdown")
    p.add_argument("--new", required=True, help="Path to newly generated changelog")
    p.add_argument("--out", required=True, help="Output merged changelog path")
    p.add_argument("--tag", required=True, help="Current release tag, e.g. v1.1.10")
    p.add_argument("--repo-url", required=True, help="Repo URL for compare links")
    p.add_argument("--prev-tag", default="", help="Fallback compare start tag")
    args = p.parse_args()

    base_body = Path(args.base).read_text(encoding="utf-8")
    new_body = Path(args.new).read_text(encoding="utf-8")
    text = merge_release_notes(
        base_body=base_body,
        new_body=new_body,
        tag=args.tag,
        repo_url=args.repo_url,
        prev_tag=args.prev_tag,
    )
    out = Path(args.out)
    out.write_text(text, encoding="utf-8", newline="\n")
    print(f"merged sections -> {out} ({len(text)} bytes)")


if __name__ == "__main__":
    main()
