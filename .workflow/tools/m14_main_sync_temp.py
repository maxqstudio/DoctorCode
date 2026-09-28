import json
import re
from pathlib import Path

PRODUCT_MAIN = "6952187538ffa4a6f3f149db440d1f6f6960ba61"

def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))

def save(path, data):
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")

state = load(".workflow/state.json")
state["status"] = "M14_MAIN_ACCEPTED"
state["working_branch"] = "main"
state["last_accepted_sha"] = PRODUCT_MAIN
state["last_accepted_branch"] = "main"
for item in [
    "M14 Deterministic Verification Contract merged through PR #13 to main at " + PRODUCT_MAIN + ".",
    "Exact M14 product main Governance Bootstrap run 36329987778 completed success.",
    "Exact M14 product main Core CI run 36329987823 completed success on ubuntu-latest, windows-latest, and macos-latest.",
    "Exact M14 product main M10 Labeled Real World Python run 36329987784 completed success on all three operating systems.",
    "Exact M14 product main M07 Python Real World run 36329987819 completed success on all three operating systems.",
    "Exact M14 product main M06 Labeled Real World Go run 36329987822 completed success on all three operating systems.",
    "Exact M14 product main Real World Go Validation run 36329987965 completed success on all three operating systems."
]:
    if item not in state["proven"]:
        state["proven"].append(item)
protection_note = (
    "GitHub main branch protection is ACTIVE through repository ruleset 24089721 ('protection'): "
    "default branch deletion and non-fast-forward updates are blocked, pull requests are required, "
    "review threads must be resolved, and no bypass actor is configured."
)
if protection_note not in state["proven"]:
    state["proven"].append(protection_note)
state["not_proven"] = [
    item for item in state["not_proven"]
    if "branch protection" not in item.lower() and "ruleset" not in item.lower()
]
state["next_authorized_actions"] = [
    "Proceed to M15 Thin Skill Adapter under the Owner directive to continue until DoctorCode is complete.",
    "Branch M15 only from the governance-normalized M14 main after exact closure acceptance.",
    "Keep the Skill adapter thin: all detection, context, and verification authority remains in the DoctorCode CLI/core.",
    "Preserve M14 product acceptance at " + PRODUCT_MAIN + " and its exact post-merge evidence as immutable historical evidence."
]
state["blocked_actions"] = [
    "Duplicating DoctorCode detector, context, or verification logic inside a Skill adapter.",
    "Executing arbitrary repository-provided commands as DoctorCode authority.",
    "Treating M14 verification as safe-delete or automatic-repair authority.",
    "Claiming required status-check contexts are enforced by the GitHub ruleset while its required-status-check list remains empty."
]
save(".workflow/state.json", state)

acceptance = load(".workflow/acceptance.json")
acceptance["evidence_boundary"] = (
    "M14 Deterministic Verification Contract is product-main accepted at " + PRODUCT_MAIN +
    " after strict governance and every accepted Linux, Windows, and macOS regression lane passed. "
    "M14 freezes a pre-repair semantic finding baseline and re-audits after repair without executing "
    "repository-provided commands. It requires target occurrence reduction, stable analyzer identities, "
    "contract self-consistency, and no new same-or-higher-severity target-path regression. Full runtime "
    "correctness, safe deletion, automatic repair, cryptographic contract authentication, and repository "
    "required-status-check context administration remains outside the M14 product proof boundary."
)
if not any(x.get("id") == "M14-MAIN-POST-MERGE" for x in acceptance["requirements"]):
    acceptance["requirements"].append({
        "id": "M14-MAIN-POST-MERGE",
        "requirement": "The exact M14 product main merge SHA must pass strict governance and every accepted cross-platform regression lane before M15 may branch.",
        "evidence": (
            "main@" + PRODUCT_MAIN + ": Governance Bootstrap 36329987778, Core CI 36329987823, "
            "M10 Labeled Real World Python 36329987784, M07 Python Real World 36329987819, "
            "M06 Labeled Real World Go 36329987822, and Real World Go Validation 36329987965 "
            "all completed success; every matrix job passed on Linux, Windows, and macOS."
        ),
        "status": "PASS"
    })
save(".workflow/acceptance.json", acceptance)

claims = load(".workflow/claims.json")
if not any(x.get("id") == "TRUTH-M14-MAIN-ACCEPTED" for x in claims["claims"]):
    claims["claims"].append({
        "id": "TRUTH-M14-MAIN-ACCEPTED",
        "claim": "M14 Deterministic Verification Contract is merged and post-merge accepted on main at " + PRODUCT_MAIN + " after strict governance and every accepted Linux, Windows, and macOS regression lane completed successfully.",
        "documents": ["PROJECT_TRUTH_SYNC.md"],
        "source_owners": [".workflow/state.json", ".workflow/acceptance.json"],
        "tests": [
            ".github/workflows/governance-bootstrap.yml",
            ".github/workflows/ci.yml",
            ".github/workflows/m10-python-labeled-real-world.yml",
            ".github/workflows/m07-python-real-world.yml",
            ".github/workflows/m06-labeled-real-world.yml",
            ".github/workflows/real-world-go.yml"
        ],
        "runtime_evidence": [
            "GitHub Actions Governance Bootstrap run 36329987778 at " + PRODUCT_MAIN,
            "GitHub Actions Core CI run 36329987823 at " + PRODUCT_MAIN,
            "GitHub Actions M10 Labeled Real World Python run 36329987784 at " + PRODUCT_MAIN,
            "GitHub Actions M07 Python Real World run 36329987819 at " + PRODUCT_MAIN,
            "GitHub Actions M06 Labeled Real World Go run 36329987822 at " + PRODUCT_MAIN,
            "GitHub Actions Real World Go Validation run 36329987965 at " + PRODUCT_MAIN
        ],
        "status": "PASS"
    })
save(".workflow/claims.json", claims)

changelog = load(".workflow/changelog.json")
if not any(x.get("title") == "M14 main accepted baseline" for x in changelog["entries"]):
    changelog["entries"].insert(0, {
        "date": "2026-09-27",
        "title": "M14 main accepted baseline",
        "type": "acceptance",
        "changes": [
            "Merged FINAL_ACCEPTED M14 through PR #13 to main at " + PRODUCT_MAIN + ".",
            "Passed strict Governance Bootstrap on the exact product main merge SHA.",
            "Passed Core CI, bounded Python labels, pinned Python public-source validation, bounded Go labels, and pinned Go public-source validation on Linux, Windows, and macOS.",
            "Authorized M15 Thin Skill Adapter while retaining DoctorCode CLI/core as the only product logic authority.",
            "Recorded active main branch protection from ruleset 24089721; deletion/non-fast-forward are blocked and pull requests are required."
        ]
    })
save(".workflow/changelog.json", changelog)

readme_path = Path("README.md")
readme = readme_path.read_text(encoding="utf-8")
readme = re.sub(
    r"\*\*M14 Deterministic Verification Contract is accepted on the development branch pending final exact-SHA promotion checks\.\*\*",
    "**M14 Deterministic Verification Contract is accepted on main at " + PRODUCT_MAIN + ".**",
    readme,
    count=1,
)
readme_path.write_text(readme, encoding="utf-8")

governance_path = Path(".github/workflows/governance-bootstrap.yml")
governance = governance_path.read_text(encoding="utf-8")
governance = re.sub(
    r"(validate_cross_document_consistency\.py --base )[0-9a-f]{40}( --require-base)",
    r"\g<1>" + PRODUCT_MAIN + r"\g<2>",
    governance,
    count=1,
)
governance_path.write_text(governance, encoding="utf-8")
