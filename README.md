# reminty

Convert React/JSX components to Go + minty.

## Status

**Alpha** — works for simple components, more complex patterns need refinement.

## Installation

```bash
go install github.com/ha1tch/reminty/cmd/reminty@latest
```

Or build from source:

```bash
git clone https://github.com/ha1tch/reminty
cd reminty
go build ./cmd/reminty
```

## Usage

```bash
# Convert a file
reminty Component.jsx

# Output to file
reminty -o component.go Component.jsx

# Analyze patterns only (no code generation)
reminty -analyze Component.jsx

# Verbose output
reminty -verbose Component.jsx

# From stdin
cat Component.jsx | reminty
```

## What It Does

### Converts JSX Structure

```jsx
// React
<div className="card">
  <h1>{title}</h1>
  <p>{description}</p>
</div>
```

```go
// minty
b.Div(mi.Class("card"),
    b.H1(title),
    b.P(description),
)
```

### Maps .map() to mi.Each()

```jsx
// React
{items.map(item => (
  <li key={item.id}>{item.name}</li>
))}
```

```go
// minty
mi.Each(items, func(item Item) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.Li(item.Name)
    }
})
```

### Converts Conditionals

```jsx
// React: {condition && <Element/>}
// minty: mi.If(condition, element)

// React: {condition ? <A/> : <B/>}
// minty: mi.IfElse(condition, a, b)
```

### Detects Patterns and Suggests mintydyn

When the tool detects React patterns, it suggests minty equivalents:

| React Pattern | mintydyn Suggestion |
|--------------|---------------------|
| Tab state + UI | `mdy.Dyn("tabs").States(...)` |
| Filter/search state | `mdy.Dyn("filter").Data(...)` |
| Form field dependencies | `mdy.Dyn("form").Rules(...)` |
| Dark mode toggle | `mi.DarkModeTailwind(...)` |
| Modal state | HTMX modal pattern |
| Pagination state | `mdy.FilterOptions{EnablePagination: true}` |

### Flags Hooks for Manual Conversion

```
// TODO: This component uses React hooks that need manual conversion:
//   - useState (line 5)
//   - useEffect (line 8)
```

With suggestions:

```
// Line 5: useState
//   → Consider: server state, mintydyn State, or HTMX pattern
// Line 8: useEffect
//   → Consider: server-side logic, OnInit hook, or HTMX trigger
```

## What It Handles

✓ JSX element structure → minty builder calls  
✓ className → mi.Class()  
✓ Common attributes (href, src, id, etc.)  
✓ Data attributes  
✓ HTMX attributes (hx-get, hx-post, etc.)  
✓ {items.map(...)} → mi.Each()  
✓ {condition && <X/>} → mi.If()  
✓ {cond ? <A/> : <B/>} → mi.IfElse()  
✓ Component props → function parameters  
✓ Import statements (parsed, noted)  
✓ Hook detection with migration suggestions  
✓ UI pattern detection with mintydyn suggestions  

## What It Flags (TODO Comments)

⚠ useState → suggests server state, mintydyn, or HTMX  
⚠ useEffect → suggests server-side logic or OnInit hook  
⚠ useContext → suggests Go context or function params  
⚠ useReducer → suggests mintydyn Rules  
⚠ useMemo/useCallback → notes these are unnecessary server-side  
⚠ Spread attributes {...props}  
⚠ Event handlers (onClick, onChange) → suggests HTMX  

## What It Doesn't Handle

✗ Complex hook logic (useReducer with complex state)  
✗ Third-party components (Material UI, Chakra, etc.)  
✗ CSS-in-JS (styled-components, emotion)  
✗ Dynamic imports  
✗ React Context with complex providers  
✗ Higher-order components  
✗ Render props patterns  
✗ TypeScript types (stripped, not converted)  

## Examples

### Simple Component

**Input (Button.jsx):**
```jsx
function Button({ text, variant }) {
  return (
    <button className={`btn btn-${variant}`}>
      {text}
    </button>
  );
}
```

**Output:**
```go
// Button component
func Button(text string, variant string) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.Button(mi.Class(/* TODO: template literal */),
            text,
        )
    }
}
```

### Component with Hooks

**Input (Counter.jsx):**
```jsx
function Counter() {
  const [count, setCount] = useState(0);
  
  return (
    <div>
      <p>Count: {count}</p>
      <button onClick={() => setCount(count + 1)}>
        Increment
      </button>
    </div>
  );
}
```

**Output:**
```go
// Counter component
// TODO: This component uses React hooks that need manual conversion:
//   - useState (line 2)
func Counter() mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.Div(
            b.P("Count: ", count),
            b.Button(mi.Attr("onClick", /* TODO */),
                "Increment",
            ),
        )
    }
}

// =============================================================================
// TRANSLATION NOTES
// =============================================================================
// Line 2: useState
//   → Consider: server state, mintydyn State, or HTMX pattern
```

**Suggested minty approach:**
```go
// Server-side counter with HTMX
func Counter(count int) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.Div(
            b.P(fmt.Sprintf("Count: %d", count)),
            b.Button(
                mi.HtmxPost("/increment"),
                mi.HtmxTarget("#counter"),
                mi.HtmxSwap("outerHTML"),
                "Increment",
            ),
        )
    }
}
```

## Options

```
-o, --output <file>   Write output to file (default: stdout)
-analyze              Only analyze patterns, don't generate code
-verbose              Show detailed analysis
-v, --version         Show version
-h, --help            Show help
```

## Philosophy

reminty is not a full transpiler. React and Go+minty are fundamentally different paradigms:

- React: Client-side state, virtual DOM, JavaScript runtime
- minty: Server-side rendering, real DOM, Go compilation

The tool helps you:
1. Convert JSX structure (the easy part)
2. Identify patterns that need rethinking (the hard part)
3. Suggest minty/mintydyn equivalents where they exist

The goal is to give you a starting point, not a finished product. You'll need to think about where state belongs, how interactions should work, and which mintydyn patterns apply.

## License

Same as minty (Apache 2.0)
