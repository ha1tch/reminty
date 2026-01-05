package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ha1tch/reminty/internal/parser"
)

// Generator produces Go code from JSX AST
type Generator struct {
	indent       int
	output       strings.Builder
	suggestions  []string
	warnings     []string
	usesFragment bool
	usesEach     bool
	usesIf       bool
	usesIfElse   bool
}

// NewGenerator creates a new code generator
func NewGenerator() *Generator {
	return &Generator{
		indent: 0,
	}
}

// Generate produces Go code from a parse result
func (g *Generator) Generate(result *parser.ParseResult) string {
	g.output.Reset()

	// Write package declaration
	g.writeln("package main")
	g.writeln("")

	// Write imports (will be adjusted based on usage)
	g.writeln("import (")
	g.writeln("\tmi \"github.com/ha1tch/minty\"")
	g.writeln(")")
	g.writeln("")

	// Generate components
	for _, comp := range result.File.Components {
		g.generateComponent(&comp)
		g.writeln("")
	}

	// Add suggestions as comments at the end
	if len(result.Suggestions) > 0 {
		g.writeln("// =============================================================================")
		g.writeln("// TRANSLATION NOTES")
		g.writeln("// =============================================================================")
		for _, s := range result.Suggestions {
			g.writef("// Line %d: %s\n", s.Line, s.ReactCode)
			g.writef("//   → %s\n", s.MintyHint)
			g.writeln("//")
		}
	}

	return g.output.String()
}

// GenerateNode generates Go code for a single node (for testing)
func (g *Generator) GenerateNode(node parser.Node) string {
	g.output.Reset()
	g.generateNode(node, "b")
	return g.output.String()
}

func (g *Generator) generateComponent(comp *parser.Component) {
	// Convert props to Go function parameters
	params := g.generateParams(comp.Props)

	// Write function signature
	g.writef("// %s component\n", comp.Name)

	// Add hook warnings as comments
	if len(comp.Hooks) > 0 {
		g.writeln("// TODO: This component uses React hooks that need manual conversion:")
		for _, hook := range comp.Hooks {
			g.writef("//   - %s (line %d)\n", hook.Type, hook.LineNumber)
		}
	}

	g.writef("func %s(%s) mi.H {\n", comp.Name, params)
	g.indent++

	g.writeIndent()
	g.write("return func(b *mi.Builder) mi.Node {\n")
	g.indent++

	if comp.Body != nil {
		g.writeIndent()
		g.write("return ")
		g.generateNode(comp.Body, "b")
		g.write("\n")
	} else {
		g.writeIndent()
		g.write("return nil // TODO: Component body not parsed\n")
	}

	g.indent--
	g.writeIndent()
	g.write("}\n")

	g.indent--
	g.write("}\n")
}

func (g *Generator) generateParams(props []parser.Prop) string {
	if len(props) == 0 {
		return ""
	}

	var params []string
	for _, prop := range props {
		// Infer type from default value or use interface{}
		typ := "interface{}"
		if prop.DefaultValue != "" {
			if prop.DefaultValue == "true" || prop.DefaultValue == "false" {
				typ = "bool"
			} else if _, err := fmt.Sscanf(prop.DefaultValue, "%d", new(int)); err == nil {
				typ = "int"
			} else {
				typ = "string"
			}
		}
		params = append(params, fmt.Sprintf("%s %s", toCamelCase(prop.Name), typ))
	}

	return strings.Join(params, ", ")
}

func (g *Generator) generateNode(node parser.Node, builder string) {
	if node == nil {
		g.write("nil")
		return
	}

	switch n := node.(type) {
	case *parser.Element:
		g.generateElement(n, builder)
	case *parser.Text:
		g.generateText(n)
	case *parser.Expression:
		g.generateExpression(n)
	case *parser.Fragment:
		g.generateFragment(n, builder)
	case *parser.MapExpr:
		g.generateMap(n, builder)
	case *parser.Conditional:
		g.generateConditional(n, builder)
	case *parser.Ternary:
		g.generateTernary(n, builder)
	default:
		g.writef("nil /* TODO: unhandled node type */")
	}
}

func (g *Generator) generateElement(elem *parser.Element, builder string) {
	tag := elem.Tag
	method := tagToMethod(tag)

	// Check if it's a component reference (PascalCase)
	if isComponentRef(tag) {
		g.writef("%s(%s)", tag, g.generateComponentArgs(elem))
		return
	}

	g.writef("%s.%s(", builder, method)

	// Generate attributes
	hasContent := false
	for _, attr := range elem.Attributes {
		if hasContent {
			g.write(", ")
		}
		g.generateAttribute(&attr)
		hasContent = true
	}

	// Generate children
	for i, child := range elem.Children {
		if hasContent || i > 0 {
			g.write(",\n")
			g.writeIndent()
			g.write("\t")
		}
		g.generateNode(child, builder)
		hasContent = true
	}

	g.write(")")
}

func (g *Generator) generateAttribute(attr *parser.Attribute) {
	if attr.IsSpread {
		g.writef("/* TODO: spread {...%s} not directly supported */", attr.SpreadExpr)
		return
	}

	name := attr.Name
	mintyAttr := attrToMinty(name)

	// String value
	if attr.Value != "" {
		if mintyAttr != "" {
			g.writef("%s(%q)", mintyAttr, attr.Value)
		} else {
			g.writef("mi.Attr(%q, %q)", name, attr.Value)
		}
		return
	}

	// Expression value
	if attr.Expression.Raw != "" {
		value := g.translateExprValue(attr.Expression.Raw)
		if mintyAttr != "" {
			g.writef("%s(%s)", mintyAttr, value)
		} else {
			g.writef("mi.Attr(%q, %s)", name, value)
		}
		return
	}

	// Boolean attribute
	if mintyAttr != "" {
		g.writef("%s()", mintyAttr)
	} else {
		g.writef("mi.Attr(%q, \"\")", name)
	}
}

func (g *Generator) generateText(text *parser.Text) {
	// Escape the text content
	g.writef("%q", text.Content)
}

func (g *Generator) generateExpression(expr *parser.Expression) {
	// Simple variable reference
	if isSimpleIdent(expr.Raw) {
		g.write(toCamelCase(expr.Raw))
		return
	}

	// More complex expression - pass through with comment
	g.writef("/* %s */", expr.Raw)
}

func (g *Generator) generateFragment(frag *parser.Fragment, builder string) {
	g.usesFragment = true

	if len(frag.Children) == 0 {
		g.write("mi.NewFragment()")
		return
	}

	g.write("mi.NewFragment(")
	for i, child := range frag.Children {
		if i > 0 {
			g.write(",")
			g.writeln("")
			g.writeIndent()
		}
		g.generateNode(child, builder)
	}
	g.write(")")
}

func (g *Generator) generateMap(m *parser.MapExpr, builder string) {
	g.usesEach = true

	// Use mi.Each or mi.EachIdx based on whether index is used
	if m.IndexVar != "" {
		g.writef("mi.EachIdx(%s, func(%s int, %s TYPE) mi.H {",
			toCamelCase(m.Collection),
			m.IndexVar,
			m.ItemVar)
	} else {
		g.writef("mi.Each(%s, func(%s TYPE) mi.H {",
			toCamelCase(m.Collection),
			m.ItemVar)
	}
	g.writeln("")
	g.indent++
	g.writeln("return func(b *mi.Builder) mi.Node {")
	g.indent++
	g.write("return ")
	g.generateNode(m.Body, "b")
	g.writeln("")
	g.indent--
	g.writeln("}")
	g.indent--
	g.write("})")
}

func (g *Generator) generateConditional(c *parser.Conditional, builder string) {
	g.usesIf = true

	condition := g.translateCondition(c.Condition)
	g.writef("mi.If(%s,", condition)
	g.writeln("")
	g.indent++
	g.writeIndent()
	g.generateNode(c.Consequent, builder)
	g.writeln(",")
	g.indent--
	g.write(")")
}

func (g *Generator) generateTernary(t *parser.Ternary, builder string) {
	g.usesIfElse = true

	condition := g.translateCondition(t.Condition)
	g.writef("mi.IfElse(%s,", condition)
	g.writeln("")
	g.indent++
	g.writeIndent()
	g.generateNode(t.Consequent, builder)
	g.write(",")
	g.writeln("")
	g.writeIndent()
	g.generateNode(t.Alternate, builder)
	g.write(",")
	g.writeln("")
	g.indent--
	g.write(")")
}

func (g *Generator) generateComponentArgs(elem *parser.Element) string {
	var args []string
	for _, attr := range elem.Attributes {
		if attr.IsSpread {
			continue
		}
		if attr.Value != "" {
			args = append(args, fmt.Sprintf("%q", attr.Value))
		} else if attr.Expression.Raw != "" {
			args = append(args, g.translateExprValue(attr.Expression.Raw))
		}
	}
	return strings.Join(args, ", ")
}

func (g *Generator) translateExprValue(expr string) string {
	// Simple identifier
	if isSimpleIdent(expr) {
		return toCamelCase(expr)
	}

	// Property access: props.name -> name
	if strings.HasPrefix(expr, "props.") {
		return toCamelCase(strings.TrimPrefix(expr, "props."))
	}

	// String concatenation or template literal - simplified
	if strings.Contains(expr, "+") || strings.Contains(expr, "`") {
		return fmt.Sprintf("/* TODO: %s */\"\"", expr)
	}

	return expr
}

func (g *Generator) translateCondition(cond string) string {
	// Simple identifier - likely a boolean
	if isSimpleIdent(cond) {
		return toCamelCase(cond)
	}

	// Property access
	if strings.HasPrefix(cond, "props.") {
		return toCamelCase(strings.TrimPrefix(cond, "props."))
	}

	// Comparison operators
	cond = strings.ReplaceAll(cond, "===", "==")
	cond = strings.ReplaceAll(cond, "!==", "!=")

	// Length check: items.length > 0
	lengthRegex := regexp.MustCompile(`(\w+)\.length\s*([><=!]+)\s*(\d+)`)
	cond = lengthRegex.ReplaceAllString(cond, "len($1) $2 $3")

	return cond
}

// Helper methods

func (g *Generator) write(s string) {
	g.output.WriteString(s)
}

func (g *Generator) writeln(s string) {
	g.output.WriteString(s)
	g.output.WriteString("\n")
}

func (g *Generator) writef(format string, args ...interface{}) {
	g.output.WriteString(fmt.Sprintf(format, args...))
}

func (g *Generator) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.output.WriteString("\t")
	}
}

// Utility functions

func tagToMethod(tag string) string {
	// Handle common HTML tags
	methods := map[string]string{
		"a":          "A",
		"abbr":       "Abbr",
		"address":    "Address",
		"article":    "Article",
		"aside":      "Aside",
		"audio":      "Audio",
		"b":          "B",
		"blockquote": "Blockquote",
		"body":       "Body",
		"br":         "Br",
		"button":     "Button",
		"canvas":     "Canvas",
		"caption":    "Caption",
		"code":       "Code",
		"col":        "Col",
		"colgroup":   "Colgroup",
		"div":        "Div",
		"dl":         "Dl",
		"dt":         "Dt",
		"dd":         "Dd",
		"em":         "Em",
		"fieldset":   "Fieldset",
		"figcaption": "Figcaption",
		"figure":     "Figure",
		"footer":     "Footer",
		"form":       "Form",
		"h1":         "H1",
		"h2":         "H2",
		"h3":         "H3",
		"h4":         "H4",
		"h5":         "H5",
		"h6":         "H6",
		"head":       "Head",
		"header":     "Header",
		"hr":         "Hr",
		"html":       "Html",
		"i":          "I",
		"iframe":     "Iframe",
		"img":        "Img",
		"input":      "Input",
		"label":      "Label",
		"legend":     "Legend",
		"li":         "Li",
		"link":       "Link",
		"main":       "Main",
		"meta":       "Meta",
		"nav":        "Nav",
		"noscript":   "Noscript",
		"ol":         "Ol",
		"optgroup":   "Optgroup",
		"option":     "Option",
		"p":          "P",
		"picture":    "Picture",
		"pre":        "Pre",
		"progress":   "Progress",
		"script":     "Script",
		"section":    "Section",
		"select":     "Select",
		"small":      "Small",
		"source":     "Source",
		"span":       "Span",
		"strong":     "Strong",
		"style":      "Style",
		"sub":        "Sub",
		"summary":    "Summary",
		"sup":        "Sup",
		"table":      "Table",
		"tbody":      "Tbody",
		"td":         "Td",
		"template":   "Template",
		"textarea":   "Textarea",
		"tfoot":      "Tfoot",
		"th":         "Th",
		"thead":      "Thead",
		"time":       "Time",
		"title":      "Title",
		"tr":         "Tr",
		"track":      "Track",
		"u":          "U",
		"ul":         "Ul",
		"video":      "Video",
		"wbr":        "Wbr",
	}

	if method, ok := methods[strings.ToLower(tag)]; ok {
		return method
	}

	// Unknown tag - use El() helper
	return fmt.Sprintf("El(%q)", tag)
}

func attrToMinty(attr string) string {
	attrs := map[string]string{
		"class":       "mi.Class",
		"className":   "mi.Class",
		"id":          "mi.ID",
		"href":        "mi.Href",
		"src":         "mi.Src",
		"alt":         "mi.Alt",
		"title":       "mi.Title",
		"type":        "mi.Type",
		"name":        "mi.Name",
		"value":       "mi.Value",
		"placeholder": "mi.Placeholder",
		"disabled":    "mi.Disabled",
		"checked":     "mi.Checked",
		"selected":    "mi.Selected",
		"required":    "mi.Required",
		"readonly":    "mi.Readonly",
		"multiple":    "mi.Multiple",
		"autofocus":   "mi.Autofocus",
		"autoplay":    "mi.Autoplay",
		"controls":    "mi.Controls",
		"loop":        "mi.Loop",
		"muted":       "mi.Muted",
		"for":         "mi.For",
		"htmlFor":     "mi.For",
		"action":      "mi.Action",
		"method":      "mi.Method",
		"target":      "mi.Target",
		"rel":         "mi.Rel",
		"role":        "mi.Role",
		"tabindex":    "mi.TabIndex",
		"tabIndex":    "mi.TabIndex",
		"style":       "mi.Style",
		"width":       "mi.Width",
		"height":      "mi.Height",
		"min":         "mi.Min",
		"max":         "mi.Max",
		"step":        "mi.Step",
		"pattern":     "mi.Pattern",
		"maxlength":   "mi.MaxLength",
		"maxLength":   "mi.MaxLength",
		"minlength":   "mi.MinLength",
		"minLength":   "mi.MinLength",
		"cols":        "mi.Cols",
		"rows":        "mi.Rows",
		"colspan":     "mi.Colspan",
		"colSpan":     "mi.Colspan",
		"rowspan":     "mi.Rowspan",
		"rowSpan":     "mi.Rowspan",
		"scope":       "mi.Scope",
		"headers":     "mi.Headers",
		"accept":      "mi.Accept",
		"enctype":     "mi.Enctype",
		"novalidate":  "mi.Novalidate",
		"noValidate":  "mi.Novalidate",
		"async":       "mi.Async",
		"defer":       "mi.Defer",
		"crossorigin": "mi.Crossorigin",
		"integrity":   "mi.Integrity",
		"loading":     "mi.Loading",
		"decoding":    "mi.Decoding",
		"srcset":      "mi.Srcset",
		"sizes":       "mi.Sizes",
		"media":       "mi.Media",
		"download":    "mi.Download",
		"hreflang":    "mi.Hreflang",
		"ping":        "mi.Ping",
		"referrerpolicy": "mi.Referrerpolicy",
		"sandbox":     "mi.Sandbox",
		"allow":       "mi.Allow",
		"allowfullscreen": "mi.Allowfullscreen",
		"frameborder": "mi.Attr(\"frameborder\"",
		"lang":        "mi.Lang",
		"translate":   "mi.Translate",
		"dir":         "mi.Dir",
		"hidden":      "mi.Hidden",
		"draggable":   "mi.Draggable",
		"spellcheck":  "mi.Spellcheck",
		"contenteditable": "mi.Contenteditable",
		// HTMX attributes
		"hx-get":       "mi.HtmxGet",
		"hx-post":      "mi.HtmxPost",
		"hx-put":       "mi.HtmxPut",
		"hx-delete":    "mi.HtmxDelete",
		"hx-patch":     "mi.HtmxPatch",
		"hx-target":    "mi.HtmxTarget",
		"hx-swap":      "mi.HtmxSwap",
		"hx-trigger":   "mi.HtmxTrigger",
		"hx-indicator": "mi.HtmxIndicator",
		"hx-push-url":  "mi.HtmxPushURL",
		"hx-select":    "mi.HtmxSelect",
		"hx-confirm":   "mi.HtmxConfirm",
		"hx-boost":     "mi.HtmxBoost",
	}

	if minty, ok := attrs[attr]; ok {
		return minty
	}

	// Data attributes
	if strings.HasPrefix(attr, "data-") {
		dataName := strings.TrimPrefix(attr, "data-")
		return fmt.Sprintf("mi.Data(%q", dataName)
	}

	// Aria attributes
	if strings.HasPrefix(attr, "aria-") {
		return fmt.Sprintf("mi.Attr(%q", attr)
	}

	return ""
}

func isComponentRef(tag string) bool {
	if len(tag) == 0 {
		return false
	}
	// PascalCase = first letter uppercase
	return tag[0] >= 'A' && tag[0] <= 'Z'
}

func isSimpleIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
				return false
			}
		} else {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return false
			}
		}
	}
	return true
}

func toCamelCase(s string) string {
	// Convert kebab-case to camelCase
	parts := strings.Split(s, "-")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}
