package handler

import (
	"strings"
	"unicode"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// directiveDocs maps directive names to short Markdown documentation strings.
// Source: https://caddyserver.com/docs/caddyfile/directives
var directiveDocs = map[string]string{
	// Core / routing
	"abort": "**abort** — aborts the request with no response.\n\n```\nabort [<matcher>]\n```",
	"error": "**error** — triggers an error in the HTTP handler chain.\n\n```\nerror [<matcher>] <status>|<message> [<status>]\n```",
	"handle": "**handle** — routes requests with mutual exclusivity.\n\n```\nhandle [<matcher>] {\n    <directives...>\n}\n```",
	"handle_errors": "**handle_errors** — defines routes for handling errors.\n\n```\nhandle_errors [<status_codes...>] {\n    <directives...>\n}\n```",
	"handle_path": "**handle_path** — like handle but strips the matched path prefix.\n\n```\nhandle_path [<matcher>] {\n    <directives...>\n}\n```",
	"invoke": "**invoke** — invokes a named route.\n\n```\ninvoke <name>\n```",
	"map": "**map** — maps an input value to one or more outputs.\n\n```\nmap [<matcher>] <source> <destinations...> {\n    [~]<input> <outputs...>\n    default <defaults...>\n}\n```",
	"method": "**method** — changes the HTTP method of the request.\n\n```\nmethod [<matcher>] <method>\n```",
	"redir": "**redir** — redirects requests.\n\n```\nredir [<matcher>] <to> [<code>]\n```",
	"request_body": "**request_body** — limits the size of request bodies.\n\n```\nrequest_body [<matcher>] {\n    max_size <size>\n}\n```",
	"respond": "**respond** — writes a hard-coded response.\n\n```\nrespond [<matcher>] [<status>|<body>] [<status>] {\n    body <text>\n    close\n}\n```",
	"rewrite": "**rewrite** — rewrites the request URI internally.\n\n```\nrewrite [<matcher>] <to>\n```",
	"route": "**route** — groups directives to apply in order.\n\n```\nroute [<matcher>] {\n    <directives...>\n}\n```",
	"try_files": "**try_files** — rewrites the request if files are not found.\n\n```\ntry_files [<matcher>] <files...> {\n    policy first_exist|smallest_size|largest_size|most_recently_modified\n}\n```",
	"uri": "**uri** — manipulates the request URI.\n\n```\nuri [<matcher>] strip_prefix|strip_suffix|replace|path_regexp <target> [<replacement> [<limit>]]\n```",
	"vars": "**vars** — sets variables for use in templates or expressions.\n\n```\nvars [<matcher>] <name> <value>\n```",

	// Reverse proxy / FastCGI
	"forward_auth": "**forward_auth** — delegates authentication to an external service.\n\n```\nforward_auth [<matcher>] [<upstreams...>] {\n    uri <to>\n    copy_headers <headers...>\n}\n```",
	"php_fastcgi": "**php_fastcgi** — serves PHP apps via FastCGI.\n\n```\nphp_fastcgi [<matcher>] <address>\n```",
	"reverse_proxy": "**reverse_proxy** — proxies requests to one or more backends.\n\n```\nreverse_proxy [<matcher>] [<upstreams...>] {\n    to <upstreams...>\n    lb_policy <name> [<options...>]\n    health_uri <uri>\n    header_up [+|-]<field> [<value>]\n    header_down [+|-]<field> [<value>]\n}\n```",

	// Static files
	"file_server": "**file_server** — serves static files from the file system.\n\n```\nfile_server [<matcher>] [browse] {\n    root <path>\n    hide <files...>\n    index <files...>\n    browse [<template_file>]\n}\n```",
	"push": "**push** — initiates HTTP/2 server push for linked resources.\n\n```\npush [<matcher>] [<resource>] {\n    [GET|HEAD] <resource>\n    headers {\n        [+|-]<field> [<value>]\n    }\n}\n```",
	"root": "**root** — sets the root path for the site.\n\n```\nroot [<matcher>] <path>\n```",

	// TLS / PKI
	"acme_server": "**acme_server** — provides an ACME server for issuing certificates.\n\n```\nacme_server [<matcher>] {\n    ca <id>\n    lifetime <duration>\n    resolvers <dns_servers...>\n}\n```",
	"tls": "**tls** — configures TLS for the site.\n\n```\ntls [<email>|<cert_file> <key_file>|off] {\n    protocols <min> [<max>]\n    ciphers <suites...>\n    curves <curves...>\n    client_auth {\n        mode [request|require|...]\n    }\n    dns <provider_name> [<options>]\n    ca <url>\n    on_demand\n}\n```",

	// Headers
	"header": "**header** — manipulates HTTP response headers.\n\n```\nheader [<matcher>] [[+|-|?|>]<field> [<value>|<find>] [<replace>]] {\n    [+|-|?|>]<field> [<value>|<find>] [<replace>]\n    defer\n}\n```",
	"request_header": "**request_header** — manipulates HTTP request headers.\n\n```\nrequest_header [<matcher>] [+|-|>]<field> [<value>]\n```",

	// Encoding / templates
	"encode": "**encode** — encodes responses (gzip, zstd, br).\n\n```\nencode [<matcher>] <formats...> {\n    gzip [<level>]\n    zstd\n    br\n    minimum_length <length>\n    match {\n        status <codes...>\n        header <field> [<value>]\n    }\n}\n```",
	"templates": "**templates** — executes response body as a Go template.\n\n```\ntemplates [<matcher>] {\n    mime <types...>\n    between <open_delim> <close_delim>\n    root <path>\n}\n```",

	// Auth
	"basicauth": "**basicauth** — enforces HTTP basic authentication.\n\n```\nbasicauth [<matcher>] [<hash_algorithm> [<realm>]] {\n    <username> <hashed_password>\n}\n```",

	// Logging
	"log": "**log** — enables access logging.\n\n```\nlog [<logger_name>] {\n    hostnames <hostnames...>\n    output file <path> {\n        roll_size <size>\n        roll_keep <count>\n    }\n    format json|console|...\n    level DEBUG|INFO|WARN|ERROR\n}\n```",
	"log_append": "**log_append** — appends a field to access log entries.\n\n```\nlog_append [<matcher>] <key> <value>\n```",
	"log_name": "**log_name** — overrides the logger name for the current site.\n\n```\nlog_name [<matcher>] <names...>\n```",
	"log_skip": "**log_skip** — skips access logging for matched requests.\n\n```\nlog_skip [<matcher>]\n```",

	// Observability
	"intercept": "**intercept** — intercepts responses and allows rewriting them.\n\n```\nintercept [<matcher>] {\n    @name <matcher>\n    handle_response [<matcher>] {\n        <directives...>\n    }\n    replace_status [<matcher>] <status_code>\n}\n```",
	"metrics": "**metrics** — exposes Prometheus metrics at an endpoint.\n\n```\nmetrics [<matcher>] {\n    disable_openmetrics\n}\n```",
	"tracing": "**tracing** — enables OpenTelemetry distributed tracing.\n\n```\ntracing [<matcher>] {\n    span <name>\n}\n```",

	// Misc
	"bind": "**bind** — overrides the interface to which the listener binds.\n\n```\nbind <hosts...>\n```",
	"import": "**import** — imports a Caddyfile snippet or file.\n\n```\nimport <pattern> [<args...>]\n```",
	"local_certs": "**local_certs** — causes all certificates to be issued locally.\n\n```\nlocal_certs\n```",
}

// Hover handles textDocument/hover.
func (h *Handler) Hover(ctx *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := string(params.TextDocument.URI)
	content, ok := h.store.Get(uri)
	if !ok {
		return nil, nil
	}

	word := wordAtPosition(content, params.Position)
	if word == "" {
		return nil, nil
	}

	doc, found := directiveDocs[word]
	if !found {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: doc,
		},
	}, nil
}

// wordAtPosition extracts the word under the cursor position.
func wordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	runes := []rune(line)
	col := int(pos.Character)
	if col > len(runes) {
		col = len(runes)
	}

	// Find start of word
	start := col
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}

	// Find end of word
	end := col
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return string(runes[start:end])
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
