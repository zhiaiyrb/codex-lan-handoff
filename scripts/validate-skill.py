#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(__file__).resolve().parents[1]
skill = root / "skills" / "codex-handoff"
text = (skill / "SKILL.md").read_text(encoding="utf-8")
errors = []
if not re.match(r"^---\nname: codex-handoff\ndescription: .+\n---\n", text):
    errors.append("SKILL.md frontmatter must contain only name and description")
for rel in [
    "agents/openai.yaml",
    "references/handoff-schema.md",
    "scripts/install.ps1",
    "scripts/install.sh",
]:
    if not (skill / rel).is_file():
        errors.append(f"missing {rel}")
if "TODO" in text:
    errors.append("SKILL.md contains TODO")
yaml = (skill / "agents" / "openai.yaml").read_text(encoding="utf-8")
if "$codex-handoff" not in yaml:
    errors.append("default_prompt must mention $codex-handoff")
if errors:
    print("\n".join(errors), file=sys.stderr)
    raise SystemExit(1)
print("skill validation passed")
