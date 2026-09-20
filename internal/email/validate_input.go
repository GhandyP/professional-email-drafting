package email

import (
	"net/mail"
	"strings"
)

type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Problems) == 0 {
		return "validation failed"
	}

	messages := make([]string, 0, len(e.Problems))
	for _, problem := range e.Problems {
		messages = append(messages, problem.Field+": "+problem.Message)
	}
	return strings.Join(messages, "; ")
}

func (in Input) Validate() error {
	problems := make([]Problem, 0, 5)

	recipient := strings.TrimSpace(in.Recipient)
	// ParseAddress accepts RFC 5322 display-name mailboxes as required here.
	if recipient == "" {
		problems = append(problems, Problem{
			Field:   "recipient",
			Message: "must be a valid email address",
		})
	} else if _, err := mail.ParseAddress(recipient); err != nil {
		problems = append(problems, Problem{
			Field:   "recipient",
			Message: "must be a valid email address",
		})
	}

	allFactsPresent := len(in.Facts) > 0
	for _, fact := range in.Facts {
		if strings.TrimSpace(fact) == "" {
			allFactsPresent = false
			break
		}
	}
	if !allFactsPresent {
		problems = append(problems, Problem{
			Field:   "facts",
			Message: "must contain at least one fact, and every fact must be non-blank",
		})
	}

	if strings.TrimSpace(in.Goal) == "" {
		problems = append(problems, Problem{
			Field:   "goal",
			Message: "must be non-blank",
		})
	}

	if !containsTone(in.Tone) {
		problems = append(problems, Problem{
			Field:   "tone",
			Message: "must be one of the declared tones",
		})
	}

	if !containsLanguage(in.Language) {
		problems = append(problems, Problem{
			Field:   "language",
			Message: "must be one of the declared languages",
		})
	}

	if len(problems) == 0 {
		return nil
	}
	return &ValidationError{Problems: problems}
}

func containsTone(want Tone) bool {
	for _, tone := range Tones() {
		if want == tone {
			return true
		}
	}
	return false
}

func containsLanguage(want Language) bool {
	for _, language := range Languages() {
		if want == language {
			return true
		}
	}
	return false
}
