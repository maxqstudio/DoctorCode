# AGENTS.md

## Purpose

This file is the repository-root operating contract for coding agents working on this project.

It is source-authored and MUST remain at repository root. It is not generated Project Truth and must not be moved under `docs/`.

## Required startup order

1. Read this `AGENTS.md`.
2. Read `PROJECT_PROFILE.yaml`.
3. Read `docs/SYSTEM_OVERVIEW.md`.
4. Read `docs/CURRENT_STATE.md`.
5. Read `docs/ROADMAP.md` and verify the current phase matches project state.
6. Read `docs/PROJECT_MANIFEST.md` and only the profile-required contracts relevant to the task.
7. Open exact source and tests only after authority and scope are clear.

## Operating rules

- Follow explicit Owner/project instructions and repository governance.
- Treat `.workflow/*.json` as machine-readable governance authority and generated uppercase Markdown under `docs/` as deterministic projections.
- Do not manually patch generated Project Truth Markdown.
- Use the smallest correct change; do not add unrelated refactors or abstractions.
- Never claim PASS without executable evidence from the required acceptance layer.
- Preserve exact tested-source lineage and fail closed on unknown authority, stale evidence, or ambiguous scope.
- Use `develop` and `verify` for iteration when safe; only `finalize` may provide final acceptance authority.

## Documentation and sequence

When source, contracts, workflow, state, or acceptance changes, update the authoritative inputs and regenerate governed documentation/evidence. If sequence governance is required, keep the current sequence session synchronized with the current source digest.

## Handoff

Leave the repository so the next human or agent can identify the current phase, exact authority, blockers, proven/not-proven facts, legal next action, and required tests without reconstructing intent from chat history.
