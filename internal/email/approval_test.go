package email

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

var approvedAt = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func TestApproveRecordsTheDraftFingerprint(t *testing.T) {
	d := validDraft()

	approval, err := Approve(d, " draft-1 ", " ana@example.com ", approvedAt)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}

	if approval.DraftID != "draft-1" {
		t.Fatalf("DraftID = %q, want %q", approval.DraftID, "draft-1")
	}
	if approval.ApprovedBy != "ana@example.com" {
		t.Fatalf("ApprovedBy = %q, want %q", approval.ApprovedBy, "ana@example.com")
	}
	if !approval.ApprovedAt.Equal(approvedAt) {
		t.Fatalf("ApprovedAt = %v, want %v", approval.ApprovedAt, approvedAt)
	}
	if approval.Fingerprint != d.Fingerprint() {
		t.Fatalf("Fingerprint = %q, want %q", approval.Fingerprint, d.Fingerprint())
	}
}

func TestApproveRejectsMissingIdentity(t *testing.T) {
	tests := []struct {
		name       string
		draftID    string
		approvedBy string
		want       error
	}{
		{name: "empty draft id", draftID: "", approvedBy: "ana@example.com", want: ErrMissingDraftID},
		{name: "blank draft id", draftID: "   ", approvedBy: "ana@example.com", want: ErrMissingDraftID},
		{name: "empty approver", draftID: "draft-1", approvedBy: "", want: ErrMissingApprover},
		{name: "blank approver", draftID: "draft-1", approvedBy: "  \t", want: ErrMissingApprover},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approval, err := Approve(validDraft(), tt.draftID, tt.approvedBy, approvedAt)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Approve() error = %v, want %v", err, tt.want)
			}
			if approval != (Approval{}) {
				t.Fatalf("Approve() = %+v, want the zero Approval", approval)
			}
		})
	}
}

func TestApprovalAuthorizesTheApprovedDraft(t *testing.T) {
	d := validDraft()

	approval, err := Approve(d, "draft-1", "ana@example.com", approvedAt)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}
	if err := approval.Authorizes(d); err != nil {
		t.Fatalf("Authorizes() = %v, want nil", err)
	}
}

func TestApprovalAuthorizesAnEqualDraft(t *testing.T) {
	approval, err := Approve(validDraft(), "draft-1", "ana@example.com", approvedAt)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}

	if err := approval.Authorizes(validDraft()); err != nil {
		t.Fatalf("Authorizes() = %v, want nil for equal content", err)
	}
}

func TestApprovalRejectsChangedContent(t *testing.T) {
	base := validDraft()
	base.Unresolved = []string{"The receipt will follow tomorrow."}

	approval, err := Approve(base, "draft-1", "ana@example.com", approvedAt)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}

	tests := []struct {
		name   string
		mutate func(*Draft)
	}{
		{name: "subject changed", mutate: func(d *Draft) { d.Subject = "Payment confirmed" }},
		{name: "body changed", mutate: func(d *Draft) { d.Body = d.Body + "\nAn extra sentence." }},
		{name: "tone changed", mutate: func(d *Draft) { d.Tone = ToneFormal }},
		{name: "language changed", mutate: func(d *Draft) { d.Language = LanguageSpanish }},
		{name: "claim fact changed", mutate: func(d *Draft) { d.Claims[0].Fact = 1 }},
		{name: "claim text changed", mutate: func(d *Draft) { d.Claims[0].Text = "Something else." }},
		{name: "claim added", mutate: func(d *Draft) {
			d.Claims = append(d.Claims, Claim{Text: "An extra sentence.", Fact: NoFact})
		}},
		{name: "claim order changed", mutate: func(d *Draft) { d.Claims[0], d.Claims[1] = d.Claims[1], d.Claims[0] }},
		{name: "unresolved added", mutate: func(d *Draft) { d.Unresolved = append(d.Unresolved, "Another item.") }},
		{name: "unresolved removed", mutate: func(d *Draft) { d.Unresolved = nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := base
			d.Claims = slices.Clone(base.Claims)
			d.Unresolved = slices.Clone(base.Unresolved)
			tt.mutate(&d)

			if err := approval.Authorizes(d); !errors.Is(err, ErrApprovalMismatch) {
				t.Fatalf("Authorizes() error = %v, want %v", err, ErrApprovalMismatch)
			}
		})
	}
}

func TestApprovalRejectsZeroValue(t *testing.T) {
	if err := (Approval{}).Authorizes(validDraft()); !errors.Is(err, ErrApprovalMismatch) {
		t.Fatalf("Authorizes() error = %v, want %v", err, ErrApprovalMismatch)
	}
}

func TestApprovalMismatchMessageNamesBothFingerprints(t *testing.T) {
	approval, err := Approve(validDraft(), "draft-1", "ana@example.com", approvedAt)
	if err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}

	other := validDraft()
	other.Subject = "Payment confirmed"

	err = approval.Authorizes(other)
	if err == nil {
		t.Fatal("Authorizes() = nil, want a mismatch")
	}
	message := err.Error()
	for _, want := range []string{approval.Fingerprint, other.Fingerprint()} {
		if !strings.Contains(message, want) {
			t.Fatalf("error message %q does not mention fingerprint %q", message, want)
		}
	}
}

func TestFingerprintIsAStableSHA256Hex(t *testing.T) {
	d := validDraft()

	first := d.Fingerprint()
	second := validDraft().Fingerprint()

	if first != second {
		t.Fatalf("Fingerprint() = %q and %q, want equal values for equal content", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("len(Fingerprint()) = %d, want 64", len(first))
	}
	if !isLowerHex(first) {
		t.Fatalf("Fingerprint() = %q, want lowercase hexadecimal", first)
	}

	changed := validDraft()
	changed.Body = changed.Body + " Extra."
	if changed.Fingerprint() == first {
		t.Fatal("Fingerprint() did not change when the body changed")
	}
}

func TestFingerprintIgnoresNilAndEmptySlices(t *testing.T) {
	withNil := validDraft()
	withNil.Claims = nil
	withNil.Unresolved = nil

	withEmpty := validDraft()
	withEmpty.Claims = []Claim{}
	withEmpty.Unresolved = []string{}

	if got, want := withNil.Fingerprint(), withEmpty.Fingerprint(); got != want {
		t.Fatalf("Fingerprint() = %q for nil slices and %q for empty slices, want equal values", got, want)
	}
}

func TestApproveDoesNotMutateTheDraft(t *testing.T) {
	d := validDraft()
	d.Unresolved = []string{"The receipt will follow tomorrow."}

	want := d
	want.Claims = slices.Clone(d.Claims)
	want.Unresolved = slices.Clone(d.Unresolved)

	if _, err := Approve(d, "draft-1", "ana@example.com", approvedAt); err != nil {
		t.Fatalf("Approve() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("Approve() mutated the draft: %+v", d)
	}
}

func isLowerHex(value string) bool {
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
