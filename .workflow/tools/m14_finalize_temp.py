import json
import re
from pathlib import Path

EXACT_SHA = "1c05cc6c5d7d5095fc29ce2bd7a56fdf92dc4b70"

def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))

def save(path, data):
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")

state = load(".workflow/state.json")
state["status"] = "M14_ACCEPTED"
for item in [
    "Final synchronized M14 branch tree " + EXACT_SHA + " passed Governance Bootstrap run 36329506408.",
    "Core CI run 36329506410 at " + EXACT_SHA + " passed on ubuntu-latest, windows-latest, and macos-latest, including deterministic verification negative and positive E2E paths.",
    "M10 Python labels run 36329506427, M07 Python Real World run 36329506385, M06 Go labels run 36329506403, and Real World Go run 36329506404 all passed on Linux, Windows, and macOS at " + EXACT_SHA + "."
]:
    if item not in state["proven"]:
        state["proven"].append(item)
state["next_authorized_actions"] = [
    "Run final exact-SHA acceptance on the finalized M14 branch tree with no temporary workflow present.",
    "Open and merge the FINAL_ACCEPTED M14 pull request to main.",
    "Rerun Governance Bootstrap, Core CI, and every accepted real-world lane on the exact main merge SHA before starting adapter/release milestones."
]
save(".workflow/state.json", state)

acceptance = load(".workflow/acceptance.json")
for item in acceptance["requirements"]:
    if item["id"] == "M14-STRICT-GOVERNANCE-FINAL":
        item["status"] = "PASS"
        item["evidence"] = (
            "At " + EXACT_SHA + ": Governance Bootstrap 36329506408, Core CI 36329506410, "
            "M10 Labeled Real World Python 36329506427, M07 Python Real World 36329506385, "
            "M06 Labeled Real World Go 36329506403, and Real World Go Validation 36329506404 "
            "all completed success; every matrix job passed on Linux, Windows, and macOS. "
            "Finalization is governance-only and will be revalidated on its exact SHA."
        )
for key in ("CROSS_DOCUMENT_CONSISTENCY", "PROJECT_STATE_SYNC"):
    acceptance.setdefault("truth_gates", {})[key] = "PASS"
save(".workflow/acceptance.json", acceptance)

session = load("docs/sequence/sessions/M14-DETERMINISTIC-VERIFICATION.json")
session["status"] = "ACCEPTED"
save("docs/sequence/sessions/M14-DETERMINISTIC-VERIFICATION.json", session)

changelog = load(".workflow/changelog.json")
title = "M14 Deterministic Verification Contract accepted baseline"
if not any(x.get("title") == title for x in changelog["entries"]):
    changelog["entries"].insert(0, {
        "date": "2026-09-27",
        "title": title,
        "type": "acceptance",
        "changes": [
            "Accepted pre-repair deterministic verification contracts and post-repair semantic occurrence verification.",
            "Accepted same-or-higher-severity target-path regression blocking.",
            "Accepted analyzer-set drift and contract metadata fail-closed behavior.",
            "Accepted public CLI negative and positive verification paths on Linux, Windows, and macOS.",
            "Kept repository command execution, safe deletion, automatic repair, and full runtime correctness outside M14 authority."
        ]
    })
save(".workflow/changelog.json", changelog)

readme_path = Path("README.md")
readme = readme_path.read_text(encoding="utf-8")
readme = re.sub(
    r"\*\*M14 Deterministic Verification Contract is validating on work/m14-deterministic-verification from governance-normalized main@79ff0ac95b3274e67b70f5dc2cda79360ed2d00a\.\*\*",
    "**M14 Deterministic Verification Contract is accepted on the development branch pending final exact-SHA promotion checks.**",
    readme,
    count=1,
)
readme_path.write_text(readme, encoding="utf-8")
