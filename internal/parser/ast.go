package parser

import protocol "github.com/tliron/glsp/protocol_3_16"

// Node is the interface implemented by every AST node.
type Node interface {
	Range() protocol.Range
}

// Token is the smallest unit produced by the lexer.
type Token struct {
	Type    TokenType
	Value   string
	Line    uint32 // 0-based
	Char    uint32 // 0-based character offset on the line
}

func (t Token) Range() protocol.Range {
	end := t.Char + uint32(len(t.Value))
	return protocol.Range{
		Start: protocol.Position{Line: t.Line, Character: t.Char},
		End:   protocol.Position{Line: t.Line, Character: end},
	}
}

// Argument is a single token value used as an argument to a directive.
type Argument struct {
	Token Token
}

func (a *Argument) Range() protocol.Range { return a.Token.Range() }

// Directive is a named directive with optional arguments and a body block.
type Directive struct {
	Name      Token
	Args      []*Argument
	Body      []*Directive // sub-directives inside { }
	StartLine uint32
	EndLine   uint32
}

func (d *Directive) Range() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: d.StartLine, Character: 0},
		End:   protocol.Position{Line: d.EndLine, Character: 0},
	}
}

// SiteBlock represents a site address block, e.g. `example.com { ... }`.
type SiteBlock struct {
	Addresses  []Token
	Directives []*Directive
	StartLine  uint32
	EndLine    uint32
}

func (s *SiteBlock) Range() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: s.StartLine, Character: 0},
		End:   protocol.Position{Line: s.EndLine, Character: 0},
	}
}

// GlobalBlock represents the global options block `{ ... }` at the top of a Caddyfile.
type GlobalBlock struct {
	Directives []*Directive
	StartLine  uint32
	EndLine    uint32
}

func (g *GlobalBlock) Range() protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: g.StartLine, Character: 0},
		End:   protocol.Position{Line: g.EndLine, Character: 0},
	}
}

// File is the root AST node for a Caddyfile.
type File struct {
	GlobalBlock *GlobalBlock // optional; nil if absent
	SiteBlocks  []*SiteBlock
}

func (f *File) Range() protocol.Range {
	if len(f.SiteBlocks) == 0 {
		return protocol.Range{}
	}
	last := f.SiteBlocks[len(f.SiteBlocks)-1]
	return protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: last.EndLine, Character: 0},
	}
}
