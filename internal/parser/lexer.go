package parser

import (
	"strings"
	"unicode"
)

// TokenType enumerates all token kinds.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	IDENT   // any unquoted word / address
	LBRACE  // {
	RBRACE  // }
	NEWLINE // \n (used to track line boundaries)
	COMMENT // # …
	STRING  // "…" or `…`
)

func (t TokenType) String() string {
	switch t {
	case EOF:
		return "EOF"
	case IDENT:
		return "IDENT"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case NEWLINE:
		return "NEWLINE"
	case COMMENT:
		return "COMMENT"
	case STRING:
		return "STRING"
	default:
		return "ILLEGAL"
	}
}

// Lexer tokenizes a Caddyfile source string.
type Lexer struct {
	src    []rune
	pos    int
	line   uint32
	char   uint32
	tokens []Token
}

// Tokenize returns all tokens for src.
func Tokenize(src string) []Token {
	l := &Lexer{src: []rune(src)}
	l.run()
	return l.tokens
}

func (l *Lexer) run() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]

		switch {
		case ch == '\n':
			l.emit(NEWLINE, "\n")
			l.pos++
			l.line++
			l.char = 0

		case ch == '\r':
			l.pos++
			// skip bare \r

		case ch == '{':
			l.emit(LBRACE, "{")
			l.pos++
			l.char++

		case ch == '}':
			l.emit(RBRACE, "}")
			l.pos++
			l.char++

		case ch == '#':
			l.lexComment()

		case ch == '"' || ch == '`':
			l.lexQuoted(ch)

		case unicode.IsSpace(ch):
			l.pos++
			l.char++

		default:
			l.lexIdent()
		}
	}
	l.emit(EOF, "")
}

func (l *Lexer) emit(t TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:  t,
		Value: value,
		Line:  l.line,
		Char:  l.char,
	})
}

func (l *Lexer) lexComment() {
	start := l.pos
	startChar := l.char
	for l.pos < len(l.src) && l.src[l.pos] != '\n' {
		l.pos++
	}
	value := string(l.src[start:l.pos])
	l.tokens = append(l.tokens, Token{
		Type:  COMMENT,
		Value: value,
		Line:  l.line,
		Char:  startChar,
	})
	l.char += uint32(len([]rune(value)))
}

func (l *Lexer) lexQuoted(quote rune) {
	startLine := l.line
	startChar := l.char
	l.pos++ // consume opening quote
	l.char++
	var sb strings.Builder
	sb.WriteRune(quote)
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		sb.WriteRune(ch)
		l.pos++
		if ch == '\n' {
			l.line++
			l.char = 0
		} else {
			l.char++
		}
		if ch == quote {
			break
		}
	}
	l.tokens = append(l.tokens, Token{
		Type:  STRING,
		Value: sb.String(),
		Line:  startLine,
		Char:  startChar,
	})
}

func (l *Lexer) lexIdent() {
	start := l.pos
	startChar := l.char
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '{' || ch == '}' || ch == '\n' || ch == '\r' || unicode.IsSpace(ch) {
			break
		}
		l.pos++
	}
	value := string(l.src[start:l.pos])
	l.tokens = append(l.tokens, Token{
		Type:  IDENT,
		Value: value,
		Line:  l.line,
		Char:  startChar,
	})
	l.char += uint32(len([]rune(value)))
}
