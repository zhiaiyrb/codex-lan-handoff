# Handoff document schema

Create UTF-8 JSON with exactly these fields. Unknown fields are rejected.

```json
{
  "schema_version": 1,
  "goal": "Concrete objective",
  "success_criteria": ["Observable acceptance condition"],
  "decisions": ["Decision already made"],
  "constraints": ["Constraint that must remain true"],
  "completed": ["Work already completed"],
  "current_state": "Concise description of the partial state",
  "repositories": [
    {
      "path": "/absolute/or/platform-native/path",
      "branch": "branch-name",
      "commit": "full-or-short-commit",
      "git_status": "Human-readable status summary; no diff content"
    }
  ],
  "validation": ["Command or check and its result"],
  "blockers": ["Known blocker"],
  "risks": ["Known risk"],
  "next_steps": ["Next concrete action"]
}
```

Required non-empty fields are `goal`, `success_criteria`, `current_state`, and `next_steps`. Arrays not relevant to a task may be empty or omitted only when tagged `omitempty` in the example (`repositories`, `blockers`, and `risks`). Keep the document below 512 KiB.
