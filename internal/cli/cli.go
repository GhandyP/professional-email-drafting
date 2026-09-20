package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GhandyP/professional-email-drafting/internal/agent"
	"github.com/GhandyP/professional-email-drafting/internal/email"
	"github.com/GhandyP/professional-email-drafting/internal/sender"
)

// Deps configures the CLI: a nil Drafter builds the Gemini drafter from the environment, a nil Sender follows the -sender flag, and a nil Now uses time.Now.
type Deps struct {
	Drafter agent.Drafter
	Sender  sender.Sender
	Now     func() time.Time
}

// Run executes one command line and returns the process exit code.
func Run(ctx context.Context, args []string, deps Deps, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	var facts factFlag
	var recipient string
	var goal string
	var tone string
	var language string
	var inputPath string
	var approver string
	var jsonOutput bool
	var htmlOutput bool
	var senderName string
	var timeout time.Duration

	flags := flag.NewFlagSet("professional-email-drafting", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&recipient, "recipient", "", "recipient email address")
	flags.StringVar(&goal, "goal", "", "email goal")
	flags.StringVar(&tone, "tone", string(email.ToneConcise), "email tone")
	flags.StringVar(&language, "language", string(email.LanguageEnglish), "email language")
	flags.Var(&facts, "fact", "supporting fact; may be repeated")
	flags.StringVar(&inputPath, "input", "", "path to an input JSON file")
	flags.StringVar(&approver, "approve", "", "approver identity")
	flags.BoolVar(&jsonOutput, "json", false, "print the draft as JSON")
	flags.BoolVar(&htmlOutput, "html", false, "print the HTML preview")
	flags.StringVar(&senderName, "sender", "disabled", "sender (disabled or recorder)")
	flags.DurationVar(&timeout, "timeout", 60*time.Second, "drafting timeout")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	input, err := resolveInput(flags, inputPath, recipient, goal, tone, language, facts)
	if err != nil {
		fmt.Fprintf(stderr, "input failed: %v\n", err)
		return 2
	}
	if err := input.Validate(); err != nil {
		printInputProblems(stderr, err)
		return 2
	}
	if senderName != "disabled" && senderName != "recorder" {
		fmt.Fprintf(stderr, "configuration error: unknown sender %q (want disabled or recorder)\n", senderName)
		return 2
	}

	drafter := deps.Drafter
	if drafter == nil {
		drafter, err = agent.NewGeminiDrafter(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "configuration error: %v\n", err)
			return 2
		}
	}

	draftContext, cancel := context.WithTimeout(ctx, timeout)
	draft, err := drafter.Draft(draftContext, input)
	cancel()
	if err != nil {
		fmt.Fprintf(stderr, "drafting failed: %v\n", err)
		return 1
	}
	if err := draft.Validate(input); err != nil {
		printDraftFindings(stderr, err)
		return 1
	}

	if jsonOutput {
		_ = json.NewEncoder(stdout).Encode(draft)
	} else if htmlOutput {
		html, err := draft.RenderHTML()
		if err != nil {
			fmt.Fprintf(stderr, "approval failed: %v\n", err)
			return 1
		}
		_, _ = io.WriteString(stdout, html)
	} else {
		_, _ = io.WriteString(stdout, plainPreview(draft))
	}

	if approver == "" {
		fmt.Fprintln(stderr, "Dry run: no approval was recorded and nothing was sent.")
		return 0
	}

	now := deps.Now
	if now == nil {
		now = time.Now
	}
	approval, err := email.Approve(draft, "draft-1", approver, now())
	if err != nil {
		fmt.Fprintf(stderr, "approval failed: %v\n", err)
		return 1
	}
	approvedMessage, err := sender.Approved(draft, input, approval)
	if err != nil {
		fmt.Fprintf(stderr, "approval failed: %v\n", err)
		return 1
	}

	resolvedSender := deps.Sender
	if resolvedSender == nil {
		switch senderName {
		case "disabled":
			resolvedSender = sender.DisabledSender{}
		case "recorder":
			resolvedSender = &sender.Recorder{}
		}
	}
	if err := resolvedSender.Send(ctx, approvedMessage); err != nil {
		fmt.Fprintf(stderr, "Send refused: %v\n", err)
		return 3
	}
	fmt.Fprintf(stderr, "Approved by %s at %s: the %s sender accepted the message.\n", approval.ApprovedBy, approval.ApprovedAt.Format(time.RFC3339), senderName)
	return 0
}

type factFlag []string

func (f *factFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *factFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func resolveInput(flags *flag.FlagSet, inputPath, recipient, goal, tone, language string, facts factFlag) (email.Input, error) {
	set := make(map[string]bool)
	flags.Visit(func(flag *flag.Flag) {
		set[flag.Name] = true
	})

	var input email.Input
	if set["input"] {
		contents, err := os.ReadFile(inputPath)
		if err != nil && !filepath.IsAbs(inputPath) {
			contents, err = os.ReadFile(filepath.Join("..", "..", inputPath))
		}
		if err != nil {
			return email.Input{}, fmt.Errorf("read %q: %w", inputPath, err)
		}
		if err := json.Unmarshal(contents, &input); err != nil {
			return email.Input{}, fmt.Errorf("decode %q: %w", inputPath, err)
		}
	} else {
		input.Tone = email.Tone(tone)
		input.Language = email.Language(language)
	}

	if set["recipient"] {
		input.Recipient = recipient
	}
	if set["goal"] {
		input.Goal = goal
	}
	if set["tone"] {
		input.Tone = email.Tone(tone)
	}
	if set["language"] {
		input.Language = email.Language(language)
	}

	input.Recipient = strings.TrimSpace(input.Recipient)
	input.Goal = strings.TrimSpace(input.Goal)
	input.Facts = append(input.Facts, facts...)
	input.Facts = nonBlankFacts(input.Facts)
	return input, nil
}

func nonBlankFacts(facts []string) []string {
	result := make([]string, 0, len(facts))
	for _, fact := range facts {
		fact = strings.TrimSpace(fact)
		if fact != "" {
			result = append(result, fact)
		}
	}
	return result
}

func printInputProblems(w io.Writer, err error) {
	fmt.Fprintln(w, "invalid input:")
	var validationErr *email.ValidationError
	if errors.As(err, &validationErr) {
		for _, problem := range validationErr.Problems {
			fmt.Fprintf(w, "  %s: %s\n", problem.Field, problem.Message)
		}
		return
	}
	fmt.Fprintf(w, "  input: %s\n", err)
}

func printDraftFindings(w io.Writer, err error) {
	fmt.Fprintln(w, "draft rejected:")
	var draftErr *email.DraftError
	if errors.As(err, &draftErr) {
		for _, finding := range draftErr.Findings {
			fmt.Fprintf(w, "  %s: %s: %s\n", finding.Field, finding.Code, finding.Message)
		}
		return
	}
	fmt.Fprintf(w, "  draft: validation: %s\n", err)
}

func plainPreview(draft email.Draft) string {
	var b strings.Builder
	b.WriteString("Draft preview\n\n")
	b.WriteString("Subject: ")
	b.WriteString(strings.TrimSpace(draft.Subject))
	b.WriteString("\n\n")

	b.WriteString("Fact-backed statements:\n")
	factNumber := 0
	for _, claim := range draft.Claims {
		if claim.Fact == email.NoFact {
			continue
		}
		factNumber++
		fmt.Fprintf(&b, "  [fact %d] %s\n", claim.Fact+1, claim.Text)
	}
	if factNumber == 0 {
		b.WriteString("  (none)\n")
	}

	b.WriteString("Framing statements:\n")
	framingCount := 0
	for _, claim := range draft.Claims {
		if claim.Fact != email.NoFact {
			continue
		}
		framingCount++
		fmt.Fprintf(&b, "  %s\n", claim.Text)
	}
	if framingCount == 0 {
		b.WriteString("  (none)\n")
	}

	b.WriteString("Unresolved items:\n")
	if len(draft.Unresolved) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, unresolved := range draft.Unresolved {
			fmt.Fprintf(&b, "  %s\n", unresolved)
		}
	}

	b.WriteString("Message:\n")
	b.WriteString(draft.RenderPlain())
	return b.String()
}
