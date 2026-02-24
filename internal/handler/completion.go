package handler

import (
	"caddy-ls/internal/parser"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// topLevelDirectives are the completion items offered at the top level of a site block.
var topLevelDirectives = []string{
	"http",
	"https",
	"tls",
	"reverse_proxy",
	"file_server",
	"encode",
	"log",
	"header",
	"root",
	"php_fastcgi",
	"redir",
	"rewrite",
	"respond",
	"route",
	"handle",
	"handle_path",
	"handle_errors",
	"basicauth",
	"templates",
}

// Completion handles textDocument/completion.
func (h *Handler) Completion(ctx *glsp.Context, params *protocol.CompletionParams) (any, error) {
	empty := []protocol.CompletionItem{}

	content, ok := h.store.Get(string(params.TextDocument.URI))
	if !ok {
		return empty, nil
	}

	// Only suggest directives when the cursor is on the first token of the
	// line (not in an argument position after an existing directive/keyword).
	if !atFirstTokenPosition(content, params.Position) {
		return empty, nil
	}

	// Use the AST to verify the cursor is at site-block level, not inside
	// a directive body block (e.g. reverse_proxy { … } or tls { … }).
	ast, _ := parser.Parse(content)
	if !atSiteBlockLevel(ast, params.Position.Line) {
		return empty, nil
	}

	kind := protocol.CompletionItemKindKeyword
	items := make([]protocol.CompletionItem, 0, len(topLevelDirectives))
	for _, name := range topLevelDirectives {
		n := name
		items = append(items, protocol.CompletionItem{
			Label: n,
			Kind:  &kind,
		})
	}
	return items, nil
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

// atSiteBlockLevel returns true when the cursor is at a position where a
// top-level site-block directive is expected: either directly inside a site
// block, or inside a container directive (handle, handle_path, route, …)
// that accepts the same directive set.
func atSiteBlockLevel(f *parser.File, cursorLine uint32) bool {
	for _, sb := range f.SiteBlocks {
		if cursorLine <= sb.StartLine || cursorLine >= sb.EndLine {
			continue
		}
		return atDirectiveListLevel(sb.Directives, cursorLine)
	}
	return false
}

// atDirectiveListLevel checks whether cursorLine is at the "top level" of a
// directive list — not inside any directive's body — or inside a container
// directive that recursively accepts the same directive set.
func atDirectiveListLevel(directives []*parser.Directive, cursorLine uint32) bool {
	for _, d := range directives {
		if !hasBody(d) || cursorLine <= d.StartLine || cursorLine >= d.EndLine {
			continue
		}
		// Cursor is inside this directive's body block.
		if containerDirectives[d.Name.Value] {
			// Container: recurse to check the inner directive list.
			return atDirectiveListLevel(d.Body, cursorLine)
		}
		return false
	}
	// Cursor is not inside any directive body → at the top level of this list.
	return true
}

// hasBody reports whether d has a body block (EndLine > StartLine),
// regardless of whether any sub-directives were parsed inside it.
func hasBody(d *parser.Directive) bool {
	return d.EndLine > d.StartLine
}
