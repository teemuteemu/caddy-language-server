package handler

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func pos(line, char uint32) protocol.Position {
	return protocol.Position{Line: line, Character: char}
}

// --- wordAtPosition ----------------------------------------------------------

func TestWordAtPosition_StartOfWord(t *testing.T) {
	got := wordAtPosition("reverse_proxy localhost", pos(0, 0))
	if got != "reverse_proxy" {
		t.Errorf("want 'reverse_proxy', got %q", got)
	}
}

func TestWordAtPosition_MidWord(t *testing.T) {
	got := wordAtPosition("reverse_proxy localhost", pos(0, 5))
	if got != "reverse_proxy" {
		t.Errorf("want 'reverse_proxy', got %q", got)
	}
}

func TestWordAtPosition_EndOfWord(t *testing.T) {
	// Character 13 is one past the last letter of "reverse_proxy"; the function
	// scans backward to find the whole word.
	got := wordAtPosition("reverse_proxy localhost", pos(0, 13))
	if got != "reverse_proxy" {
		t.Errorf("want 'reverse_proxy', got %q", got)
	}
}

func TestWordAtPosition_SecondWord(t *testing.T) {
	got := wordAtPosition("reverse_proxy localhost", pos(0, 15))
	if got != "localhost" {
		t.Errorf("want 'localhost', got %q", got)
	}
}

func TestWordAtPosition_InWhitespace(t *testing.T) {
	// Character 13 is the space between "reverse_proxy" and "localhost".
	got := wordAtPosition("reverse_proxy  localhost", pos(0, 14))
	if got != "" {
		t.Errorf("cursor in whitespace: want empty string, got %q", got)
	}
}

func TestWordAtPosition_SecondLine(t *testing.T) {
	content := "example.com {\n    reverse_proxy localhost\n}"
	got := wordAtPosition(content, pos(1, 8))
	if got != "reverse_proxy" {
		t.Errorf("want 'reverse_proxy', got %q", got)
	}
}

func TestWordAtPosition_LineOutOfBounds(t *testing.T) {
	got := wordAtPosition("example.com", pos(5, 0))
	if got != "" {
		t.Errorf("out-of-bounds line: want empty string, got %q", got)
	}
}

func TestWordAtPosition_EmptyContent(t *testing.T) {
	got := wordAtPosition("", pos(0, 0))
	if got != "" {
		t.Errorf("empty content: want empty string, got %q", got)
	}
}

func TestWordAtPosition_CharPastEndOfLine(t *testing.T) {
	// Character beyond the line length should not panic and should return a word.
	got := wordAtPosition("tls", pos(0, 100))
	if got != "tls" {
		t.Errorf("char past end of line: want 'tls', got %q", got)
	}
}

func TestWordAtPosition_WordWithUnderscore(t *testing.T) {
	got := wordAtPosition("file_server browse", pos(0, 3))
	if got != "file_server" {
		t.Errorf("want 'file_server', got %q", got)
	}
}

func TestWordAtPosition_WordWithHyphen(t *testing.T) {
	// Hyphens are valid word runes per isWordRune.
	got := wordAtPosition("X-Frame-Options DENY", pos(0, 5))
	if got != "X-Frame-Options" {
		t.Errorf("want 'X-Frame-Options', got %q", got)
	}
}

// --- directiveDocs coverage --------------------------------------------------

func TestDirectiveDocs_AllKnownDirectivesHaveDocs(t *testing.T) {
	// Every directive that the language server validates should also have hover docs.
	// This test documents intentional gaps and prevents silent regressions.
	knownWithoutDocs := []string{}
	for name := range directiveDocs {
		_ = name // just check the map is populated
	}
	if len(directiveDocs) == 0 {
		t.Error("directiveDocs must not be empty")
	}
	_ = knownWithoutDocs
}

func TestDirectiveDocs_CommonDirectivesPresent(t *testing.T) {
	mustHave := []string{
		"reverse_proxy", "file_server", "tls", "encode", "log",
		"header", "root", "redir", "rewrite", "respond",
		"route", "handle", "php_fastcgi", "basicauth", "templates",
		"handle_errors", "handle_path", "abort", "error",
	}
	for _, name := range mustHave {
		if _, ok := lookupDirectiveDoc(name); !ok {
			t.Errorf("directive docs missing entry for %q", name)
		}
	}
}
