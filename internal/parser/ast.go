package parser

// NodeType represents the type of AST node
type NodeType int

const (
	NodeComponent NodeType = iota
	NodeElement
	NodeText
	NodeExpression
	NodeFragment
	NodeMap
	NodeConditional
	NodeTernary
	NodeSpread
	NodeImport
	NodeExport
)

// Node is the interface for all AST nodes
type Node interface {
	Type() NodeType
	Line() int
}

// Component represents a React component definition
type Component struct {
	Name       string
	Props      []Prop
	Body       Node
	Hooks      []Hook
	StateVars  []StateVariable // extracted useState variables
	DerivedVars []DerivedVariable // const x = expr dependent on state
	LineNumber int
}

func (c *Component) Type() NodeType { return NodeComponent }
func (c *Component) Line() int      { return c.LineNumber }

// StateVariable represents a useState declaration
type StateVariable struct {
	Name       string // variable name (e.g., "filter")
	Setter     string // setter function name (e.g., "setFilter")
	InitValue  string // initial value as string
	InitType   string // inferred type: "string", "bool", "int", "[]interface{}"
	LineNumber int
}

// DerivedVariable represents a const derived from state
type DerivedVariable struct {
	Name       string   // variable name (e.g., "filteredUsers")
	Expression string   // the full expression
	SourceVar  string   // source collection (e.g., "users")
	Operation  string   // operation type: filter, map, find, some, every, reduce, sort, slice
	ResultType string   // inferred Go type
	DependsOn  []string // state variables it depends on
	LineNumber int
}

// Prop represents a component prop
type Prop struct {
	Name         string
	DefaultValue string
	JSType       string // for TypeScript
}

// Hook represents a React hook usage
type Hook struct {
	Type       string // useState, useEffect, useMemo, etc.
	Name       string // variable name
	InitValue  string
	Deps       []string
	Body       string
	LineNumber int
}

// EventHandler represents an event handler in JSX
type EventHandler struct {
	EventType   string   // onClick, onChange, onSubmit, etc.
	HandlerBody string   // the handler expression/function body
	SetterCalls []string // setState calls detected: ["setFilter", "setCount"]
	StateVars   []string // state variables referenced
	IsInline    bool     // true if inline arrow function
	LineNumber  int
}

// Element represents a JSX element
type Element struct {
	Tag        string
	Attributes []Attribute
	Children   []Node
	SelfClose  bool
	LineNumber int
}

func (e *Element) Type() NodeType { return NodeElement }
func (e *Element) Line() int      { return e.LineNumber }

// Attribute represents a JSX attribute
type Attribute struct {
	Name         string
	Value        string        // for string values
	Expression   Expression    // for {expression} values
	IsSpread     bool          // for {...props}
	SpreadExpr   string
	EventHandler *EventHandler // parsed event handler (if applicable)
}

// Text represents text content
type Text struct {
	Content    string
	LineNumber int
}

func (t *Text) Type() NodeType { return NodeText }
func (t *Text) Line() int      { return t.LineNumber }

// Expression represents a JS expression in JSX
type Expression struct {
	Raw        string
	Parsed     Node // if we can parse it further
	LineNumber int
}

func (e *Expression) Type() NodeType { return NodeExpression }
func (e *Expression) Line() int      { return e.LineNumber }

// Fragment represents a React fragment (<>...</> or <Fragment>)
type Fragment struct {
	Children   []Node
	LineNumber int
}

func (f *Fragment) Type() NodeType { return NodeFragment }
func (f *Fragment) Line() int      { return f.LineNumber }

// MapExpr represents {items.map(item => ...)}
type MapExpr struct {
	Collection string
	ItemVar    string
	IndexVar   string
	Body       Node
	LineNumber int
}

func (m *MapExpr) Type() NodeType { return NodeMap }
func (m *MapExpr) Line() int      { return m.LineNumber }

// Conditional represents {condition && <Element/>}
type Conditional struct {
	Condition  string
	Consequent Node
	LineNumber int
}

func (c *Conditional) Type() NodeType { return NodeConditional }
func (c *Conditional) Line() int      { return c.LineNumber }

// Ternary represents {condition ? <A/> : <B/>}
type Ternary struct {
	Condition  string
	Consequent Node
	Alternate  Node
	LineNumber int
}

func (t *Ternary) Type() NodeType { return NodeTernary }
func (t *Ternary) Line() int      { return t.LineNumber }

// Import represents an import statement
type Import struct {
	Default    string            // default import name
	Named      map[string]string // { name: alias }
	Namespace  string            // * as name
	Source     string            // module path
	LineNumber int
}

func (i *Import) Type() NodeType { return NodeImport }
func (i *Import) Line() int      { return i.LineNumber }

// File represents a complete JSX file
type File struct {
	Imports    []Import
	Components []Component
	Exports    []string
}

// ParseResult contains the parsed AST and any warnings/suggestions
type ParseResult struct {
	File        *File
	Warnings    []Warning
	Suggestions []Suggestion
}

// Warning represents a parsing warning
type Warning struct {
	Line    int
	Column  int
	Message string
}

// Suggestion represents a translation suggestion
type Suggestion struct {
	Line        int
	ReactCode   string
	MintyHint   string
	PatternType string // "useState", "useEffect", "map", "conditional", etc.
}
