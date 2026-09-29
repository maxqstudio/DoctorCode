# DoctorCode Roadmap to v1.0

This roadmap turns DoctorCode from a proven deterministic analyzer into a public, multi-language, multi-OS code-analysis product suitable for broad use.

The phase count is not a quality shortcut. A phase may split if RED/adversarial evidence exposes a safety defect that cannot be repaired cleanly inside the declared scope.

## Product acceptance principles

Every language or capability must follow the same evidence chain:

```text
declare exact scope
→ prove the capability absent or defect present
→ implement the smallest correct change
→ adversarial false-positive / false-negative hardening
→ real/repository-shaped validation where appropriate
→ Linux + Windows + macOS CI
→ exact branch acceptance
→ merge to main
→ exact-main revalidation
→ governance closure
```

DoctorCode must never imply that language recognition equals semantic support. Rule confidence and unsupported boundaries remain explicit.

## Roadmap

| Milestone | Goal | Exit criteria |
|---|---|---|
| M20 | JavaScript/TypeScript parser-grade foundation | Pure-Go parser authority integrated; JS/JSX/TS/TSX syntax validation is deterministic and fail-closed; existing M18/M19 rules remain regression-clean; AST-backed migration path proven without widening rule claims. |
| M21 | Language analyzer framework | Stable analyzer capability contract, parser/provider abstraction, rule metadata, evidence contract, deterministic IDs, parser availability and benchmark conventions reusable by later languages. |
| M22 | Rust semantic baseline | Conservative SECURITY/SIMPLIFY/LOGIC/DEADCODE evidence, BLOAT only when safe; synthetic + repository-shaped + multi-OS acceptance. |
| M23 | Java and Kotlin semantic baseline | Package/class/method scope, Maven/Gradle-aware boundaries, conservative rules and multi-OS real-project evidence. |
| M24 | C and C++ semantic baseline | Header/include/declaration-definition awareness, macro/conditional-compilation fail-closed behavior, no aggressive dead-code claims. |
| M25 | C#/.NET semantic baseline | Namespace/type/member/project/solution awareness with conservative async, nullable, credential and logic evidence. |
| M26 | Dart/Flutter semantic baseline | Imports, classes/functions/widgets, async constructs and generated-file handling with Flutter-shaped corpus evidence. |
| M27 | Scripting ecosystem | Bounded PHP, Ruby, PowerShell and Shell analyzers with language-specific parsers/rules; Python regression authority preserved. |
| M28 | Swift, Zig and MQL5 specialist support | Conservative semantic baselines for specialist languages after the common analyzer framework is stable. |
| M29 | Repository-scale intelligence | Cross-file symbol/import/dependency graph, generated/test/reference awareness, monorepo support, incremental scan/cache and deterministic repository-level evidence. |
| M30 | Developer integrations | Stable JSON/SARIF, GitHub/GitLab annotations, pre-commit, editor integration boundary, MCP/Skill protocol stability and documented exit codes/schema. |
| M31 | Production hardening | Large-repo performance/memory/cancellation, fuzzing, malformed/untrusted repo safety, path/symlink/archive protections, update/uninstall, dependency/supply-chain and backwards-compatibility gates. |
| M32 | v1.0 GA qualification | Broad multi-language real-world corpus, false-positive budgets, clean-machine install tests, documentation/tutorials, support/reporting workflow, release-signing strategy, RC acceptance and v1.0.0 publication. |

## v1.0 target support tiers

DoctorCode v1.0 should distinguish support levels rather than claim equal depth for every language.

- **Production-grade:** Go, Python, JavaScript/TypeScript.
- **Supported semantic baseline:** Rust, Java/Kotlin, C/C++, C#, Dart/Flutter.
- **Bounded specialist support:** PHP, Ruby, PowerShell, Shell, Swift, Zig, MQL5.
- Any additional recognized language remains explicitly **recognition-only** until it has its own accepted semantic milestone.

## v1.0 minimum product gates

- Windows, Linux and macOS permanent acceptance lanes pass on one exact release candidate and on the exact merged release SHA.
- Clean install, update and uninstall paths are documented and tested.
- Deterministic reruns produce stable finding identities for unchanged source.
- Malformed or adversarial input fails closed without executing analyzed source.
- Large repositories have bounded memory/runtime/cancellation behavior.
- Every rule exposes category, confidence, evidence and safe-autofix status.
- No destructive DEADCODE/BLOAT recommendation is presented as proven unless the language-specific evidence contract supports that claim.
- Public schemas, CLI behavior and integration contracts are versioned and backwards-compatibility tested.
- Release provenance/integrity claims remain precise; checksums are not described as code signing.

## Current position

- M19: **MAIN_ACCEPTED**.
- M20: **active** — JavaScript/TypeScript parser-grade foundation.
