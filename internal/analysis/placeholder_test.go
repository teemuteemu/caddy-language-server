package analysis

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// --- checkPlaceholderBalance unit tests --------------------------------------

func TestCheckPlaceholderBalance_Valid(t *testing.T) {
	cases := []string{
		"",
		"no-braces",
		"{$VAR}",
		"{http.request.uri}",
		"{args[0]}",
		"prefix{$VAR}suffix",
		"http://backend/{path}",
		`"quoted {$VAR} string"`,
		`{a}{b}`,
		`{http.vars.{path.1}}`, // nested (valid in Caddy replacer)
		`\{literal\}`,          // escaped braces
	}
	for _, s := range cases {
		if msg := checkPlaceholderBalance(s); msg != "" {
			t.Errorf("expected balanced for %q, got: %s", s, msg)
		}
	}
}

func TestCheckPlaceholderBalance_UnclosedOpen(t *testing.T) {
	cases := []string{
		"{",
		"{$VAR",
		"{http.request.uri",
		"prefix{unclosed",
		"http://backend/{path",
	}
	for _, s := range cases {
		if msg := checkPlaceholderBalance(s); msg == "" {
			t.Errorf("expected error for unclosed %q, got none", s)
		}
	}
}

func TestCheckPlaceholderBalance_UnmatchedClose(t *testing.T) {
	cases := []string{
		"}",
		"$VAR}",
		"http.request.uri}",
		"prefix}suffix",
	}
	for _, s := range cases {
		if msg := checkPlaceholderBalance(s); msg == "" {
			t.Errorf("expected error for unmatched close in %q, got none", s)
		}
	}
}

func TestCheckPlaceholderBalance_EscapedBracesIgnored(t *testing.T) {
	// \{ and \} are literal characters and must not affect bracket depth.
	if msg := checkPlaceholderBalance(`\{not a placeholder\}`); msg != "" {
		t.Errorf("escaped braces should not trigger error, got: %s", msg)
	}
	// An unescaped open after escaped sequence should still be caught.
	if msg := checkPlaceholderBalance(`\{ok\} {unclosed`); msg == "" {
		t.Error("unclosed { after escaped sequence should trigger error")
	}
}

// --- integration tests via Analyze -------------------------------------------

func TestAnalyze_ValidPlaceholders_NoError(t *testing.T) {
	cases := []string{
		// env var in arg
		"example.com {\n\treverse_proxy {$UPSTREAM}\n}\n",
		// runtime placeholder in arg
		"example.com {\n\theader X-Uri {http.request.uri}\n}\n",
		// placeholder embedded in URL arg
		"example.com {\n\treverse_proxy http://backend/{path}\n}\n",
		// placeholder in site address
		"{$SITE_ADDR} {\n\treverse_proxy localhost:8080\n}\n",
		// multiple balanced placeholders on one line
		"example.com {\n\trespond {http.request.method} 200\n}\n",
	}
	for _, src := range cases {
		diags := analyze(src)
		// Filter to error-severity placeholder diagnostics only.
		for _, d := range diags {
			if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
				t.Errorf("expected no error diagnostics for %q, got: %v", src, d.Message)
			}
		}
	}
}

func TestAnalyze_UnclosedPlaceholder_Error(t *testing.T) {
	src := "example.com {\n\treverse_proxy {$UPSTREAM\n}\n"
	diags := analyze(src)
	hasError := false
	for _, d := range diags {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected an error diagnostic for unclosed placeholder, got: %v", diags)
	}
}

func TestAnalyze_UnmatchedCloseBrace_Error(t *testing.T) {
	src := "example.com {\n\treverse_proxy $UPSTREAM}\n}\n"
	diags := analyze(src)
	hasError := false
	for _, d := range diags {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected an error diagnostic for unmatched '}', got: %v", diags)
	}
}

func TestAnalyze_UnclosedPlaceholderInSiteAddress_Error(t *testing.T) {
	// Site addresses with unbalanced braces should produce an error.
	src := "{$SITE_ADDR {\n\treverse_proxy localhost:8080\n}\n"
	diags := analyze(src)
	hasError := false
	for _, d := range diags {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected an error diagnostic for unclosed placeholder in site address, got: %v", diags)
	}
}

func TestAnalyze_UnclosedPlaceholderInNestedArg_Error(t *testing.T) {
	// Unbalanced placeholder in a subdirective arg.
	src := "example.com {\n\treverse_proxy {\n\t\theader_up X-Forwarded-Host {http.request.host\n\t}\n}\n"
	diags := analyze(src)
	hasError := false
	for _, d := range diags {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected an error diagnostic for unclosed placeholder in nested arg, got: %v", diags)
	}
}
