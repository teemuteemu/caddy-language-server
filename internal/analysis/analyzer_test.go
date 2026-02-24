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
