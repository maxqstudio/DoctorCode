import json
import re
from pathlib import Path

PRODUCT_SHA = "776bb6467fcc55f4c7192ef06aa07c5469102520"
BASE_SHA = "79ff0ac95b3274e67b70f5dc2cda79360ed2d00a"
SKILL_SHA = "9e22feddb8f94e8c0f1af6a33e14b64de5068f8f"

def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))

def save(path, data):
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")

state = load(".workflow/state.json")
state["phase"] = "M14_DETERMINISTIC_VERIFICATION_CONTRACT"
state["status"] = "M14_VALIDATING"
state["working_branch"] = "work/m14-deterministic-verification"
state["blockers"] = []
state["proven"] = [
    "M13 is MAIN_ACCEPTED with product baseline main@3fccbd4398f2a12bd5214be9ed23ee18f46c679c and governance-normalized starting main@" + BASE_SHA + ".",
    "Skill_Workflow main remains " + SKILL_SHA + ", matching DoctorCode's pinned authority.",
    "M14 initial RED Core CI run 36328542441 at d189c1496a2bc1ef016ca47f1f7e4d204397150c failed because BuildContract and Verify did not yet exist.",
    "M14 contract-integrity RED Core CI run 36328795881 at 4f145e7900a305dec4a2d277aa6dbd517fba66c0 proved a tampered target_baseline_count could otherwise be accepted.",
    "M14 verification contracts match targets by stable semantic key rule_id + path + summary and baseline occurrence count instead of exact line-bound finding ID.",
    "M14 verification fails closed when the active analyzer set differs from the contract baseline.",
    "M14 verification blocks target resolution when a new finding with severity equal to or higher than the target appears on the target path.",
    "M14 contract metadata is checked against its baseline target semantic key for count and severity consistency.",
    "M14 never executes repository-provided build, test, shell, hook, or verification command strings.",
    "M14 product candidate " + PRODUCT_SHA + " passed Core CI run 36328881587 on Linux, Windows, and macOS, including public CLI negative and positive verification paths.",
    "At the same product candidate, M10 Python labels 36328881697, M07 Python Real World 36328881562, M06 Go labels 36328881607, and Real World Go 36328881558 passed on all three operating systems."
]
state["not_proven"] = [
    "M14 semantic identity is rule_id + path + summary + occurrence count; it does not prove exact AST-node identity across arbitrary rewrites.",
    "M14 regression comparison is intentionally scoped to the selected finding path, not the entire repository or related-context files.",
    "New findings below the target severity do not block M14 PASS, although they remain visible in a normal audit.",
    "M14 contract JSON is strict and self-consistent but is not cryptographically signed or authenticated against a malicious editor.",
    "M14 analyzer-set equality detects missing or added analyzer identities, but it does not cryptographically attest analyzer implementation bytes.",
    "M14 PASS proves the declared analyzer finding count decreased without same-or-higher-severity target-file regression; it does not prove full behavioral correctness.",
    "M14 does not execute repository tests, build scripts, package hooks, shell commands, or Finding.Verification strings.",
    "M14 does not authorize safe deletion, automatic repair, or mutation."
]
state["next_authorized_actions"] = [
    "Generate and validate the M14 CURRENT sequence plus deterministic Project Truth.",
    "Run final exact-SHA Governance Bootstrap, Core CI, and every previously accepted real-world lane.",
    "After FINAL_ACCEPTED, merge M14 to main immediately and rerun all applicable acceptance lanes on the exact main merge SHA."
]
state["blocked_actions"] = [
    "Executing arbitrary repository-provided verification commands as DoctorCode authority.",
    "Treating M14 PASS as proof of unrelated runtime or business behavior.",
    "Treating verification as safe-delete or automatic-repair authority.",
    "Starting MCP or Skill adapters before M14 is MAIN_ACCEPTED."
]
save(".workflow/state.json", state)

acceptance = load(".workflow/acceptance.json")
acceptance["evidence_boundary"] = (
    "M14 Deterministic Verification Contract freezes one pre-repair finding baseline and verifies the post-repair "
    "deterministic audit without executing target-repository commands. The stable target key is rule_id + path + "
    "summary with occurrence-count reduction, analyzer-set drift fails closed, and new same-or-higher-severity "
    "findings on the target path block PASS. Product candidate " + PRODUCT_SHA +
    " passed Core CI and every accepted real-world lane across Linux, Windows, and macOS. Cryptographic contract "
    "authentication, arbitrary runtime behavior, repository test execution, safe deletion, and automatic repair "
    "remain outside M14 authority."
)
acceptance["sequence_mode"] = "DURING"
acceptance["sequence_session"] = "M14-DETERMINISTIC-VERIFICATION"
acceptance["sequence_sync_status"] = "PASS"
acceptance["runtime_status"] = "PASS"
acceptance["runtime_checks"] = [
    "go test ./...",
    "Core CI M14 smoke: audit -> exact finding ID -> contract -> repair with new SECURITY regression -> verify FAIL -> clean repair -> verify PASS",
    "go run ./cmd/doctorcode audit . --json --max-findings=10",
    "go run ./cmd/doctorcode next . --json --max-bytes=4096",
    "go test ./internal/realworld -run TestPinnedPublicRepositories -v with exact-SHA Go public paths",
    "go test ./internal/realworld -run TestM06LabeledRealWorld -v with exact-SHA bounded Go label paths",
    "go test ./internal/realworld -run TestM07PinnedPythonRepositories -v with exact-SHA Python public paths",
    "go test ./internal/realworld -run TestM10LabeledPythonRealWorld -v with exact-SHA bounded Python label paths"
]
acceptance["test_commands"] = [
    "go test ./...",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/manifest.json --json",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/m03-adversarial.json --json",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/m04-repository-shaped.json --json",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/m07-python.json --analyzer=python --json",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/m08-python-adversarial.json --analyzer=python --json",
    "go run ./cmd/doctorcode benchmark internal/benchmark/testdata/m09-python-repository-shaped.json --analyzer=python --json",
    "go vet ./...",
    "go build -trimpath ./cmd/doctorcode",
    "Core CI Go related-context smoke test",
    "Core CI Python related-context smoke test",
    "Core CI deterministic verification contract smoke test"
]
acceptance["requirements"] = [
    {
        "id": "M14-MAIN-BASELINE",
        "requirement": "M14 must branch from governance-normalized M13 main after M13_MAIN_ACCEPTED and retain the pinned Skill_Workflow authority.",
        "evidence": "work/m14-deterministic-verification starts from main@" + BASE_SHA + "; M13 product baseline remains 3fccbd4398f2a12bd5214be9ed23ee18f46c679c; Skill_Workflow remains " + SKILL_SHA + ".",
        "status": "PASS"
    },
    {
        "id": "M14-RED-CONTRACT",
        "requirement": "The verification contract API must be demonstrated absent before implementation.",
        "evidence": "Core CI run 36328542441 at d189c1496a2bc1ef016ca47f1f7e4d204397150c failed because BuildContract and Verify were undefined.",
        "status": "PASS"
    },
    {
        "id": "M14-STABLE-TARGET",
        "requirement": "Verification must not false-PASS merely because a finding line number changes.",
        "evidence": "internal/verification/verification_test.go freezes rule_id + path + summary occurrence count; a semantically identical remaining finding at a shifted line preserves current count until an occurrence is actually removed.",
        "status": "PASS"
    },
    {
        "id": "M14-REGRESSION-BLOCK",
        "requirement": "A repair that resolves the target but introduces a new same-or-higher-severity finding on the target path must fail verification.",
        "evidence": "Core CI run 36328881587 executes the public CLI negative path with a new hardcoded-credential finding and asserts passed=false before the clean repair passes.",
        "status": "PASS"
    },
    {
        "id": "M14-ANALYZER-DRIFT",
        "requirement": "A verification contract must fail closed if the analyzer identity set differs between baseline and verification.",
        "evidence": "internal/verification/verification_test.go TestVerifyFailsClosedWhenAnalyzerSetChanges passes in Core CI 36328881587.",
        "status": "PASS"
    },
    {
        "id": "M14-CONTRACT-INTEGRITY",
        "requirement": "Target metadata must agree with the stored baseline semantic key count and severity, and unknown JSON fields must be rejected.",
        "evidence": "RED Core CI 36328795881 exposed the count mismatch; repaired candidate " + PRODUCT_SHA + " validates target count and severity while cmd tests reject unknown contract fields.",
        "status": "PASS"
    },
    {
        "id": "M14-NO-ARBITRARY-EXECUTION",
        "requirement": "DoctorCode verification must re-audit deterministically and must not execute repository-provided verification, test, build, or shell command strings.",
        "evidence": "cmd/doctorcode runVerify loads a strict contract then calls engine.Default().Audit and internal/verification.Verify only; no Finding.Verification string or repository command is dispatched.",
        "status": "PASS"
    },
    {
        "id": "M14-CLI-E2E",
        "requirement": "The public contract and verify workflow must demonstrate both blocking failure and clean success on Linux, Windows, and macOS.",
        "evidence": "Core CI run 36328881587 at " + PRODUCT_SHA + " passes the deterministic verification contract smoke on all three operating systems.",
        "status": "PASS"
    },
    {
        "id": "M14-REGRESSION-MATRIX",
        "requirement": "All previously accepted detector and public-source lanes must remain passing across Linux, Windows, and macOS.",
        "evidence": "At " + PRODUCT_SHA + ": Core CI 36328881587, M10 Python labels 36328881697, M07 Python Real World 36328881562, M06 Go labels 36328881607, and Real World Go 36328881558 all passed across all three operating systems.",
        "status": "PASS"
    },
    {
        "id": "M14-STRICT-GOVERNANCE-FINAL",
        "requirement": "Generated Project Truth and M14 sequence artifacts must pass strict governance on the final exact M14 branch SHA after temporary workflows are removed.",
        "evidence": "NOT_PROVEN until the synchronized final candidate is checked.",
        "status": "NOT_PROVEN"
    }
]
acceptance.setdefault("truth_gates", {})["SEQUENCE_SYNC"] = "PASS"
acceptance["truth_gates"]["CROSS_DOCUMENT_CONSISTENCY"] = "NOT_PROVEN"
acceptance["truth_gates"]["PROJECT_STATE_SYNC"] = "NOT_PROVEN"
save(".workflow/acceptance.json", acceptance)

claims = load(".workflow/claims.json")
known_claims = {x["id"] for x in claims["claims"]}
claim_additions = [
    {
        "id": "TRUTH-M14-STABLE-VERIFICATION",
        "claim": "M14 verification freezes a pre-repair semantic target keyed by rule_id, path, and summary plus occurrence count, then requires that count to decrease after a fresh deterministic audit.",
        "documents": ["PROJECT_TRUTH_SYNC.md"],
        "source_owners": ["internal/verification/verification.go", "cmd/doctorcode/main.go"],
        "tests": ["internal/verification/verification_test.go", ".github/workflows/ci.yml"],
        "runtime_evidence": ["GitHub Actions Core CI run 36328881587 at " + PRODUCT_SHA],
        "status": "PASS"
    },
    {
        "id": "TRUTH-M14-REGRESSION-GATE",
        "claim": "M14 blocks PASS when verification introduces a new finding at least as severe as the target on the selected target path.",
        "documents": ["PROJECT_TRUTH_SYNC.md"],
        "source_owners": ["internal/verification/verification.go"],
        "tests": ["internal/verification/verification_test.go", ".github/workflows/ci.yml"],
        "runtime_evidence": ["GitHub Actions Core CI run 36328881587 at " + PRODUCT_SHA],
        "status": "PASS"
    },
    {
        "id": "TRUTH-M14-NO-ARBITRARY-EXECUTION",
        "claim": "M14 contract verification re-runs DoctorCode deterministic analyzers only and never executes repository-provided test, build, shell, hook, or Finding.Verification command strings.",
        "documents": ["PROJECT_TRUTH_SYNC.md"],
        "source_owners": ["cmd/doctorcode/main.go", "internal/verification/verification.go"],
        "tests": [".github/workflows/ci.yml"],
        "runtime_evidence": ["GitHub Actions Core CI run 36328881587 at " + PRODUCT_SHA],
        "status": "PASS"
    },
    {
        "id": "TRUTH-M14-CONTRACT-INTEGRITY",
        "claim": "M14 fails closed on analyzer-set drift, unknown contract JSON fields, and mismatched target baseline count or severity metadata.",
        "documents": ["PROJECT_TRUTH_SYNC.md"],
        "source_owners": ["cmd/doctorcode/main.go", "internal/verification/verification.go"],
        "tests": ["cmd/doctorcode/main_test.go", "internal/verification/verification_test.go"],
        "runtime_evidence": [
            "RED GitHub Actions Core CI run 36328795881 at 4f145e7900a305dec4a2d277aa6dbd517fba66c0",
            "GREEN GitHub Actions Core CI run 36328881587 at " + PRODUCT_SHA
        ],
        "status": "PASS"
    }
]
for item in claim_additions:
    if item["id"] not in known_claims:
        claims["claims"].append(item)
save(".workflow/claims.json", claims)

decisions = load(".workflow/decisions.json")
known_decisions = {x["id"] for x in decisions["decisions"]}
decision_additions = [
    {
        "id": "ADR-M14-001",
        "title": "Freeze verification before repair",
        "status": "ACCEPTED",
        "decision": "Create a JSON verification contract from the exact current finding before repair, then verify that stored contract after repair.",
        "rationale": "After a repair the original line-bound finding ID may disappear or move, so verification needs an explicit pre-repair baseline."
    },
    {
        "id": "ADR-M14-002",
        "title": "Use semantic occurrence count instead of line-bound ID",
        "status": "ACCEPTED",
        "decision": "Identify the verification target by rule_id + path + summary and require its occurrence count to decrease from the frozen baseline.",
        "rationale": "Existing finding IDs include line numbers; line-only movement must not create a false resolved result."
    },
    {
        "id": "ADR-M14-003",
        "title": "Block same-or-higher target-path regressions",
        "status": "ACCEPTED",
        "decision": "PASS requires target resolution and no newly increased semantic finding count at severity equal to or above the original target on the target path.",
        "rationale": "A repair that removes one defect by introducing an equal or worse analyzer-visible defect is not an acceptable deterministic verification result."
    },
    {
        "id": "ADR-M14-004",
        "title": "Never execute repository verification commands",
        "status": "ACCEPTED",
        "decision": "M14 verification uses a fresh DoctorCode audit only; repository-provided tests, builds, shell commands, hooks, and Finding.Verification strings are never executed as verification authority.",
        "rationale": "Analyzed repositories are untrusted input and command execution would cross the established read-only analysis boundary."
    }
]
for item in decision_additions:
    if item["id"] not in known_decisions:
        decisions["decisions"].append(item)
save(".workflow/decisions.json", decisions)

changelog = load(".workflow/changelog.json")
if not any(x.get("title") == "M14 Deterministic Verification Contract candidate" for x in changelog["entries"]):
    changelog["entries"].insert(0, {
        "date": "2026-09-27",
        "title": "M14 Deterministic Verification Contract candidate",
        "type": "development",
        "changes": [
            "Added doctorcode contract to freeze a pre-repair deterministic finding baseline.",
            "Added doctorcode verify to re-audit after repair and require semantic target-count reduction.",
            "Added target-path same-or-higher-severity regression blocking.",
            "Added analyzer-set drift, strict JSON, and baseline target metadata fail-closed checks.",
            "Kept repository-provided commands entirely outside verification execution authority.",
            "Added Linux, Windows, and macOS CLI smoke proving regression FAIL followed by clean PASS."
        ]
    })
save(".workflow/changelog.json", changelog)

architecture = load(".workflow/architecture.json")
if not any(x.get("id") == "verification_contract" for x in architecture["components"]):
    architecture["components"].append({
        "id": "verification_contract",
        "name": "Deterministic Verification Contract",
        "purpose": "Freeze one pre-repair analyzer finding baseline and verify post-repair analyzer-visible resolution without executing repository commands.",
        "depends_on": ["detector engine"],
        "owns": [
            "semantic target occurrence baseline",
            "analyzer-set compatibility check",
            "target-path regression delta",
            "verification PASS/FAIL result"
        ]
    })
for component in architecture["components"]:
    if component["id"] == "cli":
        component["purpose"] = "Human/agent interface for scan, toolchain, audit, bounded context, deterministic verification contract, verification result, and benchmark operations."
        for owned in ("verification contract serialization", "verification result rendering"):
            if owned not in component["owns"]:
                component["owns"].append(owned)
if not any(x.get("to") == "Deterministic Verification Contract" for x in architecture["data_flows"]):
    architecture["data_flows"].extend([
        {
            "from": "detector engine",
            "to": "Deterministic Verification Contract",
            "meaning": "Before repair, doctorcode contract freezes analyzer identities plus target-path semantic finding counts for one exact selected finding."
        },
        {
            "from": "repaired repository",
            "to": "detector engine",
            "meaning": "doctorcode verify performs a fresh deterministic audit of the current repository state without executing repository-provided commands."
        },
        {
            "from": "Deterministic Verification Contract",
            "to": "human or AI coding agent",
            "meaning": "Verification reports PASS only when the semantic target occurrence count decreases and no same-or-higher-severity target-path regression is introduced."
        }
    ])
if not any(x["name"] == "Verification execution boundary" for x in architecture["external_boundaries"]):
    architecture["external_boundaries"].append({
        "name": "Verification execution boundary",
        "contract": "M14 never executes repository-provided test, build, shell, hook, or Finding.Verification command strings. Verification authority is a fresh DoctorCode deterministic audit plus the frozen contract only."
    })
save(".workflow/architecture.json", architecture)

project = load(".workflow/project.json")
entry_names = {x["name"] for x in project["entry_points"]}
if "doctorcode contract" not in entry_names:
    project["entry_points"].append({
        "name": "doctorcode contract",
        "path": "cmd/doctorcode/main.go",
        "purpose": "Freeze one exact current finding into a deterministic pre-repair verification baseline."
    })
if "doctorcode verify" not in entry_names:
    project["entry_points"].append({
        "name": "doctorcode verify",
        "path": "cmd/doctorcode/main.go",
        "purpose": "Re-audit a repaired repository and deterministically verify target resolution plus bounded target-path regression constraints."
    })
save(".workflow/project.json", project)

m13 = load("docs/sequence/sessions/M13-PYTHON-RELATED-CONTEXT.json")
m13["scope"] = "HISTORICAL"
save("docs/sequence/sessions/M13-PYTHON-RELATED-CONTEXT.json", m13)

save(".workflow/workflows/FLOW-DETERMINISTIC-VERIFICATION.json", {
    "schema_version": 1,
    "flow_id": "FLOW-DETERMINISTIC-VERIFICATION",
    "title": "Deterministic repair verification",
    "purpose": "Freeze one analyzer finding baseline before repair and verify analyzer-visible resolution after repair without executing untrusted repository commands.",
    "authority": "internal/verification/verification.go, cmd/doctorcode/main.go, their tests, and .github/workflows/ci.yml",
    "entry_condition": "An exact current finding ID exists before repair and doctorcode contract is invoked against that repository state.",
    "source_owners": ["internal/verification/verification.go", "cmd/doctorcode/main.go"],
    "states": ["BASELINE_AUDITED", "CONTRACT_FROZEN", "REPAIR_EXTERNAL", "CURRENT_AUDITED", "CONTRACT_COMPARED", "RESULT_REPORTED"],
    "transitions": [
        {"from": "BASELINE_AUDITED", "to": "CONTRACT_FROZEN", "action": "Store analyzer identities, target semantic key, target baseline occurrence count, and all target-path baseline semantic counts.", "side_effects": ["Writes contract JSON only when the caller redirects CLI output."]},
        {"from": "CONTRACT_FROZEN", "to": "REPAIR_EXTERNAL", "action": "A human or coding agent performs the repair outside DoctorCode verification authority.", "side_effects": []},
        {"from": "REPAIR_EXTERNAL", "to": "CURRENT_AUDITED", "action": "doctorcode verify performs a fresh deterministic audit; repository command strings are not executed.", "side_effects": []},
        {"from": "CURRENT_AUDITED", "to": "CONTRACT_COMPARED", "action": "Require analyzer-set equality, target semantic count decrease, and no new same-or-higher-severity target-path finding-count increase.", "side_effects": []},
        {"from": "CONTRACT_COMPARED", "to": "RESULT_REPORTED", "action": "Emit deterministic JSON or text PASS/FAIL; CLI exits nonzero for a failed verification result.", "side_effects": []}
    ],
    "invariants": [
        "The contract is generated before repair from an exact current finding ID.",
        "Target semantic identity excludes line numbers and uses rule_id + path + summary plus occurrence count.",
        "Analyzer identity set must remain equal between contract and verification.",
        "Contract target metadata must agree with the stored target baseline count and severity.",
        "Unknown contract JSON fields fail closed.",
        "No repository-provided test, build, shell, hook, or Finding.Verification command is executed.",
        "PASS does not imply full runtime behavior correctness, safe deletion, or automatic repair authority."
    ],
    "tests": ["internal/verification/verification_test.go", "cmd/doctorcode/main_test.go", ".github/workflows/ci.yml"],
    "sequence_session": "M14-DETERMINISTIC-VERIFICATION",
    "critical": False,
    "failure_behavior": ["Invalid or tampered contracts and analyzer-set drift fail closed with an error; unresolved target or blocking target-path regression returns a deterministic failed verification result."],
    "restart_behavior": ["Verification is stateless beyond the explicit saved contract and can be rerun against any unchanged repaired repository state."],
    "rollback_behavior": ["M14 can be reverted without changing detector findings, evidence packets, or safe_autofix semantics."]
})

save("docs/sequence/sessions/M14-DETERMINISTIC-VERIFICATION.json", {
    "schema_version": 1,
    "session_id": "M14-DETERMINISTIC-VERIFICATION",
    "phase": "M14_DETERMINISTIC_VERIFICATION_CONTRACT",
    "mode": "DURING",
    "scope": "CURRENT",
    "critical": False,
    "status": "IN_PROGRESS",
    "implementation_base_sha": BASE_SHA,
    "plan": {"required": False, "contract": "", "frozen": False, "frozen_commit": "", "sha256": "", "diagram": ""},
    "actual": {
        "graph": "docs/sequence/generated/M14-DETERMINISTIC-VERIFICATION.actual.json",
        "diagram": "docs/sequence/generated/M14-DETERMINISTIC-VERIFICATION.actual.mmd",
        "source_digest": "",
        "entries": []
    },
    "test_traceability_required": True,
    "tests": ["internal/verification/verification_test.go", "cmd/doctorcode/main_test.go", ".github/workflows/ci.yml"],
    "runtime_trace": {"required": False, "graph": ""},
    "acceptance_report": ""
})

readme_path = Path("README.md")
readme = readme_path.read_text(encoding="utf-8")
readme = re.sub(
    r"\*\*M13 Python Related Context is accepted on main at 3fccbd4398f2a12bd5214be9ed23ee18f46c679c\.\*\*",
    "**M14 Deterministic Verification Contract is validating on work/m14-deterministic-verification from governance-normalized main@" + BASE_SHA + ".**",
    readme,
    count=1,
)
if "### M14 deterministic verification contract" not in readme:
    readme += (
        "\n### M14 deterministic verification contract\n\n"
        "M14 adds a two-step repair-verification workflow. Before editing code, save a contract with "
        "doctorcode contract <finding-id> <repo> --json redirected to a contract file. After the repair, run "
        "doctorcode verify <contract-file> <repo> --json. The verifier re-audits the current source and requires "
        "the target semantic occurrence count to decrease.\n\n"
        "Finding IDs contain line numbers, so M14 does not use disappearance of the old ID as proof. The frozen "
        "semantic target is rule_id + path + summary plus its baseline occurrence count. Verification also blocks "
        "newly increased finding counts on the target path when their severity is equal to or higher than the "
        "original target. Analyzer-set drift and inconsistent contract metadata fail closed.\n\n"
        "DoctorCode does not run repository tests, builds, shell commands, hooks, or advisory Finding.Verification "
        "strings during M14 verification. PASS means only that the declared deterministic analyzer contract passed; "
        "it is not proof of full runtime behavior, safe deletion, or automatic repair safety.\n"
    )
readme_path.write_text(readme, encoding="utf-8")
