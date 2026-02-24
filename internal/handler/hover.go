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

	// reverse_proxy subdirectives
	"to":                  "**to** *(reverse_proxy)* — upstream address(es) to proxy requests to.\n\n```\nto <upstreams...>\n```",
	"transport":           "**transport** *(reverse_proxy)* — configures the transport used to connect to upstreams.\n\n```\ntransport http|fastcgi [<options>] {\n    ...\n}\n```",
	"header_up":           "**header_up** *(reverse_proxy)* — sets, adds, or removes a request header before proxying.\n\n```\nheader_up [+|-|>]<field> [<value>]\n```",
	"header_down":         "**header_down** *(reverse_proxy)* — sets, adds, or removes a response header.\n\n```\nheader_down [+|-|>]<field> [<value>]\n```",
	"lb_policy":           "**lb_policy** *(reverse_proxy)* — load balancing policy.\n\n```\nlb_policy random|least_conn|round_robin|ip_hash|uri_hash|header <field>|cookie [<options>]\n```",
	"lb_retries":          "**lb_retries** *(reverse_proxy)* — number of times to retry selecting an upstream.\n\n```\nlb_retries <count>\n```",
	"lb_try_duration":     "**lb_try_duration** *(reverse_proxy)* — total time budget for retrying failed requests.\n\n```\nlb_try_duration <duration>\n```",
	"lb_try_interval":     "**lb_try_interval** *(reverse_proxy)* — how long to wait between retry attempts.\n\n```\nlb_try_interval <duration>\n```",
	"health_uri":          "**health_uri** *(reverse_proxy)* — URI to use for active health checks.\n\n```\nhealth_uri <uri>\n```",
	"health_port":         "**health_port** *(reverse_proxy)* — port to use for active health checks.\n\n```\nhealth_port <port>\n```",
	"health_interval":     "**health_interval** *(reverse_proxy)* — how often to perform active health checks.\n\n```\nhealth_interval <duration>\n```",
	"health_timeout":      "**health_timeout** *(reverse_proxy)* — timeout for active health check requests.\n\n```\nhealth_timeout <duration>\n```",
	"health_status":       "**health_status** *(reverse_proxy)* — expected HTTP status code for a healthy upstream.\n\n```\nhealth_status <status>\n```",
	"health_body":         "**health_body** *(reverse_proxy)* — regex that the health check response body must match.\n\n```\nhealth_body <regexp>\n```",
	"max_fails":           "**max_fails** *(reverse_proxy)* — number of failures before marking an upstream unhealthy.\n\n```\nmax_fails <count>\n```",
	"unhealthy_status":    "**unhealthy_status** *(reverse_proxy)* — response status codes that mark an upstream as unhealthy.\n\n```\nunhealthy_status <status>...\n```",
	"unhealthy_latency":   "**unhealthy_latency** *(reverse_proxy)* — latency threshold that marks an upstream as unhealthy.\n\n```\nunhealthy_latency <duration>\n```",
	"flush_interval":      "**flush_interval** *(reverse_proxy)* — how often to flush buffered response data to the client.\n\n```\nflush_interval <duration>|-1\n```",
	"trusted_proxies":     "**trusted_proxies** *(reverse_proxy)* — upstreams whose X-Forwarded-* headers should be trusted.\n\n```\ntrusted_proxies <ranges...>\n```",
	"handle_response":     "**handle_response** *(reverse_proxy)* — defines a route that handles specific upstream responses.\n\n```\nhandle_response [<matcher>] {\n    <directives...>\n}\n```",
	"replace_status":      "**replace_status** *(reverse_proxy)* — replaces the response status code.\n\n```\nreplace_status [<matcher>] <status_code>\n```",

	// tls subdirectives
	"protocols":       "**protocols** *(tls)* — minimum and maximum TLS protocol versions.\n\n```\nprotocols <min> [<max>]\n```",
	"ciphers":         "**ciphers** *(tls)* — cipher suites to use (TLS 1.2 and below).\n\n```\nciphers <suites...>\n```",
	"curves":          "**curves** *(tls)* — elliptic curves for key exchange.\n\n```\ncurves <curves...>\n```",
	"alpn":            "**alpn** *(tls)* — ALPN protocols to advertise.\n\n```\nalpn <values...>\n```",
	"ca":              "**ca** *(tls)* — ACME CA endpoint URL.\n\n```\nca <url>\n```",
	"ca_root":         "**ca_root** *(tls)* — trusted root certificate for the ACME CA.\n\n```\nca_root <pem_file>\n```",
	"dns":             "**dns** *(tls)* — DNS challenge provider for obtaining certificates.\n\n```\ndns <provider_name> [<options>]\n```",
	"resolvers":       "**resolvers** *(tls)* — custom DNS resolvers to use during challenges.\n\n```\nresolvers <ip_addresses...>\n```",
	"eab":             "**eab** *(tls)* — external account binding credentials for ACME.\n\n```\neab <key_id> <mac_key>\n```",
	"on_demand":       "**on_demand** *(tls)* — enables on-demand TLS certificate issuance.\n\n```\non_demand\n```",
	"client_auth":     "**client_auth** *(tls)* — configures mutual TLS client authentication.\n\n```\nclient_auth {\n    mode request|require|verify_if_given|require_and_verify\n    trusted_ca_certs <pem_files...>\n}\n```",
	"get_certificate": "**get_certificate** *(tls)* — obtains a certificate from a module at handshake time.\n\n```\nget_certificate <source> [<options>]\n```",

	// encode subdirectives
	"gzip":           "**gzip** *(encode)* — enables gzip compression.\n\n```\ngzip [<level>]\n```",
	"zstd":           "**zstd** *(encode)* — enables Zstandard compression.\n\n```\nzstd\n```",
	"br":             "**br** *(encode)* — enables Brotli compression.\n\n```\nbr [<quality>]\n```",
	"minimum_length": "**minimum_length** *(encode)* — minimum response size in bytes before compressing.\n\n```\nminimum_length <length>\n```",

	// log subdirectives
	"output":   "**output** *(log)* — log output module (file, stderr, stdout, discard).\n\n```\noutput file <path> {\n    roll_size <size>\n    roll_keep <count>\n    roll_keep_for <duration>\n}\n```",
	"format":   "**format** *(log)* — log format module (json, console, filter).\n\n```\nformat json|console\n```",
	"level":    "**level** *(log)* — minimum log level to emit (DEBUG, INFO, WARN, ERROR).\n\n```\nlevel DEBUG|INFO|WARN|ERROR\n```",
	"include":  "**include** *(log)* — logger names to include in this log.\n\n```\ninclude <names...>\n```",
	"exclude":  "**exclude** *(log)* — logger names to exclude from this log.\n\n```\nexclude <names...>\n```",
	"sampling": "**sampling** *(log)* — log sampling configuration.\n\n```\nsampling {\n    interval <duration>\n    first <count>\n    thereafter <count>\n}\n```",

	// file_server subdirectives
	"hide":                   "**hide** *(file_server)* — files or directories to hide from directory listings.\n\n```\nhide <files...>\n```",
	"index":                  "**index** *(file_server)* — filenames to look for as directory index files.\n\n```\nindex <files...>\n```",
	"browse":                 "**browse** *(file_server)* — enables directory browsing with an optional template.\n\n```\nbrowse [<template_file>]\n```",
	"precompressed":          "**precompressed** *(file_server)* — file encodings to look for as precompressed variants.\n\n```\nprecompressed zstd|br|gzip\n```",
	"pass_thru":              "**pass_thru** *(file_server)* — passes requests through to the next handler if a file is not found.\n\n```\npass_thru\n```",
	"disable_canonical_uris": "**disable_canonical_uris** *(file_server)* — disables redirects to canonicalize trailing slashes.\n\n```\ndisable_canonical_uris\n```",

	// request_body subdirectives
	"max_size": "**max_size** *(request_body)* — maximum request body size (e.g. 10MB).\n\n```\nmax_size <size>\n```",

	// forward_auth subdirectives
	"copy_headers":          "**copy_headers** *(forward_auth)* — response headers to copy to the original request.\n\n```\ncopy_headers <fields...>\n```",
	"trust_forward_header":  "**trust_forward_header** *(forward_auth)* — trusts existing X-Forwarded-* request headers.\n\n```\ntrust_forward_header\n```",

	// tracing subdirectives
	"span": "**span** *(tracing)* — override the default OpenTelemetry span name.\n\n```\nspan <name>\n```",

	// acme_server subdirectives
	"lifetime":   "**lifetime** *(acme_server)* — override the default certificate lifetime.\n\n```\nlifetime <duration>\n```",
	"challenges": "**challenges** *(acme_server)* — ACME challenge types to enable.\n\n```\nchallenges http-01|tls-alpn-01|dns-01\n```",

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
