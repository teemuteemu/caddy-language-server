package parser

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ParseError holds a diagnostic-friendly parse error.
type ParseError struct {
	Message string
	Rng     protocol.Range
}

func (e *ParseError) Error() string { return e.Message }

// Parse tokenizes src and builds an AST. It returns a (possibly partial) File
// along with any parse errors encountered.
func Parse(src string) (*File, []*ParseError) {
	tokens := Tokenize(src)
	p := &parser{tokens: tokens}
	return p.parseFile()
}

type parser struct {
	tokens []Token
	pos    int
	errors []*ParseError
}

// --- token navigation helpers ---

func (p *parser) peek() Token {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		if t.Type == COMMENT || t.Type == NEWLINE {
			p.pos++
			continue
		}
		return t
	}
	return Token{Type: EOF}
}

func (p *parser) next() Token {
	t := p.peek()
	if t.Type != EOF {
		p.pos++
	}
	return t
}

func (p *parser) errorf(rng protocol.Range, format string, args ...any) {
	p.errors = append(p.errors, &ParseError{
		Message: fmt.Sprintf(format, args...),
		Rng:     rng,
	})
}

// --- grammar ---

// parseFile parses the top-level structure of a Caddyfile.
// Grammar (simplified):
//
//	File        = GlobalBlock? SiteBlock*
//	GlobalBlock = "{" Directive* "}"         (when first non-comment token is "{")
//	SiteBlock   = Address+ "{" Directive* "}" | Address+ Directive*
//	Directive   = IDENT Argument* ("{" Directive* "}")?
func (p *parser) parseFile() (*File, []*ParseError) {
	f := &File{}

	// Optional global block: starts with bare "{"
	if p.peek().Type == LBRACE {
		f.GlobalBlock = p.parseGlobalBlock()
	}

	for p.peek().Type != EOF {
		sb := p.parseSiteBlock()
		if sb != nil {
			f.SiteBlocks = append(f.SiteBlocks, sb)
		}
	}

	return f, p.errors
}

func (p *parser) parseGlobalBlock() *GlobalBlock {
	lbrace := p.next() // consume "{"
	g := &GlobalBlock{StartLine: lbrace.Line}
	for {
		tok := p.peek()
		if tok.Type == EOF {
			p.errorf(tok.Range(), "unclosed global options block")
			break
		}
		if tok.Type == RBRACE {
			g.EndLine = tok.Line
			p.next() // consume "}"
			break
		}
		d := p.parseDirective()
		if d != nil {
			g.Directives = append(g.Directives, d)
		}
	}
	return g
}

func (p *parser) parseSiteBlock() *SiteBlock {
	sb := &SiteBlock{}

	// Collect address tokens until "{" or EOF
	for {
		tok := p.peek()
		if tok.Type == EOF || tok.Type == LBRACE {
			break
		}
		if tok.Type == RBRACE {
			// stray "}" — skip with error
			p.errorf(tok.Range(), "unexpected '}'")
			p.next()
			return nil
		}
		if len(sb.Addresses) == 0 {
			sb.StartLine = tok.Line
		}
		sb.Addresses = append(sb.Addresses, p.next())
	}

	if len(sb.Addresses) == 0 {
		return nil
	}

	// Expect "{"
	if p.peek().Type != LBRACE {
		p.errorf(p.peek().Range(), "expected '{' after site address(es)")
		return sb
	}
	p.next() // consume "{"

	for {
		tok := p.peek()
		if tok.Type == EOF {
			p.errorf(tok.Range(), "unclosed site block for %q", sb.Addresses[0].Value)
			break
		}
		if tok.Type == RBRACE {
			sb.EndLine = tok.Line
			p.next() // consume "}"
			break
		}
		d := p.parseDirective()
		if d != nil {
			sb.Directives = append(sb.Directives, d)
		}
	}

	return sb
}

func (p *parser) parseDirective() *Directive {
	tok := p.peek()
	if tok.Type != IDENT && tok.Type != STRING {
		// skip unexpected token
		p.errorf(tok.Range(), "expected directive name, got %s", tok.Type)
		p.next()
		return nil
	}

	name := p.next()
	d := &Directive{Name: name, StartLine: name.Line, EndLine: name.Line}

	// Collect arguments on the same line
	for {
		tok = p.peek()
		if tok.Type == EOF || tok.Type == LBRACE || tok.Type == RBRACE {
			break
		}
		// Arguments must be on the same line as the directive name
		if tok.Line != name.Line {
			break
		}
		arg := p.next()
		d.Args = append(d.Args, &Argument{Token: arg})
	}

	// Optional body block
	if p.peek().Type == LBRACE {
		p.next() // consume "{"
		for {
			tok = p.peek()
			if tok.Type == EOF {
				p.errorf(tok.Range(), "unclosed block for directive %q", name.Value)
				break
			}
			if tok.Type == RBRACE {
				d.EndLine = tok.Line
				p.next() // consume "}"
				break
			}
			sub := p.parseDirective()
			if sub != nil {
				d.Body = append(d.Body, sub)
			}
		}
	}

	return d
}
