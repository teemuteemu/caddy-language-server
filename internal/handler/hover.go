package handler

import (
	"strings"
	"unicode"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// directiveDocs maps directive names to short Markdown documentation strings.
var directiveDocs = map[string]string{
	"reverse_proxy": "**reverse_proxy** — proxies requests to one or more backends.\n\n```\nreverse_proxy [<matcher>] <upstreams...>\n```",
	"file_server":   "**file_server** — serves static files from the file system.\n\n```\nfile_server [<matcher>] [browse]\n```",
	"tls":           "**tls** — configures TLS for the site.\n\n```\ntls [<email>|<cert_file> <key_file>|off] {\n    ...\n}\n```",
	"encode":        "**encode** — encodes responses (gzip, zstd).\n\n```\nencode [<matcher>] <formats...>\n```",
	"log":           "**log** — enables access logging.\n\n```\nlog [<logger_name>] {\n    output file <path>\n}\n```",
	"header":        "**header** — manipulates HTTP response headers.\n\n```\nheader [<matcher>] [[+|-|?|>]<field> [<value>|<find>] [<replace>]]\n```",
	"root":          "**root** — sets the root path for the site.\n\n```\nroot [<matcher>] <path>\n```",
	"redir":         "**redir** — redirects requests.\n\n```\nredir [<matcher>] <to> [<code>]\n```",
	"rewrite":       "**rewrite** — rewrites the request URI internally.\n\n```\nrewrite [<matcher>] <to>\n```",
	"respond":       "**respond** — writes a hard-coded response.\n\n```\nrespond [<matcher>] [<status>|<body>] [<status>]\n```",
	"route":         "**route** — groups directives to apply in order.\n\n```\nroute [<matcher>] {\n    <directives...>\n}\n```",
	"handle":        "**handle** — routes requests with mutual exclusivity.\n\n```\nhandle [<matcher>] {\n    <directives...>\n}\n```",
	"php_fastcgi":   "**php_fastcgi** — serves PHP apps via FastCGI.\n\n```\nphp_fastcgi [<matcher>] <address>\n```",
	"basicauth":     "**basicauth** — enforces HTTP basic authentication.\n\n```\nbasicauth [<matcher>] [<hash_algorithm>] {\n    <username> <hashed_password>\n}\n```",
	"templates":     "**templates** — executes response body as a Go template.\n\n```\ntemplates [<matcher>] {\n    ...\n}\n```",
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
