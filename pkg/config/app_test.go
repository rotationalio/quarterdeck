package config_test

import (
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

func TestEmailTemplate_LoadTemplateContent(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		dir := t.TempDir()
		htmlPath := filepath.Join(dir, "welcome.html")
		textPath := filepath.Join(dir, "welcome.txt")
		require.NoError(t, os.WriteFile(htmlPath, []byte("<p>HTML body</p>"), 0644))
		require.NoError(t, os.WriteFile(textPath, []byte("Plain text body"), 0644))

		et := config.EmailTemplate{HTMLPath: htmlPath, TextPath: textPath}
		err := et.LoadTemplateContent()
		require.NoError(t, err)
		require.Equal(t, template.HTML("<p>HTML body</p>"), et.HTMLContent())
		require.Equal(t, "Plain text body", et.TextContent())
	})

	t.Run("OnlyOnePath", func(t *testing.T) {
		t.Run("HTMLOnly", func(t *testing.T) {
			dir := t.TempDir()
			htmlPath := filepath.Join(dir, "welcome.html")
			require.NoError(t, os.WriteFile(htmlPath, []byte("<p>Only HTML</p>"), 0644))

			et := config.EmailTemplate{HTMLPath: htmlPath}
			err := et.LoadTemplateContent()
			require.NoError(t, err)
			require.Equal(t, template.HTML("<p>Only HTML</p>"), et.HTMLContent())
			require.Equal(t, "", et.TextContent())
		})

		t.Run("TextOnly", func(t *testing.T) {
			dir := t.TempDir()
			textPath := filepath.Join(dir, "welcome.txt")
			require.NoError(t, os.WriteFile(textPath, []byte("Only text"), 0644))

			et := config.EmailTemplate{TextPath: textPath}
			err := et.LoadTemplateContent()
			require.NoError(t, err)
			require.Equal(t, template.HTML(""), et.HTMLContent())
			require.Equal(t, "Only text", et.TextContent())
		})
	})

	t.Run("PathError", func(t *testing.T) {
		dir := t.TempDir()
		missingPath := filepath.Join(dir, "nonexistent.txt")

		et := config.EmailTemplate{HTMLPath: missingPath, TextPath: missingPath}
		err := et.LoadTemplateContent()
		require.ErrorIs(t, err, fs.ErrNotExist, "expected file not exist error")
	})
}
