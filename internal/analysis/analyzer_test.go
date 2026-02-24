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

func TestAnalyze_SnippetBlockSkipped(t *testing.T) {
	// Snippet bodies can contain arbitrary sub-directives; they must not be validated.
	diags := analyze("(mysnippet) {\n\tunknown_directive\n\tanother_bad_one baz\n}\n")
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for snippet block, got %d: %v", len(diags), diags)
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

func TestAnalyze_ImportInSubDirectiveBody_NoWarning(t *testing.T) {
	// import is valid anywhere in a Caddyfile, including inside directive bodies.
	src := "example.com {\n\treverse_proxy {\n\t\timport my_snippet\n\t\tto localhost:8080\n\t}\n}\n"
	diags := analyze(src)
	if len(diags) != 0 {
		t.Errorf("import in directive body: expected no diagnostics, got %d: %v", len(diags), diags)
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
