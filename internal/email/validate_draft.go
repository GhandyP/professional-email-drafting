package email

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxSubjectLength is the maximum number of runes allowed in a trimmed subject.
const MaxSubjectLength = 120

// Finding describes one validation problem in a draft or its supporting input.
type Finding struct {
	Field   string
	Code    string
	Message string
}

// DraftError reports the findings produced while validating a draft.
type DraftError struct {
	Findings []Finding
}

// Error formats every finding in its field, code, and message order.
func (e *DraftError) Error() string {
	if e == nil {
		return ""
	}

	messages := make([]string, len(e.Findings))
	for i, finding := range e.Findings {
		messages[i] = fmt.Sprintf("%s: %s: %s", finding.Field, finding.Code, finding.Message)
	}
	return strings.Join(messages, "; ")
}

// CodeToneMismatch identifies a draft tone that differs from the requested tone.
const CodeToneMismatch = "tone_mismatch"

// CodeLanguageMismatch identifies a draft language that differs from the requested language.
const CodeLanguageMismatch = "language_mismatch"

// CodeEmptySubject identifies a subject that is empty after trimming.
const CodeEmptySubject = "empty_subject"

// CodeSubjectNewline identifies a subject containing a newline.
const CodeSubjectNewline = "subject_newline"

// CodeLongSubject identifies a subject longer than MaxSubjectLength runes.
const CodeLongSubject = "long_subject"

// CodeEmptyBody identifies a body that is empty after trimming.
const CodeEmptyBody = "empty_body"

// CodePlaceholder identifies placeholder syntax or placeholder words in a field.
const CodePlaceholder = "placeholder"

// CodeQuotedReply identifies quoted-reply content in a field.
const CodeQuotedReply = "quoted_reply"

// CodeUnsupportedStatement identifies a body statement without supporting evidence.
const CodeUnsupportedStatement = "unsupported_statement"

// CodeClaimEmptyText identifies a claim with empty text after trimming.
const CodeClaimEmptyText = "claim_empty_text"

// CodeClaimFactOutOfRange identifies a claim whose fact index is unusable.
const CodeClaimFactOutOfRange = "claim_fact_out_of_range"

// CodeUnresolvedEmpty identifies an unresolved entry with empty text after trimming.
const CodeUnresolvedEmpty = "unresolved_empty"

// Validate checks that a draft matches its requested metadata and support declarations.
func (d Draft) Validate(in Input) error {
	findings := make([]Finding, 0)
	add := func(field, code, message string) {
		findings = append(findings, Finding{Field: field, Code: code, Message: message})
	}

	if d.Tone != in.Tone {
		add("tone", CodeToneMismatch, "draft tone does not match the requested tone")
	}
	if d.Language != in.Language {
		add("language", CodeLanguageMismatch, "draft language does not match the requested language")
	}

	trimmedSubject := strings.TrimSpace(d.Subject)
	if trimmedSubject == "" {
		add("subject", CodeEmptySubject, "subject must not be empty")
	}
	if strings.ContainsAny(d.Subject, "\n\r") {
		add("subject", CodeSubjectNewline, "subject must be a single line")
	}
	if utf8.RuneCountInString(trimmedSubject) > MaxSubjectLength {
		add("subject", CodeLongSubject, "subject exceeds the maximum length")
	}
	if containsPlaceholder(d.Subject) {
		add("subject", CodePlaceholder, "subject contains a placeholder")
	}
	if containsQuotedReply(d.Subject) {
		add("subject", CodeQuotedReply, "subject contains quoted-reply content")
	}

	trimmedBody := strings.TrimSpace(d.Body)
	if trimmedBody == "" {
		add("body", CodeEmptyBody, "body must not be empty")
	}
	if containsPlaceholder(d.Body) {
		add("body", CodePlaceholder, "body contains a placeholder")
	}
	if containsQuotedReply(d.Body) {
		add("body", CodeQuotedReply, "body contains quoted-reply content")
	}
	for _, statement := range statements(d.Body) {
		if !supportedStatement(statement, d.Claims, d.Unresolved, len(in.Facts)) {
			add("body", CodeUnsupportedStatement, "statement is not supported by a claim or unresolved entry")
		}
	}

	for _, claim := range d.Claims {
		if strings.TrimSpace(claim.Text) == "" {
			add("claims", CodeClaimEmptyText, "claim text must not be empty")
		}
		if claim.Fact != NoFact && (claim.Fact < 0 || claim.Fact >= len(in.Facts)) {
			add("claims", CodeClaimFactOutOfRange, "claim fact index is out of range")
		}
	}
	for _, unresolved := range d.Unresolved {
		if strings.TrimSpace(unresolved) == "" {
			add("unresolved", CodeUnresolvedEmpty, "unresolved entry must not be empty")
		}
	}

	if len(findings) == 0 {
		return nil
	}
	return &DraftError{Findings: findings}
}

func statements(text string) []string {
	result := make([]string, 0)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		start := 0
		for i := 0; i < len(line); i++ {
			if line[i] != '.' && line[i] != '!' && line[i] != '?' {
				continue
			}
			if i+1 < len(line) && !nextRuneIsSpace(line[i+1:]) {
				continue
			}

			piece := strings.TrimSpace(line[start : i+1])
			if piece != "" {
				result = append(result, piece)
			}
			start = i + 1
		}

		piece := strings.TrimSpace(line[start:])
		if piece != "" {
			result = append(result, piece)
		}
	}
	return result
}

func nextRuneIsSpace(text string) bool {
	if text == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(text)
	return unicode.IsSpace(r)
}

func supportedStatement(statement string, claims []Claim, unresolved []string, factCount int) bool {
	for _, claim := range claims {
		factUsable := claim.Fact == NoFact || (claim.Fact >= 0 && claim.Fact < factCount)
		if factUsable && normalizedEqual(claim.Text, statement) {
			return true
		}
	}
	for _, entry := range unresolved {
		if normalizedEqual(entry, statement) {
			return true
		}
	}
	return false
}

func normalizedEqual(left, right string) bool {
	return strings.EqualFold(normalize(left), normalize(right))
}

func normalize(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ".!?…")
}

func containsPlaceholder(text string) bool {
	if strings.Contains(text, "{{") || strings.Contains(text, "}}") {
		return true
	}
	for i := 0; i < len(text); i++ {
		if text[i] != '<' {
			continue
		}
		for j := i + 1; j < len(text); j++ {
			if text[j] == '>' {
				if j > i+1 {
					return true
				}
				break
			}
		}
	}
	return containsPlaceholderWord(text)
}

func containsPlaceholderWord(text string) bool {
	for i := 0; i < len(text); {
		start := i
		r, size := utf8.DecodeRuneInString(text[i:])
		if !isWordRune(r) {
			i += size
			continue
		}

		i += size
		for i < len(text) {
			r, size = utf8.DecodeRuneInString(text[i:])
			if !isWordRune(r) {
				break
			}
			i += size
		}

		switch text[start:i] {
		case "TODO", "TBD", "XXX", "FIXME":
			return true
		}
	}
	return false
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func containsQuotedReply(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			return true
		}
		if strings.Contains(line, "-----Original Message-----") {
			return true
		}
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "wrote:") || strings.Contains(lowerLine, "escribió:") {
			return true
		}
	}
	return false
}
