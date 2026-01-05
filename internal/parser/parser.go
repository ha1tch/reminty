package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Parser parses JSX tokens into an AST
type Parser struct {
	tokens      []Token
	source      string // original source for regex-based extraction
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

// NewParserWithSource creates a parser with access to original source
func NewParserWithSource(tokens []Token, source string) *Parser {
	return &Parser{
		tokens: tokens,
		source: source,
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

	// Pre-extract all useState variables from source
	var allStateVars []StateVariable
	if p.source != "" {
		allStateVars = extractUseStateVars(p.source)
	}
	
	// Pre-extract all derived variables from source
	var allDerivedVars []DerivedVariable
	if p.source != "" {
		allDerivedVars = extractDerivedVars(p.source, allStateVars)
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

	// Associate state vars and derived vars with components based on line numbers
	for i := range file.Components {
		comp := &file.Components[i]
		compStart := comp.LineNumber
		compEnd := p.findComponentEnd(comp, file.Components, i)
		
		for _, sv := range allStateVars {
			if sv.LineNumber >= compStart && sv.LineNumber < compEnd {
				comp.StateVars = append(comp.StateVars, sv)
			}
		}
		
		for _, dv := range allDerivedVars {
			if dv.LineNumber >= compStart && dv.LineNumber < compEnd {
				comp.DerivedVars = append(comp.DerivedVars, dv)
			}
		}
	}

	return &ParseResult{
		File:        file,
		Warnings:    p.warnings,
		Suggestions: p.suggestions,
	}
}

// findComponentEnd returns the line where the next component starts, or a large number
func (p *Parser) findComponentEnd(comp *Component, comps []Component, idx int) int {
	if idx+1 < len(comps) {
		return comps[idx+1].LineNumber
	}
	// No next component, use a large number
	return 999999
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
		val := p.advance().Value
		// Strip surrounding quotes (lexer now includes them)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		attr.Value = val
		return attr
	}

	// Expression value
	if p.check(TokenJSXExprOpen) {
		p.advance()
		expr := p.parseExpressionContent()
		attr.Expression = expr
		
		// Check if this is an event handler
		if isEventHandler(attr.Name) {
			attr.EventHandler = parseEventHandler(attr.Name, expr.Raw, expr.LineNumber)
		}
		
		return attr
	}

	return attr
}

// isEventHandler checks if an attribute name is an event handler
func isEventHandler(name string) bool {
	return strings.HasPrefix(name, "on") && len(name) > 2 && 
		name[2] >= 'A' && name[2] <= 'Z'
}

// parseEventHandler parses an event handler expression
func parseEventHandler(eventType, body string, line int) *EventHandler {
	handler := &EventHandler{
		EventType:   eventType,
		HandlerBody: body,
		LineNumber:  line,
	}
	
	// Check for inline arrow function
	if strings.Contains(body, "=>") {
		handler.IsInline = true
	}
	
	// Extract setState calls: setX, setY, etc.
	setterPattern := regexp.MustCompile(`(set[A-Z]\w*)\s*\(`)
	setterMatches := setterPattern.FindAllStringSubmatch(body, -1)
	for _, match := range setterMatches {
		if len(match) > 1 {
			handler.SetterCalls = append(handler.SetterCalls, match[1])
		}
	}
	
	// Extract state variables referenced (simple identifiers that might be state)
	// Look for identifiers that aren't setters and aren't common keywords
	identPattern := regexp.MustCompile(`\b([a-z][a-zA-Z0-9]*)\b`)
	identMatches := identPattern.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	keywords := map[string]bool{
		"true": true, "false": true, "null": true, "undefined": true,
		"return": true, "if": true, "else": true, "const": true, "let": true,
		"var": true, "function": true, "new": true, "this": true,
		"event": true, "e": true, "target": true, "value": true,
	}
	for _, match := range identMatches {
		if len(match) > 1 {
			ident := match[1]
			if !seen[ident] && !keywords[ident] && !strings.HasPrefix(ident, "set") {
				seen[ident] = true
				handler.StateVars = append(handler.StateVars, ident)
			}
		}
	}
	
	return handler
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

// extractUseStateVars scans source for useState patterns and extracts StateVariables
func extractUseStateVars(source string) []StateVariable {
	var stateVars []StateVariable
	
	// Pattern: const [varName, setVarName] = useState(initValue)
	// Also handles: const [varName, setVarName] = useState<Type>(initValue)
	pattern := regexp.MustCompile(`const\s+\[(\w+),\s*(\w+)\]\s*=\s*useState(?:<[^>]+>)?\s*\(([^)]*)\)`)
	
	matches := pattern.FindAllStringSubmatchIndex(source, -1)
	for _, match := range matches {
		if len(match) >= 8 {
			varName := source[match[2]:match[3]]
			setterName := source[match[4]:match[5]]
			initValue := strings.TrimSpace(source[match[6]:match[7]])
			
			// Infer type from initial value
			initType := inferTypeFromValue(initValue)
			
			// Calculate line number
			lineNum := 1 + strings.Count(source[:match[0]], "\n")
			
			stateVars = append(stateVars, StateVariable{
				Name:       varName,
				Setter:     setterName,
				InitValue:  initValue,
				InitType:   initType,
				LineNumber: lineNum,
			})
		}
	}
	
	return stateVars
}

// inferTypeFromValue guesses Go type from JS initial value
func inferTypeFromValue(val string) string {
	val = strings.TrimSpace(val)
	
	// Empty or quotes = string
	if val == "" || val == `""` || val == "''" || val == "``" {
		return "string"
	}
	
	// Quoted string
	if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
		(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) ||
		(strings.HasPrefix(val, "`") && strings.HasSuffix(val, "`")) {
		return "string"
	}
	
	// Boolean
	if val == "true" || val == "false" {
		return "bool"
	}
	
	// Number
	if _, err := strconv.Atoi(val); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return "float64"
	}
	
	// Array
	if strings.HasPrefix(val, "[") {
		return "[]interface{}"
	}
	
	// Object
	if strings.HasPrefix(val, "{") {
		return "map[string]interface{}"
	}
	
	// null/undefined
	if val == "null" || val == "undefined" {
		return "interface{}"
	}
	
	// Variable reference with plural name (likely array prop)
	lowerVal := strings.ToLower(val)
	if strings.HasSuffix(lowerVal, "s") && !strings.HasSuffix(lowerVal, "ss") && 
		len(lowerVal) > 3 && isSimpleIdent(val) {
		return "[]interface{}"
	}
	if strings.Contains(lowerVal, "items") || strings.Contains(lowerVal, "list") || 
		strings.Contains(lowerVal, "data") || strings.Contains(lowerVal, "array") {
		return "[]interface{}"
	}
	
	// Default
	return "interface{}"
}

// isSimpleIdent checks if s is a simple identifier (for parser)
func isSimpleIdent(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '$') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '$') {
				return false
			}
		}
	}
	return true
}

// extractDerivedVars scans source for derived state patterns
// e.g., const filteredUsers = users.filter(user => ...)
func extractDerivedVars(source string, stateVars []StateVariable) []DerivedVariable {
	var derivedVars []DerivedVariable
	
	// Build set of known state var names for dependency tracking
	stateNames := make(map[string]bool)
	for _, sv := range stateVars {
		stateNames[sv.Name] = true
	}
	
	// Patterns for array operations:
	// const x = y.filter(...) | .map(...) | .find(...) | .some(...) | .every(...) | .reduce(...) | .sort(...)
	patterns := []struct {
		regex    *regexp.Regexp
		opType   string
		resultType string
	}{
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.filter\s*\(`),
			"filter",
			"[]interface{}",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.map\s*\(`),
			"map",
			"[]interface{}",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.find\s*\(`),
			"find",
			"interface{}",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.some\s*\(`),
			"some",
			"bool",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.every\s*\(`),
			"every",
			"bool",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.reduce\s*\(`),
			"reduce",
			"interface{}",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.sort\s*\(`),
			"sort",
			"[]interface{}",
		},
		{
			regexp.MustCompile(`const\s+(\w+)\s*=\s*(\w+)\.slice\s*\(`),
			"slice",
			"[]interface{}",
		},
	}
	
	for _, p := range patterns {
		matches := p.regex.FindAllStringSubmatchIndex(source, -1)
		for _, match := range matches {
			if len(match) >= 6 {
				varName := source[match[2]:match[3]]
				sourceName := source[match[4]:match[5]]
				
				// Skip if this is a useState destructuring (already handled)
				if strings.Contains(source[max(0, match[0]-20):match[0]], "[") {
					continue
				}
				
				// Find the full expression (up to the matching closing paren)
				exprStart := match[0]
				exprEnd := findMatchingParen(source, match[5])
				fullExpr := ""
				if exprEnd > match[5] {
					fullExpr = source[exprStart:exprEnd]
				}
				
				// Calculate line number
				lineNum := 1 + strings.Count(source[:match[0]], "\n")
				
				// Find dependencies - which state vars are referenced in the expression
				var deps []string
				for stateName := range stateNames {
					// Check if state var is used in the expression
					if strings.Contains(fullExpr, stateName) {
						deps = append(deps, stateName)
					}
				}
				// Also add source collection if it's a state var
				if stateNames[sourceName] {
					deps = append(deps, sourceName)
				}
				
				derivedVars = append(derivedVars, DerivedVariable{
					Name:       varName,
					Expression: fullExpr,
					SourceVar:  sourceName,
					Operation:  p.opType,
					ResultType: p.resultType,
					DependsOn:  deps,
					LineNumber: lineNum,
				})
			}
		}
	}
	
	return derivedVars
}

// findMatchingParen finds the position after the matching closing paren
func findMatchingParen(s string, start int) int {
	depth := 1
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (p *Parser) analyzeExpression(expr Expression) Node {
	raw := expr.Raw

	// Detect .map() pattern
	mapRegex := regexp.MustCompile(`^(\w+(?:\.\w+)*)\.map\s*\(\s*\(?\s*(\w+)(?:\s*,\s*(\w+))?\s*\)?\s*=>\s*`)
	if matches := mapRegex.FindStringSubmatch(raw); matches != nil {
		collection := matches[1]
		itemVar := matches[2]
		indexVar := ""
		if len(matches) > 3 && matches[3] != "" {
			indexVar = matches[3]
		}

		// Find the JSX body after the arrow
		bodyStart := mapRegex.FindStringIndex(raw)[1]
		bodyRaw := raw[bodyStart:]

		// Strip leading whitespace
		bodyRaw = strings.TrimLeft(bodyRaw, " \t\n\r")

		// If body starts with '(', find matching ')' and extract content
		if strings.HasPrefix(bodyRaw, "(") {
			bodyRaw = bodyRaw[1:] // skip opening paren
			// Find matching closing paren
			depth := 1
			for i, ch := range bodyRaw {
				if ch == '(' {
					depth++
				} else if ch == ')' {
					depth--
					if depth == 0 {
						bodyRaw = bodyRaw[:i]
						break
					}
				}
			}
		}

		// Strip trailing closing parens from map call
		bodyRaw = strings.TrimRight(bodyRaw, " \t\n\r)")

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
		bodyRaw := strings.TrimSpace(raw[bodyStart:])
		
		// Strip outer parentheses if present
		bodyRaw = stripOuterParens(bodyRaw)

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
			consequentRaw := strings.TrimSpace(rest[:colonIdx])
			alternateRaw := strings.TrimSpace(rest[colonIdx+1:])

			// Strip outer parentheses if present
			consequentRaw = stripOuterParens(consequentRaw)
			alternateRaw = stripOuterParens(alternateRaw)

			// Parse consequent - check if it's a .map() expression first
			var consequent Node
			if isMapExpression(consequentRaw) {
				consequent = p.analyzeExpression(Expression{Raw: consequentRaw, LineNumber: expr.LineNumber})
			} else {
				consequentLexer := NewLexer(consequentRaw)
				consequentParser := NewParser(consequentLexer.Tokenize())
				consequent = consequentParser.ParseJSX()
			}

			// Parse alternate - check if it's a .map() expression first
			var alternate Node
			if isMapExpression(alternateRaw) {
				alternate = p.analyzeExpression(Expression{Raw: alternateRaw, LineNumber: expr.LineNumber})
			} else {
				alternateLexer := NewLexer(alternateRaw)
				alternateParser := NewParser(alternateLexer.Tokenize())
				alternate = alternateParser.ParseJSX()
			}

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

// isMapExpression checks if the string looks like a .map() expression
func isMapExpression(s string) bool {
	return regexp.MustCompile(`^\w+(?:\.\w+)*\.map\s*\(`).MatchString(s)
}

// stripOuterParens removes outer parentheses from a string if balanced
func stripOuterParens(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return s
	}
	
	// Check if the outer parens are balanced
	depth := 0
	for i, ch := range s {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				// If we hit depth 0 before the end, parens aren't outer
				if i < len(s)-1 {
					return s
				}
				// Strip the outer parens
				return strings.TrimSpace(s[1 : len(s)-1])
			}
		}
	}
	return s
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
