# reminty

Express UI ideas in JSX. Ship them in Go.

reminty converts React/JSX components to Go + [minty](https://github.com/ha1tch/minty), letting web developers be productive immediately while learning Go naturally through the code they ship.

## Why

Go teams building web applications often face a choice: split into frontend/backend camps, or have everyone learn both ecosystems. minty + HTMX offers a third path—full-stack Go—but the learning curve can slow initial productivity.

reminty eliminates that curve. Write what you know (JSX), see how it maps to Go, ship working code. By the time the patterns are internalised, you're writing minty directly.

## Installation

```bash
go install github.com/ha1tch/reminty/cmd/reminty@latest
```

## Usage

```bash
reminty Component.jsx              # Convert to stdout
reminty -o component.go App.jsx    # Convert to file
reminty -analyze Component.jsx     # Pattern analysis only
reminty -verbose Component.jsx     # Full analysis + code
```

## Example

**Input (React):**
```jsx
function PostCard({ post }) {
  return (
    <article className={`card ${post.status}`}>
      <h2>{post.title}</h2>
      {post.category && <span className="tag">{post.category}</span>}
      {post.likes > 0 && <span>{post.likes} likes</span>}
    </article>
  );
}
```

**Output (Go + minty):**
```go
func PostCard(post map[string]interface{}) mi.H {
    return func(b *mi.Builder) mi.Node {
        return b.Article(mi.Class(fmt.Sprintf("card %v", mi.Str(post, "status"))),
            b.H2(mi.Str(post, "title")),
            mi.If(mi.Truthy(post["category"]), func(b *mi.Builder) mi.Node {
                return b.Span(mi.Class("tag"), mi.Str(post, "category"))
            }),
            mi.If(mi.Gt(post, "likes", 0), func(b *mi.Builder) mi.Node {
                return b.Span(mi.Str(post, "likes"), "likes")
            }),
        )
    }
}
```

The output **compiles immediately**. Items needing attention are marked `/* TODO */`.

## What Translates

| React | Go + minty |
|-------|------------|
| JSX elements | Builder calls (`b.Div`, `b.Span`) |
| `className` | `mi.Class()` |
| `{post.title}` | `mi.Str(post, "title")` |
| `{post.likes > 0 && ...}` | `mi.If(mi.Gt(post, "likes", 0), ...)` |
| `{cond ? a : b}` | `mi.IfElse(cond, a, b)` |
| `items.map(...)` | `mi.Each(items, ...)` |
| `items.length` | `len(items)` |
| Template literals | `fmt.Sprintf()` |
| Event handlers | HTMX attributes + TODO |

## What Needs Manual Work

- **Local computed variables:** `const isPublished = status === 'published'` → TODO
- **Filter predicates:** Domain logic you'd write anyway
- **HTTP handlers:** HTMX endpoints for interactivity

These are the parts that benefit from human judgment.

## The Learning Path

**Week 1:** Write JSX, run reminty, ship Go. Start predicting the output.

**Week 2:** Write simple components directly in minty. Use reminty for complex layouts.

**Week 3:** JSX becomes optional—a sketch pad for thinking through UI.

**Month 2:** Minty-native. The training wheels come off.

## Requirements

- Go 1.22+
- [minty](https://github.com/ha1tch/minty) with helpers (`mi.Str`, `mi.Truthy`, etc.)

## Documentation

See [MANUAL.md](MANUAL.md) for the complete translation reference, architectural notes, and HTMX patterns.

## License

Apache 2.0
