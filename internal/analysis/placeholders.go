package analysis

import (
	"caddy-ls/internal/parser"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// checkPlaceholderBalance returns an error message if the curly braces in s
// are unbalanced, or "" if they are balanced. Escape sequences \{ and \} are
// treated as literal characters and do not affect bracket depth.
func checkPlaceholderBalance(s string) string {
	depth := 0
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		// Skip \{ and \} escape sequences.
		if runes[i] == '\\' && i+1 < len(runes) && (runes[i+1] == '{' || runes[i+1] == '}') {
			i++
			continue
		}
		switch runes[i] {
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return "unmatched '}': no opening '{' for this placeholder"
			}
			depth--
		}
	}
	if depth > 0 {
		return "unclosed placeholder: '{' without matching '}'"
	}
	return ""
}

// placeholderDiag returns an error diagnostic if tok.Value contains unbalanced
// curly braces, otherwise nil. Standalone LBRACE/RBRACE tokens (block delimiters)
// are skipped.
func placeholderDiag(tok parser.Token) *protocol.Diagnostic {
	if tok.Type == parser.LBRACE || tok.Type == parser.RBRACE {
		return nil
	}
	msg := checkPlaceholderBalance(tok.Value)
	if msg == "" {
		return nil
	}
	sev := protocol.DiagnosticSeverityError
	return &protocol.Diagnostic{
		Range:    tok.Range(),
		Severity: &sev,
		Source:   strPtr("caddy-ls"),
		Message:  msg,
	}
}

// analyzeFilePlaceholders walks every token value in the AST and reports
// unbalanced placeholder braces.
func analyzeFilePlaceholders(f *parser.File) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	if f.GlobalBlock != nil {
		for _, d := range f.GlobalBlock.Directives {
			diags = append(diags, analyzeDirectivePlaceholders(d)...)
		}
	}

	for _, sb := range f.SiteBlocks {
		for _, addr := range sb.Addresses {
			if d := placeholderDiag(addr); d != nil {
				diags = append(diags, *d)
			}
		}
		for _, d := range sb.Directives {
			diags = append(diags, analyzeDirectivePlaceholders(d)...)
		}
	}

	return diags
}

func analyzeDirectivePlaceholders(d *parser.Directive) []protocol.Diagnostic {
	var diags []protocol.Diagnostic
	for _, arg := range d.Args {
		if diag := placeholderDiag(arg.Token); diag != nil {
			diags = append(diags, *diag)
		}
	}
	for _, sub := range d.Body {
		diags = append(diags, analyzeDirectivePlaceholders(sub)...)
	}
	return diags
}
