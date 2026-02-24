package analysis

import (
	"caddy-ls/internal/parser"
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// knownTopLevel is the set of directives valid at the site-block level.
var knownTopLevel = map[string]bool{
	"http":          true,
	"https":         true,
	"tls":           true,
	"reverse_proxy": true,
	"file_server":   true,
	"encode":        true,
	"log":           true,
	"header":        true,
	"root":          true,
	"php_fastcgi":   true,
	"redir":         true,
	"rewrite":       true,
	"respond":       true,
	"route":         true,
	"handle":        true,
	"handle_path":   true,
	"handle_errors": true,
	"basicauth":     true,
	"request_header": true,
	"templates":     true,
	"push":          true,
	"vars":          true,
	"map":           true,
	"tracing":       true,
	"acme_server":   true,
	"metrics":       true,
}

func severityError() *protocol.DiagnosticSeverity {
	s := protocol.DiagnosticSeverityError
	return &s
}

func severityWarning() *protocol.DiagnosticSeverity {
	s := protocol.DiagnosticSeverityWarning
	return &s
}

// Analyze walks the AST and returns diagnostics.
func Analyze(f *parser.File) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	for _, sb := range f.SiteBlocks {
		for _, d := range sb.Directives {
			diags = append(diags, analyzeDirective(d)...)
		}
	}

	return diags
}

func analyzeDirective(d *parser.Directive) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	name := d.Name.Value
	if !knownTopLevel[name] {
		diags = append(diags, protocol.Diagnostic{
			Range:    d.Name.Range(),
			Severity: severityWarning(),
			Source:   strPtr("caddy-ls"),
			Message:  fmt.Sprintf("unknown directive %q", name),
		})
	}

	return diags
}

func strPtr(s string) *string { return &s }
