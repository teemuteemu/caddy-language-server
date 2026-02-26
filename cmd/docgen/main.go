// docgen generates internal/handler/docs_gen.go containing Markdown documentation
// for Caddyfile directives, extracted from Caddy's source code.
//
// It handles two patterns used in Caddy:
//  1. Types with an UnmarshalCaddyfile method — the method doc comment contains
//     the directive's Caddyfile syntax.
//  2. Standalone functions registered via httpcaddyfile.RegisterDirective or
//     RegisterHandlerDirective — the function doc comment contains the syntax.
//
// Only doc comments that contain a fenced code example (tab-indented lines in
// Go doc convention) are kept; plain-text-only docs are skipped.
//
// Run via go generate from the project root:
//
//	go generate ./internal/handler/
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	caddyDir, err := findCaddyDir()
	if err != nil {
		log.Fatalf("find caddy module: %v", err)
	}

	docs, err := extractDirectiveDocs(caddyDir)
	if err != nil {
		log.Fatalf("extract docs: %v", err)
	}

	if err := writeGenFile(docs); err != nil {
		log.Fatalf("write gen file: %v", err)
	}

	fmt.Fprintf(os.Stderr, "generated docs for %d directives\n", len(docs))
}

func findCaddyDir() (string, error) {
	type modInfo struct {
		Dir string
	}
	out, err := exec.Command("go", "list", "-m", "-json", "github.com/caddyserver/caddy/v2").Output()
	if err != nil {
		return "", fmt.Errorf("go list: %w", err)
	}
	var info modInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return "", fmt.Errorf("parse json: %w", err)
	}
	if info.Dir == "" {
		return "", fmt.Errorf("module directory not found in go list output")
	}
	return info.Dir, nil
}

func extractDirectiveDocs(caddyDir string) (map[string]string, error) {
	docs := make(map[string]string)
	fset := token.NewFileSet()

	err := filepath.Walk(caddyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if name := info.Name(); name == "vendor" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // skip unparseable files
		}

		// Collect non-method function doc comments for this file.
		// Used to resolve the handler functions in RegisterDirective calls.
		funcDocs := make(map[string]string) // funcName → docText
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Doc == nil {
				continue
			}
			funcDocs[fn.Name.Name] = fn.Doc.Text()
		}

		// Pattern 1: RegisterDirective("name", handlerFunc) calls.
		// The directive name is the string literal; the doc comes from handlerFunc.
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if name := selectorName(call.Fun); name != "RegisterDirective" && name != "RegisterHandlerDirective" {
				return true
			}
			if len(call.Args) < 2 {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			directiveName := strings.Trim(lit.Value, `"`)
			if !isDirectiveName(directiveName) {
				return true
			}
			ident, ok := call.Args[1].(*ast.Ident)
			if !ok {
				return true
			}
			docText, found := funcDocs[ident.Name]
			if !found {
				return true
			}
			lines := splitLines(docText)
			if !hasCodeBlock(lines) {
				return true // skip docs without a syntax example
			}
			if _, exists := docs[directiveName]; !exists {
				docs[directiveName] = docToMarkdown(lines)
			}
			return true
		})

		// Pattern 2: UnmarshalCaddyfile methods.
		// The directive name is extracted from the first code block line.
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "UnmarshalCaddyfile" || fn.Doc == nil {
				continue
			}
			name, md := parseUnmarshalDoc(fn.Doc.Text())
			if name == "" || md == "" {
				continue
			}
			if _, exists := docs[name]; !exists {
				docs[name] = md
			}
		}

		return nil
	})
	return docs, err
}

// selectorName returns the final identifier name from an expression, handling
// both plain identifiers ("RegisterDirective") and selector expressions
// ("httpcaddyfile.RegisterDirective").
func selectorName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

// parseUnmarshalDoc extracts the directive name and Markdown from an
// UnmarshalCaddyfile doc comment. The directive name is the first word of the
// first tab-indented (code block) line.
func parseUnmarshalDoc(docText string) (name, md string) {
	lines := splitLines(docText)
	for _, line := range lines {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.Fields(trimmed)
		if len(parts) > 0 && isDirectiveName(parts[0]) {
			name = parts[0]
			break
		}
	}
	if name == "" {
		return "", ""
	}
	return name, docToMarkdown(lines)
}

// hasCodeBlock reports whether any line in lines is tab-indented (Go doc
// convention for code examples).
func hasCodeBlock(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "\t") {
			return true
		}
	}
	return false
}

// docToMarkdown converts Go doc comment lines (// markers already stripped) to
// Markdown. Tab-indented lines (code blocks in Go doc convention) are wrapped
// in fenced code blocks.
//
// Lines before the first code block are discarded: they always contain
// internal implementation notes ("UnmarshalCaddyfile sets up…", "parseFoo
// parses the X directive…") that are not useful to LSP users.
func docToMarkdown(lines []string) string {
	// Skip everything before the first tab-indented (code) line.
	firstCode := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") {
			firstCode = i
			break
		}
	}
	if firstCode >= 0 {
		lines = lines[firstCode:]
	}

	var out strings.Builder
	inCode := false

	for _, line := range lines {
		isCode := len(line) > 0 && line[0] == '\t'
		isEmpty := line == ""
		switch {
		case isCode && !inCode:
			out.WriteString("```\n")
			inCode = true
			out.WriteString(strings.TrimPrefix(line, "\t") + "\n")
		case isCode:
			out.WriteString(strings.TrimPrefix(line, "\t") + "\n")
		case isEmpty && inCode:
			// Blank lines within a code block (empty // comment lines in Go source)
			// are kept as blank lines rather than ending the block.
			out.WriteString("\n")
		case inCode:
			out.WriteString("```\n")
			inCode = false
			out.WriteString(line + "\n")
		default:
			out.WriteString(line + "\n")
		}
	}
	if inCode {
		out.WriteString("```\n")
	}

	return strings.TrimSpace(out.String())
}

// isDirectiveName reports whether s looks like a Caddyfile directive name
// (lowercase letters, digits, underscores, and hyphens, starting with a letter).
func isDirectiveName(s string) bool {
	if len(s) == 0 || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func writeGenFile(docs map[string]string) error {
	names := make([]string, 0, len(docs))
	for k := range docs {
		names = append(names, k)
	}
	sort.Strings(names)

	var buf bytes.Buffer
	buf.WriteString("// Code generated by cmd/docgen. DO NOT EDIT.\n\n")
	buf.WriteString("package handler\n\n")
	buf.WriteString("// directiveDocs maps Caddyfile directive names to Markdown documentation\n")
	buf.WriteString("// extracted from Caddy's source code.\n")
	buf.WriteString("var directiveDocs = map[string]string{\n")
	for _, name := range names {
		fmt.Fprintf(&buf, "\t%q: %q,\n", name, docs[name])
	}
	buf.WriteString("}\n")

	return os.WriteFile("docs_gen.go", buf.Bytes(), 0o644)
}
