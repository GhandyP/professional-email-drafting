package agent

import (
	"context"
	"errors"
	"iter"
	"reflect"
	"slices"
	"strings"
	"testing"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

	"github.com/GhandyP/professional-email-drafting/internal/email"
)

const validResponse = `{
  "subject": "Invoice 42 payment",
  "body": "Invoice 42 was paid on 2026-09-01.",
  "tone": "concise",
  "language": "en",
  "claims": [{"text": "Invoice 42 was paid on 2026-09-01.", "fact": 0}],
  "unresolved": []
}`

type fakeModel struct {
	response string
	err      error
	requests []*model.LLMRequest
}

func (m *fakeModel) Name() string { return "fake-model" }

func (m *fakeModel) GenerateContent(_ context.Context, req *model.LLMRequest, _ bool) iter.Seq2[*model.LLMResponse, error] {
	m.requests = append(m.requests, req)

	return func(yield func(*model.LLMResponse, error) bool) {
		if m.err != nil {
			yield(nil, m.err)
			return
		}
		yield(&model.LLMResponse{
			Content:      genai.NewContentFromText(m.response, genai.RoleModel),
			TurnComplete: true,
		}, nil)
	}
}

func testInput() email.Input {
	return email.Input{
		Recipient: "ana@example.com",
		Facts:     []string{"Invoice 42 was paid on 2026-09-01."},
		Goal:      "Confirm receipt of the payment.",
		Tone:      email.ToneConcise,
		Language:  email.LanguageEnglish,
	}
}

func newTestDrafter(t *testing.T, m model.LLM) *ADKDrafter {
	t.Helper()

	drafter, err := NewADKDrafter(m)
	if err != nil {
		t.Fatalf("NewADKDrafter() error = %v, want nil", err)
	}
	return drafter
}

func TestInstructionStatesTheGroundingRules(t *testing.T) {
	for _, want := range []string{"facts", "claims", "unresolved", "framing", "-1", "placeholder", "quoted"} {
		if !strings.Contains(Instruction, want) {
			t.Fatalf("Instruction does not mention %q:\n%s", want, Instruction)
		}
	}
}

func TestPromptCarriesFactsGoalToneAndLanguage(t *testing.T) {
	in := testInput()

	for _, want := range []string{
		in.Recipient,
		"1. " + in.Facts[0],
		in.Goal,
		string(in.Tone),
		string(in.Language),
	} {
		if !strings.Contains(Prompt(in), want) {
			t.Fatalf("Prompt() does not contain %q:\n%s", want, Prompt(in))
		}
	}
}

func TestOutputSchemaDescribesTheDraftContract(t *testing.T) {
	schema := OutputSchema()

	if schema.Type != genai.TypeObject {
		t.Fatalf("schema type = %v, want %v", schema.Type, genai.TypeObject)
	}
	for _, property := range []string{"subject", "body", "tone", "language", "claims", "unresolved"} {
		if schema.Properties[property] == nil {
			t.Fatalf("schema is missing property %q", property)
		}
	}
	wantRequired := []string{"subject", "body", "tone", "language", "claims", "unresolved"}
	if !slices.Equal(schema.Required, wantRequired) {
		t.Fatalf("schema required = %v, want %v", schema.Required, wantRequired)
	}
	if schema.Properties["claims"].Type != genai.TypeArray {
		t.Fatalf("claims type = %v, want %v", schema.Properties["claims"].Type, genai.TypeArray)
	}
	if schema.Properties["claims"].Items == nil {
		t.Fatal("claims items are missing")
	}
	if schema.Properties["claims"].Items.Properties["text"] == nil {
		t.Fatal("claims items are missing the text property")
	}
	if schema.Properties["claims"].Items.Properties["fact"] == nil {
		t.Fatal("claims items are missing the fact property")
	}
	if schema.Properties["unresolved"].Type != genai.TypeArray {
		t.Fatalf("unresolved type = %v, want %v", schema.Properties["unresolved"].Type, genai.TypeArray)
	}
	if schema.Properties["unresolved"].Items == nil {
		t.Fatal("unresolved items are missing")
	}
}

func TestOutputSchemaMatchesTheDeclaredTonesAndLanguages(t *testing.T) {
	schema := OutputSchema()

	tones := make([]string, 0, len(email.Tones()))
	for _, tone := range email.Tones() {
		tones = append(tones, string(tone))
	}
	if got := schema.Properties["tone"].Enum; !slices.Equal(got, tones) {
		t.Fatalf("tone enum = %v, want %v", got, tones)
	}

	languages := make([]string, 0, len(email.Languages()))
	for _, language := range email.Languages() {
		languages = append(languages, string(language))
	}
	if got := schema.Properties["language"].Enum; !slices.Equal(got, languages) {
		t.Fatalf("language enum = %v, want %v", got, languages)
	}
}

func TestADKDrafterParsesTheStructuredResponse(t *testing.T) {
	drafter := newTestDrafter(t, &fakeModel{response: validResponse})
	in := testInput()

	draft, err := drafter.Draft(context.Background(), in)
	if err != nil {
		t.Fatalf("Draft() error = %v, want nil", err)
	}
	if draft.Subject != "Invoice 42 payment" {
		t.Fatalf("Subject = %q, want %q", draft.Subject, "Invoice 42 payment")
	}
	if draft.Body != "Invoice 42 was paid on 2026-09-01." {
		t.Fatalf("Body = %q, want %q", draft.Body, "Invoice 42 was paid on 2026-09-01.")
	}
	if len(draft.Claims) != 1 || draft.Claims[0].Fact != 0 {
		t.Fatalf("Claims = %+v, want one claim on fact 0", draft.Claims)
	}
	if err := draft.Validate(in); err != nil {
		t.Fatalf("the parsed draft does not satisfy the domain contract: %v", err)
	}
}

func TestADKDrafterSendsThePromptToTheModel(t *testing.T) {
	fake := &fakeModel{response: validResponse}
	drafter := newTestDrafter(t, fake)

	if _, err := drafter.Draft(context.Background(), testInput()); err != nil {
		t.Fatalf("Draft() error = %v, want nil", err)
	}
	if len(fake.requests) == 0 {
		t.Fatal("the model received no request")
	}
	if !requestContains(fake.requests[0], testInput().Facts[0]) {
		t.Fatal("the model request does not carry the supplied facts")
	}
}

func TestADKDrafterRejectsAnEmptyResponse(t *testing.T) {
	drafter := newTestDrafter(t, &fakeModel{response: ""})

	draft, err := drafter.Draft(context.Background(), testInput())
	if err == nil {
		t.Fatal("Draft() = nil error, want an error for an empty response")
	}
	if !reflect.DeepEqual(draft, email.Draft{}) {
		t.Fatalf("Draft() = %+v, want the zero Draft on error", draft)
	}
}

func TestADKDrafterRejectsInvalidJSON(t *testing.T) {
	drafter := newTestDrafter(t, &fakeModel{response: "not json at all"})

	if _, err := drafter.Draft(context.Background(), testInput()); err == nil {
		t.Fatal("Draft() = nil error, want an error for a response that is not JSON")
	}
}

func TestADKDrafterPropagatesModelErrors(t *testing.T) {
	modelErr := errors.New("model exploded")
	drafter := newTestDrafter(t, &fakeModel{err: modelErr})

	_, err := drafter.Draft(context.Background(), testInput())
	if !errors.Is(err, modelErr) {
		t.Fatalf("Draft() error = %v, want %v", err, modelErr)
	}
}

func TestNewGeminiDrafterRequiresAnAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")

	drafter, err := NewGeminiDrafter(context.Background())
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("NewGeminiDrafter() error = %v, want %v", err, ErrMissingAPIKey)
	}
	if drafter != nil {
		t.Fatalf("NewGeminiDrafter() = %v, want nil", drafter)
	}
}

func TestNewGeminiDrafterBuildsWithAnAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key-that-is-never-used-for-a-request")

	drafter, err := NewGeminiDrafter(context.Background())
	if err != nil {
		t.Fatalf("NewGeminiDrafter() error = %v, want nil", err)
	}
	if drafter == nil {
		t.Fatal("NewGeminiDrafter() = nil, want a drafter")
	}
}

func requestContains(req *model.LLMRequest, want string) bool {
	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if strings.Contains(part.Text, want) {
				return true
			}
		}
	}
	return false
}
