package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// Parser parses JSX tokens into an AST
type Parser struct {
	tokens      []Token
	pos         int
	warnings    []Warning
	suggestions []Suggestion
}

// NewParser creates a new parser for the given tokens
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses a complete JSX file
func (p *Parser) Parse() *ParseResult {
	file := &File{
		Imports:    []Import{},
		Components: []Component{},
		Exports:    []string{},
	}

	for !p.isAtEnd() {
		p.skipWhitespace()
		if p.isAtEnd() {
			break
		}

		// Try to parse imports
		if p.checkIdent("import") {
			imp := p.parseImport()
			if imp != nil {
				file.Imports = append(file.Imports, *imp)
			}
			continue
		}

		// Try to parse component definitions
		if p.checkIdent("function") || p.checkIdent("const") || p.checkIdent("export") {
			comp := p.parseComponent()
			if comp != nil {
				file.Components = append(file.Components, *comp)
			}
			continue
		}

		// Skip unknown tokens
		p.advance()
	}

	return &ParseResult{
		File:        file,
		Warnings:    p.warnings,
		Suggestions: p.suggestions,
	}
}

// ParseJSX parses just a JSX element (for testing or partial conversion)
func (p *Parser) ParseJSX() Node {
	p.skipWhitespace()
	return p.parseNode()
}

func (p *Parser) parseNode() Node {
	p.skipWhitespace()

	if p.isAtEnd() {
		return nil
	}

	// JSX expression
	if p.check(TokenJSXExprOpen) {
		return p.parseExpression()
	}

	// JSX element
	if p.check(TokenTagOpen) {
		return p.parseElement()
	}

	// Text content
	return p.parseText()
}

func (p *Parser) parseElement() Node {
	if !p.match(TokenTagOpen) {
		return nil
	}

	p.skipWhitespace()

	// Check for fragment <>
	if p.check(TokenTagClose) {
		p.advance()
		return p.parseFragment()
	}

	// Get tag name
	if !p.check(TokenIdent) {
		p.addWarning("Expected tag name after <")
		return nil
	}

	tagToken := p.advance()
	tagName := tagToken.Value
	line := tagToken.Line

	elem := &Element{
		Tag:        tagName,
		Attributes: []Attribute{},
		Children:   []Node{},
		LineNumber: line,
	}

	// Parse attributes
	for !p.isAtEnd() && !p.check(TokenTagClose) && !p.check(TokenTagSelfClose) {
		p.skipWhitespace()
		if p.check(TokenTagClose) || p.check(TokenTagSelfClose) {
			break
		}

		attr := p.parseAttribute()
		if attr != nil {
			elem.Attributes = append(elem.Attributes, *attr)
		}
	}

	// Self-closing tag
	if p.match(TokenTagSelfClose) {
		elem.SelfClose = true
		return elem
	}

	// Opening tag close
	if !p.match(TokenTagClose) {
		p.addWarning("Expected > to close tag")
		return elem
	}

	// Parse children
	for !p.isAtEnd() {
		p.skipNonSignificantWhitespace()

		// Check for closing tag
		if p.check(TokenTagEnd) {
			break
		}

		child := p.parseNode()
		if child != nil {
			elem.Children = append(elem.Children, child)
		} else {
			break
		}
	}

	// Parse closing tag
	if p.match(TokenTagEnd) {
		p.skipWhitespace()
		if p.check(TokenIdent) {
			closingTag := p.advance()
			if closingTag.Value != tagName {
				p.addWarning(fmt.Sprintf("Mismatched closing tag: expected </%s>, got </%s>", tagName, closingTag.Value))
			}
		}
		p.skipWhitespace()
		p.match(TokenTagClose)
	}

	return elem
}

func (p *Parser) parseFragment() Node {
	frag := &Fragment{
		Children:   []Node{},
		LineNumber: p.current().Line,
	}

	for !p.isAtEnd() {
		p.skipNonSignificantWhitespace()

		// Check for closing </> 
		if p.check(TokenTagEnd) {
			p.advance()
			p.skipWhitespace()
			p.match(TokenTagClose)
			break
		}

		child := p.parseNode()
		if child != nil {
			frag.Children = append(frag.Children, child)
		} else {
			break
		}
	}

	return frag
}

func (p *Parser) parseAttribute() *Attribute {
	p.skipWhitespace()

	// Spread attribute {...props}
	if p.check(TokenJSXExprOpen) {
		p.advance()
		p.skipWhitespace()
		if p.match(TokenSpread) {
			// Get the identifier being spread
			var spreadExpr strings.Builder
			depth := 1
			for !p.isAtEnd() && depth > 0 {
				tok := p.advance()
				if tok.Type == TokenJSXExprOpen {
					depth++
				} else if tok.Type == TokenJSXExprClose {
					depth--
					if depth == 0 {
						break
					}
				}
				spreadExpr.WriteString(tok.Value)
			}
			return &Attribute{
				IsSpread:   true,
				SpreadExpr: strings.TrimSpace(spreadExpr.String()),
			}
		}
		// Not a spread, back up
		p.pos--
		return nil
	}

	// Regular attribute
	if !p.check(TokenIdent) {
		return nil
	}

	nameToken := p.advance()
	attr := &Attribute{
		Name: nameToken.Value,
	}

	p.skipWhitespace()

	// Boolean attribute (no value)
	if !p.check(TokenEquals) {
		return attr
	}

	p.advance() // consume =
	p.skipWhitespace()

	// String value
	if p.check(TokenString) {
		attr.Value = p.advance().Value
		return attr
	}

	// Expression value
	if p.check(TokenJSXExprOpen) {
		p.advance()
		expr := p.parseExpressionContent()
		attr.Expression = expr
		return attr
	}

	return attr
}

func (p *Parser) parseExpression() Node {
	if !p.match(TokenJSXExprOpen) {
		return nil
	}

	expr := p.parseExpressionContent()

	// Check for patterns we can translate
	node := p.analyzeExpression(expr)
	if node != nil {
		return node
	}

	return &expr
}

func (p *Parser) parseExpressionContent() Expression {
	var content strings.Builder
	depth := 1
	startLine := p.current().Line

	for !p.isAtEnd() && depth > 0 {
		tok := p.current()
		if tok.Type == TokenJSXExprOpen {
			depth++
		} else if tok.Type == TokenJSXExprClose {
			depth--
			if depth == 0 {
				p.advance()
				break
			}
		}
		content.WriteString(tok.Value)
		p.advance()
	}

	return Expression{
		Raw:        strings.TrimSpace(content.String()),
		LineNumber: startLine,
	}
}

func (p *Parser) parseText() Node {
	var content strings.Builder
	startLine := p.current().Line

	for !p.isAtEnd() {
		tok := p.current()
		if tok.Type == TokenTagOpen || tok.Type == TokenTagEnd || tok.Type == TokenJSXExprOpen {
			break
		}
		content.WriteString(tok.Value)
		p.advance()
	}

	text := strings.TrimSpace(content.String())
	if text == "" {
		return nil
	}

	return &Text{
		Content:    text,
		LineNumber: startLine,
	}
}

func (p *Parser) parseImport() *Import {
	if !p.matchIdent("import") {
		return nil
	}

	imp := &Import{
		Named:      make(map[string]string),
		LineNumber: p.current().Line,
	}

	p.skipWhitespace()

	// Default import
	if p.check(TokenIdent) && !p.checkIdent("from") {
		tok := p.advance()
		if tok.Value != "{" && tok.Value != "*" {
			imp.Default = tok.Value
			p.skipWhitespace()
			if p.check(TokenComma) {
				p.advance()
				p.skipWhitespace()
			}
		}
	}

	// Named imports { a, b, c }
	if p.check(TokenJSXExprOpen) {
		p.advance()
		for !p.isAtEnd() && !p.check(TokenJSXExprClose) {
			p.skipWhitespace()
			if p.check(TokenIdent) {
				name := p.advance().Value
				alias := name
				p.skipWhitespace()
				if p.checkIdent("as") {
					p.advance()
					p.skipWhitespace()
					if p.check(TokenIdent) {
						alias = p.advance().Value
					}
				}
				imp.Named[name] = alias
			}
			p.skipWhitespace()
			p.match(TokenComma)
		}
		p.match(TokenJSXExprClose)
	}

	// from 'module'
	p.skipWhitespace()
	if p.matchIdent("from") {
		p.skipWhitespace()
		if p.check(TokenString) {
			imp.Source = p.advance().Value
		}
	}

	// Skip to end of statement
	for !p.isAtEnd() {
		tok := p.current()
		if tok.Type == TokenIdent && (tok.Value == "import" || tok.Value == "export" || tok.Value == "function" || tok.Value == "const") {
			break
		}
		p.advance()
	}

	return imp
}

func (p *Parser) parseComponent() *Component {
	startLine := p.current().Line

	// Handle export
	isExport := p.matchIdent("export")
	if isExport {
		p.skipWhitespace()
		p.matchIdent("default")
		p.skipWhitespace()
	}

	// function ComponentName or const ComponentName
	isArrow := false
	if p.matchIdent("const") {
		isArrow = true
	} else if !p.matchIdent("function") {
		return nil
	}

	p.skipWhitespace()

	// Component name
	if !p.check(TokenIdent) {
		return nil
	}
	name := p.advance().Value

	// Skip if it doesn't look like a component (starts with lowercase and not a hook)
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' && !strings.HasPrefix(name, "use") {
		p.skipToNextStatement()
		return nil
	}

	comp := &Component{
		Name:       name,
		Props:      []Prop{},
		Hooks:      []Hook{},
		LineNumber: startLine,
	}

	p.skipWhitespace()

	// Arrow function: = (props) => or = () =>
	if isArrow {
		p.match(TokenEquals)
		p.skipWhitespace()
	}

	// Props
	if p.match(TokenLParen) {
		comp.Props = p.parseProps()
		p.match(TokenRParen)
	}

	p.skipWhitespace()

	// Arrow
	if isArrow {
		p.match(TokenArrow)
		p.skipWhitespace()
	}

	// Body - find the JSX return
	comp.Body = p.parseComponentBody(comp)

	return comp
}

func (p *Parser) parseProps() []Prop {
	var props []Prop
	p.skipWhitespace()

	// Destructured props: { prop1, prop2 }
	if p.match(TokenJSXExprOpen) {
		for !p.isAtEnd() && !p.check(TokenJSXExprClose) {
			p.skipWhitespace()
			if p.check(TokenIdent) {
				prop := Prop{Name: p.advance().Value}
				p.skipWhitespace()
				// Default value: prop = 'default'
				if p.match(TokenEquals) {
					p.skipWhitespace()
					if p.check(TokenString) {
						prop.DefaultValue = p.advance().Value
					} else {
						// Skip complex default value
						depth := 0
						var val strings.Builder
						for !p.isAtEnd() {
							tok := p.current()
							if tok.Type == TokenJSXExprOpen || tok.Type == TokenLParen {
								depth++
							} else if tok.Type == TokenJSXExprClose || tok.Type == TokenRParen {
								if depth == 0 {
									break
								}
								depth--
							} else if tok.Type == TokenComma && depth == 0 {
								break
							}
							val.WriteString(tok.Value)
							p.advance()
						}
						prop.DefaultValue = strings.TrimSpace(val.String())
					}
				}
				props = append(props, prop)
			}
			p.skipWhitespace()
			p.match(TokenComma)
		}
		p.match(TokenJSXExprClose)
	} else if p.check(TokenIdent) {
		// Single props object: props
		props = append(props, Prop{Name: p.advance().Value})
	}

	return props
}

func (p *Parser) parseComponentBody(comp *Component) Node {
	// Look for hooks and return statement
	depth := 0
	foundReturn := false

	for !p.isAtEnd() {
		tok := p.current()

		if tok.Type == TokenJSXExprOpen || (tok.Type == TokenIdent && tok.Value == "{") {
			depth++
		} else if tok.Type == TokenJSXExprClose || (tok.Type == TokenIdent && tok.Value == "}") {
			depth--
			if depth < 0 {
				break
			}
		}

		// Detect hooks
		if tok.Type == TokenIdent {
			if hook := p.detectHook(tok.Value); hook != nil {
				comp.Hooks = append(comp.Hooks, *hook)
			}
		}

		// Find return with JSX
		if tok.Type == TokenIdent && tok.Value == "return" {
			foundReturn = true
			p.advance()
			p.skipWhitespace()

			// Handle return (...) or return <...
			if p.match(TokenLParen) {
				p.skipWhitespace()
			}

			if p.check(TokenTagOpen) {
				return p.parseNode()
			}
		}

		p.advance()
	}

	if !foundReturn {
		// Arrow function with implicit return
		// Reset and try to find JSX directly
		// (simplified - in practice would need better handling)
	}

	return nil
}

func (p *Parser) detectHook(name string) *Hook {
	if !strings.HasPrefix(name, "use") {
		return nil
	}

	hook := &Hook{
		Type:       name,
		LineNumber: p.current().Line,
	}

	// Add suggestion based on hook type
	switch name {
	case "useState":
		p.addSuggestion(hook.LineNumber, name, "Consider: server state, mintydyn State, or HTMX pattern", "useState")
	case "useEffect":
		p.addSuggestion(hook.LineNumber, name, "Consider: server-side logic, OnInit hook, or HTMX trigger", "useEffect")
	case "useMemo", "useCallback":
		p.addSuggestion(hook.LineNumber, name, "Consider: Go function or method - no memoization needed server-side", "memoization")
	case "useContext":
		p.addSuggestion(hook.LineNumber, name, "Consider: function parameters or Go context.Context", "useContext")
	case "useRef":
		p.addSuggestion(hook.LineNumber, name, "Consider: mi.ID() for DOM references in mintydyn hooks", "useRef")
	case "useReducer":
		p.addSuggestion(hook.LineNumber, name, "Consider: mintydyn Rules for state machines", "useReducer")
	}

	return hook
}

func (p *Parser) analyzeExpression(expr Expression) Node {
	raw := expr.Raw

	// Detect .map() pattern
	mapRegex := regexp.MustCompile(`^(\w+(?:\.\w+)*)\.map\s*\(\s*\(?\s*(\w+)(?:\s*,\s*(\w+))?\s*\)?\s*=>\s*`)
	if matches := mapRegex.FindStringSubmatch(raw); matches != nil {
		collection := matches[1]
		itemVar := matches[2]
		indexVar := ""
		if len(matches) > 3 {
			indexVar = matches[3]
		}

		// Find the JSX body after the arrow
		bodyStart := mapRegex.FindStringIndex(raw)[1]
		bodyRaw := raw[bodyStart:]

		// Parse the body as JSX
		bodyLexer := NewLexer(bodyRaw)
		bodyTokens := bodyLexer.Tokenize()
		bodyParser := NewParser(bodyTokens)
		body := bodyParser.ParseJSX()

		return &MapExpr{
			Collection: collection,
			ItemVar:    itemVar,
			IndexVar:   indexVar,
			Body:       body,
			LineNumber: expr.LineNumber,
		}
	}

	// Detect && conditional pattern
	andRegex := regexp.MustCompile(`^(.+?)\s*&&\s*`)
	if matches := andRegex.FindStringSubmatch(raw); matches != nil {
		condition := strings.TrimSpace(matches[1])
		bodyStart := andRegex.FindStringIndex(raw)[1]
		bodyRaw := raw[bodyStart:]

		bodyLexer := NewLexer(bodyRaw)
		bodyTokens := bodyLexer.Tokenize()
		bodyParser := NewParser(bodyTokens)
		body := bodyParser.ParseJSX()

		return &Conditional{
			Condition:  condition,
			Consequent: body,
			LineNumber: expr.LineNumber,
		}
	}

	// Detect ternary pattern
	// This is tricky because ? and : can appear in nested expressions
	// Simplified detection for common cases
	ternaryRegex := regexp.MustCompile(`^([^?]+)\s*\?\s*`)
	if matches := ternaryRegex.FindStringSubmatch(raw); matches != nil {
		condition := strings.TrimSpace(matches[1])
		rest := raw[ternaryRegex.FindStringIndex(raw)[1]:]

		// Find the : separator (accounting for nesting)
		colonIdx := findTernaryColon(rest)
		if colonIdx > 0 {
			consequentRaw := rest[:colonIdx]
			alternateRaw := rest[colonIdx+1:]

			consequentLexer := NewLexer(consequentRaw)
			consequentParser := NewParser(consequentLexer.Tokenize())
			consequent := consequentParser.ParseJSX()

			alternateLexer := NewLexer(alternateRaw)
			alternateParser := NewParser(alternateLexer.Tokenize())
			alternate := alternateParser.ParseJSX()

			return &Ternary{
				Condition:  condition,
				Consequent: consequent,
				Alternate:  alternate,
				LineNumber: expr.LineNumber,
			}
		}
	}

	return nil
}

func findTernaryColon(s string) int {
	depth := 0
	for i, ch := range s {
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ':':
			if depth == 0 {
				return i
			}
		case '?':
			depth++ // nested ternary
		}
	}
	return -1
}

// Helper methods

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	tok := p.current()
	if !p.isAtEnd() {
		p.pos++
	}
	return tok
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.tokens[p.pos].Type == TokenEOF
}

func (p *Parser) check(typ TokenType) bool {
	return p.current().Type == typ
}

func (p *Parser) checkIdent(value string) bool {
	tok := p.current()
	return tok.Type == TokenIdent && tok.Value == value
}

func (p *Parser) match(typ TokenType) bool {
	if p.check(typ) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchIdent(value string) bool {
	if p.checkIdent(value) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) skipWhitespace() {
	for p.check(TokenWhitespace) {
		p.advance()
	}
}

func (p *Parser) skipNonSignificantWhitespace() {
	for p.check(TokenWhitespace) {
		ws := p.current().Value
		// Keep whitespace with newlines as potentially significant
		if !strings.Contains(ws, "\n") || strings.TrimSpace(ws) == "" {
			p.advance()
		} else {
			break
		}
	}
	p.skipWhitespace()
}

func (p *Parser) skipToNextStatement() {
	depth := 0
	for !p.isAtEnd() {
		tok := p.current()
		if tok.Type == TokenJSXExprOpen {
			depth++
		} else if tok.Type == TokenJSXExprClose {
			depth--
			if depth < 0 {
				return
			}
		}
		if depth == 0 && tok.Type == TokenIdent {
			switch tok.Value {
			case "import", "export", "function", "const", "let", "var", "class":
				return
			}
		}
		p.advance()
	}
}

func (p *Parser) addWarning(msg string) {
	p.warnings = append(p.warnings, Warning{
		Line:    p.current().Line,
		Column:  p.current().Column,
		Message: msg,
	})
}

func (p *Parser) addSuggestion(line int, reactCode, mintyHint, patternType string) {
	p.suggestions = append(p.suggestions, Suggestion{
		Line:        line,
		ReactCode:   reactCode,
		MintyHint:   mintyHint,
		PatternType: patternType,
	})
}
