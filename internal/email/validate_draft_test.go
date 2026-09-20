package email

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func validDraft() Draft {
	return Draft{
		Subject:  "Invoice 42 payment",
		Body:     "Invoice 42 was paid on 2026-09-01.\nPayment came from the operations account.",
		Tone:     ToneConcise,
		Language: LanguageEnglish,
		Claims: []Claim{
			{Text: "Invoice 42 was paid on 2026-09-01.", Fact: 0},
			{Text: "Payment came from the operations account.", Fact: 1},
		},
	}
}

func TestDraftValidateAcceptsGroundedDraft(t *testing.T) {
	if err := validDraft().Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestDraftValidateDoesNotGroundTheSubject(t *testing.T) {
	d := validDraft()
	d.Subject = "Payment confirmed"

	if err := d.Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil: the subject is a label, not a claim", err)
	}
}

func TestDraftValidateAcceptsFramingClaims(t *testing.T) {
	d := validDraft()
	d.Body = "Dear Ana,\nInvoice 42 was paid on 2026-09-01.\nBest regards,"
	d.Claims = []Claim{
		{Text: "Dear Ana,", Fact: NoFact},
		{Text: "Invoice 42 was paid on 2026-09-01.", Fact: 0},
		{Text: "Best regards,", Fact: NoFact},
	}

	if err := d.Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil for framing claims", err)
	}
}

func TestDraftValidateAcceptsUnresolvedStatement(t *testing.T) {
	d := validDraft()
	d.Body = "Invoice 42 was paid on 2026-09-01.\nThe receipt will follow tomorrow."
	d.Unresolved = []string{"The receipt will follow tomorrow."}

	if err := d.Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil for an explicitly unresolved statement", err)
	}
}

func TestDraftValidateMatchesIgnoringCaseAndTrailingTerminator(t *testing.T) {
	d := validDraft()
	d.Body = "invoice 42 was paid on 2026-09-01"
	d.Claims = []Claim{{Text: "Invoice 42 was paid on 2026-09-01.", Fact: 0}}

	if err := d.Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil after normalization", err)
	}
}

func TestDraftValidateMatchesUnresolvedIgnoringCaseAndTrailingTerminator(t *testing.T) {
	d := validDraft()
	d.Body = "The receipt will follow tomorrow"
	d.Claims = nil
	d.Unresolved = []string{"the receipt will follow tomorrow."}

	if err := d.Validate(validInput()); err != nil {
		t.Fatalf("Validate() = %v, want nil after normalization", err)
	}
}

func TestDraftValidateReportsFindings(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Draft)
		wantCodes  []string
		wantFields []string
	}{
		{
			name:       "tone mismatch",
			mutate:     func(d *Draft) { d.Tone = ToneFormal },
			wantCodes:  []string{CodeToneMismatch},
			wantFields: []string{"tone"},
		},
		{
			name:       "language mismatch",
			mutate:     func(d *Draft) { d.Language = LanguageSpanish },
			wantCodes:  []string{CodeLanguageMismatch},
			wantFields: []string{"language"},
		},
		{
			name:       "empty subject",
			mutate:     func(d *Draft) { d.Subject = "   " },
			wantCodes:  []string{CodeEmptySubject},
			wantFields: []string{"subject"},
		},
		{
			name:       "multiline subject",
			mutate:     func(d *Draft) { d.Subject = "Invoice 42\npayment" },
			wantCodes:  []string{CodeSubjectNewline},
			wantFields: []string{"subject"},
		},
		{
			name:       "long subject",
			mutate:     func(d *Draft) { d.Subject = strings.Repeat("a", MaxSubjectLength+1) },
			wantCodes:  []string{CodeLongSubject},
			wantFields: []string{"subject"},
		},
		{
			name:       "todo placeholder in subject",
			mutate:     func(d *Draft) { d.Subject = "TODO confirm the payment" },
			wantCodes:  []string{CodePlaceholder},
			wantFields: []string{"subject"},
		},
		{
			name: "brace placeholder in body",
			mutate: func(d *Draft) {
				d.Body = "Invoice 42 was paid on 2026-09-01.\nContact {{name}} for details."
				d.Claims = append(d.Claims, Claim{Text: "Contact {{name}} for details.", Fact: NoFact})
			},
			wantCodes:  []string{CodePlaceholder},
			wantFields: []string{"body"},
		},
		{
			name: "angle placeholder in body",
			mutate: func(d *Draft) {
				d.Body = "Invoice 42 was paid on 2026-09-01.\nContact <name> for details."
				d.Claims = append(d.Claims, Claim{Text: "Contact <name> for details.", Fact: NoFact})
			},
			wantCodes:  []string{CodePlaceholder},
			wantFields: []string{"body"},
		},
		{
			name: "quoted reply in body",
			mutate: func(d *Draft) {
				d.Body = "Invoice 42 was paid on 2026-09-01.\n> On Monday Ana wrote: thanks"
				d.Claims = append(d.Claims, Claim{Text: "> On Monday Ana wrote: thanks", Fact: NoFact})
			},
			wantCodes:  []string{CodeQuotedReply},
			wantFields: []string{"body"},
		},
		{
			name:       "empty body",
			mutate:     func(d *Draft) { d.Body = "\n \t" },
			wantCodes:  []string{CodeEmptyBody},
			wantFields: []string{"body"},
		},
		{
			name: "unsupported statement",
			mutate: func(d *Draft) {
				d.Body = "Invoice 42 was paid on 2026-09-01.\nThe client is very happy."
			},
			wantCodes:  []string{CodeUnsupportedStatement},
			wantFields: []string{"body"},
		},
		{
			name:       "no claims",
			mutate:     func(d *Draft) { d.Claims = nil },
			wantCodes:  []string{CodeUnsupportedStatement, CodeUnsupportedStatement},
			wantFields: []string{"body", "body"},
		},
		{
			name:       "claim fact out of range",
			mutate:     func(d *Draft) { d.Claims[0].Fact = 7 },
			wantCodes:  []string{CodeUnsupportedStatement, CodeClaimFactOutOfRange},
			wantFields: []string{"body", "claims"},
		},
		{
			name:       "claim fact below NoFact",
			mutate:     func(d *Draft) { d.Claims[0].Fact = -2 },
			wantCodes:  []string{CodeUnsupportedStatement, CodeClaimFactOutOfRange},
			wantFields: []string{"body", "claims"},
		},
		{
			name:       "blank claim text",
			mutate:     func(d *Draft) { d.Claims[0].Text = "   " },
			wantCodes:  []string{CodeUnsupportedStatement, CodeClaimEmptyText},
			wantFields: []string{"body", "claims"},
		},
		{
			name:       "blank unresolved entry",
			mutate:     func(d *Draft) { d.Unresolved = []string{"  "} },
			wantCodes:  []string{CodeUnresolvedEmpty},
			wantFields: []string{"unresolved"},
		},
		{
			name: "several findings keep a fixed order",
			mutate: func(d *Draft) {
				d.Subject = ""
				d.Tone = ToneFormal
				d.Language = LanguageSpanish
				d.Body = "The client is very happy."
				d.Claims = nil
				d.Unresolved = []string{" "}
			},
			wantCodes: []string{
				CodeToneMismatch,
				CodeLanguageMismatch,
				CodeEmptySubject,
				CodeUnsupportedStatement,
				CodeUnresolvedEmpty,
			},
			wantFields: []string{"tone", "language", "subject", "body", "unresolved"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := validDraft()
			tt.mutate(&d)

			err := d.Validate(validInput())
			if err == nil {
				t.Fatalf("Validate() = nil, want findings %v", tt.wantCodes)
			}

			var derr *DraftError
			if !errors.As(err, &derr) {
				t.Fatalf("Validate() error type = %T, want *DraftError", err)
			}
			if got := codesOf(derr.Findings); !slices.Equal(got, tt.wantCodes) {
				t.Fatalf("finding codes = %v, want %v", got, tt.wantCodes)
			}
			if got := fieldsOfFindings(derr.Findings); !slices.Equal(got, tt.wantFields) {
				t.Fatalf("finding fields = %v, want %v", got, tt.wantFields)
			}
			for _, finding := range derr.Findings {
				if finding.Message == "" {
					t.Fatalf("finding %+v has an empty message", finding)
				}
			}
		})
	}
}

func TestDraftErrorNamesEveryFinding(t *testing.T) {
	d := validDraft()
	d.Subject = ""
	d.Tone = ToneFormal

	err := d.Validate(validInput())

	var derr *DraftError
	if !errors.As(err, &derr) {
		t.Fatalf("Validate() error type = %T, want *DraftError", err)
	}
	if len(derr.Findings) != 2 {
		t.Fatalf("findings = %d, want 2", len(derr.Findings))
	}
	message := err.Error()
	for _, want := range []string{"tone", CodeToneMismatch, "subject", CodeEmptySubject} {
		if !strings.Contains(message, want) {
			t.Fatalf("error message %q does not mention %q", message, want)
		}
	}
}

func TestDraftValidateDoesNotMutateItsInputs(t *testing.T) {
	in := validInput()
	d := validDraft()
	d.Body = "The client is very happy."

	wantDraft := d
	wantDraft.Claims = slices.Clone(d.Claims)
	wantDraft.Unresolved = slices.Clone(d.Unresolved)

	if err := d.Validate(in); err == nil {
		t.Fatal("Validate() = nil, want findings")
	}
	if !reflect.DeepEqual(d, wantDraft) {
		t.Fatalf("Validate() mutated the draft: %+v", d)
	}
	if !reflect.DeepEqual(in, validInput()) {
		t.Fatalf("Validate() mutated the input: %+v", in)
	}
}

func codesOf(findings []Finding) []string {
	codes := make([]string, 0, len(findings))
	for _, finding := range findings {
		codes = append(codes, finding.Code)
	}
	return codes
}

func fieldsOfFindings(findings []Finding) []string {
	fields := make([]string, 0, len(findings))
	for _, finding := range findings {
		fields = append(fields, finding.Field)
	}
	return fields
}
