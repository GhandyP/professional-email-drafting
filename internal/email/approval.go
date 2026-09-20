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
	DraftID     string
	ApprovedBy  string
	ApprovedAt  time.Time
	Fingerprint string
}

// ErrMissingDraftID indicates that an approval is missing its draft identity.
var ErrMissingDraftID = errors.New("missing draft id")

// ErrMissingApprover indicates that an approval is missing its approver identity.
var ErrMissingApprover = errors.New("missing approver")

// ErrApprovalMismatch indicates that an approval does not match a draft.
var ErrApprovalMismatch = errors.New("approval fingerprint mismatch")

// Approve records approval for a draft after validating its identity fields.
func Approve(d Draft, draftID, approvedBy string, at time.Time) (Approval, error) {
	draftID = strings.TrimSpace(draftID)
	if draftID == "" {
		return Approval{}, fmt.Errorf("draft ID is required: %w", ErrMissingDraftID)
	}

	approvedBy = strings.TrimSpace(approvedBy)
	if approvedBy == "" {
		return Approval{}, fmt.Errorf("approver is required: %w", ErrMissingApprover)
	}

	return Approval{
		DraftID:     draftID,
		ApprovedBy:  approvedBy,
		ApprovedAt:  at,
		Fingerprint: d.Fingerprint(),
	}, nil
}

// Fingerprint returns the stable SHA-256 fingerprint of the draft content.
func (d Draft) Fingerprint() string {
	claims := d.Claims
	if claims == nil {
		claims = []Claim{}
	}
	unresolved := d.Unresolved
	if unresolved == nil {
		unresolved = []string{}
	}

	canonical, _ := json.Marshal(struct {
		Subject    string
		Body       string
		Tone       Tone
		Language   Language
		Claims     []Claim
		Unresolved []string
	}{
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
func (a Approval) Authorizes(d Draft) error {
	draftFingerprint := d.Fingerprint()
	if a.Fingerprint == draftFingerprint {
		return nil
	}
	return fmt.Errorf("approval fingerprint %q does not match draft fingerprint %q: %w", a.Fingerprint, draftFingerprint, ErrApprovalMismatch)
}
