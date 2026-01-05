package patterns

import (
	"regexp"
	"strings"

	"github.com/ha1tch/reminty/internal/parser"
)

// PatternType identifies React patterns
type PatternType string

const (
	PatternTabs           PatternType = "tabs"
	PatternAccordion      PatternType = "accordion"
	PatternFilter         PatternType = "filter"
	PatternSearch         PatternType = "search"
	PatternFormDeps       PatternType = "form-dependencies"
	PatternModal          PatternType = "modal"
	PatternDropdown       PatternType = "dropdown"
	PatternPagination     PatternType = "pagination"
	PatternInfiniteScroll PatternType = "infinite-scroll"
	PatternDarkMode       PatternType = "dark-mode"
	PatternToggle         PatternType = "toggle"
	PatternSortableTable  PatternType = "sortable-table"
)

// DetectedPattern represents a pattern found in the code
type DetectedPattern struct {
	Type        PatternType
	Line        int
	Confidence  float64 // 0.0 to 1.0
	Description string
	ReactCode   string
	MintyCode   string
	StateVars   []string // state variables involved
	DerivedVars []string // derived variables involved
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
	
	// Accordion patterns
	d.detectAccordionPattern(source)
	
	// Toggle patterns
	d.detectTogglePattern(source)
	
	// Sortable table patterns
	d.detectSortableTablePattern(source)

	return d.patterns
}

func (d *Detector) analyzeComponent(comp *parser.Component) {
	// Analyze state variables for patterns
	d.analyzeStatePatterns(comp)
	
	// Analyze derived variables for patterns
	d.analyzeDerivedPatterns(comp)
	
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

// analyzeStatePatterns detects patterns from useState variables
func (d *Detector) analyzeStatePatterns(comp *parser.Component) {
	stateNames := make(map[string]parser.StateVariable)
	for _, sv := range comp.StateVars {
		stateNames[strings.ToLower(sv.Name)] = sv
	}
	
	// Tab pattern: activeTab/selectedTab + string type
	for name, sv := range stateNames {
		if (strings.Contains(name, "tab") || strings.Contains(name, "selected")) && 
			sv.InitType == "string" {
			d.addPattern(DetectedPattern{
				Type:        PatternTabs,
				Line:        sv.LineNumber,
				Confidence:  0.85,
				Description: "Tab state with string selector",
				ReactCode:   "useState('" + sv.InitValue + "') for tab selection",
				StateVars:   []string{sv.Name},
				MintyCode: generateTabsMinty(sv.Name, sv.InitValue),
			})
		}
	}
	
	// Filter pattern: filter/search state + derived .filter()
	for name, sv := range stateNames {
		if (strings.Contains(name, "filter") || strings.Contains(name, "search") || 
			strings.Contains(name, "query")) && sv.InitType == "string" {
			// Check if there's a corresponding derived filter
			hasDerivedFilter := false
			for _, dv := range comp.DerivedVars {
				if dv.Operation == "filter" {
					hasDerivedFilter = true
					break
				}
			}
			
			confidence := 0.7
			if hasDerivedFilter {
				confidence = 0.95
			}
			
			d.addPattern(DetectedPattern{
				Type:        PatternFilter,
				Line:        sv.LineNumber,
				Confidence:  confidence,
				Description: "Filter/search with derived filtered list",
				ReactCode:   "useState for filter + .filter() derived state",
				StateVars:   []string{sv.Name},
				MintyCode:   generateFilterMinty(sv.Name),
			})
		}
	}
	
	// Modal/toggle pattern: boolean state for visibility
	for name, sv := range stateNames {
		if sv.InitType == "bool" {
			if strings.Contains(name, "modal") || strings.Contains(name, "dialog") {
				d.addPattern(DetectedPattern{
					Type:        PatternModal,
					Line:        sv.LineNumber,
					Confidence:  0.85,
					Description: "Modal visibility state",
					ReactCode:   "useState(false) for modal",
					StateVars:   []string{sv.Name},
					MintyCode:   generateModalMinty(sv.Name),
				})
			} else if strings.Contains(name, "open") || strings.Contains(name, "expanded") ||
				strings.Contains(name, "collapsed") {
				d.addPattern(DetectedPattern{
					Type:        PatternAccordion,
					Line:        sv.LineNumber,
					Confidence:  0.75,
					Description: "Accordion/collapsible state",
					ReactCode:   "useState for expand/collapse",
					StateVars:   []string{sv.Name},
					MintyCode:   generateAccordionMinty(sv.Name),
				})
			} else if strings.Contains(name, "active") || strings.Contains(name, "enabled") ||
				strings.Contains(name, "show") || strings.Contains(name, "visible") {
				d.addPattern(DetectedPattern{
					Type:        PatternToggle,
					Line:        sv.LineNumber,
					Confidence:  0.7,
					Description: "Toggle/visibility state",
					ReactCode:   "useState(boolean) for toggle",
					StateVars:   []string{sv.Name},
					MintyCode:   generateToggleMinty(sv.Name),
				})
			}
		}
	}
	
	// Pagination pattern: page number state
	for name, sv := range stateNames {
		if (strings.Contains(name, "page") || strings.Contains(name, "offset")) &&
			(sv.InitType == "int" || sv.InitType == "float64") {
			d.addPattern(DetectedPattern{
				Type:        PatternPagination,
				Line:        sv.LineNumber,
				Confidence:  0.8,
				Description: "Pagination state",
				ReactCode:   "useState for page number",
				StateVars:   []string{sv.Name},
				MintyCode:   generatePaginationMinty(sv.Name),
			})
		}
	}
	
	// Sort pattern: sort column/direction state
	for name, sv := range stateNames {
		if strings.Contains(name, "sort") {
			d.addPattern(DetectedPattern{
				Type:        PatternSortableTable,
				Line:        sv.LineNumber,
				Confidence:  0.8,
				Description: "Sortable table state",
				ReactCode:   "useState for sort column/direction",
				StateVars:   []string{sv.Name},
				MintyCode:   generateSortableMinty(sv.Name),
			})
		}
	}
}

// analyzeDerivedPatterns detects patterns from derived variables
func (d *Detector) analyzeDerivedPatterns(comp *parser.Component) {
	for _, dv := range comp.DerivedVars {
		switch dv.Operation {
		case "filter":
			// Already handled in analyzeStatePatterns if combined with filter state
			// But add if standalone
			alreadyDetected := false
			for _, p := range d.patterns {
				if p.Type == PatternFilter {
					alreadyDetected = true
					break
				}
			}
			if !alreadyDetected {
				d.addPattern(DetectedPattern{
					Type:        PatternFilter,
					Line:        dv.LineNumber,
					Confidence:  0.65,
					Description: "Client-side filtering detected",
					ReactCode:   dv.Name + " = " + dv.SourceVar + ".filter(...)",
					DerivedVars: []string{dv.Name},
					MintyCode:   generateFilterMinty("filter"),
				})
			}
		case "sort":
			d.addPattern(DetectedPattern{
				Type:        PatternSortableTable,
				Line:        dv.LineNumber,
				Confidence:  0.75,
				Description: "Client-side sorting detected",
				ReactCode:   dv.Name + " = " + dv.SourceVar + ".sort(...)",
				DerivedVars: []string{dv.Name},
				MintyCode:   generateSortableMinty("sort"),
			})
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

// detectAccordionPattern looks for accordion/collapsible patterns
func (d *Detector) detectAccordionPattern(source string) {
	accPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)accordion`),
		regexp.MustCompile(`(?i)collapsible`),
		regexp.MustCompile(`(?i)expand.*collapse|collapse.*expand`),
		regexp.MustCompile(`(?i)aria-expanded`),
	}

	for _, pattern := range accPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternAccordion,
				Line:        line,
				Confidence:  0.75,
				Description: "Accordion/collapsible pattern detected",
				ReactCode:   "Expand/collapse UI",
				MintyCode: `mdy.Dyn("accordion").
    States([]mdy.ComponentState{
        mdy.NewState("section1", "Section 1", content1),
        mdy.NewState("section2", "Section 2", content2),
    }).
    Options(mdy.AccordionOptions{
        AllowMultiple: false,
    }).
    Build()`,
			})
			break
		}
	}
}

// detectTogglePattern looks for toggle/switch patterns
func (d *Detector) detectTogglePattern(source string) {
	togglePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)toggle|switch`),
		regexp.MustCompile(`(?i)setIs\w+\(!`),
		regexp.MustCompile(`(?i)prev\s*=>\s*!prev`),
		regexp.MustCompile(`(?i)type=["']checkbox["']`),
	}

	for _, pattern := range togglePatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternToggle,
				Line:        line,
				Confidence:  0.7,
				Description: "Toggle/switch pattern detected",
				ReactCode:   "Boolean toggle state",
				MintyCode: `// Simple toggle with HTMX:
b.Button(
    mi.HtmxPost("/toggle"),
    mi.HtmxSwap("outerHTML"),
    "Toggle",
)
// Or with mintydyn:
mdy.Toggle("feature-flag", mdy.ToggleOptions{
    OnLabel:  "Enabled",
    OffLabel: "Disabled",
})`,
			})
			break
		}
	}
}

// detectSortableTablePattern looks for sortable table patterns
func (d *Detector) detectSortableTablePattern(source string) {
	sortPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)sortColumn|sortBy|sortField`),
		regexp.MustCompile(`(?i)sortDirection|sortOrder|ascending|descending`),
		regexp.MustCompile(`(?i)\.sort\s*\(`),
		regexp.MustCompile(`(?i)onClick.*sort`),
	}

	for _, pattern := range sortPatterns {
		if loc := pattern.FindStringIndex(source); loc != nil {
			line := countLines(source[:loc[0]])
			d.addPattern(DetectedPattern{
				Type:        PatternSortableTable,
				Line:        line,
				Confidence:  0.75,
				Description: "Sortable table pattern detected",
				ReactCode:   "Table sorting logic",
				MintyCode: `mdy.Dyn("table").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            SortableFields: []string{"name", "date", "status"},
        },
        Options: mdy.FilterOptions{
            EnableSort:       true,
            DefaultSortField: "name",
            DefaultSortDir:   mdy.SortAsc,
        },
    }).
    Build()`,
			})
			break
		}
	}
}

// Helper functions to generate mintydyn code suggestions

func generateTabsMinty(stateName, initValue string) string {
	return `mdy.Dyn("tabs").
    States([]mdy.ComponentState{
        mdy.ActiveState("` + initValue + `", "Tab 1", tab1Content),
        mdy.NewState("tab2", "Tab 2", tab2Content),
        mdy.NewState("tab3", "Tab 3", tab3Content),
    }).
    Theme(mdy.NewTailwindDynamicTheme()).
    Build()

// Handler for tab state:
// GET /tabs?` + stateName + `=<value> → returns updated component HTML`
}

func generateFilterMinty(stateName string) string {
	return `mdy.Dyn("filter").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            Fields: []mdy.FilterableField{
                mdy.TextField("` + stateName + `", "Search"),
            },
        },
        Options: mdy.FilterOptions{
            EnableSearch: true,
            Debounce:     300, // ms
        },
    }).
    Build()

// Handler:
// GET /filter?` + stateName + `=<value> → returns filtered results HTML`
}

func generateModalMinty(stateName string) string {
	return `// HTMX modal pattern (recommended):
b.Button(
    mi.HtmxGet("/modal-content"),
    mi.HtmxTarget("#modal-container"),
    mi.HtmxSwap("innerHTML"),
    "Open Modal",
)

// Modal container (in layout):
b.Div(mi.ID("modal-container"),
    mi.Class("fixed inset-0 z-50 hidden"),
)

// Close handler in modal content:
mi.HtmxDelete("/modal", mi.HtmxTarget("#modal-container"), mi.HtmxSwap("innerHTML"))`
}

func generateAccordionMinty(stateName string) string {
	return `mdy.Dyn("accordion").
    States([]mdy.ComponentState{
        mdy.NewState("section1", "Section 1", section1Content),
        mdy.NewState("section2", "Section 2", section2Content),
    }).
    Options(mdy.AccordionOptions{
        AllowMultiple: false,
        DefaultOpen:   "",
    }).
    Build()

// Or with HTMX:
b.Div(mi.Class("accordion"),
    b.Button(
        mi.HtmxGet("/section/1"),
        mi.HtmxTarget("#section-1-content"),
        mi.HtmxSwap("innerHTML"),
        "Section 1",
    ),
    b.Div(mi.ID("section-1-content")),
)`
}

func generateToggleMinty(stateName string) string {
	return `// Simple HTMX toggle:
b.Button(
    mi.HtmxPost("/toggle-` + stateName + `"),
    mi.HtmxSwap("outerHTML"),
    mi.Class("toggle-btn"),
    "Toggle",
)

// Handler returns updated button state:
// POST /toggle-` + stateName + ` → returns button HTML with updated state`
}

func generatePaginationMinty(stateName string) string {
	return `mdy.Dyn("list").
    Data(mdy.FilterableDataset{
        Items: items,
        Options: mdy.FilterOptions{
            EnablePagination: true,
            ItemsPerPage:     20,
        },
    }).
    Build()

// Or HTMX pagination:
b.Div(mi.ID("pagination"),
    b.Button(
        mi.HtmxGet("/items?page=1"),
        mi.HtmxTarget("#item-list"),
        "Previous",
    ),
    b.Span("Page 1 of 10"),
    b.Button(
        mi.HtmxGet("/items?page=2"),
        mi.HtmxTarget("#item-list"),
        "Next",
    ),
)`
}

func generateSortableMinty(stateName string) string {
	return `mdy.Dyn("table").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            SortableFields: []string{"name", "date", "status"},
        },
        Options: mdy.FilterOptions{
            EnableSort:       true,
            DefaultSortField: "name",
            DefaultSortDir:   mdy.SortAsc,
        },
    }).
    Build()

// Or HTMX sortable headers:
b.Th(
    mi.HtmxGet("/items?sort=name&dir=asc"),
    mi.HtmxTarget("#table-body"),
    "Name ↑",
)`
}
