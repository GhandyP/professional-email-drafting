package agent

import (
	"google.golang.org/genai"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

// OutputSchema returns the structured schema required for an email draft.
func OutputSchema() *genai.Schema {
	toneValues := make([]string, 0, len(email.Tones()))
	for _, tone := range email.Tones() {
		toneValues = append(toneValues, string(tone))
	}

	languageValues := make([]string, 0, len(email.Languages()))
	for _, language := range email.Languages() {
		languageValues = append(languageValues, string(language))
	}

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"subject": {
				Type:        genai.TypeString,
				Description: "The single-line subject of the email.",
			},
			"body": {
				Type:        genai.TypeString,
				Description: "The body of the email.",
			},
			"tone": {
				Type:   genai.TypeString,
				Format: "enum",
				Enum:   toneValues,
			},
			"language": {
				Type:   genai.TypeString,
				Format: "enum",
				Enum:   languageValues,
			},
			"claims": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"text": {Type: genai.TypeString},
						"fact": {
							Type:        genai.TypeInteger,
							Description: "The zero-based fact index, or -1 for framing text.",
						},
					},
					Required: []string{"text", "fact"},
				},
			},
			"unresolved": {
				Type:  genai.TypeArray,
				Items: &genai.Schema{Type: genai.TypeString},
			},
		},
		Required: []string{"subject", "body", "tone", "language", "claims", "unresolved"},
	}
}
