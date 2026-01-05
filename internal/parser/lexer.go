package parser

import (
	"strings"
	"unicode"
)

// TokenType represents the type of token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenError
	TokenText
	TokenWhitespace
	TokenTagOpen      // <
	TokenTagClose     // >
	TokenTagSelfClose // />
	TokenTagEnd       // </
	TokenIdent        // tag names, attribute names
	TokenString       // "..." or '...'
	TokenEquals       // =
	TokenJSXExprOpen  // {
	TokenJSXExprClose // }
	TokenDot          // .
	TokenLParen       // (
	TokenRParen       // )
	TokenArrow        // =>
	TokenComma        // ,
	TokenColon        // :
	TokenQuestion     // ?
	TokenAmpAmp       // &&
	TokenPipePipe     // ||
	TokenSpread       // ...
	TokenNumber       // 123, 45.67
	TokenTrue         // true
	TokenFalse        // false
	TokenNull         // null
	TokenUndefined    // undefined
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Value   string
	Line    int
	Column  int
	Offset  int
}

// Lexer tokenizes JSX input
type Lexer struct {
	input   string
	pos     int
	line    int
	column  int
	tokens  []Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
	}
}

// Tokenize processes the input and returns all tokens
func (l *Lexer) Tokenize() []Token {
	for l.pos < len(l.input) {
		l.scanToken()
	}
	l.emit(TokenEOF, "")
	return l.tokens
}

func (l *Lexer) emit(typ TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:   typ,
		Value:  value,
		Line:   l.line,
		Column: l.column,
		Offset: l.pos,
	})
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) peekN(n int) string {
	end := l.pos + n
	if end > len(l.input) {
		end = len(l.input)
	}
	return l.input[l.pos:end]
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	ch := l.input[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

func (l *Lexer) scanToken() {
	ch := l.peek()

	// Whitespace
	if unicode.IsSpace(rune(ch)) {
		l.scanWhitespace()
		return
	}

	// JSX expression
	if ch == '{' {
		l.advance()
		l.emit(TokenJSXExprOpen, "{")
		return
	}
	if ch == '}' {
		l.advance()
		l.emit(TokenJSXExprClose, "}")
		return
	}

	// Tags
	if ch == '<' {
		if l.peekN(2) == "</" {
			l.advance()
			l.advance()
			l.emit(TokenTagEnd, "</")
			return
		}
		l.advance()
		l.emit(TokenTagOpen, "<")
		return
	}

	if l.peekN(2) == "/>" {
		l.advance()
		l.advance()
		l.emit(TokenTagSelfClose, "/>")
		return
	}

	if ch == '>' {
		l.advance()
		l.emit(TokenTagClose, ">")
		return
	}

	// Operators and punctuation
	if ch == '=' {
		if l.peekN(2) == "=>" {
			l.advance()
			l.advance()
			l.emit(TokenArrow, "=>")
			return
		}
		l.advance()
		l.emit(TokenEquals, "=")
		return
	}

	if ch == '.' {
		if l.peekN(3) == "..." {
			l.advance()
			l.advance()
			l.advance()
			l.emit(TokenSpread, "...")
			return
		}
		l.advance()
		l.emit(TokenDot, ".")
		return
	}

	if ch == '(' {
		l.advance()
		l.emit(TokenLParen, "(")
		return
	}
	if ch == ')' {
		l.advance()
		l.emit(TokenRParen, ")")
		return
	}
	if ch == ',' {
		l.advance()
		l.emit(TokenComma, ",")
		return
	}
	if ch == ':' {
		l.advance()
		l.emit(TokenColon, ":")
		return
	}
	if ch == '?' {
		l.advance()
		l.emit(TokenQuestion, "?")
		return
	}

	if l.peekN(2) == "&&" {
		l.advance()
		l.advance()
		l.emit(TokenAmpAmp, "&&")
		return
	}
	if l.peekN(2) == "||" {
		l.advance()
		l.advance()
		l.emit(TokenPipePipe, "||")
		return
	}

	// Strings
	if ch == '"' || ch == '\'' || ch == '`' {
		l.scanString(ch)
		return
	}

	// Numbers
	if unicode.IsDigit(rune(ch)) {
		l.scanNumber()
		return
	}

	// Identifiers and keywords
	if isIdentStart(ch) {
		l.scanIdent()
		return
	}

	// Unknown - treat as text
	l.advance()
	l.emit(TokenText, string(ch))
}

func (l *Lexer) scanWhitespace() {
	start := l.pos
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.peek())) {
		l.advance()
	}
	l.emit(TokenWhitespace, l.input[start:l.pos])
}

func (l *Lexer) scanString(quote byte) {
	l.advance() // consume opening quote
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '\\' {
			l.advance()
			l.advance() // skip escaped char
			continue
		}
		if ch == quote {
			value := l.input[start:l.pos]
			l.advance() // consume closing quote
			l.emit(TokenString, value)
			return
		}
		l.advance()
	}
	// Unterminated string
	l.emit(TokenError, "unterminated string")
}

func (l *Lexer) scanNumber() {
	start := l.pos
	for l.pos < len(l.input) {
		ch := l.peek()
		if !unicode.IsDigit(rune(ch)) && ch != '.' {
			break
		}
		l.advance()
	}
	l.emit(TokenNumber, l.input[start:l.pos])
}

func (l *Lexer) scanIdent() {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.peek()) {
		l.advance()
	}
	value := l.input[start:l.pos]

	// Check for keywords
	switch value {
	case "true":
		l.emit(TokenTrue, value)
	case "false":
		l.emit(TokenFalse, value)
	case "null":
		l.emit(TokenNull, value)
	case "undefined":
		l.emit(TokenUndefined, value)
	default:
		l.emit(TokenIdent, value)
	}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_' || ch == '$'
}

func isIdentChar(ch byte) bool {
	return isIdentStart(ch) ||
		(ch >= '0' && ch <= '9') ||
		ch == '-' // for kebab-case attributes
}

// TokenName returns a human-readable name for a token type
func TokenName(t TokenType) string {
	names := map[TokenType]string{
		TokenEOF:          "EOF",
		TokenError:        "Error",
		TokenText:         "Text",
		TokenWhitespace:   "Whitespace",
		TokenTagOpen:      "TagOpen",
		TokenTagClose:     "TagClose",
		TokenTagSelfClose: "TagSelfClose",
		TokenTagEnd:       "TagEnd",
		TokenIdent:        "Ident",
		TokenString:       "String",
		TokenEquals:       "Equals",
		TokenJSXExprOpen:  "JSXExprOpen",
		TokenJSXExprClose: "JSXExprClose",
		TokenDot:          "Dot",
		TokenLParen:       "LParen",
		TokenRParen:       "RParen",
		TokenArrow:        "Arrow",
		TokenComma:        "Comma",
		TokenColon:        "Colon",
		TokenQuestion:     "Question",
		TokenAmpAmp:       "AmpAmp",
		TokenPipePipe:     "PipePipe",
		TokenSpread:       "Spread",
		TokenNumber:       "Number",
		TokenTrue:         "True",
		TokenFalse:        "False",
		TokenNull:         "Null",
		TokenUndefined:    "Undefined",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "Unknown"
}

// IsJSKeyword checks if an identifier is a JS keyword we care about
func IsJSKeyword(s string) bool {
	keywords := map[string]bool{
		"const": true, "let": true, "var": true,
		"function": true, "return": true,
		"if": true, "else": true,
		"for": true, "while": true,
		"import": true, "export": true, "default": true,
		"from": true, "as": true,
		"class": true, "extends": true,
		"new": true, "this": true,
		"typeof": true, "instanceof": true,
	}
	return keywords[strings.ToLower(s)]
}
