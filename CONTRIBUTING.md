# Contributing to DoctorCode

DoctorCode accepts narrowly scoped contributions that improve deterministic code intelligence, evidence quality, portability, testing, documentation, or safety without weakening fail-closed behavior.

## Before changing code

Read `AGENTS.md`, `PROJECT_PROFILE.yaml`, `docs/SYSTEM_OVERVIEW.md`, `docs/CURRENT_STATE.md`, and `docs/ROADMAP.md` before editing source. Treat `.workflow/*.json` as governance authority and generated uppercase Markdown under `docs/` as projections that must be regenerated rather than hand-patched.

## Change scope

Keep each pull request focused on the smallest correct change. Do not bundle unrelated refactors, new abstractions, configuration, or features. Unsupported semantics must fail closed rather than being guessed.

## Branches and pull requests

Work on a branch and merge through a pull request. A green merge button is not acceptance: the exact candidate SHA must pass the repository's required evidence and cross-platform workflows before merge, and accepted phase changes are revalidated on `main` after merge.

## Validation

Run the tests and acceptance gates relevant to the change. Changes to analyzers must preserve existing corpus gates and include negative/adversarial coverage when the proof boundary changes. Do not claim PASS without executable evidence from the exact tested source.

## Documentation and governance

When behavior, contracts, workflow state, sequence evidence, or acceptance changes, update the authoritative inputs and regenerate governed Project Truth. Keep `docs/ROADMAP.md` synchronized with `.workflow/roadmap.json` and do not advance phases without accepted evidence.

## Security

Do not publish secrets, private user data, exploit details, or sensitive vulnerability information in issues or pull requests. Follow `SECURITY.md` for security reporting.

## License

Contributions are subject to the repository's Owner-approved license decision and tracked license file. Do not infer, replace, or broaden licensing terms through a contribution.
