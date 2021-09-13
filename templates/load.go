package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

const (
	bodySuffix    = "_body.html"
	subjectSuffix = "_subject.txt"
)

//go:embed sources/*
var Sources embed.FS

func Load() (Templates, error) {
	templates := make(Templates)
	entries, err := fs.ReadDir(Sources, "sources")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), bodySuffix) {
			name := strings.TrimSuffix(entry.Name(), bodySuffix)

			// Load the html body
			body, err := Sources.ReadFile(fmt.Sprintf("sources/%s", entry.Name()))
			if err != nil {
				return nil, err
			}

			// Inline the css
			html, err := inlineCSS(body)
			if err != nil {
				return nil, err
			}

			// Load the email subject template
			expectedSubjectFilename := fmt.Sprintf("sources/%s%s", name, subjectSuffix)
			subject, err := Sources.ReadFile(expectedSubjectFilename)
			if err != nil {
				return nil, err
			}

			template, err := NewPrecompiledTemplate(TemplateName(name), string(subject), html)
			if err != nil {
				return nil, err
			}

			templates[template.Name()] = template
		}
	}

	return templates, nil
}
