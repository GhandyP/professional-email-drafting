package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GhandyP/professional-email-drafting/internal/email"
	"github.com/GhandyP/professional-email-drafting/internal/sender"
)

var fixedNow = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

type fakeDrafter struct {
	draft email.Draft
	err   error
}

func (f fakeDrafter) Draft(context.Context, email.Input) (email.Draft, error) {
	return f.draft, f.err
}

type capturingDrafter struct {
	draft email.Draft
	input email.Input
}

func (c *capturingDrafter) Draft(_ context.Context, in email.Input) (email.Draft, error) {
	c.input = in
	return c.draft, nil
}

func testDraft() email.Draft {
	return email.Draft{
		Subject:  "Invoice 42 payment",
		Body:     "Invoice 42 was paid on 2026-09-01.",
		Tone:     email.ToneConcise,
		Language: email.LanguageEnglish,
		Claims:   []email.Claim{{Text: "Invoice 42 was paid on 2026-09-01.", Fact: 0}},
	}
}

func testArgs(extra ...string) []string {
	args := []string{
		"-recipient", "ana@example.com",
		"-fact", "Invoice 42 was paid on 2026-09-01.",
		"-goal", "Confirm receipt of the payment.",
	}
	return append(args, extra...)
}

func run(t *testing.T, deps Deps, args ...string) (int, string, string) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), args, deps, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func recorderDeps(recorder *sender.Recorder) Deps {
	return Deps{
		Drafter: fakeDrafter{draft: testDraft()},
		Sender:  recorder,
		Now:     func() time.Time { return fixedNow },
	}
}

func TestRunRejectsMissingInput(t *testing.T) {
	code, stdout, stderr := run(t, recorderDeps(&sender.Recorder{}))

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %s)", code, stderr)
	}
	for _, want := range []string{"recipient", "facts", "goal"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not mention %q", stderr, want)
		}
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestRunReportsInputProblems(t *testing.T) {
	args := []string{"-recipient", "not-an-address", "-goal", "Confirm it.", "-tone", "shouty"}

	code, _, stderr := run(t, recorderDeps(&sender.Recorder{}), args...)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %s)", code, stderr)
	}
	for _, want := range []string{"recipient", "facts", "tone"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q does not mention %q", stderr, want)
		}
	}
}

func TestRunDryRunPrintsThePreviewAndSendsNothing(t *testing.T) {
	recorder := &sender.Recorder{}

	code, stdout, stderr := run(t, recorderDeps(recorder), testArgs()...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}

	for _, want := range []string{
		"Draft preview",
		"Subject: Invoice 42 payment",
		"Fact-backed statements:",
		"[fact 1] Invoice 42 was paid on 2026-09-01.",
		"Framing statements:",
		"Unresolved items:",
		"Message:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
		}
	}
	if !strings.Contains(stderr, "Dry run") {
		t.Fatalf("stderr %q does not announce the dry run", stderr)
	}
	if got := len(recorder.Messages()); got != 0 {
		t.Fatalf("the recorder captured %d messages, want 0 without approval", got)
	}
}

func TestRunWithoutApprovalNeverSendsEvenWithASenderConfigured(t *testing.T) {
	recorder := &sender.Recorder{}

	code, _, stderr := run(t, recorderDeps(recorder), testArgs()...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if got := len(recorder.Messages()); got != 0 {
		t.Fatalf("the recorder captured %d messages, want 0", got)
	}
}

func TestRunReportsDraftFindings(t *testing.T) {
	draft := testDraft()
	draft.Body = "The client is very happy."

	deps := Deps{
		Drafter: fakeDrafter{draft: draft},
		Sender:  &sender.Recorder{},
		Now:     func() time.Time { return fixedNow },
	}

	code, stdout, stderr := run(t, deps, testArgs()...)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, email.CodeUnsupportedStatement) {
		t.Fatalf("stderr %q does not report %q", stderr, email.CodeUnsupportedStatement)
	}
	if strings.Contains(stdout, "Draft preview") {
		t.Fatalf("stdout printed a preview for a rejected draft:\n%s", stdout)
	}
}

func TestRunReportsDrafterFailure(t *testing.T) {
	deps := Deps{
		Drafter: fakeDrafter{err: errors.New("model exploded")},
		Sender:  &sender.Recorder{},
		Now:     func() time.Time { return fixedNow },
	}

	code, _, stderr := run(t, deps, testArgs()...)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "model exploded") {
		t.Fatalf("stderr %q does not report the drafter failure", stderr)
	}
}

func TestRunRecordsAnApprovedSend(t *testing.T) {
	recorder := &sender.Recorder{}

	code, _, stderr := run(t, recorderDeps(recorder), testArgs("-approve", "ana@example.com")...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}

	recorded := recorder.Messages()
	if len(recorded) != 1 {
		t.Fatalf("the recorder captured %d messages, want 1", len(recorded))
	}
	if recorded[0].Subject != "Invoice 42 payment" {
		t.Fatalf("recorded subject = %q, want %q", recorded[0].Subject, "Invoice 42 payment")
	}
	if recorded[0].ApprovedBy != "ana@example.com" {
		t.Fatalf("recorded approver = %q, want %q", recorded[0].ApprovedBy, "ana@example.com")
	}
	if !recorded[0].ApprovedAt.Equal(fixedNow) {
		t.Fatalf("recorded approval time = %v, want %v", recorded[0].ApprovedAt, fixedNow)
	}
}

func TestRunRefusesToSendWithTheDisabledSender(t *testing.T) {
	deps := Deps{
		Drafter: fakeDrafter{draft: testDraft()},
		Sender:  sender.DisabledSender{},
		Now:     func() time.Time { return fixedNow },
	}

	code, _, stderr := run(t, deps, testArgs("-approve", "ana@example.com")...)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "disabled") {
		t.Fatalf("stderr %q does not explain the refusal", stderr)
	}
}

func TestRunPrintsTheDraftAsJSON(t *testing.T) {
	code, stdout, stderr := run(t, recorderDeps(&sender.Recorder{}), testArgs("-json")...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}

	var draft email.Draft
	if err := json.Unmarshal([]byte(stdout), &draft); err != nil {
		t.Fatalf("stdout is not a JSON draft: %v\n%s", err, stdout)
	}
	if draft.Subject != "Invoice 42 payment" {
		t.Fatalf("decoded subject = %q, want %q", draft.Subject, "Invoice 42 payment")
	}
}

func TestRunPrintsTheHTMLPreview(t *testing.T) {
	code, stdout, stderr := run(t, recorderDeps(&sender.Recorder{}), testArgs("-html")...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "<!doctype html>") {
		t.Fatalf("stdout is not an HTML document:\n%s", stdout)
	}
	if !strings.Contains(stdout, "<h1>Invoice 42 payment</h1>") {
		t.Fatalf("stdout lacks the subject heading:\n%s", stdout)
	}
}

func TestRunReadsTheInputFromAJSONFile(t *testing.T) {
	code, stdout, stderr := run(t, recorderDeps(&sender.Recorder{}), "-input", "testdata/email.json")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Subject: Invoice 42 payment") {
		t.Fatalf("stdout lacks the preview:\n%s", stdout)
	}
}

func TestRunPassesTheInputToTheDrafter(t *testing.T) {
	drafter := &capturingDrafter{draft: testDraft()}
	deps := Deps{
		Drafter: drafter,
		Sender:  &sender.Recorder{},
		Now:     func() time.Time { return fixedNow },
	}

	code, _, stderr := run(t, deps, "-input", "testdata/email.json")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if drafter.input.Recipient != "ana@example.com" {
		t.Fatalf("drafter recipient = %q, want %q", drafter.input.Recipient, "ana@example.com")
	}
	if len(drafter.input.Facts) != 2 {
		t.Fatalf("drafter facts = %d, want 2 from the fixture", len(drafter.input.Facts))
	}
}

func TestRunAcceptsRepeatedFacts(t *testing.T) {
	drafter := &capturingDrafter{draft: testDraft()}
	deps := Deps{
		Drafter: drafter,
		Sender:  &sender.Recorder{},
		Now:     func() time.Time { return fixedNow },
	}

	args := []string{
		"-recipient", "ana@example.com",
		"-fact", "Invoice 42 was paid on 2026-09-01.",
		"-fact", "Payment came from the operations account.",
		"-goal", "Confirm receipt of the payment.",
	}

	code, _, stderr := run(t, deps, args...)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if len(drafter.input.Facts) != 2 {
		t.Fatalf("drafter facts = %d, want 2", len(drafter.input.Facts))
	}
}

func TestRunRejectsAnUnknownSender(t *testing.T) {
	code, _, stderr := run(t, recorderDeps(&sender.Recorder{}), testArgs("-sender", "smtp")...)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "sender") {
		t.Fatalf("stderr %q does not mention the sender", stderr)
	}
}

func TestRunRequiresAnAPIKeyWhenNoDrafterIsInjected(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")

	deps := Deps{
		Sender: &sender.Recorder{},
		Now:    func() time.Time { return fixedNow },
	}

	code, _, stderr := run(t, deps, testArgs()...)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr, "GOOGLE_API_KEY") {
		t.Fatalf("stderr %q does not mention the missing key", stderr)
	}
}
