package analysis

import (
	"caddy-ls/internal/parser"
	"fmt"
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// knownSubDirectiveParent maps subdirectives that are only valid inside a specific
// parent to that parent's name. Used to produce better "wrong level" diagnostics
// when one of these appears at the top of a site block.
var knownSubDirectiveParent = map[string]string{
	// reverse_proxy sub-directives
	"to":                    "reverse_proxy",
	"transport":             "reverse_proxy",
	"header_up":             "reverse_proxy",
	"header_down":           "reverse_proxy",
	"lb_policy":             "reverse_proxy",
	"lb_retries":            "reverse_proxy",
	"lb_try_duration":       "reverse_proxy",
	"lb_try_interval":       "reverse_proxy",
	"health_uri":            "reverse_proxy",
	"health_port":           "reverse_proxy",
	"health_interval":       "reverse_proxy",
	"health_timeout":        "reverse_proxy",
	"health_status":         "reverse_proxy",
	"health_body":           "reverse_proxy",
	"max_fails":             "reverse_proxy",
	"unhealthy_status":      "reverse_proxy",
	"unhealthy_latency":     "reverse_proxy",
	"flush_interval":        "reverse_proxy",
	"buffer_requests":       "reverse_proxy",
	"buffer_responses":      "reverse_proxy",
	"max_buffer_size":       "reverse_proxy",
	"trusted_proxies":       "reverse_proxy",
	"handle_response":       "reverse_proxy",
	"replace_status":        "reverse_proxy",
	// tls sub-directives
	"protocols":       "tls",
	"ciphers":         "tls",
	"curves":          "tls",
	"alpn":            "tls",
	"load":            "tls",
	"ca":              "tls",
	"ca_root":         "tls",
	"key_type":        "tls",
	"dns":             "tls",
	"resolvers":       "tls",
	"eab":             "tls",
	"on_demand":       "tls",
	"client_auth":     "tls",
	"get_certificate": "tls",
	// encode sub-directives
	"gzip": "encode",
	"zstd": "encode",
	"br":   "encode",
	// log sub-directives
	"output": "log",
	"format": "log",
	"level":  "log",
}

// knownSubDirectives maps a directive name to the set of subdirective names valid
// inside its body block.  A nil value means the body is freeform and should not
// be validated (e.g. basicauth username/hash pairs, header field operations).
// Directives not present in this map have their bodies skipped silently.
var knownSubDirectives = map[string]map[string]bool{
	"reverse_proxy": {
		// upstream selection
		"to": true, "dynamic": true,
		// transport
		"transport": true,
		// headers
		"header_up": true, "header_down": true,
		// load balancing
		"lb_policy": true, "lb_retries": true,
		"lb_try_duration": true, "lb_try_interval": true, "lb_retry_match": true,
		// active health checks
		"health_uri": true, "health_port": true, "health_interval": true,
		"health_timeout": true, "health_status": true, "health_body": true,
		"health_passes": true, "health_fails": true, "health_headers": true,
		"health_request_body": true,
		// passive health checks
		"max_fails": true, "fail_duration": true,
		"unhealthy_status": true, "unhealthy_latency": true, "unhealthy_request_count": true,
		// streaming / buffering
		"flush_interval": true, "trusted_proxies": true,
		"request_buffers": true, "response_buffers": true,
		"stream_timeout": true, "stream_close_delay": true,
		// older aliases still accepted by Caddy
		"buffer_requests": true, "buffer_responses": true, "max_buffer_size": true,
		// response handling
		"handle_response": true, "replace_status": true,
		"copy_response": true, "copy_response_headers": true,
	},
	"tls": {
		"protocols": true, "ciphers": true, "curves": true, "alpn": true,
		"load": true, "ca": true, "ca_root": true, "key_type": true,
		"dns": true, "propagation_delay": true, "propagation_timeout": true,
		"resolvers": true, "dns_challenge_override_domain": true,
		"eab": true, "on_demand": true, "client_auth": true,
		"issuer": true, "get_certificate": true,
		"insecure_secrets_log": true, "reuse_private_keys": true,
	},
	"encode": {
		"gzip": true, "zstd": true, "br": true, "minimum_length": true, "match": true,
	},
	"log": {
		"hostnames": true, "output": true, "format": true, "level": true,
		"include": true, "exclude": true, "sampling": true,
	},
	"file_server": {
		"fs": true, "root": true, "hide": true, "index": true, "browse": true,
		"precompressed": true, "status": true, "disable_canonical_uris": true,
		"pass_thru": true,
	},
	"php_fastcgi": {
		"root": true, "split": true, "env": true, "resolve_root_symlink": true,
		"dial_timeout": true, "read_timeout": true, "write_timeout": true,
		"capture_stderr": true, "index": true, "try_files": true,
	},
	"request_body": {
		"max_size": true,
	},
	"forward_auth": {
		"uri": true, "copy_headers": true, "header_up": true, "header_down": true,
		"trust_forward_header": true,
	},
	"acme_server": {
		"ca": true, "lifetime": true, "resolvers": true, "challenges": true,
	},
	"templates": {
		"mime_type": true, "delimiters": true, "root": true, "extensions": true,
	},
	"tracing": {
		"span": true,
	},
	// freeform bodies – structure is user-defined, not validated
	"basicauth":      nil,
	"header":         nil,
	"request_header": nil,
	"map":            nil,
}

// containerDirectives are top-level directives whose body contains site-level
// directives (routing blocks). Their contents are validated the same way as a
// site block rather than against a fixed subdirective set.
var containerDirectives = map[string]bool{
	"handle":        true,
	"handle_errors": true,
	"handle_path":   true,
	"route":         true,
}

// KnownTopLevel is the set of directives valid at the site-block level.
// Source: https://caddyserver.com/docs/caddyfile/directives
var KnownTopLevel = map[string]bool{
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

// KnownGlobalOptions is the set of directives valid inside the global options block.
// Source: https://caddyserver.com/docs/caddyfile/options
var KnownGlobalOptions = map[string]bool{
	"debug":              true,
	"http_port":          true,
	"https_port":         true,
	"default_bind":       true,
	"grace_period":       true,
	"shutdown_delay":     true,
	"admin":              true,
	"on_demand_tls":      true,
	"storage":            true,
	"acme_ca":            true,
	"acme_ca_root":       true,
	"acme_dns":           true,
	"acme_eab":           true,
	"cert_issuer":        true,
	"skip_install_trust": true,
	"email":              true,
	"ocsp_stapling":      true,
	"ocsp_interval":      true,
	"preferred_chains":   true,
	"key_type":           true,
	"auto_https":         true,
	"metrics":            true,
	"tracing":            true,
	"servers":            true,
	"log":                true,
	"order":              true,
	"local_certs":        true,
	"persist_config":     true,
	"pki":                true,
	"import":             true,
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

	if f.GlobalBlock != nil {
		for _, d := range f.GlobalBlock.Directives {
			diags = append(diags, analyzeGlobalDirective(d)...)
		}
	}

	for _, sb := range f.SiteBlocks {
		if isSnippet(sb) {
			continue
		}
		for _, d := range sb.Directives {
			diags = append(diags, analyzeSiteDirective(d)...)
		}
	}

	return diags
}

func analyzeGlobalDirective(d *parser.Directive) []protocol.Diagnostic {
	name := d.Name.Value
	if strings.HasPrefix(name, "@") {
		return nil
	}
	if !KnownGlobalOptions[name] {
		msg := fmt.Sprintf("unknown global option %q", name)
		return []protocol.Diagnostic{{
			Range:    d.Name.Range(),
			Severity: severityWarning(),
			Source:   strPtr("caddy-ls"),
			Message:  msg,
		}}
	}
	return nil
}

func analyzeSiteDirective(d *parser.Directive) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	name := d.Name.Value
	// Named matcher declarations (@name) are always valid inside a site block.
	if strings.HasPrefix(name, "@") {
		return diags
	}
	if !KnownTopLevel[name] {
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
		// Don't attempt to validate the body of an unknown directive.
		return diags
	}

	// Validate subdirectives inside the body block.
	diags = append(diags, analyzeDirectiveBody(name, d.Body)...)
	return diags
}

// analyzeDirectiveBody validates the subdirectives inside a directive's body block.
func analyzeDirectiveBody(parentName string, body []*parser.Directive) []protocol.Diagnostic {
	if len(body) == 0 {
		return nil
	}

	// Container directives hold site-level directives in their body.
	if containerDirectives[parentName] {
		var diags []protocol.Diagnostic
		for _, sub := range body {
			diags = append(diags, analyzeSiteDirective(sub)...)
		}
		return diags
	}

	subDirs, known := knownSubDirectives[parentName]
	if !known || subDirs == nil {
		// Either we have no subdirective list for this directive, or it is
		// explicitly marked as freeform (nil). Skip body validation.
		return nil
	}

	var diags []protocol.Diagnostic
	for _, sub := range body {
		subName := sub.Name.Value
		// Matcher declarations and import are always valid inside any block.
		if strings.HasPrefix(subName, "@") || subName == "import" {
			continue
		}
		if !subDirs[subName] {
			diags = append(diags, protocol.Diagnostic{
				Range:    sub.Name.Range(),
				Severity: severityWarning(),
				Source:   strPtr("caddy-ls"),
				Message:  fmt.Sprintf("unknown subdirective %q for %q", subName, parentName),
			})
		}
		// Sub-subdirective bodies (e.g. transport http { … }) are not validated
		// further to avoid false positives on module-specific syntax.
	}
	return diags
}

func strPtr(s string) *string { return &s }
