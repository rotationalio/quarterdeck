package emails

import (
	"embed"
	"html/template"
	"io/fs"
	"path/filepath"
)

const (
	// Email templates must be provided in this directory and are loaded at compile time
	templatesDir = "templates"

	// Partials are included when rendering templates for composability and reuse
	partialsDir = "partials"
)

//go:embed templates/*.html templates/*.txt templates/partials/*html
var files embed.FS

// Load templates
func LoadTemplates() (templates map[string]*template.Template) {
	var (
		err           error
		templateFiles []fs.DirEntry
	)

	templates = make(map[string]*template.Template)
	if templateFiles, err = fs.ReadDir(files, templatesDir); err != nil {
		panic(err)
	}

	// Each template needs to be parsed independently to ensure that define directives
	// are not overriden if they have the same name; e.g. to use the base template.
	for _, file := range templateFiles {
		if file.IsDir() {
			continue
		}

		// Each template will be accessible by its base name in the global map
		patterns := make([]string, 0, 2)
		patterns = append(patterns, filepath.Join(templatesDir, file.Name()))
		switch filepath.Ext(file.Name()) {
		case ".html":
			patterns = append(patterns, filepath.Join(templatesDir, partialsDir, "*.html"))
		}

		templates[file.Name()] = template.Must(template.ParseFS(files, patterns...))
	}

	return templates
}
