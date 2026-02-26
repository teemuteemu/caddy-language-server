package handler

import (
	"caddy-ls/internal/analysis"
	"caddy-ls/internal/parser"
	"sort"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// topLevelDirectives is built from the authoritative KnownTopLevel set so that
// completion items are always in sync with the analyzer's validation rules.
var topLevelDirectives = func() []string {
	names := make([]string, 0, len(analysis.KnownTopLevel))
	for name := range analysis.KnownTopLevel {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}()

// Completion handles textDocument/completion.
func (h *Handler) Completion(ctx *glsp.Context, params *protocol.CompletionParams) (any, error) {
	empty := []protocol.CompletionItem{}

	content, ok := h.store.Get(string(params.TextDocument.URI))
	if !ok {
		return empty, nil
	}

	// When the cursor is in the argument position of an "import" directive,
	// suggest snippet names defined in the current file.
	if partial, ok := importArgPrefix(content, params.Position); ok {
		ast, _ := parser.Parse(content)
		return snippetCompletions(ast, partial), nil
	}

	// Only suggest directives when the cursor is on the first token of the
	// line (not in an argument position after an existing directive/keyword).
	if !atFirstTokenPosition(content, params.Position) {
		return empty, nil
	}

	ast, _ := parser.Parse(content)
	names := completionNamesAt(ast, params.Position.Line)
	if names == nil {
		return empty, nil
	}

	kind := protocol.CompletionItemKindKeyword
	items := make([]protocol.CompletionItem, 0, len(names))
	for _, name := range names {
		n := name
		item := protocol.CompletionItem{
			Label: n,
			Kind:  &kind,
		}
		if doc, ok := directiveDocs[n]; ok {
			item.Documentation = protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: doc,
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// importArgPrefix reports whether the cursor is in the first-argument position
// of an "import" directive on the current line. If so, it returns the partial
// snippet name typed so far (may be empty) and true.
func importArgPrefix(content string, pos protocol.Position) (string, bool) {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return "", false
	}
	line := lines[pos.Line]
	col := int(pos.Character)
	if col > len(line) {
		col = len(line)
	}
	// Normalise indentation.
	prefix := strings.TrimLeft(line[:col], " \t")
	// Must start with "import" followed by at least one space/tab.
	rest, found := strings.CutPrefix(prefix, "import")
	if !found || len(rest) == 0 || (rest[0] != ' ' && rest[0] != '\t') {
		return "", false
	}
	// The (partial) first argument typed so far.
	arg := strings.TrimLeft(rest, " \t")
	// If arg already contains whitespace the cursor is past the first argument.
	if strings.ContainsAny(arg, " \t") {
		return "", false
	}
	return arg, true
}

// snippetCompletions returns CompletionItems for all snippet names defined in f
// whose name starts with partial.
func snippetCompletions(f *parser.File, partial string) []protocol.CompletionItem {
	names := analysis.CollectSnippetNames(f)
	kind := protocol.CompletionItemKindModule
	items := make([]protocol.CompletionItem, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, partial) {
			n := name
			items = append(items, protocol.CompletionItem{
				Label: n,
				Kind:  &kind,
			})
		}
	}
	return items
}

// atFirstTokenPosition reports whether the cursor is still within the first
// non-whitespace token of the current line (i.e. the user is typing a
// directive name, not an argument to one).
func atFirstTokenPosition(content string, pos protocol.Position) bool {
	lines := strings.Split(content, "\n")
	lineIdx := int(pos.Line)
	if lineIdx >= len(lines) {
		return false
	}
	line := lines[lineIdx]
	charIdx := int(pos.Character)
	if charIdx > len(line) {
		charIdx = len(line)
	}
	// Strip leading whitespace; if any whitespace remains, the cursor has
	// already moved past the first token into an argument position.
	trimmed := strings.TrimLeft(line[:charIdx], " \t")
	return !strings.ContainsAny(trimmed, " \t")
}

// containerDirectives is the set of directives whose body accepts the same
// top-level directive set as a site block (routing containers).
var containerDirectives = map[string]bool{
	"handle":        true,
	"handle_path":   true,
	"handle_errors": true,
	"route":         true,
}

// completionNamesAt returns the sorted list of names to complete at cursorLine,
// or nil when the cursor is not in a completable position (outside all site
// blocks, on an address line, or inside a freeform/unknown directive body).
func completionNamesAt(f *parser.File, cursorLine uint32) []string {
	for _, sb := range f.SiteBlocks {
		if cursorLine <= sb.StartLine || cursorLine >= sb.EndLine {
			continue
		}
		return directiveNamesAt(sb.Directives, cursorLine)
	}
	return nil
}

// directiveNamesAt walks a directive list and returns the names to complete at
// cursorLine. It recurses into container directives and returns subdirective
// names when the cursor is inside a directive with known subdirectives.
func directiveNamesAt(directives []*parser.Directive, cursorLine uint32) []string {
	for _, d := range directives {
		if !hasBody(d) || cursorLine <= d.StartLine || cursorLine >= d.EndLine {
			continue
		}
		// Cursor is inside this directive's body block.
		if containerDirectives[d.Name.Value] {
			return directiveNamesAt(d.Body, cursorLine)
		}
		subDirs, known := analysis.SubDirectivesFor(d.Name.Value)
		if !known || subDirs == nil {
			// Unknown or freeform directive — no completions.
			return nil
		}
		names := make([]string, 0, len(subDirs))
		for name := range subDirs {
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}
	// Not inside any directive body → site-block level.
	return topLevelDirectives
}

// hasBody reports whether d has a body block (EndLine > StartLine),
// regardless of whether any sub-directives were parsed inside it.
func hasBody(d *parser.Directive) bool {
	return d.EndLine > d.StartLine
}
