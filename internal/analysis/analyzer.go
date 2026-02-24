package analysis

import (
	"caddy-ls/internal/parser"
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// knownTopLevel is the set of directives valid at the site-block level.
// Source: https://caddyserver.com/docs/caddyfile/directives
var knownTopLevel = map[string]bool{
	// Core / routing
	"abort":          true,
	"error":          true,
	"handle":         true,
	"handle_errors":  true,
	"handle_path":    true,
	"invoke":         true,
	"map":            true,
	"method":         true,
	"redir":          true,
	"request_body":   true,
	"respond":        true,
	"rewrite":        true,
	"route":          true,
	"try_files":      true,
	"uri":            true,
	"vars":           true,
	// Reverse proxy / fastcgi
	"forward_auth":  true,
	"php_fastcgi":   true,
	"reverse_proxy": true,
	// Static files
	"file_server": true,
	"push":        true,
	"root":        true,
	// TLS / PKI
	"acme_server": true,
	"tls":         true,
	// Headers
	"header":         true,
	"request_header": true,
	// Encoding / templates
	"encode":    true,
	"templates": true,
	// Auth
	"basicauth": true,
	// Logging
	"log":        true,
	"log_append": true,
	"log_skip":   true,
	"log_name":   true,
	// Observability
	"intercept": true,
	"metrics":   true,
	"tracing":   true,
	// Misc
	"bind":        true,
	"import":      true,
	"local_certs": true,
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
	// Named matcher declarations (@name) are always valid inside a site block.
	if strings.HasPrefix(name, "@") {
		return diags
	}
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
