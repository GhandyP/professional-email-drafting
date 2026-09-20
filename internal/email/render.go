package email

import (
	"html/template"
	"strings"
)

const draftHTMLTemplate = `<!doctype html>
<html lang="{{.Language}}">
<head>
<meta charset="utf-8">
<title>{{.Subject}}</title>
</head>
<body>
<h1>{{.Subject}}</h1>
{{range .Paragraphs}}<p>{{range $index, $line := .}}{{if $index}}<br>{{end}}{{$line}}{{end}}</p>
{{end}}</body>
</html>`

var draftHTML = template.Must(template.New("draft").Parse(draftHTMLTemplate))

// RenderPlain renders the draft as a plain-text message.
func (d Draft) RenderPlain() string {
	return "Subject: " + strings.TrimSpace(d.Subject) + "\n\n" + strings.Trim(d.Body, "\n") + "\n"
}

// RenderHTML renders the draft as an escaped HTML preview document.
func (d Draft) RenderHTML() (string, error) {
	data := struct {
		Language   string
		Subject    string
		Paragraphs [][]string
	}{
		Language:   string(d.Language),
		Subject:    strings.TrimSpace(d.Subject),
		Paragraphs: htmlParagraphs(d.Body),
	}

	var output strings.Builder
	if err := draftHTML.Execute(&output, data); err != nil {
		return "", err
	}
	return output.String(), nil
}

func htmlParagraphs(body string) [][]string {
	paragraphs := make([][]string, 0)
	paragraph := make([]string, 0)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(paragraph) > 0 {
				paragraphs = append(paragraphs, paragraph)
				paragraph = nil
			}
			continue
		}
		paragraph = append(paragraph, line)
	}
	if len(paragraph) > 0 {
		paragraphs = append(paragraphs, paragraph)
	}
	return paragraphs
}
