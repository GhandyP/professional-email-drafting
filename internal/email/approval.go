package email

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Approval binds an approval identity and timestamp to a draft fingerprint.
type Approval struct {
	draftID     string
	approvedBy  string
	approvedAt  time.Time
	fingerprint string
}

// ErrMissingDraftID indicates that an approval is missing its draft identity.
var ErrMissingDraftID = errors.New("missing draft id")

// ErrMissingApprover indicates that an approval is missing its approver identity.
var ErrMissingApprover = errors.New("missing approver")

// ErrApprovalMismatch indicates that an approval does not match a draft.
var ErrApprovalMismatch = errors.New("approval fingerprint mismatch")

// DraftID returns the identity of the approved draft.
func (a Approval) DraftID() string {
	return a.draftID
}

// ApprovedBy returns the identity of the approver.
func (a Approval) ApprovedBy() string {
	return a.approvedBy
}

// ApprovedAt returns the time when the approval was recorded.
func (a Approval) ApprovedAt() time.Time {
	return a.approvedAt
}

// Fingerprint returns the draft fingerprint recorded by the approval.
func (a Approval) Fingerprint() string {
	return a.fingerprint
}

// Approve records approval for a draft after validating its identity fields.
func Approve(d Draft, recipient, draftID, approvedBy string, at time.Time) (Approval, error) {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return Approval{}, fmt.Errorf("draft ID is required: %w", ErrMissingDraftID)
	}

	approvedBy = strings.TrimSpace(approvedBy)
	if approvedBy == "" {
		return Approval{}, fmt.Errorf("approver is required: %w", ErrMissingApprover)
	}

	return Approval{
		draftID:     draftID,
		approvedBy:  approvedBy,
		approvedAt:  at,
		fingerprint: d.Fingerprint(recipient),
	}, nil
}

// Fingerprint returns the stable SHA-256 fingerprint of the draft content and recipient.
func (d Draft) Fingerprint(recipient string) string {
	claims := d.Claims
	if claims == nil {
		claims = []Claim{}
	}
	unresolved := d.Unresolved
	if unresolved == nil {
		unresolved = []string{}
	}

	canonical, _ := json.Marshal(struct {
		Recipient  string
		Subject    string
		Body       string
		Tone       Tone
		Language   Language
		Claims     []Claim
		Unresolved []string
	}{
		Recipient:  recipient,
		Subject:    d.Subject,
		Body:       d.Body,
		Tone:       d.Tone,
		Language:   d.Language,
		Claims:     claims,
		Unresolved: unresolved,
	})

	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:])
}

// Authorizes reports whether the approval matches the draft's fingerprint.
func (a Approval) Authorizes(d Draft, recipient string) error {
	draftFingerprint := d.Fingerprint(recipient)
	if a.fingerprint == draftFingerprint {
		return nil
	}
	return fmt.Errorf("approval fingerprint %q does not match draft fingerprint %q: %w", a.fingerprint, draftFingerprint, ErrApprovalMismatch)
}
