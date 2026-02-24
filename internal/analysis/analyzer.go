package analysis

import (
	"caddy-ls/internal/parser"
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// knownSubDirectiveParent maps directives that are only valid inside a parent
// block to the name of that parent. When one of these appears at the top level
// of a site block the analyzer emits a placement hint instead of "unknown".
var knownSubDirectiveParent = map[string]string{
	// reverse_proxy sub-directives
	"to":                "reverse_proxy",
	"transport":         "reverse_proxy",
	"header_up":         "reverse_proxy",
	"header_down":       "reverse_proxy",
	"lb_policy":         "reverse_proxy",
	"lb_retries":        "reverse_proxy",
	"lb_try_duration":   "reverse_proxy",
	"lb_try_interval":   "reverse_proxy",
	"health_uri":        "reverse_proxy",
	"health_port":       "reverse_proxy",
	"health_interval":   "reverse_proxy",
	"health_timeout":    "reverse_proxy",
	"health_status":     "reverse_proxy",
	"health_body":       "reverse_proxy",
	"max_fails":         "reverse_proxy",
	"unhealthy_status":  "reverse_proxy",
	"unhealthy_latency": "reverse_proxy",
	"flush_interval":    "reverse_proxy",
	"buffer_requests":   "reverse_proxy",
	"buffer_responses":  "reverse_proxy",
	"max_buffer_size":   "reverse_proxy",
	"trusted_proxies":   "reverse_proxy",
	"handle_response":   "reverse_proxy",
	"replace_status":    "reverse_proxy",
	// tls sub-directives
	"protocols":         "tls",
	"ciphers":           "tls",
	"curves":            "tls",
	"alpn":              "tls",
	"load":              "tls",
	"ca":                "tls",
	"ca_root":           "tls",
	"key_type":          "tls",
	"dns":               "tls",
	"resolvers":         "tls",
	"eab":               "tls",
	"on_demand":         "tls",
	"client_auth":       "tls",
	"get_certificate":   "tls",
	// encode sub-directives
	"gzip":  "encode",
	"zstd":  "encode",
	"br":    "encode",
	// log sub-directives
	"output": "log",
	"format": "log",
	"level":  "log",
}

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

// isSnippet reports whether a site block is a Caddy snippet definition,
// e.g. "(my_snippet) { ... }". Snippets can contain arbitrary sub-directives
// so their contents must not be validated as top-level directives.
func isSnippet(sb *parser.SiteBlock) bool {
	return len(sb.Addresses) > 0 && strings.HasPrefix(sb.Addresses[0].Value, "(")
}

// Analyze walks the AST and returns diagnostics.
func Analyze(f *parser.File) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	for _, sb := range f.SiteBlocks {
		if isSnippet(sb) {
			continue
		}
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
		var msg string
		if parent, ok := knownSubDirectiveParent[name]; ok {
			msg = fmt.Sprintf("%q must appear inside a %q block, not at the site level", name, parent)
		} else {
			msg = fmt.Sprintf("unknown directive %q", name)
		}
		diags = append(diags, protocol.Diagnostic{
			Range:    d.Name.Range(),
			Severity: severityWarning(),
			Source:   strPtr("caddy-ls"),
			Message:  msg,
		})
	}

	return diags
}

func strPtr(s string) *string { return &s }
