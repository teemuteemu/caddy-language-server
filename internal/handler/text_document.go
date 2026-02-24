package handler

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidOpen handles textDocument/didOpen.
func (h *Handler) DidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	text := params.TextDocument.Text
	h.store.Open(uri, text)
	h.Analyze(ctx, uri, text)
	return nil
}

// DidChange handles textDocument/didChange (full sync).
func (h *Handler) DidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	if len(params.ContentChanges) == 0 {
		return nil
	}
	// With full sync, the last change contains the full document text.
	change := params.ContentChanges[len(params.ContentChanges)-1]
	var text string
	switch c := change.(type) {
	case protocol.TextDocumentContentChangeEvent:
		text = c.Text
	case protocol.TextDocumentContentChangeEventWhole:
		text = c.Text
	}
	h.store.Update(uri, text)
	h.Analyze(ctx, uri, text)
	return nil
}

// DidSave handles textDocument/didSave.
func (h *Handler) DidSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	var text string
	if params.Text != nil {
		text = *params.Text
		h.store.Update(uri, text)
	} else {
		var ok bool
		text, ok = h.store.Get(uri)
		if !ok {
			return nil
		}
	}
	h.Analyze(ctx, uri, text)
	return nil
}

// DidClose handles textDocument/didClose.
func (h *Handler) DidClose(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	h.store.Close(uri)
	return nil
}
