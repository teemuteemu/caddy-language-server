package handler

// directiveDocsExtra provides documentation for the small number of directives
// whose Caddy source code lacks a syntax doc comment and are therefore absent
// from the generated directiveDocs map.
var directiveDocsExtra = map[string]string{
	"abort": "abort parses the abort directive.\n\n```\nabort [<matcher>]\n```\n\nAborts the HTTP request with no response.",

	"handle": "handle sets up a mutually-exclusive request handler.\n\n```\nhandle [<matcher>] {\n    <directives...>\n}\n```\n\nLike route, but handlers are mutually exclusive by default based on their matcher.",

	"handle_errors": "handle_errors defines routes for handling HTTP errors.\n\n```\nhandle_errors [<status_codes...>] {\n    <directives...>\n}\n```\n\nStatus codes can be 3-digit numbers, 4xx, 5xx, or a range like 500-599.",

	"invoke": "invoke invokes a named route defined elsewhere in the config.\n\n```\ninvoke <name>\n```",

	"route": "route groups directives that are applied in order, without reordering.\n\n```\nroute [<matcher>] {\n    <directives...>\n}\n```\n\nUnlike handle, directives inside a route block are applied in the order they appear.",
}
