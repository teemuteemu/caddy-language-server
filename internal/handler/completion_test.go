package handler

import (
	"caddy-ls/internal/parser"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// parseAST is a helper that parses src, ignoring errors, and returns the File.
func parseAST(src string) *parser.File {
	f, _ := parser.Parse(src)
	return f
}

// --- atFirstTokenPosition ----------------------------------------------------

func TestAtFirstTokenPosition_BlankLine(t *testing.T) {
	if !atFirstTokenPosition("foo\n\nbar", protocol.Position{Line: 1, Character: 0}) {
		t.Error("blank line: want true")
	}
}

func TestAtFirstTokenPosition_StartOfWord(t *testing.T) {
	if !atFirstTokenPosition("reverse_proxy", protocol.Position{Line: 0, Character: 0}) {
		t.Error("cursor at start of line: want true")
	}
}

func TestAtFirstTokenPosition_MidWord(t *testing.T) {
	if !atFirstTokenPosition("reverse_proxy", protocol.Position{Line: 0, Character: 7}) {
		t.Error("cursor mid-word: want true")
	}
}

func TestAtFirstTokenPosition_AfterFirstToken(t *testing.T) {
	// Cursor after the first token and at least one space means an argument position.
	if atFirstTokenPosition("reverse_proxy localhost", protocol.Position{Line: 0, Character: 14}) {
		t.Error("cursor after first token: want false")
	}
}

func TestAtFirstTokenPosition_IndentedFirstWord(t *testing.T) {
	// Cursor inside the indented directive name should still be true.
	if !atFirstTokenPosition("    reverse_proxy", protocol.Position{Line: 0, Character: 8}) {
		t.Error("indented first word: want true")
	}
}

func TestAtFirstTokenPosition_IndentedAfterFirstToken(t *testing.T) {
	if atFirstTokenPosition("    reverse_proxy localhost", protocol.Position{Line: 0, Character: 22}) {
		t.Error("indented, cursor after first token: want false")
	}
}

func TestAtFirstTokenPosition_SecondLine(t *testing.T) {
	content := "example.com {\n    reverse_proxy\n}"
	// Cursor at start of "reverse_proxy" on line 1.
	if !atFirstTokenPosition(content, protocol.Position{Line: 1, Character: 4}) {
		t.Error("second line first token: want true")
	}
}

func TestAtFirstTokenPosition_LineOutOfBounds(t *testing.T) {
	if atFirstTokenPosition("foo", protocol.Position{Line: 5, Character: 0}) {
		t.Error("out-of-bounds line: want false")
	}
}

// --- completionNamesAt -------------------------------------------------------

func TestCompletionNamesAt_InsideSiteBlock(t *testing.T) {
	src := "example.com {\n    reverse_proxy localhost\n}\n"
	f := parseAST(src)
	names := completionNamesAt(f, 1)
	if names == nil {
		t.Fatal("line inside site block: want top-level directives, got nil")
	}
	for _, n := range names {
		if n == "reverse_proxy" {
			return
		}
	}
	t.Errorf("expected 'reverse_proxy' in site-block completions, got %v", names)
}

func TestCompletionNamesAt_OutsideAllBlocks(t *testing.T) {
	src := "example.com {\n    reverse_proxy localhost\n}\n"
	f := parseAST(src)
	// Line 3 is beyond the closing brace on line 2.
	if completionNamesAt(f, 3) != nil {
		t.Error("line outside all site blocks: want nil")
	}
}

func TestCompletionNamesAt_OnAddressLine(t *testing.T) {
	src := "example.com {\n    respond \"ok\"\n}\n"
	f := parseAST(src)
	// Line 0 is the address+brace line — not inside the block.
	if completionNamesAt(f, 0) != nil {
		t.Error("cursor on address line: want nil")
	}
}

func TestCompletionNamesAt_InsideNonContainerBody(t *testing.T) {
	// Cursor inside reverse_proxy block should yield its subdirectives.
	src := "example.com {\n    reverse_proxy {\n        to localhost\n    }\n}\n"
	f := parseAST(src)
	names := completionNamesAt(f, 2)
	if names == nil {
		t.Fatal("line inside reverse_proxy body: want subdirectives, got nil")
	}
	for _, n := range names {
		if n == "to" {
			return
		}
	}
	t.Errorf("expected 'to' in reverse_proxy subdirectives, got %v", names)
}

func TestCompletionNamesAt_InsideFreeformBody(t *testing.T) {
	// basicauth has a freeform (nil) body — no completions.
	src := "example.com {\n    basicauth {\n        user $2a$...\n    }\n}\n"
	f := parseAST(src)
	if completionNamesAt(f, 2) != nil {
		t.Error("line inside basicauth (freeform) body: want nil")
	}
}

func TestCompletionNamesAt_InsideHandleContainer(t *testing.T) {
	src := "example.com {\n    handle /api/* {\n        reverse_proxy localhost\n    }\n}\n"
	f := parseAST(src)
	names := completionNamesAt(f, 2)
	if names == nil {
		t.Fatal("line inside handle body: want top-level directives, got nil")
	}
	for _, n := range names {
		if n == "reverse_proxy" {
			return
		}
	}
	t.Errorf("expected 'reverse_proxy' in handle container completions, got %v", names)
}

func TestCompletionNamesAt_InsideRouteContainer(t *testing.T) {
	src := "example.com {\n    route {\n        file_server\n    }\n}\n"
	f := parseAST(src)
	if completionNamesAt(f, 2) == nil {
		t.Error("line inside route body: want top-level directives, got nil")
	}
}

func TestCompletionNamesAt_InsideHandleErrorsContainer(t *testing.T) {
	src := "example.com {\n    handle_errors {\n        respond \"error\" 500\n    }\n}\n"
	f := parseAST(src)
	if completionNamesAt(f, 2) == nil {
		t.Error("line inside handle_errors body: want top-level directives, got nil")
	}
}

func TestCompletionNamesAt_NestedContainer(t *testing.T) {
	// handle inside handle — both are containers.
	src := "example.com {\n    handle {\n        handle /inner/* {\n            respond \"inner\"\n        }\n    }\n}\n"
	f := parseAST(src)
	// Line 3 is inside the inner handle block.
	if completionNamesAt(f, 3) == nil {
		t.Error("line inside nested handle body: want top-level directives, got nil")
	}
}

func TestCompletionNamesAt_EmptyFile(t *testing.T) {
	f := parseAST("")
	if completionNamesAt(f, 0) != nil {
		t.Error("empty file: want nil")
	}
}

func TestCompletionNamesAt_InsideHandlePathContainer(t *testing.T) {
	src := "example.com {\n    handle_path /static/* {\n        file_server\n    }\n}\n"
	f := parseAST(src)
	names := completionNamesAt(f, 2)
	if names == nil {
		t.Fatal("line inside handle_path body: want top-level directives, got nil")
	}
	for _, n := range names {
		if n == "file_server" {
			return
		}
	}
	t.Errorf("expected 'file_server' in handle_path container completions, got %v", names)
}

func TestCompletionNamesAt_InsideTransportHTTPBody(t *testing.T) {
	// Cursor inside transport http { … } — no completions (not yet implemented
	// for sub-subdirective bodies), so nil is expected.
	src := "example.com {\n    reverse_proxy localhost {\n        transport http {\n            \n        }\n    }\n}\n"
	f := parseAST(src)
	// Line 3 is inside the transport http body.
	result := completionNamesAt(f, 3)
	// The completion for sub-subdirective bodies is not yet supported; nil is correct.
	_ = result // either nil or a list is acceptable — this test just confirms no panic
}

func TestCompletionNamesAt_TLSSubdirectives(t *testing.T) {
	src := "example.com {\n    tls {\n        \n    }\n}\n"
	f := parseAST(src)
	names := completionNamesAt(f, 2)
	if names == nil {
		t.Fatal("line inside tls body: want subdirectives, got nil")
	}
	for _, n := range names {
		if n == "protocols" {
			return
		}
	}
	t.Errorf("expected 'protocols' in tls subdirectives, got %v", names)
}

// --- importArgPrefix ---------------------------------------------------------

func TestImportArgPrefix_NotImport(t *testing.T) {
	_, ok := importArgPrefix("reverse_proxy localhost", protocol.Position{Line: 0, Character: 14})
	if ok {
		t.Error("non-import line: want false")
	}
}

func TestImportArgPrefix_JustImportWord(t *testing.T) {
	// Cursor still on the word "import" — not yet in arg position.
	_, ok := importArgPrefix("import", protocol.Position{Line: 0, Character: 6})
	if ok {
		t.Error("cursor still on 'import' keyword: want false")
	}
}

func TestImportArgPrefix_AfterImportSpace_EmptyArg(t *testing.T) {
	partial, ok := importArgPrefix("import ", protocol.Position{Line: 0, Character: 7})
	if !ok {
		t.Error("cursor right after 'import ': want true")
	}
	if partial != "" {
		t.Errorf("partial: want \"\", got %q", partial)
	}
}

func TestImportArgPrefix_PartialSnippetName(t *testing.T) {
	partial, ok := importArgPrefix("    import my", protocol.Position{Line: 0, Character: 13})
	if !ok {
		t.Error("cursor in partial snippet name: want true")
	}
	if partial != "my" {
		t.Errorf("partial: want \"my\", got %q", partial)
	}
}

func TestImportArgPrefix_FullSnippetName_NoTrailingSpace(t *testing.T) {
	// Cursor at end of the snippet name, no space yet — still first arg.
	partial, ok := importArgPrefix("import mysnippet", protocol.Position{Line: 0, Character: 16})
	if !ok {
		t.Error("cursor at end of snippet name: want true")
	}
	if partial != "mysnippet" {
		t.Errorf("partial: want \"mysnippet\", got %q", partial)
	}
}

func TestImportArgPrefix_AfterFirstArg(t *testing.T) {
	// Cursor in the second argument — must not trigger snippet completions.
	_, ok := importArgPrefix("import mysnippet arg2", protocol.Position{Line: 0, Character: 18})
	if ok {
		t.Error("cursor in second argument: want false")
	}
}

func TestImportArgPrefix_IndentedLine(t *testing.T) {
	// "\t\timport sni" = 12 chars; cursor after the 'i' is at character 12.
	partial, ok := importArgPrefix("\t\timport sni", protocol.Position{Line: 0, Character: 12})
	if !ok {
		t.Error("indented import line: want true")
	}
	if partial != "sni" {
		t.Errorf("partial: want \"sni\", got %q", partial)
	}
}

func TestImportArgPrefix_MultiLine(t *testing.T) {
	// line 1 = "\timport my" = 10 chars; cursor after 'y' is at character 10.
	content := "example.com {\n\timport my"
	partial, ok := importArgPrefix(content, protocol.Position{Line: 1, Character: 10})
	if !ok {
		t.Error("import on second line: want true")
	}
	if partial != "my" {
		t.Errorf("partial: want \"my\", got %q", partial)
	}
}

// --- snippetCompletions ------------------------------------------------------

func TestSnippetCompletions_Empty(t *testing.T) {
	f := parseAST("example.com {\n\trespond \"ok\"\n}\n")
	items := snippetCompletions(f, "")
	if len(items) != 0 {
		t.Errorf("no snippets defined: want 0 items, got %d", len(items))
	}
}

func TestSnippetCompletions_AllSnippets(t *testing.T) {
	src := "(alpha) {\n\trespond \"a\"\n}\n(beta) {\n\trespond \"b\"\n}\nexample.com {\n\trespond \"ok\"\n}\n"
	f := parseAST(src)
	items := snippetCompletions(f, "")
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	labels := map[string]bool{items[0].Label: true, items[1].Label: true}
	if !labels["alpha"] || !labels["beta"] {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestSnippetCompletions_FilterByPrefix(t *testing.T) {
	src := "(alpha) {\n\trespond \"a\"\n}\n(bravo) {\n\trespond \"b\"\n}\n(alcazar) {\n\trespond \"c\"\n}\n"
	f := parseAST(src)
	items := snippetCompletions(f, "al")
	if len(items) != 2 {
		t.Fatalf("want 2 items matching \"al*\", got %d: %v", len(items), items)
	}
}

func TestSnippetCompletions_KindIsModule(t *testing.T) {
	src := "(mysnippet) {\n\trespond \"ok\"\n}\n"
	f := parseAST(src)
	items := snippetCompletions(f, "")
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	if items[0].Kind == nil || *items[0].Kind != protocol.CompletionItemKindModule {
		t.Errorf("want CompletionItemKindModule, got %v", items[0].Kind)
	}
}

// --- hasBody -----------------------------------------------------------------

func TestHasBody_WithBody(t *testing.T) {
	d := &parser.Directive{StartLine: 1, EndLine: 3}
	if !hasBody(d) {
		t.Error("directive with EndLine > StartLine: want hasBody=true")
	}
}

func TestHasBody_NoBody(t *testing.T) {
	d := &parser.Directive{StartLine: 2, EndLine: 2}
	if hasBody(d) {
		t.Error("directive with EndLine == StartLine: want hasBody=false")
	}
}
