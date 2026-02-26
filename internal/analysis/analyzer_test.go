package analysis

import (
	"caddy-ls/internal/parser"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// analyze is a helper that parses src and runs Analyze on the result.
func analyze(src string) []protocol.Diagnostic {
	f, _ := parser.Parse(src)
	return Analyze(f)
}

// hasMsg reports whether any diagnostic message contains all the given substrings.
func hasMsg(diags []protocol.Diagnostic, subs ...string) bool {
	for _, d := range diags {
		match := true
		for _, s := range subs {
			if !strings.Contains(d.Message, s) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// --- site-block directive validation -----------------------------------------

func TestAnalyze_KnownDirectiveNoWarning(t *testing.T) {
	for _, name := range []string{"reverse_proxy", "file_server", "tls", "encode", "log", "root", "redir"} {
		diags := analyze("example.com {\n\t" + name + "\n}\n")
		if len(diags) != 0 {
			t.Errorf("directive %q: expected no diagnostics, got %d: %v", name, len(diags), diags)
		}
	}
}

func TestAnalyze_UnknownDirectiveWarning(t *testing.T) {
	diags := analyze("example.com {\n\tfoobar\n}\n")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if *diags[0].Severity != protocol.DiagnosticSeverityWarning {
		t.Errorf("expected Warning severity, got %v", *diags[0].Severity)
	}
	if !strings.Contains(diags[0].Message, `"foobar"`) {
		t.Errorf("message should name the directive, got: %q", diags[0].Message)
	}
}

func TestAnalyze_SubDirectivePlacementHint(t *testing.T) {
	// "to" is only valid inside reverse_proxy; at site level it should get a
	// placement hint rather than a generic "unknown" message.
	diags := analyze("example.com {\n\tto localhost:8080\n}\n")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if *diags[0].Severity != protocol.DiagnosticSeverityWarning {
		t.Errorf("expected Warning severity")
	}
	if !hasMsg(diags, `"to"`, "reverse_proxy") {
		t.Errorf("expected placement hint mentioning parent directive, got: %q", diags[0].Message)
	}
}

func TestAnalyze_NamedMatcherNoWarning(t *testing.T) {
	// @name declarations and references must not trigger "unknown directive".
	diags := analyze("example.com {\n\t@api path /api/*\n\treverse_proxy @api localhost:8080\n}\n")
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for named matcher, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_SnippetBody_Validated(t *testing.T) {
	// Truly unknown directives (not in KnownTopLevel and not in knownSubDirectiveParent)
	// must still be flagged inside snippets.
	diags := analyze("(mysnippet) {\n\tunknown_directive\n\tanother_bad_one baz\n}\n")
	if len(diags) != 2 {
		t.Errorf("expected 2 diagnostics for unknown directives in snippet, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_SnippetBody_ValidDirective_NoWarning(t *testing.T) {
	diags := analyze("(mysnippet) {\n\treverse_proxy localhost:8080\n}\n")
	if len(diags) != 0 {
		t.Errorf("valid directive in snippet: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_SnippetBody_SubDirectiveLevelTokens_NoWarning(t *testing.T) {
	// Snippets may be imported inside a parent directive block (e.g. reverse_proxy),
	// so subdirective-level tokens must not produce "must appear inside X" warnings.
	cases := []string{
		"transport", "header_up", "header_down", "lb_policy", "health_uri",
		"protocols", "ciphers", "alpn",
		"gzip", "zstd",
		"output", "format", "level",
	}
	for _, sub := range cases {
		src := "(mysnippet) {\n\t" + sub + "\n}\n"
		diags := analyze(src)
		if len(diags) != 0 {
			t.Errorf("snippet with subdirective token %q: expected no diagnostics, got %d: %v", sub, len(diags), diags)
		}
	}
}

func TestAnalyze_SiteLevel_SubDirectivePlacementHint_StillWorks(t *testing.T) {
	// Outside a snippet the placement hint must still fire.
	diags := analyze("example.com {\n\ttransport http\n}\n")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "transport", "reverse_proxy") {
		t.Errorf("expected placement hint, got: %q", diags[0].Message)
	}
}

func TestAnalyze_MultipleUnknownDirectives(t *testing.T) {
	src := "example.com {\n\treverse_proxy localhost\n\tbad_one\n\tbad_two\n}\n"
	diags := analyze(src)
	if len(diags) != 2 {
		t.Errorf("expected 2 diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_DiagnosticRange(t *testing.T) {
	// The diagnostic range must point at the directive name token.
	diags := analyze("example.com {\n\tbaddir\n}\n")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	rng := diags[0].Range
	// "baddir" is on line 1 (0-based), indented by one tab (char 1).
	if rng.Start.Line != 1 {
		t.Errorf("range start line: want 1, got %d", rng.Start.Line)
	}
	if rng.Start.Character != 1 {
		t.Errorf("range start char: want 1, got %d", rng.Start.Character)
	}
}

func TestAnalyze_EmptyFile(t *testing.T) {
	diags := analyze("")
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for empty file, got %d", len(diags))
	}
}

func TestAnalyze_MultipleSiteBlocks(t *testing.T) {
	src := "a.com {\n\treverse_proxy localhost\n}\nb.com {\n\tbaddir\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Errorf("expected 1 diagnostic (from b.com block), got %d: %v", len(diags), diags)
	}
}

// --- global options block validation -----------------------------------------

func TestAnalyze_GlobalKnownOptionNoWarning(t *testing.T) {
	for _, name := range []string{"email", "http_port", "https_port", "admin", "storage", "log"} {
		diags := analyze("{\n\t" + name + " foo\n}\nexample.com {\n\trespond \"ok\"\n}\n")
		if len(diags) != 0 {
			t.Errorf("global option %q: expected no diagnostics, got %d: %v", name, len(diags), diags)
		}
	}
}

func TestAnalyze_GlobalUnknownOptionWarning(t *testing.T) {
	diags := analyze("{\n\tunknown_global_thing foo\n}\n")
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if *diags[0].Severity != protocol.DiagnosticSeverityWarning {
		t.Errorf("expected Warning severity")
	}
}

func TestAnalyze_GlobalAndSiteErrors(t *testing.T) {
	// Both a bad global option and a bad site directive should each produce a diagnostic.
	src := "{\n\tbad_global\n}\nexample.com {\n\tbad_site\n}\n"
	diags := analyze(src)
	if len(diags) != 2 {
		t.Errorf("expected 2 diagnostics, got %d: %v", len(diags), diags)
	}
}

// --- subdirective body validation --------------------------------------------

func TestAnalyze_ValidSubDirective_NoWarning(t *testing.T) {
	cases := []struct {
		parent string
		sub    string
	}{
		{"reverse_proxy", "to localhost:8080"},
		{"reverse_proxy", "lb_policy round_robin"},
		{"reverse_proxy", "health_uri /healthz"},
		{"tls", "protocols tls1.2 tls1.3"},
		{"tls", "ciphers TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		{"encode", "gzip"},
		{"encode", "zstd"},
		{"log", "output file /var/log/access.log"},
		{"log", "level DEBUG"},
		{"file_server", "root /var/www"},
		{"file_server", "browse"},
		{"php_fastcgi", "root /var/www/php"},
		{"request_body", "max_size 10MB"},
		{"forward_auth", "uri https://auth.example.com/check"},
		{"tracing", "span my-span"},
	}
	for _, tc := range cases {
		src := "example.com {\n\t" + tc.parent + " {\n\t\t" + tc.sub + "\n\t}\n}\n"
		diags := analyze(src)
		if len(diags) != 0 {
			t.Errorf("%s > %s: expected no diagnostics, got %d: %v", tc.parent, tc.sub, len(diags), diags)
		}
	}
}

func TestAnalyze_InvalidSubDirective_Warning(t *testing.T) {
	cases := []struct {
		parent string
		badSub string
	}{
		{"reverse_proxy", "totally_invalid_sub"},
		{"tls", "not_a_tls_option"},
		{"encode", "deflate"},
		{"log", "unknown_log_setting"},
		{"file_server", "bad_fs_sub"},
		{"php_fastcgi", "nonexistent"},
		{"request_body", "invalid"},
		{"forward_auth", "bad_sub"},
		{"tracing", "not_a_span"},
	}
	for _, tc := range cases {
		src := "example.com {\n\t" + tc.parent + " {\n\t\t" + tc.badSub + "\n\t}\n}\n"
		diags := analyze(src)
		if len(diags) != 1 {
			t.Errorf("%s > %s: expected 1 diagnostic, got %d: %v", tc.parent, tc.badSub, len(diags), diags)
			continue
		}
		if !hasMsg(diags, `"`+tc.badSub+`"`, tc.parent) {
			t.Errorf("%s > %s: diagnostic message should name both the subdirective and parent, got: %q",
				tc.parent, tc.badSub, diags[0].Message)
		}
	}
}

func TestAnalyze_ContainerDirective_ValidContents(t *testing.T) {
	for _, container := range []string{"handle", "handle_errors", "handle_path", "route"} {
		src := "example.com {\n\t" + container + " {\n\t\treverse_proxy localhost:8080\n\t}\n}\n"
		diags := analyze(src)
		if len(diags) != 0 {
			t.Errorf("%s with valid site directive: expected no diagnostics, got %d: %v", container, len(diags), diags)
		}
	}
}

func TestAnalyze_ContainerDirective_InvalidContents(t *testing.T) {
	for _, container := range []string{"handle", "handle_errors", "route"} {
		src := "example.com {\n\t" + container + " {\n\t\tnot_a_directive\n\t}\n}\n"
		diags := analyze(src)
		if len(diags) != 1 {
			t.Errorf("%s with invalid directive: expected 1 diagnostic, got %d: %v", container, len(diags), diags)
		}
	}
}

func TestAnalyze_FreeformBody_NoWarning(t *testing.T) {
	// basicauth, header, map have freeform bodies that must not be validated.
	cases := []string{
		"example.com {\n\tbasicauth {\n\t\tBob $2y$10$abc\n\t}\n}\n",
		"example.com {\n\theader {\n\t\tX-Custom-Header value\n\t\t-X-Remove-Me\n\t}\n}\n",
		"example.com {\n\tmap {path} {output} {\n\t\t/foo bar\n\t\tdefault baz\n\t}\n}\n",
	}
	for _, src := range cases {
		diags := analyze(src)
		if len(diags) != 0 {
			t.Errorf("freeform body should not produce diagnostics, got %d: %v", len(diags), diags)
		}
	}
}

func TestAnalyze_ImportInSubDirectiveBody_DefinedSnippet_NoWarning(t *testing.T) {
	// import inside a non-container directive body is valid when the snippet exists.
	src := "(my_snippet) {\n\theader_up X-Foo bar\n}\nexample.com {\n\treverse_proxy {\n\t\timport my_snippet\n\t\tto localhost:8080\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 0 {
		t.Errorf("import of defined snippet in directive body: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_ImportInSubDirectiveBody_UndefinedSnippet_Warning(t *testing.T) {
	// import inside a non-container directive body flags an undefined snippet.
	src := "example.com {\n\treverse_proxy {\n\t\timport ghost\n\t\tto localhost:8080\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("import of undefined snippet in directive body: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "ghost") {
		t.Errorf("diagnostic should name the undefined snippet, got: %q", diags[0].Message)
	}
}

func TestAnalyze_MatcherInSubDirectiveBody_NoWarning(t *testing.T) {
	// @matcher declarations inside a container directive body must not be flagged.
	src := "example.com {\n\thandle {\n\t\t@api path /api/*\n\t\treverse_proxy @api localhost:8080\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 0 {
		t.Errorf("matcher in container body: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

// --- snippet collection ------------------------------------------------------

func TestCollectSnippetNames_Empty(t *testing.T) {
	f, _ := parser.Parse("example.com {\n\trespond \"ok\"\n}\n")
	names := CollectSnippetNames(f)
	if len(names) != 0 {
		t.Errorf("expected no snippets, got %v", names)
	}
}

func TestCollectSnippetNames_Single(t *testing.T) {
	f, _ := parser.Parse("(mysnippet) {\n\trespond \"ok\"\n}\n")
	names := CollectSnippetNames(f)
	if len(names) != 1 || names[0] != "mysnippet" {
		t.Errorf("expected [mysnippet], got %v", names)
	}
}

func TestCollectSnippetNames_Multiple(t *testing.T) {
	src := "(alpha) {\n\trespond \"a\"\n}\n(beta) {\n\trespond \"b\"\n}\nexample.com {\n\trespond \"ok\"\n}\n"
	f, _ := parser.Parse(src)
	names := CollectSnippetNames(f)
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("expected [alpha beta] (sorted), got %v", names)
	}
}

// --- import validation -------------------------------------------------------

func TestAnalyze_ImportKnownSnippet_NoWarning(t *testing.T) {
	src := "(mysnippet) {\n\trespond \"ok\"\n}\nexample.com {\n\timport mysnippet\n}\n"
	diags := analyze(src)
	if len(diags) != 0 {
		t.Errorf("import of defined snippet: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_ImportUndefinedSnippet_Warning(t *testing.T) {
	src := "example.com {\n\timport no_such_snippet\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("import of undefined snippet: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "no_such_snippet") {
		t.Errorf("diagnostic should name the undefined snippet, got: %q", diags[0].Message)
	}
}

func TestAnalyze_ImportFilePath_NoWarning(t *testing.T) {
	// Paths and globs reference external files and must not be validated.
	cases := []string{
		"example.com {\n\timport /etc/caddy/common.conf\n}\n",
		"example.com {\n\timport ./snippets/*.conf\n}\n",
		"example.com {\n\timport ../shared/tls.conf\n}\n",
	}
	for _, src := range cases {
		diags := analyze(src)
		if len(diags) != 0 {
			t.Errorf("file import: expected no diagnostics, got %d: %v", len(diags), diags)
		}
	}
}

func TestAnalyze_ImportInContainerDirective_ValidatesSnippet(t *testing.T) {
	// import inside handle/route is validated because those are container directives.
	defined := "(mysnippet) {\n\trespond \"ok\"\n}\nexample.com {\n\thandle {\n\t\timport mysnippet\n\t}\n}\n"
	if diags := analyze(defined); len(diags) != 0 {
		t.Errorf("import of defined snippet in handle: expected no diagnostics, got %d: %v", len(diags), diags)
	}

	undefined := "example.com {\n\thandle {\n\t\timport ghost\n\t}\n}\n"
	if diags := analyze(undefined); len(diags) != 1 {
		t.Errorf("import of undefined snippet in handle: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_ImportGlobalLevel_ValidatesSnippet(t *testing.T) {
	// import at global-options level is also validated.
	undefined := "{\n\timport no_such_global_snippet\n}\nexample.com {\n\trespond \"ok\"\n}\n"
	diags := analyze(undefined)
	if len(diags) != 1 {
		t.Errorf("import of undefined snippet at global level: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
}

// --- sub-subdirective validation (e.g. transport http { … }) ------------------

func TestAnalyze_TransportHTTP_ValidSubSub_NoWarning(t *testing.T) {
	cases := []string{
		"example.com {\n\treverse_proxy localhost:8080 {\n\t\ttransport http {\n\t\t\tread_buffer 4kb\n\t\t}\n\t}\n}\n",
		"example.com {\n\treverse_proxy localhost:8080 {\n\t\ttransport http {\n\t\t\tdial_timeout 5s\n\t\t\ttls\n\t\t\tversions h2\n\t\t}\n\t}\n}\n",
		"example.com {\n\treverse_proxy localhost:8080 {\n\t\ttransport http {\n\t\t\tkeepalive 30s\n\t\t\tmax_conns_per_host 10\n\t\t}\n\t}\n}\n",
	}
	for _, src := range cases {
		if diags := analyze(src); len(diags) != 0 {
			t.Errorf("valid transport http subdirectives: expected no diagnostics, got %d: %v", len(diags), diags)
		}
	}
}

func TestAnalyze_TransportHTTP_UnknownSubSub_Warning(t *testing.T) {
	src := "example.com {\n\treverse_proxy http://{$LOCALHOST_GATEWAY}:3000 {\n\t\ttransport http {\n\t\t\tfoo\n\t\t}\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("unknown transport http subdirective: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "foo", "transport http") {
		t.Errorf("expected message mentioning %q and %q, got: %q", "foo", "transport http", diags[0].Message)
	}
}

func TestAnalyze_TransportFastCGI_ValidSubSub_NoWarning(t *testing.T) {
	src := "example.com {\n\treverse_proxy localhost:9000 {\n\t\ttransport fastcgi {\n\t\t\troot /var/www\n\t\t\tsplit .php\n\t\t}\n\t}\n}\n"
	if diags := analyze(src); len(diags) != 0 {
		t.Errorf("valid transport fastcgi subdirectives: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_TransportFastCGI_UnknownSubSub_Warning(t *testing.T) {
	src := "example.com {\n\treverse_proxy localhost:9000 {\n\t\ttransport fastcgi {\n\t\t\tbad_option\n\t\t}\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("unknown transport fastcgi subdirective: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "bad_option", "transport fastcgi") {
		t.Errorf("expected message mentioning %q and %q, got: %q", "bad_option", "transport fastcgi", diags[0].Message)
	}
}

// --- SubDirectivesFor --------------------------------------------------------

func TestSubDirectivesFor_KnownParent(t *testing.T) {
	subs, ok := SubDirectivesFor("reverse_proxy")
	if !ok {
		t.Fatal("reverse_proxy: expected ok=true")
	}
	if !subs["to"] || !subs["transport"] {
		t.Errorf("reverse_proxy: expected 'to' and 'transport' in subdirectives")
	}
}

func TestSubDirectivesFor_FreeformParent(t *testing.T) {
	subs, ok := SubDirectivesFor("basicauth")
	if !ok {
		t.Fatal("basicauth: expected ok=true (freeform)")
	}
	if subs != nil {
		t.Errorf("basicauth: expected nil subdirectives (freeform), got %v", subs)
	}
}

func TestSubDirectivesFor_UnknownParent(t *testing.T) {
	_, ok := SubDirectivesFor("not_a_known_directive")
	if ok {
		t.Error("unknown parent: expected ok=false")
	}
}

// --- parseSnippetName --------------------------------------------------------

func TestParseSnippetName_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"(mysnippet)", "mysnippet"},
		{"(my_snippet)", "my_snippet"},
		{"(a)", "a"},
	}
	for _, tc := range cases {
		name, ok := parseSnippetName(tc.input)
		if !ok || name != tc.want {
			t.Errorf("parseSnippetName(%q): want (%q, true), got (%q, %v)", tc.input, tc.want, name, ok)
		}
	}
}

func TestParseSnippetName_Invalid(t *testing.T) {
	// Empty parens "()" have len 2; single-char "(x)" is ok but "()" is not.
	cases := []string{"example.com", "localhost:8080", "", "()", "noparens"}
	for _, c := range cases {
		if _, ok := parseSnippetName(c); ok {
			t.Errorf("parseSnippetName(%q): expected false", c)
		}
	}
}

// --- isFileImport ------------------------------------------------------------

func TestIsFileImport_Paths(t *testing.T) {
	cases := []string{
		"/etc/caddy/common.conf",
		"./snippets/tls.conf",
		"../shared/common",
		"snippets/*.conf",
		`C:\caddy\conf`,
	}
	for _, c := range cases {
		if !isFileImport(c) {
			t.Errorf("isFileImport(%q): expected true", c)
		}
	}
}

func TestIsFileImport_SnippetNames(t *testing.T) {
	cases := []string{"mysnippet", "common_tls", "backend", "tls_opts"}
	for _, c := range cases {
		if isFileImport(c) {
			t.Errorf("isFileImport(%q): expected false", c)
		}
	}
}

// --- isCaddyPlaceholder ------------------------------------------------------

func TestIsCaddyPlaceholder_Valid(t *testing.T) {
	cases := []string{"{$VAR}", "{http.request.uri}", "{args[0]}"}
	for _, c := range cases {
		if !isCaddyPlaceholder(c) {
			t.Errorf("isCaddyPlaceholder(%q): expected true", c)
		}
	}
}

func TestIsCaddyPlaceholder_Invalid(t *testing.T) {
	cases := []string{"mysnippet", "{unclosed", "nobraces", ""}
	for _, c := range cases {
		if isCaddyPlaceholder(c) {
			t.Errorf("isCaddyPlaceholder(%q): expected false", c)
		}
	}
}

// --- multiple subdirective errors in one body --------------------------------

func TestAnalyze_MultipleUnknownSubDirectives(t *testing.T) {
	src := "example.com {\n\treverse_proxy {\n\t\tbad_one\n\t\tbad_two\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 2 {
		t.Errorf("two unknown subdirectives: expected 2 diagnostics, got %d: %v", len(diags), diags)
	}
}

// --- freeform body: @name matcher should not warn ----------------------------

func TestAnalyze_FreeformBody_MatcherNoWarning(t *testing.T) {
	// basicauth is freeform (nil subdirectives); body is not validated at all.
	src := "example.com {\n\tbasicauth {\n\t\t@admin ...\n\t}\n}\n"
	if diags := analyze(src); len(diags) != 0 {
		t.Errorf("matcher inside freeform body: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

// --- handle_path is a container directive ------------------------------------

func TestAnalyze_HandlePathContainer_ValidContents(t *testing.T) {
	src := "example.com {\n\thandle_path /static/* {\n\t\tfile_server\n\t}\n}\n"
	if diags := analyze(src); len(diags) != 0 {
		t.Errorf("handle_path with valid site directive: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_HandlePathContainer_UnknownDirective_Warning(t *testing.T) {
	src := "example.com {\n\thandle_path /static/* {\n\t\tnot_a_directive\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("handle_path with unknown directive: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "not_a_directive") {
		t.Errorf("expected message mentioning the unknown directive, got: %q", diags[0].Message)
	}
}

// --- transport without known arg: no false positive --------------------------

func TestAnalyze_TransportUnknownArg_BodyNotValidated(t *testing.T) {
	// "transport" with no arg or unknown arg — body is not validated because
	// the "transport:<arg>" key is absent from knownSubSubDirectives.
	src := "example.com {\n\treverse_proxy localhost:8080 {\n\t\ttransport {\n\t\t\tanything_goes\n\t\t}\n\t}\n}\n"
	diags := analyze(src)
	for _, d := range diags {
		if hasMsg([]protocol.Diagnostic{d}, "anything_goes") {
			t.Errorf("body of transport (no arg) must not be validated, got: %q", d.Message)
		}
	}
}

// --- import inside transport http body ---------------------------------------

func TestAnalyze_TransportHTTP_ImportDefinedSnippet_NoWarning(t *testing.T) {
	// Snippet body uses a known subdirective-level token (header_up is in
	// knownSubDirectiveParent) so it is accepted inside snippets.
	src := "(tls_opts) {\n\theader_up X-Foo bar\n}\n" +
		"example.com {\n\treverse_proxy localhost {\n\t\ttransport http {\n\t\t\timport tls_opts\n\t\t}\n\t}\n}\n"
	if diags := analyze(src); len(diags) != 0 {
		t.Errorf("import of defined snippet inside transport http: expected no diagnostics, got %d: %v", len(diags), diags)
	}
}

func TestAnalyze_TransportHTTP_ImportUndefinedSnippet_Warning(t *testing.T) {
	src := "example.com {\n\treverse_proxy localhost {\n\t\ttransport http {\n\t\t\timport no_such\n\t\t}\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 1 {
		t.Fatalf("import of undefined snippet inside transport http: expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !hasMsg(diags, "no_such") {
		t.Errorf("expected message mentioning snippet name, got: %q", diags[0].Message)
	}
}

// --- KnownTopLevel / KnownGlobalOptions maps ----------------------------------

func TestKnownTopLevel_NotEmpty(t *testing.T) {
	if len(KnownTopLevel) == 0 {
		t.Error("KnownTopLevel must not be empty")
	}
}

func TestKnownGlobalOptions_NotEmpty(t *testing.T) {
	if len(KnownGlobalOptions) == 0 {
		t.Error("KnownGlobalOptions must not be empty")
	}
}
