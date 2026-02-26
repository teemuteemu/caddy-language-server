package handler

import (
	"strings"
	"unicode"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// lookupDirectiveDoc returns the Markdown documentation for a directive name.
// It checks the generated map first, then the hand-maintained fallback map.
func lookupDirectiveDoc(name string) (string, bool) {
	if doc, ok := directiveDocs[name]; ok {
		return doc, true
	}
	doc, ok := directiveDocsExtra[name]
	return doc, ok
}

// Hover handles textDocument/hover.
func (h *Handler) Hover(ctx *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := string(params.TextDocument.URI)
	content, ok := h.store.Get(uri)
	if !ok {
		return nil, nil
	}

	word := wordAtPosition(content, params.Position)
	if word == "" {
		return nil, nil
	}

	doc, found := lookupDirectiveDoc(word)
	if !found {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: doc,
		},
	}, nil
}

// wordAtPosition extracts the word under the cursor position.
func wordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	runes := []rune(line)
	col := int(pos.Character)
	if col > len(runes) {
		col = len(runes)
	}

	// Find start of word
	start := col
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}

	// Find end of word
	end := col
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return string(runes[start:end])
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
