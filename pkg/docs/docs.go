package docs

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

type Documentation []*DocsPage

type DocsPage struct {
	Title    string
	Template string
	Href     string
	Icon     string
	Pages    []*SubPage
}

type SubPage struct {
	Title    string
	Template string
	Rel      string
	parent   *DocsPage
	href     string
}

// Add all new documentation pages here.
var docs = Documentation{
	{
		Title:    "Introduction",
		Template: "docs/introduction/index.html",
		Href:     "",
		Icon:     "atlas",
		Pages: []*SubPage{
			{
				Title:    "Getting Started",
				Template: "docs/introduction/getting-started.html",
				Rel:      "getting-started",
			},
			{
				Title:    "Walkthrough",
				Template: "docs/introduction/walkthrough.html",
				Rel:      "walkthrough",
			},
		},
	},
	{
		Title:    "Security Model",
		Template: "docs/security/index.html",
		Href:     "security",
		Icon:     "shield-alt",
	},
}

func Routes(g *gin.RouterGroup) {
	basePath := g.BasePath()
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	for _, page := range docs {
		// Ensure the page href has no / prefix or suffix
		page.Href = strings.Trim(page.Href, "/")

		// Add Top Level Page
		g.GET(page.Href, page.Handler)

		// Add any sub-pages
		for _, subPage := range page.Pages {
			// Ensure the subpage href has no / prefix or suffix
			subPage.Rel = strings.Trim(subPage.Rel, "/")
			subPage.parent = page

			// Create the subpage route
			g.GET(subPage.Href(), subPage.Handler())

			// Update the internal href with the base path for sidebar processing.
			if page.Href != "" {
				subPage.href = basePath + page.Href + "/" + subPage.Rel
			} else {
				subPage.href = basePath + subPage.Rel
			}
		}

		// Update the href with the base path for template handling.
		page.Href = basePath + page.Href
	}
}

func (d *DocsPage) Handler(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Docs"] = docs
	ctx["Page"] = d
	c.HTML(http.StatusOK, d.Template, ctx)
}

func (d *SubPage) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := scene.New(c)
		ctx["Docs"] = docs
		ctx["Parent"] = d.parent
		ctx["Page"] = d
		c.HTML(http.StatusOK, d.Template, ctx)
	}
}

func (d *SubPage) Href() string {
	if d.href == "" {
		return d.parent.Href + "/" + d.Rel
	}
	return d.href
}

func (d *DocsPage) Active(p any) bool {
	switch p := p.(type) {
	case *DocsPage:
		return d == p
	case *SubPage:
		return d == p.parent
	}
	return false
}

func (d *SubPage) Active(p any) bool {
	switch p := p.(type) {
	case *DocsPage:
		return d.parent == p
	case *SubPage:
		return d == p
	}
	return false
}

// Return documentation for testing.
func Docs() Documentation {
	return docs
}
