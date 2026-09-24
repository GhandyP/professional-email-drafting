# professional-email-drafting

[![CI](https://github.com/GhandyP/professional-email-drafting/actions/workflows/ci.yml/badge.svg)](https://github.com/GhandyP/professional-email-drafting/actions/workflows/ci.yml)

Draft a professional email from facts you supply, under a contract that refuses to state anything the facts do not support — and never send without a recorded human approval.

A small, complete Go example built on [Google ADK for Go](https://github.com/google/adk-go) v2.2.0 and Gemini. It exists for the two hard parts of an LLM email feature: keeping the model inside the facts, and keeping a human between the model and the send button.

## The two guarantees

**Grounded content.** Every sentence in the draft body must be matched by a claim that points at one of the supplied facts, or be listed as unresolved. Validation also rejects placeholder text, quoted-reply leakage, multi-line or oversized subjects, and claim fact indexes that do not exist. It proves provenance, not entailment: it can prove that every body sentence is matched to a claim citing an existing fact, and it cannot prove that the fact entails the sentence. That judgement stays with the reviewer, which is why the preview prints the cited fact under each statement.

**No send without approval.** `sender.Sender` accepts only `sender.ApprovedMessage`, a type whose approval flag is unexported and is set solely by `sender.Approved`, after the draft validates and the approval fingerprint matches the exact draft content. The only adapter wired in by default refuses every send, so nothing leaves the process.

The dry-run preview makes both guarantees visible: fact-backed statements, framing statements, and unresolved items print in separate groups, with the cited fact under each statement, so a reviewer sees in seconds which lines the model could not ground and which fact each remaining line leans on.

## Quick start

Requirements: Go 1.26.5 or newer, and a `GOOGLE_API_KEY` from [Google AI Studio](https://aistudio.google.com/apikey).

```bash
export GOOGLE_API_KEY="your-key"

# Dry run: draft, validate, preview. Nothing is sent.
go run . -input testdata/email.json

# The same draft as JSON, or as an escaped HTML document.
go run . -input testdata/email.json -json
go run . -input testdata/email.json -html

# Record an approval, then watch the default sender refuse to deliver.
go run . -input testdata/email.json -approve ana@example.com

# Approve and record the message with the in-memory recorder instead.
go run . -input testdata/email.json -approve ana@example.com -sender recorder
```

Facts can also come from flags:

```bash
go run . \
  -recipient ana@example.com \
  -fact "Invoice 42 was paid on 2026-09-01." \
  -goal "Confirm receipt of the payment." \
  -tone concise -language en
```

## Package map

| Path | Responsibility |
| --- | --- |
| `main.go` | Signal-aware entry point; delegates to the CLI. |
| `internal/email` | Domain contract: input and draft types, validation, rendering, approval fingerprints. No I/O. |
| `internal/agent` | ADK wiring: output schema, grounded instruction, prompt, and the Gemini-backed drafter. |
| `internal/sender` | Send boundary: `ApprovedMessage`, the disabled sender, and an in-memory recorder. |
| `internal/cli` | Flag parsing, the preview, the approval gate, and the exit codes. |
| `testdata/email.json` | Example input fixture, pinned to the contract by `main_test.go`. |

## CLI reference

| Flag | Default | Meaning |
| --- | --- | --- |
| `-recipient` | – | Recipient mailbox; required. |
| `-fact` | – | Supporting fact; repeat for each fact. At least one is required. |
| `-goal` | – | What the email must achieve; required. |
| `-tone` | `concise` | One of `concise`, `formal`, `friendly`. |
| `-language` | `en` | One of `en`, `es`. |
| `-input` | – | Path to a JSON input file; explicit scalar flags override its fields and `-fact` values are appended. |
| `-approve` | – | Approver identity. Without it the run is a dry run and no sender is called. |
| `-json` | `false` | Print the draft as JSON on stdout instead of the preview. |
| `-html` | `false` | Print the HTML preview on stdout instead of the plain preview. |
| `-sender` | `disabled` | `disabled` or `recorder`. |
| `-timeout` | `60s` | Drafting timeout. |

Exit codes: `0` success, `1` rejected draft or drafting failure, `2` usage, input, or configuration error, `3` refused send. Data goes to stdout; narration and errors go to stderr.

## Verification status

Verified offline, with `GOPROXY=off` and no `GOOGLE_API_KEY` present:

```bash
go build ./... && go vet ./... && go test ./...

# The suite was verified with the network and the key switched off:
env -u GOOGLE_API_KEY GOPROXY=off go test ./...
```

CI runs the same checks — `gofmt`, `go vet`, `go build`, and `go test -race` — on every push and every pull request.

The suite reports 76 tests (122 including subtests) passing. It covers input and draft validation, rendering and escaping, approval fingerprints, the sender boundary, the CLI pipeline with an injected drafter, and the ADK path through a fake model — so `llmagent`, the output schema, the runner turn, and the structured decode are all exercised without a network call.

Not verified: the live Gemini request. No test reaches the network, and this repository has not run the model against the real API. Treat the Gemini transport as untested until you run it with your own key.

## Design decisions

- **Framing text is declared, not assumed.** A salutation or sign-off is a claim with fact index `-1` (`email.NoFact`), so the preview prints it apart from the fact-backed statements instead of pretending it is grounded.
- **The subject is a label, not an assertion.** It is validated for shape, placeholders, and quoted content, but deliberately not grounded; the body carries the claims.
- **Approval is bound to content and recipient.** `email.Approval` stores a SHA-256 fingerprint over the draft and the recipient, and its fields are unexported, so only `email.Approve` can record a decision. Editing the draft, or changing who receives it, invalidates the approval.
- **The domain rejects blank facts; the CLI filters them.** User input is trimmed at the boundary, and the validator keeps the contract strict.
- **There is exactly one send boundary.** The module contains no SMTP, HTTP, or process call. A real transport means implementing `sender.Sender` and passing it explicitly.

## Lineage

This project started as case 02 of the `adk-go-enterprise-cookbook` and is now standalone: it owns its ADK wiring and depends only on the ADK and genai modules.

## License

MIT — see [LICENSE](LICENSE).
