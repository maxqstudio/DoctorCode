# DoctorCode

DoctorCode is a deterministic, LLM-optional code intelligence engine designed to turn large repositories into small, evidence-backed findings that even small local models can consume efficiently.

The project focuses on five defect domains:

1. **BLOAT** — unnecessary code and structural complexity.
2. **SECURITY** — security weaknesses and unsafe data/control flows.
3. **SIMPLIFY** — behavior-preserving simplification opportunities.
4. **LOGIC** — contradictions, unreachable branches, lifecycle/resource defects, and other logic risks.
5. **DEADCODE** — unused or stale functions, classes, modules, and unreachable code islands, with conservative deletion evidence.

## Design principles

- Deterministic analysis first; LLM reasoning is optional.
- Small, bounded evidence packets instead of whole-repository prompts.
- Precision over recall for destructive recommendations.
- Never claim safe deletion from a zero-reference count alone.
- Cross-platform core: Windows, Linux, and macOS.
- Language/compiler support through adapters rather than vendor lock-in.
- GitHub Actions is the public repository CI and acceptance runner.
- Project governance follows `maxqstudio/Skill_Workflow`.

## Status

DoctorCode is in bootstrap development. Current support claims are intentionally narrow until CI evidence exists.

## Support

If DoctorCode is useful to you, optional donations can support continued development:

- [Saweria](https://saweria.co/maxq)
- [PayPal](https://paypal.me/JacksonJackson1501)

Donations do not affect access to the public repository or its features.
