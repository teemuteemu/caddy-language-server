package parser

import (
	"testing"
)

// ---- helpers ----------------------------------------------------------------

func mustParse(t *testing.T, src string) *File {
	t.Helper()
	f, errs := Parse(src)
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	return f
}

func assertNoErrors(t *testing.T, errs []*ParseError) {
	t.Helper()
	if len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
}

// ---- site block tests -------------------------------------------------------

func TestParse_SimpleSiteBlock(t *testing.T) {
	src := "example.com {\n\troot * /var/www\n}\n"
	f := mustParse(t, src)

	if len(f.SiteBlocks) != 1 {
		t.Fatalf("want 1 site block, got %d", len(f.SiteBlocks))
	}
	sb := f.SiteBlocks[0]
	if len(sb.Addresses) != 1 || sb.Addresses[0].Value != "example.com" {
		t.Errorf("unexpected addresses: %v", sb.Addresses)
	}
	if len(sb.Directives) != 1 {
		t.Fatalf("want 1 directive, got %d", len(sb.Directives))
	}
	d := sb.Directives[0]
	if d.Name.Value != "root" {
		t.Errorf("directive name: want 'root', got %q", d.Name.Value)
	}
	if len(d.Args) != 2 || d.Args[0].Token.Value != "*" || d.Args[1].Token.Value != "/var/www" {
		t.Errorf("directive args: %v", d.Args)
	}
}

func TestParse_MultipleAddresses(t *testing.T) {
	src := "example.com www.example.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)

	if len(f.SiteBlocks) != 1 {
		t.Fatalf("want 1 site block, got %d", len(f.SiteBlocks))
	}
	sb := f.SiteBlocks[0]
	if len(sb.Addresses) != 2 {
		t.Fatalf("want 2 addresses, got %d", len(sb.Addresses))
	}
}

func TestParse_MultipleSiteBlocks(t *testing.T) {
	src := "a.com {\n\trespond \"a\"\n}\nb.com {\n\trespond \"b\"\n}\n"
	f := mustParse(t, src)

	if len(f.SiteBlocks) != 2 {
		t.Fatalf("want 2 site blocks, got %d", len(f.SiteBlocks))
	}
	if f.SiteBlocks[0].Addresses[0].Value != "a.com" {
		t.Errorf("first block address: want a.com, got %q", f.SiteBlocks[0].Addresses[0].Value)
	}
	if f.SiteBlocks[1].Addresses[0].Value != "b.com" {
		t.Errorf("second block address: want b.com, got %q", f.SiteBlocks[1].Addresses[0].Value)
	}
}

// ---- global block tests -----------------------------------------------------

func TestParse_GlobalBlock(t *testing.T) {
	src := "{\n\temail admin@example.com\n}\nexample.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)

	if f.GlobalBlock == nil {
		t.Fatal("expected global block, got nil")
	}
	if len(f.GlobalBlock.Directives) != 1 {
		t.Fatalf("want 1 global directive, got %d", len(f.GlobalBlock.Directives))
	}
	if f.GlobalBlock.Directives[0].Name.Value != "email" {
		t.Errorf("global directive: want 'email', got %q", f.GlobalBlock.Directives[0].Name.Value)
	}
	if len(f.SiteBlocks) != 1 {
		t.Errorf("want 1 site block after global, got %d", len(f.SiteBlocks))
	}
}

func TestParse_NoGlobalBlock(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)

	if f.GlobalBlock != nil {
		t.Errorf("expected no global block, got one")
	}
}

// ---- directive argument tests -----------------------------------------------

func TestParse_DirectiveNoArgs(t *testing.T) {
	src := "example.com {\n\tfile_server\n}\n"
	f := mustParse(t, src)

	d := f.SiteBlocks[0].Directives[0]
	if d.Name.Value != "file_server" {
		t.Errorf("want 'file_server', got %q", d.Name.Value)
	}
	if len(d.Args) != 0 {
		t.Errorf("want 0 args, got %d", len(d.Args))
	}
}

func TestParse_DirectiveMultipleArgs(t *testing.T) {
	src := "example.com {\n\theader Content-Type text/plain\n}\n"
	f := mustParse(t, src)

	d := f.SiteBlocks[0].Directives[0]
	if len(d.Args) != 2 {
		t.Fatalf("want 2 args, got %d", len(d.Args))
	}
	if d.Args[0].Token.Value != "Content-Type" {
		t.Errorf("arg[0]: want 'Content-Type', got %q", d.Args[0].Token.Value)
	}
	if d.Args[1].Token.Value != "text/plain" {
		t.Errorf("arg[1]: want 'text/plain', got %q", d.Args[1].Token.Value)
	}
}

func TestParse_DirectiveWithBody(t *testing.T) {
	src := "example.com {\n\treverse_proxy {\n\t\theader_up Host {upstream_hostport}\n\t}\n}\n"
	f := mustParse(t, src)

	d := f.SiteBlocks[0].Directives[0]
	if d.Name.Value != "reverse_proxy" {
		t.Errorf("want 'reverse_proxy', got %q", d.Name.Value)
	}
	if len(d.Body) != 1 {
		t.Fatalf("want 1 sub-directive, got %d", len(d.Body))
	}
	if d.Body[0].Name.Value != "header_up" {
		t.Errorf("sub-directive: want 'header_up', got %q", d.Body[0].Name.Value)
	}
}

// ---- placeholder tests -------------------------------------------------------

func TestParse_EnvVarPlaceholderInArg(t *testing.T) {
	src := "example.com {\n\treverse_proxy http://{$LOCALHOST_GATEWAY}:3000\n}\n"
	f, errs := Parse(src)
	assertNoErrors(t, errs)

	d := f.SiteBlocks[0].Directives[0]
	if d.Name.Value != "reverse_proxy" {
		t.Fatalf("directive name: want 'reverse_proxy', got %q", d.Name.Value)
	}
	if len(d.Args) != 1 {
		t.Fatalf("want 1 arg, got %d: %v", len(d.Args), d.Args)
	}
	if d.Args[0].Token.Value != "http://{$LOCALHOST_GATEWAY}:3000" {
		t.Errorf("arg value: got %q", d.Args[0].Token.Value)
	}
}

func TestParse_StandalonePlaceholderArg(t *testing.T) {
	src := "example.com {\n\treverse_proxy {$UPSTREAM}\n}\n"
	f, errs := Parse(src)
	assertNoErrors(t, errs)

	d := f.SiteBlocks[0].Directives[0]
	if len(d.Args) != 1 || d.Args[0].Token.Value != "{$UPSTREAM}" {
		t.Errorf("unexpected args: %v", d.Args)
	}
}

func TestParse_PlaceholderInHeader(t *testing.T) {
	src := "example.com {\n\theader X-Real-IP {http.request.remote.host}\n}\n"
	f, errs := Parse(src)
	assertNoErrors(t, errs)

	d := f.SiteBlocks[0].Directives[0]
	if len(d.Args) != 2 {
		t.Fatalf("want 2 args, got %d", len(d.Args))
	}
	if d.Args[1].Token.Value != "{http.request.remote.host}" {
		t.Errorf("arg[1]: got %q", d.Args[1].Token.Value)
	}
}

// ---- comment / whitespace tests ---------------------------------------------

func TestParse_CommentsIgnored(t *testing.T) {
	src := "# top comment\nexample.com {\n\t# inline comment\n\trespond \"ok\" # trailing\n}\n"
	f := mustParse(t, src)

	if len(f.SiteBlocks) != 1 {
		t.Fatalf("want 1 site block, got %d", len(f.SiteBlocks))
	}
	if len(f.SiteBlocks[0].Directives) != 1 {
		t.Fatalf("want 1 directive, got %d", len(f.SiteBlocks[0].Directives))
	}
}

// ---- error recovery tests ---------------------------------------------------

func TestParse_UnclosedBlock(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n"
	_, errs := Parse(src)
	if len(errs) == 0 {
		t.Error("expected parse error for unclosed block, got none")
	}
}

func TestParse_StrayClosingBrace(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n}\n}\n"
	_, errs := Parse(src)
	if len(errs) == 0 {
		t.Error("expected parse error for stray '}', got none")
	}
}

func TestParse_UnclosedGlobalBlock(t *testing.T) {
	src := "{\n\temail admin@example.com\n"
	_, errs := Parse(src)
	if len(errs) == 0 {
		t.Error("expected parse error for unclosed global block, got none")
	}
}

// ---- line number tests ------------------------------------------------------

func TestParse_DirectiveLineNumbers(t *testing.T) {
	src := "example.com {\n\troot * /var/www\n\tfile_server\n}\n"
	f := mustParse(t, src)

	dirs := f.SiteBlocks[0].Directives
	if dirs[0].StartLine != 1 {
		t.Errorf("root StartLine: want 1, got %d", dirs[0].StartLine)
	}
	if dirs[1].StartLine != 2 {
		t.Errorf("file_server StartLine: want 2, got %d", dirs[1].StartLine)
	}
}

func TestParse_SiteBlockLineNumbers(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)

	sb := f.SiteBlocks[0]
	if sb.StartLine != 0 {
		t.Errorf("StartLine: want 0, got %d", sb.StartLine)
	}
	if sb.EndLine != 2 {
		t.Errorf("EndLine: want 2, got %d", sb.EndLine)
	}
}

// ---- empty / minimal inputs -------------------------------------------------

func TestParse_EmptyInput(t *testing.T) {
	f := mustParse(t, "")
	if f.GlobalBlock != nil {
		t.Error("expected nil global block for empty input")
	}
	if len(f.SiteBlocks) != 0 {
		t.Errorf("expected 0 site blocks for empty input, got %d", len(f.SiteBlocks))
	}
}

func TestParse_OnlyComments(t *testing.T) {
	src := "# just a comment\n# another\n"
	f := mustParse(t, src)
	if len(f.SiteBlocks) != 0 {
		t.Errorf("expected 0 site blocks, got %d", len(f.SiteBlocks))
	}
}

// ---- ParseError -------------------------------------------------------------

func TestParseError_ErrorMessage(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n" // unclosed
	_, errs := Parse(src)
	if len(errs) == 0 {
		t.Fatal("expected at least one parse error")
	}
	// Error() must return a non-empty string matching the Message field.
	for _, e := range errs {
		if e.Error() != e.Message {
			t.Errorf("Error() = %q, want %q", e.Error(), e.Message)
		}
		if e.Error() == "" {
			t.Error("Error() must not be empty")
		}
	}
}

func TestParse_MultipleErrors(t *testing.T) {
	// Two stray closing braces should each produce a parse error.
	src := "}\n}\n"
	_, errs := Parse(src)
	if len(errs) < 2 {
		t.Errorf("expected >=2 parse errors, got %d: %v", len(errs), errs)
	}
}

// ---- empty / edge-case inputs -----------------------------------------------

func TestParse_EmptyGlobalBlock(t *testing.T) {
	src := "{\n}\nexample.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)
	if f.GlobalBlock == nil {
		t.Fatal("expected global block, got nil")
	}
	if len(f.GlobalBlock.Directives) != 0 {
		t.Errorf("empty global block: want 0 directives, got %d", len(f.GlobalBlock.Directives))
	}
	if len(f.SiteBlocks) != 1 {
		t.Errorf("want 1 site block, got %d", len(f.SiteBlocks))
	}
}

// ---- nested directive bodies ------------------------------------------------

func TestParse_ThreeLevelNesting(t *testing.T) {
	// handle > reverse_proxy > transport
	src := "example.com {\n\thandle {\n\t\treverse_proxy {\n\t\t\ttransport http {\n\t\t\t\ttls\n\t\t\t}\n\t\t}\n\t}\n}\n"
	f := mustParse(t, src)

	if len(f.SiteBlocks) != 1 {
		t.Fatalf("want 1 site block, got %d", len(f.SiteBlocks))
	}
	handle := f.SiteBlocks[0].Directives[0]
	if handle.Name.Value != "handle" {
		t.Fatalf("want 'handle', got %q", handle.Name.Value)
	}
	if len(handle.Body) != 1 {
		t.Fatalf("handle body: want 1, got %d", len(handle.Body))
	}
	rp := handle.Body[0]
	if rp.Name.Value != "reverse_proxy" {
		t.Fatalf("want 'reverse_proxy', got %q", rp.Name.Value)
	}
	if len(rp.Body) != 1 {
		t.Fatalf("reverse_proxy body: want 1, got %d", len(rp.Body))
	}
	transport := rp.Body[0]
	if transport.Name.Value != "transport" {
		t.Fatalf("want 'transport', got %q", transport.Name.Value)
	}
	if len(transport.Args) != 1 || transport.Args[0].Token.Value != "http" {
		t.Errorf("transport args: want [http], got %v", transport.Args)
	}
	if len(transport.Body) != 1 || transport.Body[0].Name.Value != "tls" {
		t.Errorf("transport body: want [tls], got %v", transport.Body)
	}
}

// ---- File.Range() -----------------------------------------------------------

func TestFileRange_EmptyFile(t *testing.T) {
	f := mustParse(t, "")
	rng := f.Range()
	if rng.Start.Line != 0 || rng.Start.Character != 0 {
		t.Errorf("empty file range start: want {0,0}, got %v", rng.Start)
	}
}

func TestFileRange_NonEmpty(t *testing.T) {
	src := "example.com {\n\trespond \"ok\"\n}\n"
	f := mustParse(t, src)
	rng := f.Range()
	if rng.Start.Line != 0 {
		t.Errorf("file range start line: want 0, got %d", rng.Start.Line)
	}
	if rng.End.Line == 0 {
		t.Errorf("file range end line: want >0, got 0")
	}
}
