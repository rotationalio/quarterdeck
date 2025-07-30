package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin/render"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/x/typecase"
)

//go:embed all:static
//go:embed all:templates
var content embed.FS

const (
	partials           = "partials/**/*.html"
	partialsComponents = "partials/components/**/*.html"
)

var (
	includes        = []string{"*.html", "components/*.html"}
	partialsInclude = []string{} // TODO: include partialsComponents if needed

	excludes = map[string]struct{}{
		"partials":   {},
		"components": {},
	}
)

// Static creates StaticFS routes that is used to serve static files.
// For use in gin, call router.StaticFS("/static", web.Static()).
func Static() http.FileSystem {
	staticFiles, err := fs.Sub(content, "static")
	if err != nil {
		panic(errors.Fmt("failed to create static file system: %w", err))
	}
	return http.FS(staticFiles)
}

// Templates returns a Filesystem that contains the HTML templates.
func Templates() fs.FS {
	templateFiles, err := fs.Sub(content, "templates")
	if err != nil {
		panic(errors.Fmt("failed to create static file system: %w", err))
	}
	return templateFiles
}

// Creates a new template renderer from the default templates. This is used to render
// the HTML templates in the templates folder and include additional functions for
// managing rendering in the web application.
func HTMLRender(fsys fs.FS) (render *Render, err error) {
	render = &Render{
		templates: make(map[string]*template.Template),
	}

	var entries []fs.DirEntry
	if entries, err = fs.ReadDir(fsys, "."); err != nil {
		return nil, err
	}

	// Search for all top-level directories to collect templates for (layouts)
	for _, entry := range entries {
		// Skip any excluded directories or files at the top level.
		name := entry.Name()
		if _, ok := excludes[name]; ok || !entry.IsDir() {
			continue
		}

		// Create the includes patterns for the layout directory.
		pattern := fmt.Sprintf("%s/**/*.html", name)
		patternInclude := make([]string, 0, len(includes)+2)
		patternInclude = append(patternInclude, includes...)

		if components := fmt.Sprintf("%s/components/*.html", name); globExists(fsys, components) {
			patternInclude = append(patternInclude, components)
		}

		// Ensure the current layout template is last in the list of templates.
		patternInclude = append(patternInclude, fmt.Sprintf("%s/*.html", name))

		// Add the templates to the renderer.
		if err = render.AddPattern(fsys, pattern, patternInclude...); err != nil {
			return nil, err
		}
	}

	// Add the partials to the templates.
	// Partials are independently rendered with other templates included.
	if err = render.AddPattern(fsys, partials, partialsInclude...); err != nil {
		return nil, err
	}

	return render, nil
}

type Render struct {
	templates map[string]*template.Template
	funcs     template.FuncMap
}

func (r *Render) Instance(name string, data any) render.Render {
	return &render.HTML{
		Template: r.templates[name],
		Name:     filepath.Base(name),
		Data:     data,
	}
}

func (r *Render) AddPattern(fsys fs.FS, pattern string, includes ...string) (err error) {
	var names []string
	if names, err = fs.Glob(fsys, pattern); err != nil {
		return err
	}

	for _, name := range names {
		patterns := make([]string, 0, len(includes)+1)
		patterns = append(patterns, includes...)
		patterns = append(patterns, name)

		tmpl := template.New(name).Funcs(r.FuncMap())
		if r.templates[name], err = tmpl.ParseFS(fsys, patterns...); err != nil {
			return err
		}

		log.Trace().Str("template", name).Strs("patterns", patterns).Msg("parsed template")
	}
	return nil
}

func (r *Render) FuncMap() template.FuncMap {
	if r.funcs == nil {
		r.funcs = template.FuncMap{
			"uppercase": strings.ToUpper,
			"lowercase": strings.ToLower,
			"titlecase": titlecase,
			"rfc3339":   rfc3339,
		}
	}
	return r.funcs
}

func globExists(fsys fs.FS, pattern string) (exists bool) {
	names, _ := fs.Glob(fsys, pattern)
	return len(names) > 0
}

// ===========================================================================
// Template Helpers Functions
// ===========================================================================

func titlecase(s string) string {
	return typecase.Title(s)
}

func rfc3339(t time.Time) string {
	return t.Format(time.RFC3339)
}
