package handler

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialize handles the LSP initialize request and returns server capabilities.
func (h *Handler) Initialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	return protocol.InitializeResult{
		Capabilities: h.CreateServerCapabilities(),
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "caddy-ls",
			Version: strPtr(version),
		},
	}, nil
}

// Initialized is called after the client acknowledges initialize.
func (h *Handler) Initialized(ctx *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

// Shutdown gracefully shuts the server down.
func (h *Handler) Shutdown(ctx *glsp.Context) error {
	return nil
}

// SetTrace updates the trace level (no-op for now).
func (h *Handler) SetTrace(ctx *glsp.Context, params *protocol.SetTraceParams) error {
	return nil
}

// CreateServerCapabilities returns the capabilities advertised to the client.
func (h *Handler) CreateServerCapabilities() protocol.ServerCapabilities {
	syncKind := protocol.TextDocumentSyncKindFull
	triggerChars := []string{"."}

	return protocol.ServerCapabilities{
		TextDocumentSync: &protocol.TextDocumentSyncOptions{
			OpenClose: boolPtr(true),
			Change:    &syncKind,
			Save:      &protocol.SaveOptions{IncludeText: boolPtr(true)},
		},
		HoverProvider: true,
		CompletionProvider: &protocol.CompletionOptions{
			TriggerCharacters: triggerChars,
		},
	}
}

func boolPtr(b bool) *bool { return &b }
