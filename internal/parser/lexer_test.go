package parser

import (
	"testing"
)

func TestTokenize_BasicTokenTypes(t *testing.T) {
	tokens := Tokenize("foo { bar }\n")
	want := []struct {
		typ TokenType
		val string
	}{
		{IDENT, "foo"},
		{LBRACE, "{"},
		{IDENT, "bar"},
		{RBRACE, "}"},
		{NEWLINE, "\n"},
		{EOF, ""},
	}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(tokens), len(want), tokens)
	}
	for i, w := range want {
		if tokens[i].Type != w.typ || tokens[i].Value != w.val {
			t.Errorf("token[%d]: got (%s %q), want (%s %q)", i, tokens[i].Type, tokens[i].Value, w.typ, w.val)
		}
	}
}

func TestTokenize_Comment(t *testing.T) {
	tokens := Tokenize("# this is a comment\nfoo")
	if tokens[0].Type != COMMENT {
		t.Errorf("expected COMMENT, got %s", tokens[0].Type)
	}
	if tokens[0].Value != "# this is a comment" {
		t.Errorf("unexpected comment value: %q", tokens[0].Value)
	}
	if tokens[1].Type != NEWLINE {
		t.Errorf("expected NEWLINE after comment, got %s", tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Value != "foo" {
		t.Errorf("expected IDENT 'foo', got %s %q", tokens[2].Type, tokens[2].Value)
	}
}

func TestTokenize_QuotedStrings(t *testing.T) {
	tokens := Tokenize(`"hello world" ` + "`backtick string`")
	if tokens[0].Type != STRING {
		t.Errorf("expected STRING, got %s", tokens[0].Type)
	}
	if tokens[1].Type != STRING {
		t.Errorf("expected STRING, got %s", tokens[1].Type)
	}
}

func TestTokenize_LineAndCharPositions(t *testing.T) {
	tokens := Tokenize("foo\nbar")
	// foo: line 0, char 0
	if tokens[0].Line != 0 || tokens[0].Char != 0 {
		t.Errorf("foo: want line=0 char=0, got line=%d char=%d", tokens[0].Line, tokens[0].Char)
	}
	// bar: line 1, char 0
	bar := tokens[2] // [foo, \n, bar, EOF]
	if bar.Line != 1 || bar.Char != 0 {
		t.Errorf("bar: want line=1 char=0, got line=%d char=%d", bar.Line, bar.Char)
	}
}

func TestTokenize_CharOffsets(t *testing.T) {
	tokens := Tokenize("foo bar")
	// foo at char 0, bar at char 4
	if tokens[0].Char != 0 {
		t.Errorf("foo char: want 0, got %d", tokens[0].Char)
	}
	if tokens[1].Char != 4 {
		t.Errorf("bar char: want 4, got %d", tokens[1].Char)
	}
}

func TestTokenize_EnvVarPlaceholder(t *testing.T) {
	tokens := Tokenize(`http://{$LOCALHOST_GATEWAY}:3000`)
	// Should be a single IDENT token, not split at the braces.
	if len(tokens) != 2 { // IDENT + EOF
		t.Fatalf("got %d tokens, want 2 (IDENT + EOF): %v", len(tokens), tokens)
	}
	if tokens[0].Type != IDENT {
		t.Errorf("expected IDENT, got %s", tokens[0].Type)
	}
	if tokens[0].Value != `http://{$LOCALHOST_GATEWAY}:3000` {
		t.Errorf("unexpected value: %q", tokens[0].Value)
	}
}

func TestTokenize_StandalonePlaceholder(t *testing.T) {
	tokens := Tokenize(`{$UPSTREAM}`)
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2: %v", len(tokens), tokens)
	}
	if tokens[0].Type != IDENT || tokens[0].Value != `{$UPSTREAM}` {
		t.Errorf("got (%s %q), want IDENT {$UPSTREAM}", tokens[0].Type, tokens[0].Value)
	}
}

func TestTokenize_RuntimePlaceholder(t *testing.T) {
	tokens := Tokenize(`{http.request.remote.host}`)
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2: %v", len(tokens), tokens)
	}
	if tokens[0].Value != `{http.request.remote.host}` {
		t.Errorf("unexpected value: %q", tokens[0].Value)
	}
}

func TestTokenize_PlaceholderInPath(t *testing.T) {
	tokens := Tokenize(`/var/www/{env.APP_DIR}/public`)
	if len(tokens) != 2 {
		t.Fatalf("got %d tokens, want 2: %v", len(tokens), tokens)
	}
	if tokens[0].Value != `/var/www/{env.APP_DIR}/public` {
		t.Errorf("unexpected value: %q", tokens[0].Value)
	}
}

func TestTokenize_BlockBraceNotPlaceholder(t *testing.T) {
	// A bare '{' followed by a newline or space is a block brace, not a placeholder.
	tokens := Tokenize("example.com {\n}")
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	want := []TokenType{IDENT, LBRACE, NEWLINE, RBRACE, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("got tokens %v, want types %v", tokens, want)
	}
	for i, w := range want {
		if types[i] != w {
			t.Errorf("token[%d]: got %s, want %s", i, types[i], w)
		}
	}
}

func TestTokenize_Empty(t *testing.T) {
	tokens := Tokenize("")
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("empty input: want [EOF], got %v", tokens)
	}
}

func TestTokenize_CRLF(t *testing.T) {
	// \r should be skipped; \n should produce NEWLINE.
	tokens := Tokenize("foo\r\nbar")
	// Expect: IDENT foo, NEWLINE, IDENT bar, EOF
	if len(tokens) != 4 {
		t.Fatalf("got %d tokens, want 4: %v", len(tokens), tokens)
	}
	if tokens[1].Type != NEWLINE {
		t.Errorf("expected NEWLINE, got %s", tokens[1].Type)
	}
}

func TestTokenize_MultipleAddresses(t *testing.T) {
	tokens := Tokenize("example.com www.example.com {")
	if tokens[0].Value != "example.com" {
		t.Errorf("unexpected first token: %q", tokens[0].Value)
	}
	if tokens[1].Value != "www.example.com" {
		t.Errorf("unexpected second token: %q", tokens[1].Value)
	}
	if tokens[2].Type != LBRACE {
		t.Errorf("expected LBRACE, got %s", tokens[2].Type)
	}
}
