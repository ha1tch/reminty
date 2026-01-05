package patterns

import (
	"regexp"
	"strings"

	"github.com/ha1tch/reminty/internal/parser"
)

// PatternType identifies React patterns
type PatternType string

const (
	PatternTabs         PatternType = "tabs"
	PatternAccordion    PatternType = "accordion"
	PatternFilter       PatternType = "filter"
	PatternSearch       PatternType = "search"
	PatternFormDeps     PatternType = "form-dependencies"
	PatternModal        PatternType = "modal"
	PatternDropdown     PatternType = "dropdown"
	PatternPagination   PatternType = "pagination"
	PatternInfiniteScroll PatternType = "infinite-scroll"
	PatternDarkMode     PatternType = "dark-mode"
)

// DetectedPattern represents a pattern found in the code
type DetectedPattern struct {
	Type        PatternType
	Line        int
	Confidence  float64 // 0.0 to 1.0
	Description string
	ReactCode   string
	MintyCode   string
}

// Detector analyzes React code for patterns
type Detector struct {
	patterns []DetectedPattern
}

// NewDetector creates a new pattern detector
func NewDetector() *Detector {
	return &Detector{
		patterns: []DetectedPattern{},
	}
}

// Analyze looks for patterns in a parse result
func (d *Detector) Analyze(result *parser.ParseResult) []DetectedPattern {
	d.patterns = []DetectedPattern{}

	for _, comp := range result.File.Components {
		d.analyzeComponent(&comp)
	}

	return d.patterns
}

// AnalyzeSource analyzes raw source code for patterns
func (d *Detector) AnalyzeSource(source string) []DetectedPattern {
	d.patterns = []DetectedPattern{}

	// Tab patterns
	d.detectTabsPattern(source)

	// Filter/search patterns
	d.detectFilterPattern(source)

	// Form dependency patterns
	d.detectFormDepsPattern(source)

	// Modal patterns
	d.detectModalPattern(source)

	// Dark mode patterns
	d.detectDarkModePattern(source)

	// Pagination patterns
	d.detectPaginationPattern(source)

	return d.patterns
}

func (d *Detector) analyzeComponent(comp *parser.Component) {
	// Check hooks for patterns
	for _, hook := range comp.Hooks {
		switch hook.Type {
		case "useState":
			d.analyzeStateUsage(hook, comp)
		case "useEffect":
			d.analyzeEffectUsage(hook, comp)
		}
	}
}

func (d *Detector) analyzeStateUsage(hook parser.Hook, comp *parser.Component) {
	// Look for common state patterns
	name := strings.ToLower(hook.Name)

	if strings.Contains(name, "tab") || strings.Contains(name, "active") {
		d.addPattern(DetectedPattern{
			Type:        PatternTabs,
			Line:        hook.LineNumber,
			Confidence:  0.7,
			Description: "Tab state management detected",
			ReactCode:   "useState for active tab",
			MintyCode: `mdy.Dyn("tabs").
    States([]mdy.ComponentState{
        mdy.ActiveState("tab1", "Tab 1", content1),
        mdy.NewState("tab2", "Tab 2", content2),
    }).
    Build()`,
		})
	}

	if strings.Contains(name, "filter") || strings.Contains(name, "search") || strings.Contains(name, "query") {
		d.addPattern(DetectedPattern{
			Type:        PatternFilter,
			Line:        hook.LineNumber,
			Confidence:  0.8,
			Description: "Filter/search state detected",
			ReactCode:   "useState for filter/search value",
			MintyCode: `mdy.Dyn("search").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            Fields: []mdy.FilterableField{
                mdy.TextField("search", "Search"),
            },
        },
    }).
    Build()`,
		})
	}

	if strings.Contains(name, "modal") || strings.Contains(name, "open") || strings.Contains(name, "show") {
		d.addPattern(DetectedPattern{
			Type:        PatternModal,
			Line:        hook.LineNumber,
			Confidence:  0.6,
			Description: "Modal/dialog state detected",
			ReactCode:   "useState for modal visibility",
			MintyCode: `// Consider HTMX for modal:
b.Button(
    mi.HtmxGet("/modal-content"),
    mi.HtmxTarget("#modal"),
    mi.HtmxSwap("innerHTML"),
    "Open Modal",
)`,
		})
	}

	if strings.Contains(name, "dark") || strings.Contains(name, "theme") {
		d.addPattern(DetectedPattern{
			Type:        PatternDarkMode,
			Line:        hook.LineNumber,
			Confidence:  0.9,
			Description: "Dark mode/theme state detected",
			ReactCode:   "useState for theme",
			MintyCode: `darkMode := mi.DarkModeTailwind(mi.DarkModeSVGIcons())
// In <head>:
darkMode.Script(b)
// Toggle button:
darkMode.Toggle(b, mi.Class("p-2 rounded"))`,
		})
	}

	if strings.Contains(name, "page") || strings.Contains(name, "offset") || strings.Contains(name, "limit") {
		d.addPattern(DetectedPattern{
			Type:        PatternPagination,
			Line:        hook.LineNumber,
			Confidence:  0.7,
			Description: "Pagination state detected",
			ReactCode:   "useState for pagination",
			MintyCode: `mdy.Dyn("list").
    Data(mdy.FilterableDataset{
        Items: items,
        Options: mdy.FilterOptions{
            EnablePagination: true,
            ItemsPerPage:     20,
        },
    }).
    Build()`,
		})
	}
}

func (d *Detector) analyzeEffectUsage(hook parser.Hook, comp *parser.Component) {
	// Effects often indicate side effects that should be server-side
	d.addPattern(DetectedPattern{
		Type:        PatternType("effect"),
		Line:        hook.LineNumber,
		Confidence:  0.5,
		Description: "useEffect detected - consider server-side alternative",
		ReactCode:   "useEffect for side effects",
		MintyCode:   "// Most useEffect logic belongs server-side in Go handlers",
	})
}

func (d *Detector) detectTabsPattern(source string) {
	// Look for tab-related patterns
	tabPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)role=["']tablist["']`),
		regexp.MustCompile(`(?i)role=["']tab["']`),
		regexp.MustCompile(`(?i)aria-selected`),
		regexp.MustCompile(`(?i)className=.*tab.*active`),
		regexp.MustCompile(`(?i)activeTab|selectedTab|currentTab`),
	}

	for _, pattern := range tabPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternTabs,
				Line:        line,
				Confidence:  0.8,
				Description: "Tab UI pattern detected",
				ReactCode:   pattern.String(),
				MintyCode: `mdy.Dyn("tabs").
    States([]mdy.ComponentState{
        mdy.ActiveState("tab1", "Tab 1", content1),
        mdy.NewState("tab2", "Tab 2", content2),
    }).
    Theme(mdy.NewTailwindDynamicTheme()).
    Build()`,
			})
			break
		}
	}
}

func (d *Detector) detectFilterPattern(source string) {
	filterPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\.filter\s*\(`),
		regexp.MustCompile(`(?i)searchTerm|filterValue|query`),
		regexp.MustCompile(`(?i)type=["']search["']`),
		regexp.MustCompile(`(?i)onChange.*filter`),
	}

	for _, pattern := range filterPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternFilter,
				Line:        line,
				Confidence:  0.7,
				Description: "Filter/search pattern detected",
				ReactCode:   "Client-side filtering",
				MintyCode: `mdy.Dyn("filter").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            Fields: []mdy.FilterableField{
                mdy.TextField("search", "Search"),
                mdy.SelectField("category", "Category", categories),
            },
        },
        Options: mdy.FilterOptions{
            EnableSearch: true,
        },
    }).
    Build()`,
			})
			break
		}
	}
}

func (d *Detector) detectFormDepsPattern(source string) {
	depPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)disabled=\{.*\}`),
		regexp.MustCompile(`(?i)hidden.*&&`),
		regexp.MustCompile(`(?i)style=\{.*display.*none`),
		regexp.MustCompile(`(?i)showIf|hideIf|visibleWhen`),
	}

	for _, pattern := range depPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternFormDeps,
				Line:        line,
				Confidence:  0.6,
				Description: "Form field dependency pattern detected",
				ReactCode:   "Conditional field visibility",
				MintyCode: `mdy.Dyn("form").
    Rules([]mdy.DependencyRule{
        mdy.ShowWhen("field1", "equals", "value", "dependent-field"),
        mdy.EnableWhen("checkbox", "equals", true, "submit-btn"),
    }).
    Build()`,
			})
			break
		}
	}
}

func (d *Detector) detectModalPattern(source string) {
	modalPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)role=["']dialog["']`),
		regexp.MustCompile(`(?i)aria-modal`),
		regexp.MustCompile(`(?i)Modal|Dialog`),
		regexp.MustCompile(`(?i)isOpen|showModal|modalOpen`),
	}

	for _, pattern := range modalPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternModal,
				Line:        line,
				Confidence:  0.7,
				Description: "Modal/dialog pattern detected",
				ReactCode:   "Modal component",
				MintyCode: `// HTMX modal pattern:
b.Button(
    mi.HtmxGet("/modal-content"),
    mi.HtmxTarget("#modal-container"),
    mi.HtmxSwap("innerHTML"),
    "Open",
)
// Modal container in layout:
b.Div(mi.ID("modal-container"))`,
			})
			break
		}
	}
}

func (d *Detector) detectDarkModePattern(source string) {
	darkPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)darkMode|darkTheme|isDark`),
		regexp.MustCompile(`(?i)theme.*dark|dark.*theme`),
		regexp.MustCompile(`(?i)prefers-color-scheme`),
		regexp.MustCompile(`(?i)toggleTheme|toggleDark`),
	}

	for _, pattern := range darkPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternDarkMode,
				Line:        line,
				Confidence:  0.9,
				Description: "Dark mode pattern detected",
				ReactCode:   "Theme toggle logic",
				MintyCode: `// Tailwind dark mode:
darkMode := mi.DarkModeTailwind(
    mi.DarkModeDefault("system"),
    mi.DarkModeSVGIcons(),
)
// In <head> (before body renders):
darkMode.Script(b)
// Toggle button:
darkMode.Toggle(b, mi.Class("p-2 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-700"))`,
			})
			break
		}
	}
}

func (d *Detector) detectPaginationPattern(source string) {
	pagePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)pagination|paginate`),
		regexp.MustCompile(`(?i)pageNumber|currentPage|page\s*=`),
		regexp.MustCompile(`(?i)nextPage|prevPage|previousPage`),
		regexp.MustCompile(`(?i)itemsPerPage|pageSize|limit`),
	}

	for _, pattern := range pagePatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternPagination,
				Line:        line,
				Confidence:  0.75,
				Description: "Pagination pattern detected",
				ReactCode:   "Pagination state/logic",
				MintyCode: `mdy.Dyn("list").
    Data(mdy.FilterableDataset{
        Items: items,
        Options: mdy.FilterOptions{
            EnablePagination: true,
            ItemsPerPage:     20,
        },
    }).
    Build()
// Or use HTMX for server-side pagination:
b.Button(
    mi.HtmxGet("/items?page=2"),
    mi.HtmxTarget("#item-list"),
    mi.HtmxSwap("innerHTML"),
    "Next Page",
)`,
			})
			break
		}
	}
}

func (d *Detector) addPattern(p DetectedPattern) {
	// Avoid duplicates
	for _, existing := range d.patterns {
		if existing.Type == p.Type && existing.Line == p.Line {
			return
		}
	}
	d.patterns = append(d.patterns, p)
}

func countLines(s string) int {
	return strings.Count(s, "\n") + 1
}
