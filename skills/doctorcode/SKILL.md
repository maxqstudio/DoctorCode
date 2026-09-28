---
name: doctorcode
description: Guides agents through evidence-bounded code diagnosis and repair using the DoctorCode CLI as the deterministic authority. Use when reviewing code for bloat, security, simplification, logic, dead or stale code, when preparing a small repair from a DoctorCode finding, or when verifying that a repair removed the selected finding without introducing an equal-or-higher-severity regression on its target file.
---

# DoctorCode

## Overview

Use DoctorCode as the deterministic evidence layer around an agent or human code repair.

The skill does not contain detector logic. It orchestrates the installed `doctorcode` CLI and keeps the agent inside the evidence boundaries already enforced by the core.

The normal lifecycle is:

```text
audit
  -> choose one current finding
  -> context
  -> contract before editing
  -> make the smallest justified repair
  -> verify after editing
  -> report deterministic evidence
```

## When to Use

Use this skill when:

- diagnosing DoctorCode findings in a repository;
- reviewing BLOAT, SECURITY, SIMPLIFY, LOGIC, or DEADCODE / UNUSED / STALE findings;
- preparing bounded source context for a human or coding agent;
- repairing one selected current finding;
- checking whether a repair satisfies the deterministic DoctorCode verification contract;
- an agent needs a compact evidence packet instead of loading a whole repository.

Do not use this skill as:

- a replacement for project-specific tests, compilers, linters, or runtime acceptance;
- authority to delete code merely because a reference count is zero;
- authority to mutate code automatically;
- a substitute for the DoctorCode CLI/core.

## Authority Boundaries

Treat these rules as invariants:

1. The `doctorcode` CLI/core is the only DoctorCode product-logic authority.
2. Findings are evidence, not permission to edit.
3. `safe_autofix=false` means no DoctorCode finding authorizes automatic mutation.
4. Zero conservative references do not prove runtime unreachability or safe deletion.
5. Context packets are bounded review evidence, not whole-program dependency proofs.
6. Security findings may intentionally omit source excerpts.
7. Verification does not execute repository-provided build, test, shell, hook, or verification command strings.
8. Project-specific tests and runtime checks remain separate evidence and follow the repository's own trust and governance policy.

## Workflow

### 1. Audit the current repository state

Run:

```sh
doctorcode audit <repo> --json
```

Use the current output as the finding authority. Do not carry forward a finding from an older source state without re-auditing.

Select one finding ID to work on. Preserve its:

- finding ID;
- rule ID;
- path;
- severity;
- summary;
- confidence and proof boundary.

Do not reinterpret `SUSPICIOUS` as proven behavior.

### 2. Retrieve bounded context for that exact finding

Run:

```sh
doctorcode context <finding-id> <repo> --json --max-bytes=4096
```

The exact byte budget may be adjusted when the task requires it, but keep it intentionally bounded.

Use the packet as the preferred source context. For supported Go and Python findings, the packet may include deterministic related production or test references. Unsupported or ambiguous cases can remain primary-source-only.

If the finding ID is stale or unknown, stop and re-run `doctorcode audit`. Do not guess a replacement finding.

### 3. Freeze the verification contract before editing

Before changing the target source, run:

```sh
doctorcode contract <finding-id> <repo> --json > doctorcode-contract.json
```

The contract freezes the pre-repair analyzer set and semantic finding baseline.

Do not generate the contract after the repair. A post-edit baseline cannot prove that the selected pre-edit finding decreased.

Keep the contract unchanged through the repair attempt.

### 4. Make the smallest evidence-supported repair

Change only what is necessary to address the selected finding.

Preserve these constraints:

- do not expand scope because nearby code also looks improvable;
- do not delete code solely from a DEADCODE or zero-reference signal;
- do not weaken tests, validation, or security controls to make a finding disappear;
- do not change DoctorCode metadata or the saved contract to manufacture a PASS;
- do not treat related-context references as a complete call graph or import graph.

If the finding is ambiguous, preserve the code and report the ambiguity instead of forcing a repair.

### 5. Verify against the pre-repair contract

After the edit, run:

```sh
doctorcode verify doctorcode-contract.json <repo> --json
```

Interpret the result narrowly.

A DoctorCode verification PASS means:

- the selected semantic finding occurrence count decreased from its baseline;
- the analyzer identity set still matches the contract;
- the contract is internally consistent;
- no new finding with severity equal to or higher than the target appeared on the target path.

A PASS does **not** prove:

- full behavioral correctness;
- that unrelated files are regression-free;
- runtime correctness;
- safe deletion;
- that lower-severity findings did not appear;
- that project-specific tests pass.

A nonzero `doctorcode verify` exit or JSON `passed: false` is a failed deterministic verification gate. Repair the code or report the blocker; do not edit the contract to bypass it.

### 6. Run project acceptance separately when authorized

DoctorCode verification and repository acceptance answer different questions.

After DoctorCode verification, run only the project-specific build, tests, lint, or runtime checks authorized by that project's own workflow. Treat those results as separate evidence.

Never execute an untrusted repository-provided command merely because a DoctorCode finding contains a verification hint.

### 7. Report evidence

A completed repair report should identify:

- selected finding ID and rule;
- target path;
- context budget used;
- pre-repair contract captured;
- files actually changed;
- DoctorCode verification result;
- any separate project test/runtime evidence;
- remaining uncertainty or unsupported dynamic behavior.

Do not report DoctorCode PASS when only repository tests passed, and do not report project runtime PASS when only DoctorCode verification passed.

## Failure Handling

### Stale finding ID

Re-run `doctorcode audit`, locate the current finding from deterministic evidence, and create a fresh pre-repair contract before editing.

### Analyzer-set drift

Treat verification as failed closed. Do not compare contracts across a changed analyzer identity set.

### New blocking finding

If `new_blocking_findings` is non-empty, the repair is not DoctorCode-verified even when the original target decreased.

### Ambiguous dynamic behavior

Reflection, generated linkage, plugins, monkey-patching, dynamic imports, external callers, and other runtime mechanisms may sit outside the static proof boundary. Preserve uncertainty explicitly.

### Insufficient context

Increase the bounded `--max-bytes` budget only as needed or inspect the exact referenced source directly. Do not replace missing evidence with an invented dependency relation.

## Common Rationalizations

| Rationalization | Required response |
|---|---|
| "The finding disappeared, so the repair is done." | Run `doctorcode verify` against the contract captured before editing. |
| "The line number changed, so verification cannot work." | M14 verification uses the semantic rule/path/summary baseline rather than exact line-bound finding identity. |
| "Zero references means I can delete it." | Zero conservative references are not proof of runtime unreachability. |
| "The repository tests pass, so DoctorCode verification is unnecessary." | The gates prove different things; preserve both when both are required. |
| "DoctorCode PASS proves the feature still works." | DoctorCode PASS is intentionally narrower than behavioral/runtime correctness. |
| "I can edit the contract because the code moved." | Contract mutation destroys the pre-repair baseline; re-audit and restart the repair attempt instead. |

## Red Flags

Stop and reassess if you are:

- editing before capturing the contract;
- selecting a finding from stale output;
- changing multiple unrelated files for one finding;
- treating `SUSPICIOUS` as proven;
- claiming dead code is safe to remove from static references alone;
- bypassing a failed `doctorcode verify`;
- copying detector or verifier rules into this skill;
- using DoctorCode as authority to execute untrusted repository commands;
- reporting runtime correctness from static verification alone.

## Verification Checklist

Before declaring the DoctorCode portion of a repair complete, confirm:

- [ ] `doctorcode audit` ran against the current repository state.
- [ ] One exact current finding was selected.
- [ ] `doctorcode context` supplied bounded evidence for that finding.
- [ ] `doctorcode contract` was captured before the edit.
- [ ] The repair stayed within the selected evidence scope.
- [ ] `doctorcode verify` returned PASS after the edit.
- [ ] No DoctorCode boundary was inflated into a stronger claim.
- [ ] Any project-specific tests or runtime acceptance are reported separately.
