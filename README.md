# 02 — Draft professional emails

> **Status: standalone port in progress.** This directory is being turned into an
> independent project with the module path `github.com/GhandyP/professional-email-drafting`.
> Everything below is the original cookbook case note, kept as the design input for the
> port, so the *Run the current prototype* section is stale: it described the shared
> `internal/adkhelper` of `adk-go-enterprise-cookbook`, which a standalone module cannot
> import. The CLI is not wired yet, and `go build ./...` is the only verified command
> today. The *Missing elements* section below is the work this port executes.

## Goal

Draft a professional email from supplied facts, a goal, a tone, and a language without adding unsupported claims or sending anything automatically.

## Run the current prototype

From this case directory:

```bash
export GOOGLE_API_KEY="your-key"
go run .
```

`internal/adkhelper.Run` uses ADK Go v2.2.0, Gemini, and `gemini-flash-latest`; set `ADK_MODEL` to override the model.

Sample prompt:

```text
Draft a polite follow-up from these three facts and keep the tone concise.
```

## What exists now

- `main.go (exists)` launches one Gemini ADK agent through the shared helper.
- The input is free-form text and the output is free-form draft text; recipient, facts, tone, and language are not validated.
- This is intentionally text-only: no email provider, template store, approval record, or sender is connected.

## Missing elements

- **Contract:** Define `DraftEmailInput{recipient, facts[], goal, tone, language}` -> `DraftEmail{subject, body, claims[], unresolved[]}`.
- **ADK layer:** Add a response schema and a session for the thread; keep the agent instruction grounded in `facts`, with a future `send_email` function tool disabled by default.
- **Domain boundary:** Add template/brand-policy lookup and an email-sender adapter that accepts only an approved draft ID.
- **Fixtures and validation:** Test missing facts, placeholders, quoted replies, HTML/plain-text rendering, and unsupported claims with golden fixtures.
- **Safety:** Show a dry-run preview and require a human approval event before any send; protect recipient and account data.

## Target structure

```text
02-professional-email-drafting/
├── main.go (exists)
├── README.md (exists)
├── agent.go (next)
├── schema.go (next)
├── policy.go (next)
├── adapters/sender.go (next)
├── testdata/email.json (next)
└── main_test.go (next)
```

## Implementation order

1. Model facts, recipient, tone, language, and the draft output.
2. Add schema validation, policy checks, and a thread session.
3. Implement draft rendering and placeholder/claim validation.
4. Add fixtures for short, multilingual, and incomplete requests.
5. Add a disabled sender tool and approval-gated dry-run workflow.

## Done when

- [ ] The draft contains only supplied or explicitly marked unresolved facts.
- [ ] Subject, body, language, tone, and claims pass schema validation.
- [ ] Tests cover missing facts, placeholders, and quoted replies.
- [ ] No message leaves the system without a recorded human approval.

## Next improvement

Define `DraftEmail` first, then validate the current model response against it before adding a sender.
