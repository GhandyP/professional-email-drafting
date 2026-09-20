package email

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDraftRenderPlainMatchesTheMessageLayout(t *testing.T) {
	d := validDraft()

	want := "Subject: Invoice 42 payment\n\n" +
		"Invoice 42 was paid on 2026-09-01.\n" +
		"Payment came from the operations account.\n"

	if got := d.RenderPlain(); got != want {
		t.Fatalf("RenderPlain() = %q, want %q", got, want)
	}
}

func TestDraftRenderPlainKeepsBodyIndentation(t *testing.T) {
	d := validDraft()
	d.Body = "  indented line"

	want := "Subject: Invoice 42 payment\n\n  indented line\n"

	if got := d.RenderPlain(); got != want {
		t.Fatalf("RenderPlain() = %q, want %q", got, want)
	}
}

func TestDraftRenderPlainIgnoresSurroundingNewlines(t *testing.T) {
	padded := validDraft()
	padded.Body = "\n\nInvoice 42 was paid on 2026-09-01.\n\n"

	plain := validDraft()
	plain.Body = "Invoice 42 was paid on 2026-09-01."

	if got, want := padded.RenderPlain(), plain.RenderPlain(); got != want {
		t.Fatalf("RenderPlain() = %q, want %q", got, want)
	}
}

func TestDraftRenderPlainTrimsTheSubject(t *testing.T) {
	d := validDraft()
	d.Subject = "  Invoice 42 payment  "

	if got, want := d.RenderPlain(), validDraft().RenderPlain(); got != want {
		t.Fatalf("RenderPlain() = %q, want %q", got, want)
	}
}

func TestDraftRenderPlainDoesNotMutateTheDraft(t *testing.T) {
	d := validDraft()
	d.Body = "\n\nPadded body\n\n"

	want := d
	want.Claims = slices.Clone(d.Claims)
	want.Unresolved = slices.Clone(d.Unresolved)

	d.RenderPlain()

	if !reflect.DeepEqual(d, want) {
		t.Fatalf("RenderPlain() mutated the draft: %+v", d)
	}
}

func TestDraftRenderHTMLDocumentShape(t *testing.T) {
	d := validDraft()
	d.Body = "Invoice 42 was paid on 2026-09-01."

	out, err := d.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML() error = %v, want nil", err)
	}

	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		`<meta charset="utf-8">`,
		"<title>Invoice 42 payment</title>",
		"<h1>Invoice 42 payment</h1>",
		"<p>Invoice 42 was paid on 2026-09-01.</p>",
		"</body>",
		"</html>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("RenderHTML() output does not contain %q:\n%s", want, out)
		}
	}
}

func TestDraftRenderHTMLUsesTheDraftLanguage(t *testing.T) {
	d := validDraft()
	d.Language = LanguageSpanish

	out, err := d.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML() error = %v, want nil", err)
	}
	if !strings.Contains(out, `<html lang="es">`) {
		t.Fatalf("RenderHTML() output does not declare the draft language:\n%s", out)
	}
}

func TestDraftRenderHTMLEscapesSubjectAndBody(t *testing.T) {
	d := validDraft()
	d.Subject = "Tom & Ana"
	d.Body = "<script>alert(1)</script>"

	out, err := d.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML() error = %v, want nil", err)
	}

	if strings.Contains(out, "<script>") {
		t.Fatalf("RenderHTML() emitted unescaped script content:\n%s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("RenderHTML() did not escape the body:\n%s", out)
	}
	if !strings.Contains(out, "Tom &amp; Ana") {
		t.Fatalf("RenderHTML() did not escape the subject:\n%s", out)
	}
}

func TestDraftRenderHTMLParagraphsAndLineBreaks(t *testing.T) {
	d := validDraft()
	d.Body = "First line\nSecond line\n\nSecond paragraph."

	out, err := d.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML() error = %v, want nil", err)
	}

	if got := strings.Count(out, "<p>"); got != 2 {
		t.Fatalf("paragraph count = %d, want 2:\n%s", got, out)
	}
	if got := strings.Count(out, "<br>"); got != 1 {
		t.Fatalf("line break count = %d, want 1:\n%s", got, out)
	}
	if !strings.Contains(out, "<p>First line<br>Second line</p>") {
		t.Fatalf("RenderHTML() did not keep the paragraph lines together:\n%s", out)
	}
	if !strings.Contains(out, "<p>Second paragraph.</p>") {
		t.Fatalf("RenderHTML() did not render the second paragraph:\n%s", out)
	}
}

func TestDraftRenderHTMLCollapsesBlankLines(t *testing.T) {
	d := validDraft()
	d.Body = "A\n\n\n\nB"

	out, err := d.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML() error = %v, want nil", err)
	}

	if got := strings.Count(out, "<p>"); got != 2 {
		t.Fatalf("paragraph count = %d, want 2:\n%s", got, out)
	}
}
