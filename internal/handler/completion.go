package handler

import (
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
	kind := protocol.CompletionItemKindKeyword
	items := make([]protocol.CompletionItem, 0, len(topLevelDirectives))
	for _, name := range topLevelDirectives {
		n := name // capture
		items = append(items, protocol.CompletionItem{
			Label: n,
			Kind:  &kind,
		})
	}
	return items, nil
}
