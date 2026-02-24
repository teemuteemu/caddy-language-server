package parser

import (
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// TokenType enumerates all token kinds.
type TokenType int

const (
	ILLEGAL TokenType = iota
	EOF
	IDENT   // any unquoted word / address / directive name
	LBRACE  // {
	RBRACE  // }
	NEWLINE // retained for enum compatibility; not produced by Caddy's tokenizer
	COMMENT // retained for enum compatibility; not produced by Caddy's tokenizer
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

// buildLineStarts returns a slice where lineStarts[i] is the byte offset of
// the first character of line i (0-based) within src.
func buildLineStarts(src string) []int {
	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// Tokenize uses Caddy's official Caddyfile tokenizer and enriches each token
// with column information derived by scanning the source text.
//
// Note: COMMENT and NEWLINE tokens are not produced because Caddy's tokenizer
// strips comments and does not emit newlines as separate tokens. The NEWLINE
// and COMMENT enum values are retained for backward compatibility only.
func Tokenize(src string) []Token {
	caddyTokens, err := caddyfile.Tokenize([]byte(src), "Caddyfile")
	if err != nil {
		// Return just an EOF so the parser can report errors gracefully.
		return []Token{{Type: EOF}}
	}
	return addColumns(src, caddyTokens)
}

// addColumns converts a slice of Caddy tokens into our internal Token slice,
// computing a column position for each token by scanning the source text in
// forward order.
//
// Caddy's Token carries only a line number (1-based). We find the column by
// searching for the token's text forward from the end of the last token on the
// same line, starting at least from the line's first byte. This correctly
// handles duplicate tokens on the same line and skips over comment text that
// Caddy has already stripped from the token stream.
func addColumns(src string, caddyTokens []caddyfile.Token) []Token {
	lineStarts := buildLineStarts(src)
	result := make([]Token, 0, len(caddyTokens)+1)

	// lineEnd[line0] is the byte offset just past the end of the last token
	// we matched on line0. Used to avoid re-matching an earlier occurrence of
	// the same text.
	lineEnd := make(map[uint32]int)

	for _, ct := range caddyTokens {
		if ct.Line <= 0 {
			continue
		}
		line0 := uint32(ct.Line - 1) // Caddy is 1-based; we are 0-based

		lineStart := 0
		if int(line0) < len(lineStarts) {
			lineStart = lineStarts[line0]
		}

		// Search starts at the line start or after the previous token on this
		// line, whichever is later.
		searchFrom := lineStart
		if prev, ok := lineEnd[line0]; ok && prev > searchFrom {
			searchFrom = prev
		}

		var (
			tt    TokenType
			value string
			col   uint32
		)

		if ct.Quoted() {
			tt = STRING
			// Locate the opening quote (or heredoc marker) in the source.
			qpos := -1
			for i := searchFrom; i < len(src); i++ {
				ch := src[i]
				if ch == '"' || ch == '`' {
					qpos = i
					break
				}
				// Heredoc: << marker
				if ch == '<' && i+1 < len(src) && src[i+1] == '<' {
					qpos = i
					break
				}
			}

			if qpos >= 0 {
				col = uint32(qpos - lineStart)
				if src[qpos] == '<' {
					// Heredoc: value is just the opening <<MARKER line.
					end := qpos
					for end < len(src) && src[end] != '\n' {
						end++
					}
					value = src[qpos:end]
					lineEnd[line0] = end
				} else {
					// Regular quoted string: read through matching closing quote.
					q := src[qpos]
					end := qpos + 1
					for end < len(src) && src[end] != q {
						if src[end] == '\n' {
							break // unterminated; stop at newline
						}
						end++
					}
					if end < len(src) && src[end] == q {
						end++ // include closing quote
					}
					value = src[qpos:end]
					lineEnd[line0] = end
				}
			} else {
				// Fallback: reconstruct a quoted value from the token text.
				value = `"` + ct.Text + `"`
			}
		} else {
			switch ct.Text {
			case "{":
				tt = LBRACE
			case "}":
				tt = RBRACE
			default:
				tt = IDENT
			}
			value = ct.Text

			idx := strings.Index(src[searchFrom:], ct.Text)
			if idx >= 0 {
				absPos := searchFrom + idx
				col = uint32(absPos - lineStart)
				lineEnd[line0] = absPos + len(ct.Text)
			}
		}

		result = append(result, Token{
			Type:  tt,
			Value: value,
			Line:  line0,
			Char:  col,
		})
	}

	result = append(result, Token{Type: EOF})
	return result
}
