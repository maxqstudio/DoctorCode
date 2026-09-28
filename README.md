# DoctorCode

DoctorCode is a deterministic, LLM-optional code intelligence tool that turns repositories into small, evidence-backed findings for humans and small local models.

It targets five code-maintenance problems: **BLOAT**, **SECURITY**, **SIMPLIFY**, **LOGIC**, and **DEADCODE**.

## Why DoctorCode

```text
repository
  -> deterministic analyzers
  -> evidence-backed findings
  -> bounded evidence packet
  -> human or optional LLM
  -> independent verification
```

The LLM is not the source of truth.

## Current status

**M16 Thin MCP Adapter is accepted on protected `main` at `2e76c9e635a8a68507fea9d44438f74e96d8e754`. M17 Release and Install Hardening is the next authorized milestone.** Go and Python semantic coverage remain deliberately narrow; transport adapters do not enlarge core proof boundaries.

| Language | Recognition | Toolchain detection | Built-in semantic rules |
|---|---:|---:|---:|
| Go | Yes | Yes | **Yes — narrow rules, three synthetic regression gates + pinned compatibility + bounded real-world labels** |
| Python | Yes | Yes | **Yes — narrow stdlib-AST rules for all five categories; adversarial + repository-shaped gated; host Python required** |
| JavaScript / TypeScript | Yes | Yes | Not yet |
| Rust | Yes | Yes | Not yet |
| Java / Kotlin | Yes | Yes | Not yet |
| C / C++ | Yes | Yes | Not yet |
| C# | Yes | Yes | Not yet |
| Dart | Yes | Yes | Not yet |
| PHP / Ruby | Yes | Yes | Not yet |
| PowerShell / Shell | Yes | Yes | Not yet |
| Swift / Zig | Yes | Yes | Not yet |
| MQL5 | Yes | Yes when MetaEditor is present | Not yet |

**Recognition does not mean semantic-analysis support.** Capabilities are kept separate to avoid false support claims.

### Go detector rules

| Category | M01 rule | Confidence boundary |
|---|---|---|
| DEADCODE | unexported hand-maintained package-level function with zero lexical references | `HIGH`, never `PROVEN_UNUSED`; generated declarations excluded |
| LOGIC | repeated side-effect-free condition in one if/else-if chain with binding-aware identifier comparison | `HIGH` |
| SIMPLIFY | opposite boolean-return branches | `PROVEN` for that structural rewrite |
| SECURITY | hardcoded literal assigned to a credential-like identifier | `SUSPICIOUS`, literal always redacted |
| BLOAT | one-call unexported pass-through wrapper with one lexical reference | `SUSPICIOUS` |

No current rule performs automatic fixing or deletion.

### M02 precision regression gate

DoctorCode now includes a labeled Go benchmark that is executed as a blocking CI step:

```bash
./doctorcode benchmark internal/benchmark/testdata/manifest.json --json
```

The accepted M02 corpus contains **9 curated cases**. On the accepted candidate it reports **5 TP, 0 FP, and 0 FN** across the five current rules. That is a regression result for this exact corpus only; it is **not** a claim of 100% real-world precision or recall.

The benchmark fails when an expected finding is missing, when an unexpected finding appears, or when the configured corpus threshold is missed.

### M03 adversarial regression gate

M03 adds a second independent 11-case corpus:

```bash
./doctorcode benchmark internal/benchmark/testdata/m03-adversarial.json --json
```

Before repair, this gate exposed **4 false negatives** that the M02 corpus did not catch: local-name shadowing, a selector/method-name collision, and a duplicate pure condition hidden by redundant parentheses. After the targeted repairs, the M03 corpus reports **6 TP, 0 FP, and 0 FN**, while the original M02 corpus remains passing.

These results remain bounded to the tracked corpora. They do not establish general real-world precision or recall.

### M04 repository-shaped regression gate

M04 adds 12 mini-repositories that exercise interactions a single-file corpus cannot cover:

```bash
./doctorcode benchmark internal/benchmark/testdata/m04-repository-shaped.json --json
```

The corpus covers cross-file references, separate packages with colliding names, same-package and external tests, build-tag-visible references, assembly fail-closed behavior, generated-file references, and nested package findings. The accepted M04 result is **7 TP, 0 FP, and 0 FN**.

The first M04 attempt intentionally remained blocking when one fixture was mislabeled: the detector correctly found a zero-reference function inside the build-tag fixture. The fixture was corrected; the detector was not weakened. **M04 changes evaluation evidence only and makes no Go detector implementation change.**

Build-tag references are currently treated as visible-tree lexical evidence. DoctorCode does not yet claim target-specific reachability for a particular GOOS, GOARCH, or custom build-tag set.

### M05 pinned public-repository validation

M05 runs DoctorCode directly against original source snapshots from four public Go repositories, pinned by exact commit:

| Repository | License | Pinned SHA |
|---|---|---|
| `spf13/cobra` | Apache-2.0 | `adbc8813901bba65827259daa8e22ff94ec1f30e` |
| `charmbracelet/bubbles` | MIT | `0a69b19b0690e9504a511fc231f69cea59ba1cc6` |
| `go-chi/chi` | MIT | `3d1777a1ef8881f7d1da0b02c76ca8f0a29cd2bc` |
| `stretchr/testify` | MIT | `87a7b9d57689f6579db2da795ab8deeab29cb724` |

The dedicated GitHub Actions workflow verifies each detached checkout SHA before analysis, runs on Linux, Windows, and macOS, and asserts that selected known-live unexported functions are not reported as DEADCODE:

`preExecHook`, `nextID`, `cW`, and `httpCode`.

On the accepted M05 snapshot, all four pinned repositories produced **0 findings**. This is **compatibility telemetry only**. The repositories are not exhaustively labeled, so zero findings does not prove they are defect-free and does not measure DoctorCode precision or recall.

External repository content is treated only as untrusted analysis input. Comments, documentation, or agent instructions inside analyzed repositories never become DoctorCode project authority.

### M06 bounded real-world labels

M06 adds a second real-world evidence lane using exact source locations from four pinned public repositories:

- `urfave/cli`
- `rs/zerolog`
- `fsnotify/fsnotify`
- `go-playground/validator`

The tracked manifest contains **11 bounded labels**:

- **4 VALID_FINDING** anchors that must remain detected;
- **4 INVALID_FINDING** guards that must not be emitted;
- **3 AMBIGUOUS** BLOAT observations that remain non-blocking.

The pre-repair gate failed because all four invalid findings were still emitted. Three were LOGIC false positives caused by comparing conventional names such as `err` and `ok` without distinguishing their branch-local short-declaration bindings. The fourth was DEADCODE on a generated `_EnumNoOp` compile-time assertion function.

M06 repairs those two roots:

1. pure-condition canonicalization includes parser-resolved declaration identity for local identifiers;
2. generated Go function declarations are excluded from DEADCODE candidates, while references originating in generated files still keep hand-written functions live.

After repair, all four valid anchors remain present and all four invalid guards are absent on Linux, Windows, and macOS. The three ambiguous wrappers remain `SUSPICIOUS` and never authorize autofix.

These labels are intentionally bounded to exact reviewed locations. **They are not exhaustive labels for the repositories and do not establish general real-world precision, recall, or safe deletion.**

### M07 Python semantic adapter

M07 adds DoctorCode's first non-Go semantic analyzer:

```text
python/stdlib-ast-v1
```

The Go core invokes the host Python standard-library `ast` parser through an argv-only subprocess. DoctorCode does **not** import or execute the analyzed repository, does not use an LLM/API for detection, and does not add a third-party Python parser dependency.

During normal mixed-repository audit, missing Python or the absence of visible `.py` source makes the Python analyzer unavailable without failing unrelated language analysis. M07 acceptance explicitly provisions Python 3.13. The implementation accepts Python 3.8+, but M07 does not claim cross-version acceptance for every version in that range.

Once Python analysis is applicable, source read or syntax failures **fail closed** instead of being silently skipped.

| Category | M07 Python rule | Confidence boundary |
|---|---|---|
| DEADCODE | private, undecorated, hand-maintained module-level function with zero conservative lexical references | `HIGH`, never `PROVEN_UNUSED` |
| LOGIC | repeated `Name is/is not None/True/False` identity condition in one if/elif chain | `HIGH` |
| SIMPLIFY | opposite boolean-return branches | `PROVEN` for the narrow structural rewrite |
| SECURITY | hardcoded literal assigned to a credential-like identifier | `SUSPICIOUS`; literal always redacted |
| BLOAT | private one-call pass-through wrapper with one conservative lexical reference | `SUSPICIOUS` |

The M07 labeled Python corpus contains **6 cases** and reports **5 TP, 0 FP, and 0 FN** on the accepted candidate: one true positive for each rule. These are corpus-scoped regression metrics only.

A separate three-OS public-source gate checks exact-SHA snapshots of `pallets/click`, `encode/httpx`, `psf/requests`, and `pallets/flask`. The repositories must parse and analyze successfully, checkout provenance is verified, and Flask `src/flask/cli.py:691` `_path_is_ancestor` is tracked as one bounded `PY-DEADCODE-PRIVATE-ZERO-REF` anchor.

That Flask anchor proves the narrow zero-lexical-reference observation only. It does **not** prove safe deletion. Dynamic imports, reflection, plugin registration, external callers, and other runtime behaviors remain outside M07's proof boundary.

Python semantic analysis currently covers `.py` source. Recognition of `.pyi` files does **not** mean `.pyi` semantic-analysis support.

### M08 Python adversarial regression gate

M08 adds a second independent Python corpus with 10 deliberately adversarial cases:

```bash
./doctorcode benchmark internal/benchmark/testdata/m08-python-adversarial.json --analyzer=python --json
```

The first M08 run failed with **5 false positives and 2 false negatives** while the accepted M07 corpus still passed. The failures exposed four concrete roots:

1. Python identity constants were checked with equality-compatible membership, so integer `1` could be confused with `True`;
2. DEADCODE ignored intentional string-based references through `__all__`, `getattr`, and `globals()`;
3. private-function reference counts were global by identifier name, so same-named functions in unrelated modules could hide one another;
4. obvious `not_secret` / `not-secret` placeholder literals still triggered the narrow SECURITY rule.

M08 repairs those roots without adding new detector categories. Python private-function references are now candidate/module-aware for recognized imports and module attributes, while selected dynamic/export patterns contribute conservative reference evidence.

The accepted M08 corpus reports **2 TP, 0 FP, and 0 FN**. Only two cases are positive anchors; the other eight are adversarial negative guards. The accepted M07 corpus remains **5 TP, 0 FP, and 0 FN**, and exact-SHA click/httpx/requests/Flask validation remains passing on Linux, Windows, and macOS.

These metrics are corpus-scoped. Recognized dynamic reference handling is intentionally incomplete and does not resolve arbitrary reflection, `eval`/`exec`, plugin loaders, or external callers. Zero conservative references still never means safe deletion.

### M09 Python repository-shaped regression gate

M09 adds **14 mini-repositories** that exercise Python package/import topology which single-file and adversarial snippets do not cover:

```bash
./doctorcode benchmark internal/benchmark/testdata/m09-python-repository-shaped.json --analyzer=python --json
```

The corpus includes conventional `src/` layouts, package `__init__.py` re-exports, relative and aliased imports, dotted module access, same-named functions in separate packages, test-only references, generated references, namespace packages, `TYPE_CHECKING`, and conditional imports.

The final pre-repair M09 run reported **2 TP, 5 FP, and 3 FN**. The repair introduced four bounded topology changes:

1. files below a leading `src/` are indexed under both repository-relative and conventional package module names;
2. direct module-level `from ... import ...` re-exports propagate to a bounded fixed point so package facades resolve to the original candidate;
3. import/export liveness is tracked separately from executable usage, so DEADCODE can stay conservative without inflating the BLOAT one-use count;
4. dotted attributes are resolved through explicit import roots, while unresolved name fallback is allowed only when exactly one candidate has that private name.

The accepted M09 corpus reports **5 TP, 0 FP, and 0 FN**. M07 and M08 remain passing.

These results remain fixture-scoped. DoctorCode does not infer arbitrary package roots, `sys.path` mutation, star-import semantics, import hooks, runtime symbol assignment, or a whole-program Python import graph. Zero conservative references still never authorizes deletion.

### M10 bounded real-world Python labels

M10 adds a dedicated exact-SHA label gate over selected Click and Flask source locations. The tracked set contains **2 VALID_FINDING anchors** and **4 INVALID_FINDING guards**. The valid anchors are the Flask private helpers `src/flask/app.py:74::_make_timedelta` and `src/flask/cli.py:691::_path_is_ancestor`, each validating only the narrow `PY-DEADCODE-PRIVATE-ZERO-REF` claim. The invalid anchors are private helpers under pinned Click/Flask test trees that must remain excluded from production DEADCODE candidates.

The accepted M10 label run reports **valid=2, invalid=4** on Linux, Windows, and macOS. This is a bounded regression gate only: it is not an exhaustive review of either repository, does not establish general Python precision or recall, and does not prove that either valid finding is safe to delete. External repositories remain untrusted read-only analysis input and their code is not executed by the M10 workflow.

## Quick start

Requires Go 1.24+ to build from source.

```bash
git clone https://github.com/maxqstudio/DoctorCode.git
cd DoctorCode
go build -o doctorcode ./cmd/doctorcode
```

Repository/language inventory:

```bash
./doctorcode scan .
```

Host compiler/runtime capability inventory:

```bash
./doctorcode toolchains .
```

Run deterministic findings:

```bash
./doctorcode audit .
./doctorcode audit . --json --max-findings=20
```

### M11 finding-specific context compiler

M11 adds exact finding-specific retrieval without adding a second analysis authority. First obtain a deterministic finding ID from `doctorcode audit`, then request a bounded packet for that exact current finding:

```bash
./doctorcode audit . --json --max-findings=20
./doctorcode context <finding-id> . --json --max-bytes=4096
```

`context` re-audits the current source state and fails if the supplied ID is absent, so a stale ID is never silently redirected to a different finding. It reuses the same byte-bounded packet builder as `next`; it does not invoke an LLM, mutate code, or change detector confidence.

M11 also hardens excerpt reads: repository root and selected source path are resolved through symlinks and the resolved target must remain inside the resolved repository root. Blocking tests cover both `../` traversal and a symlink pointing outside the repository.

This milestone is deliberately a **single-finding context foundation**. It does not yet infer transitive dependencies, collect related tests/symbols across multiple files, migrate stale IDs, or claim exact tokenizer counts.
### M12 bounded Go related context

M12 upgrades evidence packets to schema version 2 for eligible Go findings. When the selected finding is inside a top-level non-method Go function, DoctorCode reuses that source AST binding and adds deterministic references from the same package. Same-package test references are labeled separately as test_reference; local shadow bindings and external test packages are not guessed as related.

Related excerpts are optional context after the primary selected source. DoctorCode emits at most eight related excerpts, keeps related_total for discovered locations, and marks the packet truncated when the byte budget or cap omits context. The encoded packet still cannot exceed --max-bytes.

The repository boundary is checked before related discovery. A parent-traversal path is rejected before an outside Go file can be parsed, and every emitted related excerpt is independently repository-contained.

M12 is intentionally Go-only related-context enrichment. Python and other languages retain the M11 primary-source-only packet until an explicit parity milestone.

Give a small LLM one highest-priority bounded evidence packet instead of the repository:

```bash
./doctorcode next . --json --max-bytes=4096
```

The byte cap is deliberate: exact token counts vary by model/tokenizer, while a byte budget is deterministic.

## Safety boundaries

- `PROVEN` means the narrow structural claim is mechanically established.
- `HIGH` means strong evidence exists but the proven boundary is incomplete.
- `SUSPICIOUS` means review is warranted; it is not a proven defect.
- `UNKNOWN` means evidence is insufficient.
- Zero references never authorizes deletion by itself.
- Go DEADCODE fails closed for visible assembly/cgo/linkage escape hatches.
- Go repository analysis skips source symlink entries rather than following them outside the selected tree.
- Benchmark case roots must resolve inside the benchmark manifest directory.
- Security findings never include the detected credential literal in evidence.
- Security `next` packets omit source excerpts by default.
- Python semantic analysis fails closed when an applicable `.py` file cannot be read or parsed.
- Python DEADCODE is conservative lexical evidence only; dynamic imports, reflection, plugins, and external callers can exist outside the visible tree.
- Python `.pyi` files may be recognized by the scanner but are not semantically analyzed by the current Python adapter.

## Cross-platform acceptance

GitHub Actions is the CI/runtime acceptance authority; the Owner PC is not used. Every FINAL_ACCEPTED phase is promoted to `main`, and the same accepted lanes must pass again on the exact `main` merge SHA before the next phase starts.

Blocking runners:

- `ubuntu-latest`
- `windows-latest`
- `macos-latest`

Core CI runs unit/corpus tests, the M02 baseline, M03 adversarial, M04 repository-shaped, M07 Python semantic, M08 Python adversarial, and M09 Python repository-shaped benchmark gates, `go vet`, CLI build, scan, toolchain detection, audit smoke, bounded next-packet smoke, and the M12 exact finding-specific related-context smoke. Separate Go compatibility, bounded Go real-world label, Python public-source, and bounded M10 Python label workflows also run on Linux, Windows, and macOS for `main` and every `work/**` phase branch.

## Design principles

- Deterministic analysis first; LLM reasoning is optional.
- Small evidence packets instead of whole-repository prompts.
- Precision over recall for destructive recommendations.
- Never claim safe deletion from zero-reference count alone.
- Language/compiler support is adapter-based, not inferred from extensions.
- Missing optional toolchains are reported as unavailable.
- Project governance follows `maxqstudio/Skill_Workflow`.

## Documentation

Canonical generated docs live under `docs/`:

- [System overview](docs/SYSTEM_OVERVIEW.md)
- [Current state](docs/CURRENT_STATE.md)
- [Project manifest](docs/PROJECT_MANIFEST.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Acceptance matrix](docs/TEST_ACCEPTANCE_MATRIX.md)
- [Project truth synchronization](docs/PROJECT_TRUTH_SYNC.md)

The upstream semantic/governance source is `.workflow/*.json`; generated Markdown is not edited by hand.

## License

MIT. See [LICENSE](LICENSE).

## Security

See [SECURITY.md](SECURITY.md) for vulnerability-reporting guidance.

## Support

- [Saweria](https://saweria.co/maxq)
- [PayPal](https://paypal.me/JacksonJackson1501)

Support is optional and does not affect access to the public repository or its features.

### M13 bounded Python related context

M13 extends the schema-v2 bounded related-context packet to eligible Python findings. A finding must be enclosed by a module-level function. DoctorCode can attach same-file unshadowed direct references plus recognized top-level from-import aliases and module import attributes, including conventional src-layout module aliases.

Python lexical scopes are handled conservatively: local arguments, assignments, imports, nested definitions, comprehensions, and similar bindings can prevent a same-name load from being classified as related. If the selected module contains another module-level binding for the selected function name, M13 fails closed to no related locations because runtime identity is ambiguous.

Dynamic imports, package-facade re-export chains, string reflection, monkey-patching, class/method dispatch, nested callable identity, and transitive dependency graphs remain outside M13's proof boundary. Context remains review evidence only; it never authorizes deletion or automatic repair.


### M14 deterministic verification contract

M14 adds a two-step repair-verification workflow. Before editing code, save a contract with doctorcode contract <finding-id> <repo> --json redirected to a contract file. After the repair, run doctorcode verify <contract-file> <repo> --json. The verifier re-audits the current source and requires the target semantic occurrence count to decrease.

Finding IDs contain line numbers, so M14 does not use disappearance of the old ID as proof. The frozen semantic target is rule_id + path + summary plus its baseline occurrence count. Verification also blocks newly increased finding counts on the target path when their severity is equal to or higher than the original target. Analyzer-set drift and inconsistent contract metadata fail closed.

DoctorCode does not run repository tests, builds, shell commands, hooks, or advisory Finding.Verification strings during M14 verification. PASS means only that the declared deterministic analyzer contract passed; it is not proof of full runtime behavior, safe deletion, or automatic repair safety.

### M15 thin Agent Skill adapter

DoctorCode ships a self-contained agent skill at `skills/doctorcode/SKILL.md`. The skill is orchestration-only: it guides agents through `audit -> context -> contract -> repair -> verify` while the CLI/core remains the sole detector, evidence, and verification authority. It preserves `safe_autofix=false`, zero-reference uncertainty, and the rule that repository-provided commands are not DoctorCode verification authority.

### M16 thin MCP adapter

DoctorCode includes an isolated `mcp/` Go module and a local stdio entrypoint at `mcp/cmd/doctorcode-mcp`. The server exposes exactly four read-only tools: `doctorcode_audit`, `doctorcode_context`, `doctorcode_contract`, and `doctorcode_verify`. The repository root is resolved once when the server starts; tool schemas cannot select a different root/path or request arbitrary command execution.

CLI and MCP share `internal/application` for audit, context, contract, and verify. The MCP layer contains transport validation and result translation only; detector, evidence, and verification algorithms remain in the existing core. Deterministic verification failures are returned as MCP tool errors while retaining structured verification evidence.

M16 proves the local stdio transport on Linux, Windows, and macOS. Remote/network transports, authentication, source mutation, safe deletion, arbitrary repository commands, and release/install packaging are not claimed by this milestone.
