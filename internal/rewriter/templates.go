package rewriter

import (
	"bytes"
	"text/template"
)

// DisclaimerData provides contextual variables for disclaimer template rendering.
type DisclaimerData struct {
	EntityName     string
	Attribute      string
	AssertedValue  string
	ExpectedValue  string
	SourceDoc      string
}

// RenderDisclaimer interpolates factual variables into the configured disclaimer template.
func RenderDisclaimer(tmplStr string, data DisclaimerData) (string, error) {
	if tmplStr == "" {
		tmplStr = " [Correction: Official enterprise policy is {{.ExpectedValue}}, not {{.AssertedValue}}]"
	}

	tmpl, err := template.New("disclaimer").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
