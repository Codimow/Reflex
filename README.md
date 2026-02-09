# Reflex ⚡

**The last dev server you'll ever need.**

Reflex watches your files and automatically restarts commands when changes are detected. No more manual restarts. No more context switching. Just code.

## Installation

```bash
go install github.com/Codimow/Reflex/cmd/reflex@latest
```

## Usage

```bash
reflex "npm run dev"
```

That's it. Reflex watches your project and restarts the command whenever files change.

### Examples

```bash
# Run a Node.js dev server
reflex "npm run dev"

# Run a Go application
reflex "go run ."

# Run tests on change
reflex "npm test"

# Custom command with arguments
reflex "python -m flask run --debug"
```

## Features

- **🔄 Automatic Restart** — Detects file changes and restarts your command instantly
- **👀 Smart File Watching** — Watches only relevant files, ignores noise
- **🖥️ Beautiful TUI** — Clean terminal interface with real-time output
- **⚡ Fast** — Minimal overhead, maximum productivity
- **🎯 Process Management** — Graceful shutdown and restart handling
- **📁 Recursive Watching** — Monitors your entire project tree
- **🚫 Debouncing** — Prevents restart storms from rapid saves

## Default Watched Extensions

Reflex watches these file types by default:

| Category | Extensions |
|----------|------------|
| JavaScript | `.js`, `.jsx` |
| TypeScript | `.ts`, `.tsx` |
| Styles | `.css`, `.scss` |
| Data | `.json` |
| Markup | `.html`, `.mdx` |
| Frameworks | `.vue`, `.svelte` |

## Configuration

### Watch Specific Extensions

```bash
reflex --ext ".go,.mod" "go run ."
```

### Ignore Patterns

```bash
reflex --ignore "node_modules,dist,.git" "npm run dev"
```

### Custom Delay

```bash
reflex --delay 500ms "npm run dev"
```

## Why Reflex?

| Feature | Reflex | nodemon | watchexec |
|---------|--------|---------|-----------|
| TUI Interface | ✅ | ❌ | ❌ |
| Zero Config | ✅ | ⚠️ | ⚠️ |
| Cross-platform | ✅ | ✅ | ✅ |
| Language Agnostic | ✅ | ⚠️ | ✅ |

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT © [Codimow](https://github.com/Codimow)
