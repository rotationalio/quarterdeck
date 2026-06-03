package tests

import (
	"testing"
)

func TestLoremIpsum(t *testing.T) {
	t.Logf("Name: %s", lorem.Name())
	t.Logf("Email: %s", lorem.Email())
	t.Logf("Words: %s", lorem.Words(10))
	t.Logf("Sentence: %s", lorem.Sentence(5, 12))
	t.Logf("Paragraph: %s", lorem.Paragraph(3, 5, 12))
}
