package tests

import (
	"math/rand"
	"strings"
	"time"

	"go.rtnl.ai/x/typecase"
)

func init() {
	lorem = NewLoremIpsum()
}

var lorem *LoremIpsum

// Global source pool of standard Lorem Ipsum words
var wordPool = []string{
	"lorem", "ipsum", "dolor", "sit", "amet", "consectetur", "adipiscing", "elit",
	"sed", "do", "eiusmod", "tempor", "incididunt", "ut", "labore", "et", "dolore",
	"magna", "aliqua", "ut", "enim", "ad", "minim", "veniam", "quis", "nostrud",
	"exercitation", "ullamco", "laboris", "nisi", "ut", "aliquip", "ex", "ea",
	"commodo", "consequat", "duis", "aute", "irure", "dolor", "in", "reprehenderit",
	"in", "voluptate", "velit", "esse", "cillum", "dolore", "eu", "fugiat", "nulla",
	"pariatur", "excepteur", "sint", "occaecat", "cupidatat", "non", "proident",
	"sunt", "in", "culpa", "qui", "officia", "deserunt", "mollit", "anim", "id", "est",
	"laborum", "jane", "john", "doe", "com", "net", "org", "io", "ai", "dev", "test",
	"example", "user", "admin", "manager", "supervisor", "employee", "customer",
	"frank", "support", "system", "fred", "amanda", "jill", "alice", "bob", "hackensack",
	"az", "re",
}

// LoremIpsum holds the configuration for text creation
type LoremIpsum struct {
	r *rand.Rand
}

// NewLoremIpsum initializes the random number generator
func NewLoremIpsum() *LoremIpsum {
	source := rand.NewSource(time.Now().UnixNano())
	return &LoremIpsum{r: rand.New(source)}
}

// Words returns a space-separated string containing n random words
func (g *LoremIpsum) Words(count int) string {
	if count <= 0 {
		return ""
	}

	words := make([]string, count)
	for i := 0; i < count; i++ {
		words[i] = wordPool[g.r.Intn(len(wordPool))]
	}

	return strings.Join(words, " ")
}

// Sentence creates a single capitalized string ending with a period
func (g *LoremIpsum) Sentence(minWords, maxWords int) string {
	wordCount := minWords
	if maxWords > minWords {
		wordCount = minWords + g.r.Intn(maxWords-minWords+1)
	}

	rawText := g.Words(wordCount)
	if rawText == "" {
		return ""
	}

	// Capitalize the first letter and add a period at the end
	return strings.ToUpper(string(rawText[0])) + rawText[1:] + "."
}

// Paragraph joins multiple randomized sentences together
func (g *LoremIpsum) Paragraph(sentenceCount, minWords, maxWords int) string {
	sentences := make([]string, sentenceCount)
	for i := 0; i < sentenceCount; i++ {
		sentences[i] = g.Sentence(minWords, maxWords)
	}
	return strings.Join(sentences, " ")
}

func (g *LoremIpsum) Email() string {
	return g.Words(1) + "@" + g.Words(1) + "." + g.Words(1)
}

func (g *LoremIpsum) Name() string {
	return typecase.Title(g.Words(2))
}
