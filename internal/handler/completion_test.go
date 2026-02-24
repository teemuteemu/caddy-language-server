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

// --- atSiteBlockLevel --------------------------------------------------------

func TestAtSiteBlockLevel_InsideSiteBlock(t *testing.T) {
	src := "example.com {\n    reverse_proxy localhost\n}\n"
	f := parseAST(src)
	if !atSiteBlockLevel(f, 1) {
		t.Error("line inside site block: want true")
	}
}

func TestAtSiteBlockLevel_OutsideAllBlocks(t *testing.T) {
	src := "example.com {\n    reverse_proxy localhost\n}\n"
	f := parseAST(src)
	// Line 3 is beyond the closing brace on line 2.
	if atSiteBlockLevel(f, 3) {
		t.Error("line outside all site blocks: want false")
	}
}

func TestAtSiteBlockLevel_OnAddressLine(t *testing.T) {
	src := "example.com {\n    respond \"ok\"\n}\n"
	f := parseAST(src)
	// Line 0 is the address+brace line; atSiteBlockLevel should return false because
	// cursorLine (0) == sb.StartLine (0) and the check is cursorLine <= sb.StartLine.
	if atSiteBlockLevel(f, 0) {
		t.Error("cursor on address line: want false")
	}
}

func TestAtSiteBlockLevel_InsideNonContainerBody(t *testing.T) {
	// reverse_proxy is NOT a container — its body is for sub-directives only.
	src := "example.com {\n    reverse_proxy {\n        to localhost\n    }\n}\n"
	f := parseAST(src)
	// Line 2 is inside the reverse_proxy block.
	if atSiteBlockLevel(f, 2) {
		t.Error("line inside reverse_proxy body: want false")
	}
}

func TestAtSiteBlockLevel_InsideHandleContainer(t *testing.T) {
	src := "example.com {\n    handle /api/* {\n        reverse_proxy localhost\n    }\n}\n"
	f := parseAST(src)
	// Line 2 is inside handle { }, which is a container directive.
	if !atSiteBlockLevel(f, 2) {
		t.Error("line inside handle body: want true")
	}
}

func TestAtSiteBlockLevel_InsideRouteContainer(t *testing.T) {
	src := "example.com {\n    route {\n        file_server\n    }\n}\n"
	f := parseAST(src)
	if !atSiteBlockLevel(f, 2) {
		t.Error("line inside route body: want true")
	}
}

func TestAtSiteBlockLevel_InsideHandleErrorsContainer(t *testing.T) {
	src := "example.com {\n    handle_errors {\n        respond \"error\" 500\n    }\n}\n"
	f := parseAST(src)
	if !atSiteBlockLevel(f, 2) {
		t.Error("line inside handle_errors body: want true")
	}
}

func TestAtSiteBlockLevel_NestedContainer(t *testing.T) {
	// handle inside handle — both are containers.
	src := "example.com {\n    handle {\n        handle /inner/* {\n            respond \"inner\"\n        }\n    }\n}\n"
	f := parseAST(src)
	// Line 3 is inside the inner handle block.
	if !atSiteBlockLevel(f, 3) {
		t.Error("line inside nested handle body: want true")
	}
}

func TestAtSiteBlockLevel_EmptyFile(t *testing.T) {
	f := parseAST("")
	if atSiteBlockLevel(f, 0) {
		t.Error("empty file: want false")
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
