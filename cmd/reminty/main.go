package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ha1tch/reminty/internal/generator"
	"github.com/ha1tch/reminty/internal/parser"
	"github.com/ha1tch/reminty/internal/patterns"
)

const version = "0.1.0"

func main() {
	// Flags
	var (
		outputFile   string
		analyzeOnly  bool
		showVersion  bool
		showHelp     bool
		verbose      bool
	)

	flag.StringVar(&outputFile, "o", "", "Output file (default: stdout)")
	flag.StringVar(&outputFile, "output", "", "Output file (default: stdout)")
	flag.BoolVar(&analyzeOnly, "analyze", false, "Only analyze patterns, don't generate code")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `reminty - Convert React/JSX to Go + minty

Usage:
  reminty [options] <input.jsx>
  reminty [options] < input.jsx
  cat input.jsx | reminty [options]

Options:
  -o, --output <file>   Write output to file (default: stdout)
  -analyze              Only analyze patterns, don't generate code
  -verbose              Show detailed analysis
  -v, --version         Show version
  -h, --help            Show this help

Examples:
  reminty Component.jsx                    # Convert and print to stdout
  reminty -o component.go Component.jsx    # Convert to file
  reminty -analyze Component.jsx           # Show pattern analysis only
  cat Component.jsx | reminty              # Read from stdin

The tool will:
  1. Parse JSX structure and convert to minty builder calls
  2. Detect React patterns and suggest minty/mintydyn equivalents
  3. Flag hooks (useState, useEffect) with migration guidance
  4. Convert .map() to mi.Each(), conditionals to mi.If()/mi.IfElse()

Not supported (flagged as TODO):
  - Complex hooks (useReducer, useContext with complex state)
  - Third-party component libraries (Material UI, etc.)
  - CSS-in-JS (styled-components, emotion)
  - Dynamic imports

`)
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("reminty version %s\n", version)
		os.Exit(0)
	}

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Get input
	var input string
	var inputName string

	if flag.NArg() > 0 {
		// Read from file
		inputFile := flag.Arg(0)
		inputName = filepath.Base(inputFile)
		data, err := os.ReadFile(inputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		input = string(data)
	} else {
		// Read from stdin
		inputName = "stdin"
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		input = string(data)
	}

	if strings.TrimSpace(input) == "" {
		fmt.Fprintln(os.Stderr, "Error: No input provided")
		flag.Usage()
		os.Exit(1)
	}

	// Parse
	lexer := parser.NewLexer(input)
	tokens := lexer.Tokenize()

	if verbose {
		fmt.Fprintf(os.Stderr, "Parsed %d tokens from %s\n", len(tokens), inputName)
	}

	p := parser.NewParserWithSource(tokens, input)
	result := p.Parse()

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d components, %d imports\n",
			len(result.File.Components), len(result.File.Imports))
	}

	// Detect patterns
	detector := patterns.NewDetector()
	detectedPatterns := detector.AnalyzeSource(input)

	// Also analyze the parsed result
	parsedPatterns := detector.Analyze(result)
	detectedPatterns = append(detectedPatterns, parsedPatterns...)

	if verbose || analyzeOnly {
		printPatternAnalysis(detectedPatterns, result)
	}

	if analyzeOnly {
		os.Exit(0)
	}

	// Generate code
	gen := generator.NewGenerator()
	output := gen.Generate(result)

	// Add pattern suggestions as comments
	if len(detectedPatterns) > 0 {
		output += "\n// =============================================================================\n"
		output += "// DETECTED PATTERNS - CONSIDER USING MINTYDYN\n"
		output += "// =============================================================================\n"
		for _, p := range detectedPatterns {
			output += fmt.Sprintf("//\n// %s (line %d, confidence: %.0f%%)\n", p.Description, p.Line, p.Confidence*100)
			output += fmt.Sprintf("// React: %s\n", p.ReactCode)
			output += "// Minty equivalent:\n"
			for _, line := range strings.Split(p.MintyCode, "\n") {
				output += fmt.Sprintf("//   %s\n", line)
			}
		}
	}

	// Write output
	if outputFile != "" {
		err := os.WriteFile(outputFile, []byte(output), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Written to %s\n", outputFile)
	} else {
		fmt.Print(output)
	}
}

func printPatternAnalysis(patterns []patterns.DetectedPattern, result *parser.ParseResult) {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "=== PATTERN ANALYSIS ===")
	fmt.Fprintln(os.Stderr, "")

	// Hooks
	for _, comp := range result.File.Components {
		if len(comp.Hooks) > 0 {
			fmt.Fprintf(os.Stderr, "Component: %s\n", comp.Name)
			fmt.Fprintln(os.Stderr, "  Hooks detected:")
			for _, hook := range comp.Hooks {
				fmt.Fprintf(os.Stderr, "    - %s (line %d)\n", hook.Type, hook.LineNumber)
			}
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// Suggestions from parsing
	if len(result.Suggestions) > 0 {
		fmt.Fprintln(os.Stderr, "Hook migration suggestions:")
		for _, s := range result.Suggestions {
			fmt.Fprintf(os.Stderr, "  Line %d: %s\n", s.Line, s.ReactCode)
			fmt.Fprintf(os.Stderr, "    → %s\n", s.MintyHint)
		}
		fmt.Fprintln(os.Stderr, "")
	}

	// Detected patterns
	if len(patterns) > 0 {
		fmt.Fprintln(os.Stderr, "Detected UI patterns:")
		for _, p := range patterns {
			confidence := ""
			if p.Confidence >= 0.8 {
				confidence = "HIGH"
			} else if p.Confidence >= 0.6 {
				confidence = "MEDIUM"
			} else {
				confidence = "LOW"
			}
			fmt.Fprintf(os.Stderr, "  [%s] %s (line %d)\n", confidence, p.Description, p.Line)
			fmt.Fprintf(os.Stderr, "    React: %s\n", p.ReactCode)
			fmt.Fprintln(os.Stderr, "    Minty suggestion:")
			for _, line := range strings.Split(p.MintyCode, "\n") {
				if strings.TrimSpace(line) != "" {
					fmt.Fprintf(os.Stderr, "      %s\n", line)
				}
			}
			fmt.Fprintln(os.Stderr, "")
		}
	}

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Fprintln(os.Stderr, "Warnings:")
		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "  Line %d: %s\n", w.Line, w.Message)
		}
		fmt.Fprintln(os.Stderr, "")
	}
}
