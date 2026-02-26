package analysis

import (
	"caddy-ls/internal/parser"
	"fmt"
	"sort"
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

// knownSubSubDirectives maps a "subdirective:arg" key to the set of valid
// sub-subdirective names inside its body block.  The key is formed from the
// subdirective name and its first argument (e.g. "transport:http").
var knownSubSubDirectives = map[string]map[string]bool{
	"transport:http": {
		// buffering
		"read_buffer": true, "write_buffer": true, "max_response_header": true,
		// proxy
		"proxy_protocol": true, "network_proxy": true,
		// timeouts
		"dial_timeout": true, "dial_fallback_delay": true,
		"response_header_timeout": true, "expect_continue_timeout": true,
		"read_timeout": true, "write_timeout": true,
		// DNS
		"resolvers": true,
		// TLS
		"tls": true, "tls_client_auth": true, "tls_insecure_skip_verify": true,
		"tls_curves": true, "tls_timeout": true, "tls_trust_pool": true,
		"tls_server_name": true, "tls_renegotiation": true, "tls_except_ports": true,
		// keepalive
		"keepalive": true, "keepalive_interval": true,
		"keepalive_idle_conns": true, "keepalive_idle_conns_per_host": true,
		// misc
		"versions": true, "compression": true, "max_conns_per_host": true,
	},
	"transport:fastcgi": {
		"root": true, "split": true, "env": true,
		"resolve_root_symlink": true, "dial_timeout": true,
		"read_timeout": true, "write_timeout": true, "capture_stderr": true,
	},
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

// SubDirectivesFor returns the set of valid subdirective names for parentName.
// ok is false when the parent is unknown to the analyzer; the returned map is
// nil when the body is freeform (no sub-directive validation applies).
func SubDirectivesFor(parentName string) (subs map[string]bool, ok bool) {
	subs, ok = knownSubDirectives[parentName]
	return
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

// analyzer holds per-file state used during a single analysis pass.
type analyzer struct {
	snippets map[string]bool // snippet names defined in the file (without parens)
}

// CollectSnippetNames returns the names of all snippets defined in f, without
// the surrounding parentheses, sorted alphabetically. The completion handler
// uses this to suggest snippet names after "import".
func CollectSnippetNames(f *parser.File) []string {
	var names []string
	for _, sb := range f.SiteBlocks {
		if len(sb.Addresses) == 0 {
			continue
		}
		if name, ok := parseSnippetName(sb.Addresses[0].Value); ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// parseSnippetName extracts the snippet name from an address token like
// "(mysnippet)", returning ("mysnippet", true), or ("", false) if the address
// is not a snippet definition.
func parseSnippetName(addr string) (string, bool) {
	if strings.HasPrefix(addr, "(") && strings.HasSuffix(addr, ")") && len(addr) > 2 {
		return addr[1 : len(addr)-1], true
	}
	return "", false
}

// collectSnippets builds a lookup map for O(1) snippet-name validation.
func collectSnippets(f *parser.File) map[string]bool {
	names := CollectSnippetNames(f)
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// isFileImport reports whether an import argument is a file path or glob
// pattern. File imports are not validated against the snippet registry.
func isFileImport(arg string) bool {
	return strings.Contains(arg, "/") ||
		strings.Contains(arg, "*") ||
		strings.Contains(arg, "\\") ||
		strings.HasPrefix(arg, ".")
}

// isCaddyPlaceholder reports whether arg is a Caddy runtime placeholder
// like {$ENV_VAR} or {http.request.uri}. These cannot be resolved statically.
func isCaddyPlaceholder(arg string) bool {
	return strings.HasPrefix(arg, "{") && strings.HasSuffix(arg, "}")
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
	a := &analyzer{snippets: collectSnippets(f)}
	var diags []protocol.Diagnostic

	if f.GlobalBlock != nil {
		for _, d := range f.GlobalBlock.Directives {
			diags = append(diags, a.analyzeGlobalDirective(d)...)
		}
	}

	for _, sb := range f.SiteBlocks {
		// Snippets can be imported at any nesting level (e.g. inside a
		// reverse_proxy block), so their bodies may legitimately contain
		// subdirective-level tokens. Pass inSnippet=true to suppress the
		// "must appear inside X" placement hint for those tokens.
		inSnippet := isSnippet(sb)
		for _, d := range sb.Directives {
			diags = append(diags, a.analyzeSiteDirective(d, inSnippet)...)
		}
	}

	diags = append(diags, analyzeFilePlaceholders(f)...)

	return diags
}

func (a *analyzer) analyzeGlobalDirective(d *parser.Directive) []protocol.Diagnostic {
	name := d.Name.Value
	if strings.HasPrefix(name, "@") {
		return nil
	}
	if !KnownGlobalOptions[name] {
		return []protocol.Diagnostic{{
			Range:    d.Name.Range(),
			Severity: severityWarning(),
			Source:   strPtr("caddy-ls"),
			Message:  fmt.Sprintf("unknown global option %q", name),
		}}
	}
	if name == "import" {
		return a.analyzeImport(d)
	}
	return nil
}

// analyzeSiteDirective validates a directive at site-block (or snippet) level.
// inSnippet is true when the directive is inside a snippet definition; in that
// case known subdirective tokens (e.g. "transport", "header_up") are silently
// accepted because the snippet may be imported inside their parent directive.
func (a *analyzer) analyzeSiteDirective(d *parser.Directive, inSnippet bool) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	name := d.Name.Value
	// Named matcher declarations (@name) are always valid inside a site block.
	if strings.HasPrefix(name, "@") {
		return diags
	}
	if !KnownTopLevel[name] {
		// Inside a snippet we don't know the import context, so a token that
		// belongs to a known parent directive is accepted without complaint.
		if inSnippet {
			if _, ok := knownSubDirectiveParent[name]; ok {
				return diags
			}
		}
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

	// import is handled separately so the snippet reference can be validated.
	if name == "import" {
		diags = append(diags, a.analyzeImport(d)...)
		return diags
	}

	// Validate subdirectives inside the body block.
	diags = append(diags, a.analyzeDirectiveBody(name, d.Body, inSnippet)...)
	return diags
}

// analyzeDirectiveBody validates the subdirectives inside a directive's body block.
func (a *analyzer) analyzeDirectiveBody(parentName string, body []*parser.Directive, inSnippet bool) []protocol.Diagnostic {
	if len(body) == 0 {
		return nil
	}

	// Container directives hold site-level directives in their body.
	if containerDirectives[parentName] {
		var diags []protocol.Diagnostic
		for _, sub := range body {
			diags = append(diags, a.analyzeSiteDirective(sub, inSnippet)...)
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
		// Matcher declarations are always valid inside any block.
		if strings.HasPrefix(subName, "@") {
			continue
		}
		// import is valid anywhere; validate its snippet reference.
		if subName == "import" {
			diags = append(diags, a.analyzeImport(sub)...)
			continue
		}
		if !subDirs[subName] {
			diags = append(diags, protocol.Diagnostic{
				Range:    sub.Name.Range(),
				Severity: severityWarning(),
				Source:   strPtr("caddy-ls"),
				Message:  fmt.Sprintf("unknown subdirective %q for %q", subName, parentName),
			})
			continue
		}
		// Validate sub-subdirective bodies when we know the schema
		// (e.g. transport http { … }, transport fastcgi { … }).
		if len(sub.Body) > 0 {
			subKey := subName
			if len(sub.Args) > 0 {
				subKey = subName + ":" + sub.Args[0].Token.Value
			}
			if subSubDirs, ok := knownSubSubDirectives[subKey]; ok {
				diags = append(diags, a.analyzeNestedBody(subSubDirs, sub, parentName)...)
			}
		}
	}
	return diags
}

// analyzeNestedBody validates the body of a subdirective (e.g. "transport http")
// against a known set of valid names.  parent is the subdirective node and
// grandparentName is its containing directive (e.g. "reverse_proxy").
func (a *analyzer) analyzeNestedBody(validDirs map[string]bool, parent *parser.Directive, grandparentName string) []protocol.Diagnostic {
	qualifiedParent := parent.Name.Value
	if len(parent.Args) > 0 {
		qualifiedParent += " " + parent.Args[0].Token.Value
	}
	var diags []protocol.Diagnostic
	for _, sub := range parent.Body {
		subName := sub.Name.Value
		if strings.HasPrefix(subName, "@") {
			continue
		}
		if subName == "import" {
			diags = append(diags, a.analyzeImport(sub)...)
			continue
		}
		if !validDirs[subName] {
			diags = append(diags, protocol.Diagnostic{
				Range:    sub.Name.Range(),
				Severity: severityWarning(),
				Source:   strPtr("caddy-ls"),
				Message:  fmt.Sprintf("unknown subdirective %q for %q %q", subName, grandparentName, qualifiedParent),
			})
		}
	}
	return diags
}

// analyzeImport validates the snippet reference in an import directive.
// File paths, glob patterns, and Caddy placeholders are accepted without
// checking; bare names are validated against the snippets defined in the file.
func (a *analyzer) analyzeImport(d *parser.Directive) []protocol.Diagnostic {
	if len(d.Args) == 0 {
		return nil
	}
	arg := d.Args[0].Token.Value
	if isFileImport(arg) || isCaddyPlaceholder(arg) {
		return nil
	}
	if !a.snippets[arg] {
		return []protocol.Diagnostic{{
			Range:    d.Args[0].Range(),
			Severity: severityWarning(),
			Source:   strPtr("caddy-ls"),
			Message:  fmt.Sprintf("undefined snippet %q", arg),
		}}
	}
	return nil
}

func strPtr(s string) *string { return &s }
