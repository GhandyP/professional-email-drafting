package sender

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

var approvedAt = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func testInput() email.Input {
	return email.Input{
		Recipient: "ana@example.com",
		Facts:     []string{"Invoice 42 was paid on 2026-09-01."},
		Goal:      "Confirm receipt of the payment.",
		Tone:      email.ToneConcise,
		Language:  email.LanguageEnglish,
	}
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

func testApproval(t *testing.T, d email.Draft) email.Approval {
	t.Helper()

	approval, err := email.Approve(d, "draft-1", "ana@example.com", approvedAt)
	if err != nil {
		t.Fatalf("email.Approve() error = %v, want nil", err)
	}
	return approval
}

func testApprovedMessage(t *testing.T) ApprovedMessage {
	t.Helper()

	d := testDraft()
	msg, err := Approved(d, testInput(), testApproval(t, d))
	if err != nil {
		t.Fatalf("Approved() error = %v, want nil", err)
	}
	return msg
}

func TestApprovedRendersTheMessage(t *testing.T) {
	msg := testApprovedMessage(t)

	if !msg.Authorized() {
		t.Fatal("Authorized() = false, want true")
	}

	got := msg.Message()
	if got.DraftID != "draft-1" {
		t.Fatalf("DraftID = %q, want %q", got.DraftID, "draft-1")
	}
	if got.To != "ana@example.com" {
		t.Fatalf("To = %q, want %q", got.To, "ana@example.com")
	}
	if got.Subject != "Invoice 42 payment" {
		t.Fatalf("Subject = %q, want %q", got.Subject, "Invoice 42 payment")
	}
	if got.Plain == "" {
		t.Fatal("Plain is empty, want the rendered plain-text message")
	}
	if got.HTML == "" {
		t.Fatal("HTML is empty, want the rendered HTML preview")
	}
	if got.ApprovedBy != "ana@example.com" {
		t.Fatalf("ApprovedBy = %q, want %q", got.ApprovedBy, "ana@example.com")
	}
	if !got.ApprovedAt.Equal(approvedAt) {
		t.Fatalf("ApprovedAt = %v, want %v", got.ApprovedAt, approvedAt)
	}
}

func TestApprovedRejectsAnInvalidDraft(t *testing.T) {
	d := testDraft()
	d.Body = "The client is very happy."

	msg, err := Approved(d, testInput(), testApproval(t, d))
	if err == nil {
		t.Fatal("Approved() = nil error, want the draft findings")
	}

	var derr *email.DraftError
	if !errors.As(err, &derr) {
		t.Fatalf("Approved() error type = %T, want *email.DraftError", err)
	}
	if msg.Authorized() {
		t.Fatal("Authorized() = true, want false for a rejected draft")
	}
}

func TestApprovedRejectsAMismatchedApproval(t *testing.T) {
	other := testDraft()
	other.Subject = "Payment confirmed"

	msg, err := Approved(testDraft(), testInput(), testApproval(t, other))
	if !errors.Is(err, email.ErrApprovalMismatch) {
		t.Fatalf("Approved() error = %v, want %v", err, email.ErrApprovalMismatch)
	}
	if msg.Authorized() {
		t.Fatal("Authorized() = true, want false for a mismatched approval")
	}
}

func TestZeroApprovedMessageIsNotAuthorized(t *testing.T) {
	if (ApprovedMessage{}).Authorized() {
		t.Fatal("Authorized() = true, want false for the zero value")
	}
}

func TestDisabledSenderRefusesApprovedMessages(t *testing.T) {
	err := DisabledSender{}.Send(context.Background(), testApprovedMessage(t))
	if !errors.Is(err, ErrSendingDisabled) {
		t.Fatalf("Send() error = %v, want %v", err, ErrSendingDisabled)
	}
}

func TestSendersRejectUnapprovedMessages(t *testing.T) {
	tests := []struct {
		name   string
		sender Sender
	}{
		{name: "disabled sender", sender: DisabledSender{}},
		{name: "recorder", sender: &Recorder{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.sender.Send(context.Background(), ApprovedMessage{}); !errors.Is(err, ErrNotApproved) {
				t.Fatalf("Send() error = %v, want %v", err, ErrNotApproved)
			}
		})
	}
}

func TestRecorderRecordsApprovedMessages(t *testing.T) {
	recorder := &Recorder{}
	msg := testApprovedMessage(t)

	if err := recorder.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}

	recorded := recorder.Messages()
	if len(recorded) != 1 {
		t.Fatalf("len(Messages()) = %d, want 1", len(recorded))
	}
	if recorded[0].Subject != msg.Message().Subject {
		t.Fatalf("recorded subject = %q, want %q", recorded[0].Subject, msg.Message().Subject)
	}

	recorded[0].Subject = "tampered"
	if again := recorder.Messages(); again[0].Subject != msg.Message().Subject {
		t.Fatalf("Messages() exposed internal state: %q", again[0].Subject)
	}
}

func TestRecorderRejectsUnapprovedMessages(t *testing.T) {
	recorder := &Recorder{}

	if err := recorder.Send(context.Background(), ApprovedMessage{}); !errors.Is(err, ErrNotApproved) {
		t.Fatalf("Send() error = %v, want %v", err, ErrNotApproved)
	}
	if got := len(recorder.Messages()); got != 0 {
		t.Fatalf("len(Messages()) = %d, want 0", got)
	}
}
