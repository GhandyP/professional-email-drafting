package email

import (
	"errors"
	"slices"
	"testing"
)

func validInput() Input {
	return Input{
		Recipient: "ana@example.com",
		Facts: []string{
			"Invoice 42 was paid on 2026-09-01.",
			"Payment came from the operations account.",
		},
		Goal:     "Confirm receipt of the payment.",
		Tone:     ToneConcise,
		Language: LanguageEnglish,
	}
}

func TestInputValidateAcceptsCompleteInput(t *testing.T) {
	in := validInput()

	if err := in.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestInputValidateAcceptsDisplayNameRecipient(t *testing.T) {
	in := validInput()
	in.Recipient = "Ana Pérez <ana@example.com>"

	if err := in.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for an RFC 5322 mailbox", err)
	}
}

func TestInputValidateReportsEveryProblemInFieldOrder(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Input)
		want   []string
	}{
		{
			name:   "missing recipient",
			mutate: func(in *Input) { in.Recipient = "" },
			want:   []string{"recipient"},
		},
		{
			name:   "blank recipient",
			mutate: func(in *Input) { in.Recipient = "   " },
			want:   []string{"recipient"},
		},
		{
			name:   "malformed recipient",
			mutate: func(in *Input) { in.Recipient = "ana at example.com" },
			want:   []string{"recipient"},
		},
		{
			name:   "no facts",
			mutate: func(in *Input) { in.Facts = nil },
			want:   []string{"facts"},
		},
		{
			name:   "blank fact among valid facts",
			mutate: func(in *Input) { in.Facts = []string{"ok", "   ", "\t"} },
			want:   []string{"facts"},
		},
		{
			name:   "only blank facts",
			mutate: func(in *Input) { in.Facts = []string{"   ", "\t"} },
			want:   []string{"facts"},
		},
		{
			name:   "missing goal",
			mutate: func(in *Input) { in.Goal = "" },
			want:   []string{"goal"},
		},
		{
			name:   "blank goal",
			mutate: func(in *Input) { in.Goal = "  \n " },
			want:   []string{"goal"},
		},
		{
			name:   "unknown tone",
			mutate: func(in *Input) { in.Tone = "shouty" },
			want:   []string{"tone"},
		},
		{
			name:   "empty tone",
			mutate: func(in *Input) { in.Tone = "" },
			want:   []string{"tone"},
		},
		{
			name:   "unknown language",
			mutate: func(in *Input) { in.Language = "xx" },
			want:   []string{"language"},
		},
		{
			name:   "empty language",
			mutate: func(in *Input) { in.Language = "" },
			want:   []string{"language"},
		},
		{
			name: "several problems keep field order",
			mutate: func(in *Input) {
				in.Recipient = ""
				in.Goal = ""
				in.Language = "xx"
			},
			want: []string{"recipient", "goal", "language"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.mutate(&in)

			err := in.Validate()
			if err == nil {
				t.Fatalf("Validate() = nil, want problems for %v", tt.want)
			}

			var verr *ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("Validate() error type = %T, want *ValidationError", err)
			}
			if got := fieldsOf(verr.Problems); !slices.Equal(got, tt.want) {
				t.Fatalf("problem fields = %v, want %v", got, tt.want)
			}
			if err.Error() == "" {
				t.Fatal("ValidationError.Error() is empty")
			}
		})
	}
}

func TestInputValidateAcceptsEveryDeclaredToneAndLanguage(t *testing.T) {
	for _, tone := range Tones() {
		for _, language := range Languages() {
			in := validInput()
			in.Tone = tone
			in.Language = language

			if err := in.Validate(); err != nil {
				t.Fatalf("Validate() = %v for tone %q and language %q", err, tone, language)
			}
		}
	}
}

func fieldsOf(problems []Problem) []string {
	fields := make([]string, 0, len(problems))
	for _, problem := range problems {
		fields = append(fields, problem.Field)
	}
	return fields
}
