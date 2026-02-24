package handler

import (
	"caddy-ls/internal/analysis"
	"caddy-ls/internal/parser"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const version = "0.0.1"

// Analyze parses and analyzes content, then publishes diagnostics for uri.
func (h *Handler) Analyze(ctx *glsp.Context, uri, content string) {
	ast, parseErrors := parser.Parse(content)

	diags := []protocol.Diagnostic{}

	// Convert parse errors to diagnostics
	for _, pe := range parseErrors {
		severity := protocol.DiagnosticSeverityError
		diags = append(diags, protocol.Diagnostic{
			Range:    pe.Rng,
			Severity: &severity,
			Source:   strPtr("caddy-ls"),
			Message:  pe.Message,
		})
	}

	// Run semantic analysis
	diags = append(diags, analysis.Analyze(ast)...)

	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func strPtr(s string) *string { return &s }
