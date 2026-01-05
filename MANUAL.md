# reminty Manual

## The Fundamental Problem

React and Go+minty represent fundamentally different approaches to web development:

| Aspect | React | Go + minty |
|--------|-------|------------|
| Execution | Client-side JavaScript | Server-side Go |
| State | In-browser, component-local | Server-managed, request-scoped |
| Rendering | Virtual DOM diffing | Direct HTML generation |
| Interactivity | JavaScript event handlers | HTMX + server round-trips |
| Updates | Re-render on state change | Full or partial page reload |

**There is no 1:1 translation.** React components are programs that run in the browser and maintain state across interactions. Minty components are functions that produce HTML once per request.

reminty helps bridge this gap by:
1. Converting what can be converted (structure, expressions, type-safe access)
2. Providing scaffolding for what needs rethinking (state, events)
3. Suggesting minty/mintydyn patterns where they apply

---

## What Translates Directly

### JSX Structure → Builder Calls

JSX elements map to minty builder methods:

```jsx
// React
<div className="container">
  <h1>Title</h1>
  <p>Content</p>
</div>
```

```go
// minty
b.Div(mi.Class("container"),
    b.H1("Title"),
    b.P("Content"),
)
```

### Props → Function Parameters

React component props become Go function parameters with intelligent type inference:

```jsx
function Button({ text, variant, disabled, count, items, user }) {
  // ...
}
```

```go
func Button(text string, variant string, disabled bool, count int, items []interface{}, user map[string]interface{}) mi.H {
    // ...
}
```

**Type inference rules:**
- Names starting with `is`, `has`, `show`, `can` → `bool`
- Names like `count`, `index`, `num`, `size` → `int`
- Plural names (`items`, `users`, `posts`) → `[]interface{}`
- Singular object names (`user`, `post`, `item`, `task`) → `map[string]interface{}`
- Everything else → `string`

### Object Property Access → Type-Safe Helpers

reminty uses minty's helper functions for safe property access on objects:

```jsx
// React
<h1>{post.title}</h1>
<span>{post.views} views</span>
<div className={`card ${post.status}`}>
```

```go
// minty
b.H1(mi.Str(post, "title"))
b.Span(mi.Str(post, "views"), "views")
b.Div(mi.Class(fmt.Sprintf("card %v", mi.Str(post, "status"))))
```

**Available helpers:**
| Helper | Purpose | Example |
|--------|---------|---------|
| `mi.Str(m, "key")` | Safe string access | `mi.Str(post, "title")` |
| `mi.Int(m, "key")` | Safe int access | `mi.Int(post, "views")` |
| `mi.Bool(m, "key")` | Safe bool access | `mi.Bool(post, "active")` |
| `mi.Float(m, "key")` | Safe float64 access | `mi.Float(product, "price")` |

### Truthy Checks → mi.Truthy

JavaScript's truthy evaluation translates to `mi.Truthy()`:

```jsx
// React
{post.category && <span>{post.category}</span>}
{user.avatar && <img src={user.avatar} />}
```

```go
// minty
mi.If(mi.Truthy(post["category"]), func(b *mi.Builder) mi.Node {
    return b.Span(mi.Str(post, "category"))
})
mi.If(mi.Truthy(user["avatar"]), func(b *mi.Builder) mi.Node {
    return b.Img(mi.Src(mi.Str(user, "avatar")))
})
```

`mi.Truthy()` returns false for: `nil`, `false`, `0`, `0.0`, `""`, empty slices.

### Numeric Comparisons → mi.Gt, mi.Lt, etc.

```jsx
// React
{post.likes > 0 && <span>{post.likes} likes</span>}
{item.stock >= 10 && <span>In stock</span>}
```

```go
// minty
mi.If(mi.Gt(post, "likes", 0), func(b *mi.Builder) mi.Node {
    return b.Span(mi.Str(post, "likes"), "likes")
})
mi.If(mi.Gte(item, "stock", 10), func(b *mi.Builder) mi.Node {
    return b.Span("In stock")
})
```

**Comparison helpers:**
- `mi.Gt(m, "key", n)` — greater than
- `mi.Gte(m, "key", n)` — greater than or equal
- `mi.Lt(m, "key", n)` — less than
- `mi.Lte(m, "key", n)` — less than or equal
- `mi.Eq(m, "key", "val")` — string equality
- `mi.Ne(m, "key", "val")` — string inequality

### Template Literals → fmt.Sprintf

```jsx
// React
<div className={`card card-${variant}`}>
<p>{`Showing ${count} of ${total}`}</p>
```

```go
// minty
b.Div(mi.Class(fmt.Sprintf("card card-%v", variant)))
b.P(fmt.Sprintf("Showing %v of %v", count, total))
```

With object properties:
```jsx
<div className={`post-card ${post.status}`}>
```

```go
b.Div(mi.Class(fmt.Sprintf("post-card %v", mi.Str(post, "status"))))
```

### Ternary in Attributes → Inline Functions

```jsx
// React
<div className={isActive ? "card active" : "card"}>
<button className={filter === 'all' ? 'btn active' : 'btn'}>
```

```go
// minty
b.Div(mi.Class(func() string { if isActive { return "card active" }; return "card" }()))
b.Button(mi.Class(func() string { if filter == "all" { return "btn active" }; return "btn" }()))
```

### Array Length → len()

```jsx
// React
{items.length > 0 && <List items={items} />}
<p>Showing {filtered.length} of {items.length}</p>
```

```go
// minty
mi.If(len(items) > 0, List(items))
b.P("Showing", len(filtered), "of", len(items))
```

### Comparisons

```jsx
// React
{activeTab === 'home' && <Home />}
{status !== 'draft' && <Published />}
```

```go
// minty
mi.If(activeTab == "home", Home())
mi.If(status != "draft", Published())
```

### .map() Iterations

```jsx
// React
{posts.map(post => (
  <PostCard key={post.id} post={post} />
))}
```

```go
// minty
mi.Each(posts, func(postVal interface{}) mi.H {
    post := postVal.(map[string]interface{})
    return PostCard(post)
})
```

**Notes:**
- `key` prop is automatically filtered (React-specific)
- Component calls are detected and generated correctly
- Type assertion comment suggests using your own struct type

### Conditionals

```jsx
// React: logical AND
{isVisible && <Modal />}

// React: ternary
{isActive ? <Active /> : <Inactive />}
```

```go
// minty
mi.If(isVisible, Modal())
mi.IfElse(isActive, Active(), Inactive())
```

### Map in Ternary

```jsx
// React
{items.length > 0
  ? items.map(item => <Card item={item} />)
  : <p>No items</p>
}
```

```go
// minty
mi.IfElse(len(items) > 0,
    func(b *mi.Builder) mi.Node {
        nodes := mi.Each(items, func(itemVal interface{}) mi.H {
            item := itemVal.(map[string]interface{})
            return Card(item)
        })
        children := make([]interface{}, len(nodes))
        for i, n := range nodes { children[i] = n }
        return b.Div(children...)
    },
    func(b *mi.Builder) mi.Node {
        return b.P("No items")
    },
)
```

---

## What Doesn't Translate (and Why)

### useState

**React:**
```jsx
const [count, setCount] = useState(0);
```

**Why it doesn't translate:** `useState` creates persistent state that survives re-renders. In server-side rendering, there's no persistent state between requests.

**reminty's solution:**

1. **Converts state to function parameters:**
   ```go
   func Counter(count int) mi.H { ... }
   ```

2. **Adds guidance comments:**
   ```go
   // State converted to parameters. Original setters:
   //   setCount → use HTMX to update count parameter
   ```

3. **For event handlers, generates HTMX:**
   ```go
   b.Button(mi.HtmxPost("/update-count"), mi.HtmxSwap("outerHTML"), "+")
   ```

**Your responsibility:** 
- Store state server-side (session, database, URL params)
- Create HTTP handlers that update state and return new HTML

### Local Computed Variables

**React:**
```jsx
const isPublished = post.status === 'published';
const isDraft = post.status === 'draft';
{isPublished && <ViewButton />}
```

**Why it doesn't fully translate:** These are local scope bindings that reference other expressions. reminty doesn't track local variable definitions.

**reminty's solution:** Generates TODO placeholder:
```go
mi.If(false /* TODO: isPublished */, func(b *mi.Builder) mi.Node {
    return b.Button("View")
})
```

**Your fix:**
```go
isPublished := mi.Eq(post, "status", "published")
mi.If(isPublished, func(b *mi.Builder) mi.Node {
    return b.Button("View")
})
```

### Filter/Reduce Operations

**React:**
```jsx
const published = posts.filter(p => p.status === 'published');
const totalViews = posts.reduce((sum, p) => sum + p.views, 0);
```

**Why it partially translates:** The concept exists in both worlds, but JavaScript arrow function predicates can't be automatically converted to Go.

**reminty's solution:** Generates scaffolding:
```go
var published []interface{} // TODO: implement filter
for _, item := range posts {
    // TODO: add filter condition
    // Original: posts.filter(p => p.status === 'published')
    published = append(published, item)
}

var totalViews interface{} // TODO: implement reduce from posts
```

**Your fix using minty helpers:**
```go
published := mi.FilterItems(posts, mi.Where("status", "published"))
totalViews := mi.Sum(posts, "views")
```

### Event Handlers

**React:**
```jsx
<button onClick={() => setCount(count + 1)}>+</button>
<input onChange={(e) => setFilter(e.target.value)} />
```

**Why they don't translate:** JavaScript event handlers run in the browser. Go code runs on the server.

**reminty's solution:** Generates HTMX with TODO:
```go
b.Button(mi.HtmxPost("/click-action") /* TODO: () => setCount(count + 1) */, "+")
```

**Your responsibility:** Create HTTP handlers and wire up HTMX.

---

## Minty Helper Functions Reference

These helpers are used by reminty and available for manual use:

### Type-Safe Map Access
```go
mi.Str(m, "key")    // Safe string, returns "" if missing
mi.Int(m, "key")    // Safe int, returns 0 if missing
mi.Float(m, "key")  // Safe float64
mi.Bool(m, "key")   // Safe bool, returns false if missing
```

### Truthy/Conditional
```go
mi.Truthy(v)                    // JavaScript-like truthy check
mi.IfTruthy(v, thenNode)        // Render if truthy
mi.IfTruthyElse(v, then, else)  // Render one or the other
```

### Comparisons
```go
mi.Eq(m, "key", "val")   // String equality
mi.Ne(m, "key", "val")   // String inequality
mi.Gt(m, "key", n)       // Greater than (int)
mi.Gte(m, "key", n)      // Greater or equal
mi.Lt(m, "key", n)       // Less than
mi.Lte(m, "key", n)      // Less or equal
mi.Contains(m, "key", "substr")   // String contains
mi.ContainsI(m, "key", "substr")  // Case-insensitive
```

### Collection Operations
```go
mi.FilterItems(items, pred)  // Filter by predicate
mi.Count(items, pred)        // Count matching
mi.Sum(items, "key")         // Sum int field
mi.SumFloat(items, "key")    // Sum float field
mi.Avg(items, "key")         // Average
mi.Find(items, pred)         // First match
mi.Any(items, pred)          // True if any match
mi.All(items, pred)          // True if all match
```

### Predicates
```go
mi.Where("key", "val")       // Field equals value
mi.WhereNot("key", "val")    // Field not equals
mi.WhereGt("key", n)         // Field > n
mi.WhereTruthy("key")        // Field is truthy
mi.WhereContains("key", "s") // Field contains string
mi.And(p1, p2, ...)          // All predicates true
mi.Or(p1, p2, ...)           // Any predicate true
```

### Sorting
```go
mi.SortBy(items, "key", mi.Asc)   // Sort ascending
mi.SortBy(items, "key", mi.Desc)  // Sort descending
mi.SortByMulti(items,             // Multi-field sort
    mi.SortField{"status", mi.Asc},
    mi.SortField{"date", mi.Desc},
)
```

### Utilities
```go
mi.Pluck(items, "key")       // Extract field as []string
mi.PluckInt(items, "key")    // Extract field as []int
mi.GroupItems(items, "key")  // Group by field value
mi.ToMap(v)                  // Safe cast to map
mi.ToSlice(v)                // Safe cast to slice
```

---

## Pattern Solutions

reminty detects common React patterns and suggests minty/mintydyn equivalents.

### Tabs

**React:**
```jsx
const [activeTab, setActiveTab] = useState('overview');
```

**Suggestion:**
```go
mdy.Dyn("tabs").
    States([]mdy.ComponentState{
        mdy.ActiveState("overview", "Overview", overviewContent),
        mdy.NewState("details", "Details", detailsContent),
    }).
    Build()
```

### Filter/Search

**React:**
```jsx
const [search, setSearch] = useState('');
const filtered = items.filter(item => item.name.includes(search));
```

**Suggestion:**
```go
// Using minty helpers
filtered := mi.FilterItems(items, mi.WhereContainsI("name", search))

// Or mintydyn for full UI
mdy.Dyn("filter").
    Data(mdy.FilterableDataset{
        Items: items,
        Schema: mdy.FilterSchema{
            Fields: []mdy.FilterableField{
                mdy.TextField("search", "Search"),
            },
        },
    }).
    Build()
```

### Modal

**React:**
```jsx
const [isOpen, setIsOpen] = useState(false);
{isOpen && <Modal onClose={() => setIsOpen(false)} />}
```

**HTMX solution:**
```go
// Trigger
b.Button(mi.HtmxGet("/modal-content"), mi.HtmxTarget("#modal"), "Open")

// Container
b.Div(mi.ID("modal"))

// Close (in modal content)
b.Button(mi.HtmxDelete("/modal"), mi.HtmxTarget("#modal"), mi.HtmxSwap("innerHTML"), "Close")
```

### Dark Mode

**React:**
```jsx
const [isDark, setIsDark] = useState(false);
```

**Minty solution:**
```go
darkMode := mi.DarkModeTailwind(
    mi.DarkModeDefault("system"),
    mi.DarkModeSVGIcons(),
)
darkMode.Script(b) // In <head>
darkMode.Toggle(b) // Toggle button
```

---

## Migration Strategy

### 1. Start with Pure Components

Begin with components that have no state or effects—just props in, JSX out. These translate cleanly.

### 2. Replace Interface Types

After translation, replace `map[string]interface{}` with proper Go structs:

```go
// Generated
func PostCard(post map[string]interface{}) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.H1(mi.Str(post, "title"))
    }
}

// Improved
type Post struct {
    Title  string
    Status string
    Views  int
}

func PostCard(post Post) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.H1(post.Title)
    }
}
```

### 3. Fill in Filter/Sort Logic

Replace TODO scaffolding with minty helpers:

```go
// Generated
var published []interface{} // TODO: implement filter
for _, item := range posts {
    // TODO: add filter condition
    published = append(published, item)
}

// Fixed
published := mi.FilterItems(posts, mi.Where("status", "published"))
```

### 4. Wire Up HTMX

For each `/* TODO: handler */` comment:
1. Create an HTTP handler
2. Return the updated HTML fragment
3. Configure HTMX target and swap strategy

---

## Limitations

reminty does not handle:

- **TypeScript types:** Stripped during parsing
- **CSS-in-JS:** styled-components, emotion ignored
- **Higher-order components:** `withRouter(Component)` patterns
- **Render props:** `<DataProvider render={data => ...} />`
- **Portals:** `ReactDOM.createPortal`
- **Refs:** `useRef` (different paradigm)
- **Suspense/lazy loading:** Client-side code splitting
- **Error boundaries:** Different error handling model
- **Local const bindings:** `const x = expr` inside components

For these patterns, manual conversion is required.

---

## Command Reference

```bash
reminty [options] <input.jsx>

Options:
  -o, --output <file>   Write to file (default: stdout)
  -analyze              Pattern analysis only, no code
  -verbose              Show analysis + code
  -v, --version         Version info
  -h, --help            This help

Examples:
  reminty Component.jsx                   # Convert, print to stdout
  reminty -o out.go Component.jsx         # Convert to file
  reminty -analyze Component.jsx          # Analyze patterns only
  reminty -verbose Component.jsx          # Full analysis + code
  cat Component.jsx | reminty             # Read from stdin
```
