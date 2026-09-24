# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project follows
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-09-24

First release: a working example of an LLM email feature that keeps the model inside
the facts and keeps a human between the model and the send button.

### Added

- **Grounded drafting**: every body sentence must be matched by a claim that cites one of
  the supplied facts, or be listed as `unresolved`. Placeholders, quoted-reply leakage,
  multi-line or oversized subjects, and unusable claim fact indexes are rejected.
  Framing text such as a salutation or a sign-off is declared with fact index `-1`.
- **Approval bound to content and recipient**: `email.Approval` stores a SHA-256
  fingerprint over the draft and the recipient, and its fields are unexported so only
  `email.Approve` can record a decision. Editing a draft, or changing who receives it,
  invalidates the approval.
- **A send boundary that cannot be bypassed**: `sender.Sender` accepts only
  `sender.ApprovedMessage`, whose approval flag is unexported and set solely by
  `sender.Approved`. The default adapter refuses every send; an in-memory recorder is
  available behind `-sender recorder`.
- **ADK v2 agent** with a `genai` output schema whose tone and language enums come from
  the declared domain values, a stable grounded instruction, and a prompt that carries
  the case facts.
- **CLI** with a dry-run preview that separates fact-backed statements (printing the
  cited fact under each one), framing statements, and unresolved items; `-json` and
  `-html` output; exit codes that separate usage and configuration errors (2), rejected
  drafts and drafting failures (1), and refused sends (3).
- **Example fixture** `testdata/email.json`, pinned to the contract by `main_test.go`.
- **Offline test suite**: 76 tests, 122 including subtests, covering validation,
  rendering and escaping, fingerprints, the sender boundary, the CLI pipeline with an
  injected drafter, and the ADK path through a fake model.
- **CI**: `gofmt`, `go vet`, `go build`, and `go test -race` on every push and pull request.

### Security

- Nothing can leave the process without an approval record bound to the exact draft and
  recipient; the only built-in adapters either refuse to send or record in memory.
- HTML rendering goes through `html/template`, so escaping is context-aware.

### Verification status

Verified offline with `GOPROXY=off` and no `GOOGLE_API_KEY`. The live Gemini request has
never been run in this repository: treat the model transport as unverified until you run
`go run . -input testdata/email.json` with your own key.

[0.1.0]: https://github.com/GhandyP/professional-email-drafting/releases/tag/v0.1.0
