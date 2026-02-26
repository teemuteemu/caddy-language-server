package parser

import (
	"testing"
)

func TestTokenize_BasicTokenTypes(t *testing.T) {
	// Caddy's tokenizer strips comments and does not emit NEWLINE tokens.
	// "foo { bar }\n" produces: IDENT, LBRACE, IDENT, RBRACE, EOF.
	tokens := Tokenize("foo { bar }\n")
	want := []struct {
		typ TokenType
		val string
	}{
		{IDENT, "foo"},
		{LBRACE, "{"},
		{IDENT, "bar"},
		{RBRACE, "}"},
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
	// Caddy's tokenizer strips comments; only the non-comment tokens remain.
	tokens := Tokenize("# this is a comment\nfoo")
	if len(tokens) != 2 { // IDENT "foo" + EOF
		t.Fatalf("expected 2 tokens (IDENT + EOF), got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Type != IDENT || tokens[0].Value != "foo" {
		t.Errorf("expected IDENT 'foo', got %s %q", tokens[0].Type, tokens[0].Value)
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
	// Caddy's tokenizer does not emit NEWLINE tokens, so the token slice is
	// [foo, bar, EOF] rather than [foo, NEWLINE, bar, EOF].
	tokens := Tokenize("foo\nbar")
	// foo: line 0, char 0
	if tokens[0].Line != 0 || tokens[0].Char != 0 {
		t.Errorf("foo: want line=0 char=0, got line=%d char=%d", tokens[0].Line, tokens[0].Char)
	}
	// bar: line 1, char 0
	bar := tokens[1] // [foo, bar, EOF]
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
	// Caddy's tokenizer does not emit NEWLINE tokens.
	tokens := Tokenize("example.com {\n}")
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	want := []TokenType{IDENT, LBRACE, RBRACE, EOF}
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
	// \r\n line endings are handled; foo and bar end up on separate lines.
	// Caddy's tokenizer does not emit NEWLINE tokens.
	tokens := Tokenize("foo\r\nbar")
	// Expect: IDENT foo, IDENT bar, EOF
	if len(tokens) != 3 {
		t.Fatalf("got %d tokens, want 3: %v", len(tokens), tokens)
	}
	if tokens[0].Type != IDENT || tokens[0].Value != "foo" {
		t.Errorf("token[0]: want IDENT 'foo', got %s %q", tokens[0].Type, tokens[0].Value)
	}
	if tokens[1].Type != IDENT || tokens[1].Value != "bar" {
		t.Errorf("token[1]: want IDENT 'bar', got %s %q", tokens[1].Type, tokens[1].Value)
	}
	if tokens[0].Line != 0 || tokens[1].Line != 1 {
		t.Errorf("line numbers: foo line=%d, bar line=%d; want 0, 1", tokens[0].Line, tokens[1].Line)
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

func TestTokenize_TabIndentedToken(t *testing.T) {
	// A tab-indented directive: the column must reflect the position after the tab.
	tokens := Tokenize("example.com {\n\tfile_server\n}")
	// Expect: example.com, {, file_server, }, EOF
	var fsToken *Token
	for i := range tokens {
		if tokens[i].Value == "file_server" {
			fsToken = &tokens[i]
			break
		}
	}
	if fsToken == nil {
		t.Fatal("file_server token not found")
	}
	if fsToken.Line != 1 {
		t.Errorf("file_server line: want 1, got %d", fsToken.Line)
	}
	// Tab is 1 byte at offset 0; file_server starts at byte offset 1 → char 1.
	if fsToken.Char != 1 {
		t.Errorf("file_server char: want 1, got %d", fsToken.Char)
	}
}

func TestTokenize_QuotedStringColumnOffset(t *testing.T) {
	// "respond" starts at char 0; "hello world" (quoted) starts at char 8.
	tokens := Tokenize(`respond "hello world"`)
	if len(tokens) < 3 {
		t.Fatalf("expected >=3 tokens, got %d", len(tokens))
	}
	if tokens[0].Value != "respond" || tokens[0].Char != 0 {
		t.Errorf("respond: want char=0, got char=%d", tokens[0].Char)
	}
	if tokens[1].Type != STRING {
		t.Errorf("second token: want STRING, got %s", tokens[1].Type)
	}
	if tokens[1].Char != 8 {
		t.Errorf("quoted string: want char=8, got char=%d", tokens[1].Char)
	}
}

func TestTokenize_DuplicateTokensOnOneLine(t *testing.T) {
	// "foo foo" — two identical tokens; each must get the correct column.
	tokens := Tokenize("foo foo")
	if len(tokens) < 3 {
		t.Fatalf("expected >=3 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Char != 0 {
		t.Errorf("first foo: want char=0, got char=%d", tokens[0].Char)
	}
	if tokens[1].Char != 4 {
		t.Errorf("second foo: want char=4, got char=%d", tokens[1].Char)
	}
}

// ---- buildLineStarts --------------------------------------------------------

func TestBuildLineStarts_NoNewlines(t *testing.T) {
	starts := buildLineStarts("hello")
	if len(starts) != 1 || starts[0] != 0 {
		t.Errorf("no newlines: want [0], got %v", starts)
	}
}

func TestBuildLineStarts_SingleNewline(t *testing.T) {
	starts := buildLineStarts("foo\nbar")
	if len(starts) != 2 || starts[0] != 0 || starts[1] != 4 {
		t.Errorf("single newline: want [0 4], got %v", starts)
	}
}

func TestBuildLineStarts_TrailingNewline(t *testing.T) {
	starts := buildLineStarts("foo\n")
	// The trailing newline creates an entry for an (empty) second line.
	if len(starts) != 2 || starts[0] != 0 || starts[1] != 4 {
		t.Errorf("trailing newline: want [0 4], got %v", starts)
	}
}

func TestBuildLineStarts_Empty(t *testing.T) {
	starts := buildLineStarts("")
	if len(starts) != 1 || starts[0] != 0 {
		t.Errorf("empty string: want [0], got %v", starts)
	}
}

func TestBuildLineStarts_MultipleNewlines(t *testing.T) {
	starts := buildLineStarts("a\nb\nc")
	want := []int{0, 2, 4}
	if len(starts) != len(want) {
		t.Fatalf("want %v, got %v", want, starts)
	}
	for i, w := range want {
		if starts[i] != w {
			t.Errorf("starts[%d]: want %d, got %d", i, w, starts[i])
		}
	}
}
