package web

import (
	"fmt"
	"html/template"
	"io/fs"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// Maintenance returns an HTML template renderer that only has the maintenance mode
// index.html template inside of it for fast rendering.
func Maintenance() (render *Render, err error) {
	var fsys fs.FS
	if fsys, err = fs.Sub(content, "templates"); err != nil {
		return nil, errors.Fmt("failed to create maintenance file system: %w", err)
	}

	render = &Render{
		templates: make(map[string]*template.Template),
	}

	if err = render.AddPattern(fsys, "maintenance/index.html"); err != nil {
		return nil, errors.Fmt("failed to add maintenance index template: %w", err)
	}

	fmt.Printf("%v\n", render.templates)
	return render, nil
}
