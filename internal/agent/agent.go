package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	adkagent "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

// Instruction is the stable instruction used by the email drafting agent.
const Instruction = `You draft one professional email from facts supplied by the caller.

Rules:
- Use only the supplied facts. Never invent a detail, a number, a date, or a name.
- Every sentence in the body must appear in claims, its claim text must match that
  sentence exactly, and its fact index must point at the supplied fact it uses.
- Mark framing text such as a salutation or a sign-off with fact index -1, so a
  reviewer can see it is not a fact-backed assertion.
- Anything you cannot support with a fact must be listed in unresolved instead of
  being written as if it were true.
- Never leave placeholder text such as angle-bracket tokens, TODO, TBD, XXX, or FIXME.
- Never quote an earlier message or include quoted-reply markers.
- Keep the subject a single line of at most 120 characters.
- Answer with the structured draft only.`

// Drafter produces a structured email draft from an input request.
type Drafter interface {
	Draft(ctx context.Context, in email.Input) (email.Draft, error)
}

// ADKDrafter produces drafts through a Google ADK runner.
type ADKDrafter struct {
	runner         *runner.Runner
	sessionCounter atomic.Uint64
}

var _ Drafter = (*ADKDrafter)(nil)

var errEmptyResponse = errors.New("empty draft response")

// ErrMissingAPIKey reports that GOOGLE_API_KEY was not configured.
var ErrMissingAPIKey = errors.New("missing GOOGLE_API_KEY")

// NewADKDrafter constructs an ADK-backed drafter for the supplied model.
func NewADKDrafter(m model.LLM) (*ADKDrafter, error) {
	a, err := llmagent.New(llmagent.Config{
		Name:         "email_drafter",
		Description:  "drafting professional email from supplied facts",
		Model:        m,
		Instruction:  Instruction,
		OutputSchema: OutputSchema(),
	})
	if err != nil {
		return nil, fmt.Errorf("create email drafter agent: %w", err)
	}

	r, err := runner.NewInMemory("professional-email-drafting", a)
	if err != nil {
		return nil, fmt.Errorf("create email drafter runner: %w", err)
	}
	return &ADKDrafter{runner: r}, nil
}

// NewGeminiDrafter constructs an ADK drafter backed by the configured Gemini model.
func NewGeminiDrafter(ctx context.Context) (*ADKDrafter, error) {
	key := strings.TrimSpace(os.Getenv("GOOGLE_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("create Gemini drafter: %w", ErrMissingAPIKey)
	}

	m, err := gemini.NewModel(ctx, modelName(), &genai.ClientConfig{APIKey: key})
	if err != nil {
		return nil, fmt.Errorf("create Gemini model: %w", err)
	}
	return NewADKDrafter(m)
}

// Prompt formats an email input as the drafting request sent to the model.
func Prompt(in email.Input) string {
	var b strings.Builder
	b.WriteString("Draft the email for the request below.\n\n")
	b.WriteString("Recipient: ")
	b.WriteString(in.Recipient)
	b.WriteString("\nGoal: ")
	b.WriteString(in.Goal)
	b.WriteString("\nTone: ")
	b.WriteString(string(in.Tone))
	b.WriteString("\nLanguage: ")
	b.WriteString(string(in.Language))
	b.WriteString("\n\nFacts:\n")
	for i, fact := range in.Facts {
		fmt.Fprintf(&b, "%d. %s\n", i+1, fact)
	}
	return b.String()
}

// Draft runs one ADK turn and decodes its structured response.
func (d *ADKDrafter) Draft(ctx context.Context, in email.Input) (email.Draft, error) {
	message := genai.NewContentFromText(Prompt(in), genai.RoleUser)
	sessionID := fmt.Sprintf("draft-%d", d.sessionCounter.Add(1))
	var responseText string

	for event, err := range d.runner.Run(ctx, "user", sessionID, message, adkagent.RunConfig{}) {
		if err != nil {
			return email.Draft{}, fmt.Errorf("run email drafter: %w", err)
		}
		if event == nil || event.Content == nil {
			continue
		}
		responseText = contentText(event.Content)
	}

	if strings.TrimSpace(responseText) == "" {
		return email.Draft{}, fmt.Errorf("run email drafter: %w", errEmptyResponse)
	}

	var draft email.Draft
	if err := json.Unmarshal([]byte(responseText), &draft); err != nil {
		return email.Draft{}, fmt.Errorf("decode email draft: %w", err)
	}
	return draft, nil
}

func modelName() string {
	if name := strings.TrimSpace(os.Getenv("ADK_MODEL")); name != "" {
		return name
	}
	return "gemini-flash-latest"
}

func contentText(content *genai.Content) string {
	var b strings.Builder
	for _, part := range content.Parts {
		if part != nil {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}
