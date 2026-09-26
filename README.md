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

**M05 pinned public-repository validation is accepted on the development branch.** Current semantic coverage remains deliberately narrow; labeled corpus metrics and unlabeled public-repository compatibility evidence are kept separate.

| Language | Recognition | Toolchain detection | Built-in semantic rules |
|---|---:|---:|---:|
| Go | Yes | Yes | **Yes — narrow rules, three labeled regression gates + pinned public-repository validation** |
| Python | Yes | Yes | Not yet |
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
| DEADCODE | unexported package-level function with zero lexical references | `HIGH`, never `PROVEN_UNUSED` |
| LOGIC | repeated side-effect-free condition in one if/else-if chain | `HIGH` |
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

## Cross-platform acceptance

GitHub Actions is the CI/runtime acceptance authority; the Owner PC is not used.

Blocking runners:

- `ubuntu-latest`
- `windows-latest`
- `macos-latest`

Core CI runs unit/corpus tests, the M02 baseline, M03 adversarial, and M04 repository-shaped benchmark gates, `go vet`, CLI build, scan, toolchain detection, audit smoke, and bounded-packet smoke. A separate M05 matrix runs pinned public-repository validation on the same Linux, Windows, and macOS runner set.

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
